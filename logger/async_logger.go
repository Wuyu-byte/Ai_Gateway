package logger

import (
	"context"
	"log"
	"sync"
	"time"

	"ai-gateway/metrics"
	"ai-gateway/model"
	"ai-gateway/repository"
)

type Config struct {
	QueueSize     int
	WorkerCount   int
	BatchSize     int
	FlushInterval time.Duration
}

type AsyncUsageLogger struct {
	repo    *repository.UsageRepository
	metrics *metrics.Collector
	config  Config
	queue   chan *model.UsageLog
	wg      sync.WaitGroup
}

func NewAsyncUsageLogger(repo *repository.UsageRepository, cfg Config, collector *metrics.Collector) *AsyncUsageLogger {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 1024
	}
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 2
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 20
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = time.Second
	}

	return &AsyncUsageLogger{
		repo:    repo,
		metrics: collector,
		config:  cfg,
		queue:   make(chan *model.UsageLog, cfg.QueueSize),
	}
}

func (l *AsyncUsageLogger) Start(ctx context.Context) {
	for i := 0; i < l.config.WorkerCount; i++ {
		l.wg.Add(1)
		go l.runWorker(ctx)
	}
}

func (l *AsyncUsageLogger) Stop() {
	close(l.queue)
	l.wg.Wait()
}

func (l *AsyncUsageLogger) Enqueue(usage *model.UsageLog) bool {
	select {
	case l.queue <- usage:
		return true
	default:
		if l.metrics != nil {
			l.metrics.AsyncLogDropped.Inc()
		}
		return false
	}
}

func (l *AsyncUsageLogger) runWorker(ctx context.Context) {
	defer l.wg.Done()

	ticker := time.NewTicker(l.config.FlushInterval)
	defer ticker.Stop()

	buffer := make([]*model.UsageLog, 0, l.config.BatchSize)
	flush := func() {
		if len(buffer) == 0 {
			return
		}

		if err := l.repo.BatchCreate(context.Background(), buffer); err != nil {
			log.Printf("async usage logger batch insert failed: %v", err)
		}
		buffer = buffer[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case usage, ok := <-l.queue:
			if !ok {
				flush()
				return
			}
			buffer = append(buffer, usage)
			if len(buffer) >= l.config.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}
