package manager

import (
	"context"
	"fmt"
	"sync"
	"time"
	"ulak/internal/config"
	"ulak/internal/models"
	"ulak/internal/worker/populator"
	"ulak/internal/worker/processor"

	log "github.com/sirupsen/logrus"
)

type WorkerManager struct {
	populator  *populator.MessagePopulatorWorker
	processor  *processor.MessageProcessorWorker
	messagesCh chan *models.Message
	ticker     *time.Ticker
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
	cfg        config.Server
}

func NewWorkerManager(
	ctx context.Context,
	populatorWorker *populator.MessagePopulatorWorker,
	processorWorker *processor.MessageProcessorWorker,
	messagesCh chan *models.Message,
	ticker *time.Ticker,
	cfg config.Server,
) *WorkerManager {
	workerCtx, cancel := context.WithCancel(ctx)

	return &WorkerManager{
		populator:  populatorWorker,
		processor:  processorWorker,
		messagesCh: messagesCh,
		ticker:     ticker,
		ctx:        workerCtx,
		cancel:     cancel,
		cfg:        cfg,
	}
}

func (m *WorkerManager) Start() error {
	if err := m.populator.Start(m.ctx, m.ticker, m.messagesCh); err != nil {
		return fmt.Errorf("failed to start populator worker: %w", err)
	}
	log.Println("Populator worker started")

	// Start processor worker
	if err := m.processor.Start(m.ctx, m.messagesCh, nil); err != nil {
		return fmt.Errorf("failed to start processor worker: %w", err)
	}
	log.Println("Processor worker started")

	return nil
}

func (m *WorkerManager) Stop() {
	// Stop populator worker first (stops fetching new messages)
	if err := m.populator.Stop(m.cfg.WorkerStopTimeout); err != nil {
		log.Printf("Error stopping populator worker: %v", err)
	} else {
		log.Println("Populator worker stopped successfully")
	}

	// Stop processor worker
	if err := m.processor.Stop(m.cfg.WorkerStopTimeout); err != nil {
		log.Printf("Error stopping processor worker: %v", err)
	} else {
		log.Println("Processor worker stopped successfully")
	}
}

func (m *WorkerManager) GetStatus() WorkerStatusResponse {
	return WorkerStatusResponse{
		IsPopulatorActive: m.populator.IsRunning(),
		IsProcessorActive: m.processor.IsRunning(),
	}
}
