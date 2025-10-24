package processor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"ulak/internal/config"
	"ulak/internal/models"
	"ulak/internal/store/keyval"

	"golang.org/x/time/rate"

	log "github.com/sirupsen/logrus"
)

type MessageProcessorWorker struct {
	cfg     config.MessageWorker
	kvStore keyval.KeyValueStore

	// Metrics
	processedCount atomic.Int64
	failedCount    atomic.Int64

	// Lifecycle management
	cancel context.CancelFunc
	done   chan struct{}
	err    error
	mu     sync.Mutex
}

func NewMessageProcessor(cfg config.MessageWorker, store keyval.KeyValueStore) *MessageProcessorWorker {
	return &MessageProcessorWorker{
		cfg:     cfg,
		kvStore: store,
		done:    make(chan struct{}),
	}
}

// GetMetrics returns current processing metrics
func (p *MessageProcessorWorker) GetMetrics() (processed, failed int64) {
	return p.processedCount.Load(), p.failedCount.Load()
}

// processMessage handles individual message processing with error recovery
func (p *MessageProcessorWorker) processMessage(ctx context.Context, msg *models.Message) error {
	// TODO: Implement actual message processing logic
	log.WithFields(log.Fields{
		"message_id": msg.ID,
		"offset":     msg.Offset,
	}).Debug("Processing message")

	println(msg)

	return nil
}

func (p *MessageProcessorWorker) Process(ctx context.Context, messagesCh chan *models.Message) error {
	rateLimit := rate.Every(time.Minute / time.Duration(p.cfg.RateLimitPerMinute))
	throttle := rate.NewLimiter(rateLimit, p.cfg.RateLimitBurst)

	log.WithFields(log.Fields{
		"rate_limit_per_minute": p.cfg.RateLimitPerMinute,
		"burst":                 p.cfg.RateLimitBurst,
	}).Info("MessageProcessorWorker rate limiter configured")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case msg, ok := <-messagesCh:
			if !ok {
				// Channel closed, exit gracefully
				log.Info("Message channel closed, processor shutting down")
				return nil
			}

			// Apply rate limiting before processing
			if err := throttle.Wait(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					log.Info("Context cancelled during rate limit wait,")
					return err
				}
				log.WithError(err).Error("Rate limiter error")
				return err
			}

			// Process message with error recovery
			if err := p.processMessage(ctx, msg); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"message_id": msg.ID,
					"offset":     msg.Offset,
				}).Error("Failed to process message")
				p.failedCount.Add(1)
				// TODO: Add retry logic or dead letter queue
			} else {
				p.processedCount.Add(1)
			}
		}
	}
}

func (p *MessageProcessorWorker) Start(ctx context.Context, messagesCh chan *models.Message, wg *sync.WaitGroup) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		return fmt.Errorf("worker already running")
	}

	workerCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.done = make(chan struct{})
	p.err = nil

	log.Info("Starting MessageProcessorWorker")

	// Start the worker goroutine
	go func() {
		defer close(p.done)
		if wg != nil {
			defer wg.Done()
		}

		err := p.Process(workerCtx, messagesCh)

		p.mu.Lock()
		p.err = err
		p.cancel = nil
		p.mu.Unlock()

		if err != nil && !errors.Is(err, context.Canceled) {
			log.WithError(err).Error("MessageProcessorWorker stopped with error")
		} else {
			log.Info("MessageProcessorWorker stopped gracefully")
		}

		// Log final metrics
		processed, failed := p.GetMetrics()
		log.WithFields(log.Fields{
			"total_processed": processed,
			"total_failed":    failed,
		}).Info("MessageProcessorWorker final metrics")
	}()

	return nil
}

func (p *MessageProcessorWorker) Stop(timeout time.Duration) error {
	p.mu.Lock()
	cancel := p.cancel
	done := p.done
	p.mu.Unlock()

	if cancel == nil {
		return fmt.Errorf("worker not running")
	}

	log.Info("Stopping MessageProcessorWorker")

	// Request shutdown
	cancel()

	// Wait for completion with timeout
	select {
	case <-done:
		log.Info("MessageProcessorWorker stopped successfully")
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.err

	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for worker to stop after %v", timeout)
	}
}

func (p *MessageProcessorWorker) Wait() error {
	p.mu.Lock()
	done := p.done
	p.mu.Unlock()

	if done == nil {
		return fmt.Errorf("worker not started")
	}

	<-done

	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

func (p *MessageProcessorWorker) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cancel != nil
}
