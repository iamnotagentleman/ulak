package populator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"ulak/internal/config"
	"ulak/internal/models"
	"ulak/internal/store/message"

	log "github.com/sirupsen/logrus"
)

var (
	// Singleton populatorInstance
	populatorInstance *MessagePopulatorWorker
	once              sync.Once
)

type MessagePopulatorWorker struct {
	cfg                 config.MessageWorker
	store               message.MessageStore
	lastProcessedOffset atomic.Int64
	isProcessing        atomic.Bool

	cancel context.CancelFunc
	done   chan struct{}
	err    error
	mu     sync.Mutex
}

func GetMessagePopulator(cfg config.MessageWorker, store message.MessageStore) *MessagePopulatorWorker {
	once.Do(func() {
		populatorInstance = &MessagePopulatorWorker{
			cfg:   cfg,
			store: store,
			done:  make(chan struct{}),
		}
		log.Info("MessagePopulatorWorker singleton populatorInstance created")
	})
	return populatorInstance
}

func (p *MessagePopulatorWorker) GetOffset() int64 {
	return p.lastProcessedOffset.Load()
}

func (p *MessagePopulatorWorker) fetchPendingMessages(ctx context.Context, limit int) []*models.Message {
	status := models.StatusPending
	currentOffset := p.GetOffset()
	messages, err := p.store.ListByOffset(ctx, currentOffset, status, limit)

	if err != nil {
		// TODO add alert mechanism & sentry error catch
		log.WithError(err).Error("error listing messages")
	}

	return messages
}

func (p *MessagePopulatorWorker) sendMessage(ctx context.Context, messagesCh chan *models.Message, message *models.Message) error {
	timeout := time.Duration(p.cfg.MessageChannelSendTimeout) * time.Second
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case messagesCh <- message:
		return nil
	case <-timer.C:
		return fmt.Errorf("timeout sending message %s after %v", message.ID, timeout)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *MessagePopulatorWorker) populate(ctx context.Context, ticker *time.Ticker, messagesCh chan *models.Message) error {
	for {
		select {
		case <-ticker.C:
			log.Info("MessagePopulatorWorker checking for messages")
			// Skip tick if still processing previous batch
			if !p.isProcessing.CompareAndSwap(false, true) {
				log.Warn("skipping tick: still processing previous message batch")
				continue
			}
			fetchLimit := p.cfg.MaxRecordsPerRead

			// Process messages (fetch only what fits in available space)
			messages := p.fetchPendingMessages(ctx, fetchLimit)

			// Check channel occupancy and apply backpressure if needed
			messagesChCap := cap(messagesCh)
			messagesChLen := len(messagesCh)
			availableArea := messagesChCap - messagesChLen
			messageCount := len(messages)

			if messageCount > availableArea {
				log.WithFields(log.Fields{
					"channelCapacity": messagesChCap,
					"channelLength":   messagesChLen,
					"availableArea":   availableArea,
					"messageCount":    messageCount,
					"requestedFetch":  fetchLimit,
				}).Warn("channel near capacity, pausing populator")

				p.isProcessing.Store(false)
				continue
			}

			for _, msg := range messages {
				err := p.sendMessage(ctx, messagesCh, msg)

				if err != nil {
					log.WithError(err).Error("error processing message")
				}

			}

			if len(messages) > 0 {
				newOffset := messages[len(messages)-1].Offset
				p.lastProcessedOffset.Store(newOffset)
			}

			// Mark processing as complete
			p.isProcessing.Store(false)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *MessagePopulatorWorker) Start(parentCtx context.Context, ticker *time.Ticker, messagesCh chan *models.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		return fmt.Errorf("worker already running")
	}

	ctx, cancel := context.WithCancel(parentCtx)
	p.cancel = cancel
	p.done = make(chan struct{})
	p.err = nil

	log.Info("starting MessagePopulatorWorker")

	// Start the worker goroutine
	go func() {
		defer close(p.done)

		err := p.populate(ctx, ticker, messagesCh)

		p.mu.Lock()
		p.err = err
		p.cancel = nil
		p.mu.Unlock()

		if err != nil && !errors.Is(err, context.Canceled) {
			log.WithError(err).Error("MessagePopulatorWorker stopped with error")
		} else {
			log.Info("MessagePopulatorWorker stopped gracefully")
		}
	}()

	return nil
}

func (p *MessagePopulatorWorker) Stop(timeout time.Duration) error {
	p.mu.Lock()
	cancel := p.cancel
	done := p.done
	p.mu.Unlock()

	if cancel == nil {
		return fmt.Errorf("worker not running")
	}

	log.Info("stopping MessagePopulatorWorker")

	// Request shutdown
	cancel()

	// Wait for completion with timeout
	select {
	case <-done:
		log.Info("MessagePopulatorWorker stopped successfully")
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.err

	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for worker to stop after %v", timeout)
	}
}

func (p *MessagePopulatorWorker) Wait() error {
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

func (p *MessagePopulatorWorker) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cancel != nil
}
