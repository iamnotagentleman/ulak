package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"ulak/internal/config"
	"ulak/internal/enums"
	"ulak/internal/models"
	"ulak/internal/store/keyval"
	"ulak/internal/store/message"
	"ulak/pkg/notification"

	"golang.org/x/time/rate"

	log "github.com/sirupsen/logrus"
)

type MessageProcessorWorker struct {
	cfg                 config.MessageWorker
	kvStore             keyval.KeyValueStore
	msgStore            message.MessageStore
	notificationService notification.NotificationService
	// Metrics
	processedCount atomic.Int64
	failedCount    atomic.Int64

	// Lifecycle management
	cancel context.CancelFunc
	done   chan struct{}
	err    error
	mu     sync.Mutex
}

func NewMessageProcessor(cfg config.MessageWorker, kvStore keyval.KeyValueStore, msgStore message.MessageStore, notificationService notification.NotificationService) *MessageProcessorWorker {
	return &MessageProcessorWorker{
		cfg:                 cfg,
		kvStore:             kvStore,
		msgStore:            msgStore,
		notificationService: notificationService,
		done:                make(chan struct{}),
	}
}

// GetMetrics returns current processing metrics
func (p *MessageProcessorWorker) GetMetrics() (processed, failed int64) {
	return p.processedCount.Load(), p.failedCount.Load()
}

// processMessage handles individual message processing with error recovery
func (p *MessageProcessorWorker) processMessage(ctx context.Context, msg *models.Message) error {
	log.WithFields(log.Fields{
		"message_id": msg.ID,
		"offset":     msg.Offset,
	}).Debug("Processing message")

	// Begin database transaction
	tx := p.msgStore.GetDB().Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// transaction roll back on error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.WithFields(log.Fields{
				"message_id": msg.ID,
				"panic":      r,
			}).Error("Panic during message processing, transaction rolled back")
		}
	}()

	// Send notification Step
	input := notification.Input{
		To:      msg.To,
		Content: msg.Message,
	}

	resp, err := p.notificationService.SendNotification(ctx, input)
	if err != nil {
		tx.Rollback()
		log.WithFields(log.Fields{
			"message_id": msg.ID,
			"error":      err,
		}).Error("Failed to send notification, transaction rolled back")

		msg.Status = enums.StatusFailed
		newTx := p.msgStore.GetDB().Begin()
		if newTx.Error == nil {
			if updateErr := newTx.WithContext(ctx).Save(msg).Error; updateErr != nil {
				newTx.Rollback()
				log.WithFields(log.Fields{
					"message_id": msg.ID,
					"error":      updateErr,
				}).Error("Failed to update message status to FAILED")
			} else {
				newTx.Commit()
			}
		}

		return fmt.Errorf("notification failed: %w", err)
	}

	msg.Status = enums.StatusSent
	if err := tx.WithContext(ctx).Save(msg).Error; err != nil {
		tx.Rollback()
		log.WithFields(log.Fields{
			"message_id": msg.ID,
			"error":      err,
		}).Error("Failed to update message status, transaction rolled back")
		return fmt.Errorf("failed to update message: %w", err)
	}

	// Redis Data Write Step
	redisData := map[string]interface{}{
		"response":     resp,
		"processed_at": time.Now().Unix(),
	}

	redisJSON, err := json.Marshal(redisData)
	if err != nil {
		tx.Rollback()
		log.WithFields(log.Fields{
			"message_id": msg.ID,
			"error":      err,
		}).Error("Failed to marshal Redis data, transaction rolled back")

		msg.Status = enums.StatusFailed
		newTx := p.msgStore.GetDB().Begin()
		if newTx.Error == nil {
			if updateErr := newTx.WithContext(ctx).Save(msg).Error; updateErr != nil {
				newTx.Rollback()
				log.WithFields(log.Fields{
					"message_id": msg.ID,
					"error":      updateErr,
				}).Error("Failed to update message status to FAILED")
			} else {
				newTx.Commit()
			}
		}

		return fmt.Errorf("failed to marshal redis data: %w", err)
	}

	if err := p.kvStore.SetMessageDelivered(ctx, msg.ID.String(), string(redisJSON), p.cfg.RedisMessageTTLSeconds); err != nil {
		tx.Rollback()
		log.WithFields(log.Fields{
			"message_id": msg.ID,
			"error":      err,
		}).Error("Failed to store notification response in Redis, transaction rolled back")

		msg.Status = enums.StatusFailed
		newTx := p.msgStore.GetDB().Begin()
		if newTx.Error == nil {
			if updateErr := newTx.WithContext(ctx).Save(msg).Error; updateErr != nil {
				newTx.Rollback()
				log.WithFields(log.Fields{
					"message_id": msg.ID,
					"error":      updateErr,
				}).Error("Failed to update message status to FAILED")
			} else {
				newTx.Commit()
			}
		}

		return fmt.Errorf("failed to store in redis: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.WithFields(log.Fields{
			"message_id": msg.ID,
			"error":      err,
		}).Error("Failed to commit transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.WithFields(log.Fields{
		"message_id": msg.ID,
		"offset":     msg.Offset,
		"status":     msg.Status,
	}).Info("Message processed successfully")

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
