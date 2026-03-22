package scheduler

import (
	"context"
	"errors"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"ai-gateway/metrics"
	"ai-gateway/provider"
)

var ErrNoHealthyProvider = errors.New("no healthy provider available")

type Config struct {
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration
}

type ProviderState struct {
	Name         string    `json:"name"`
	Healthy      bool      `json:"healthy"`
	AvgLatencyMS float64   `json:"avg_latency_ms"`
	LastChecked  time.Time `json:"last_checked"`
	LastError    string    `json:"last_error,omitempty"`
}

type Scheduler struct {
	registry *provider.Registry
	config   Config
	metrics  *metrics.Collector

	mu      sync.RWMutex
	states  map[string]*ProviderState
	counter uint64
}

func New(registry *provider.Registry, cfg Config, collector *metrics.Collector) *Scheduler {
	states := make(map[string]*ProviderState)
	for _, item := range registry.All() {
		states[item.Name()] = &ProviderState{
			Name:         item.Name(),
			Healthy:      true,
			AvgLatencyMS: 0,
		}
	}

	return &Scheduler{
		registry: registry,
		config:   cfg,
		metrics:  collector,
		states:   states,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.runHealthChecks(ctx)

	ticker := time.NewTicker(s.config.HealthCheckInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runHealthChecks(ctx)
			}
		}
	}()
}

func (s *Scheduler) Select(model string) (provider.Provider, error) {
	candidates := s.registry.ProvidersForModel(model)
	if len(candidates) == 0 {
		return nil, provider.ErrModelNotSupported
	}

	s.mu.RLock()
	healthy := make([]provider.Provider, 0, len(candidates))
	for _, candidate := range candidates {
		state, ok := s.states[candidate.Name()]
		if !ok || state.Healthy {
			healthy = append(healthy, candidate)
		}
	}
	s.mu.RUnlock()

	if len(healthy) == 0 {
		return nil, ErrNoHealthyProvider
	}

	type scoredProvider struct {
		item    provider.Provider
		latency float64
	}

	scored := make([]scoredProvider, 0, len(healthy))
	s.mu.RLock()
	for _, candidate := range healthy {
		latency := math.MaxFloat64
		if state, ok := s.states[candidate.Name()]; ok && state.AvgLatencyMS > 0 {
			latency = state.AvgLatencyMS
		}
		scored = append(scored, scoredProvider{item: candidate, latency: latency})
	}
	s.mu.RUnlock()

	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].latency < scored[j].latency
	})

	bestLatency := scored[0].latency
	window := make([]provider.Provider, 0, len(scored))
	for _, item := range scored {
		if item.latency == math.MaxFloat64 || bestLatency == math.MaxFloat64 || item.latency-bestLatency <= 20 {
			window = append(window, item.item)
		}
	}
	if len(window) == 0 {
		window = append(window, scored[0].item)
	}

	index := atomic.AddUint64(&s.counter, 1)
	return window[(index-1)%uint64(len(window))], nil
}

func (s *Scheduler) RecordResult(providerName string, latency time.Duration, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.states[providerName]
	if !ok {
		state = &ProviderState{Name: providerName}
		s.states[providerName] = state
	}

	state.LastChecked = time.Now()
	if success {
		state.Healthy = true
		state.LastError = ""
		latencyMS := float64(latency.Milliseconds())
		if latencyMS <= 0 {
			latencyMS = 1
		}
		if state.AvgLatencyMS == 0 {
			state.AvgLatencyMS = latencyMS
		} else {
			state.AvgLatencyMS = state.AvgLatencyMS*0.7 + latencyMS*0.3
		}
	} else {
		state.Healthy = false
		state.LastError = "last runtime call failed"
	}

	if s.metrics != nil {
		s.metrics.SetProviderHealth(providerName, state.Healthy)
		s.metrics.SetProviderLatency(providerName, state.AvgLatencyMS)
	}
}

func (s *Scheduler) Snapshot() []ProviderState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ProviderState, 0, len(s.states))
	for _, item := range s.states {
		result = append(result, *item)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

func (s *Scheduler) runHealthChecks(ctx context.Context) {
	var wg sync.WaitGroup
	for _, item := range s.registry.All() {
		wg.Add(1)
		go func(p provider.Provider) {
			defer wg.Done()

			checkCtx, cancel := context.WithTimeout(ctx, s.config.HealthCheckTimeout)
			defer cancel()

			status, err := p.HealthCheck(checkCtx)
			if err != nil && status == nil {
				status = &provider.HealthStatus{
					Healthy:      false,
					CheckedAt:    time.Now(),
					ErrorMessage: err.Error(),
				}
			}

			s.mu.Lock()
			state, ok := s.states[p.Name()]
			if !ok {
				state = &ProviderState{Name: p.Name()}
				s.states[p.Name()] = state
			}
			state.LastChecked = time.Now()
			if status != nil {
				state.Healthy = status.Healthy
				state.LastError = status.ErrorMessage
				if status.Latency > 0 {
					latencyMS := float64(status.Latency.Milliseconds())
					if state.AvgLatencyMS == 0 {
						state.AvgLatencyMS = latencyMS
					} else {
						state.AvgLatencyMS = state.AvgLatencyMS*0.7 + latencyMS*0.3
					}
				}
			}
			s.mu.Unlock()

			if s.metrics != nil {
				healthy := status != nil && status.Healthy
				latency := 0.0
				if status != nil && status.Latency > 0 {
					latency = float64(status.Latency.Milliseconds())
				}
				s.metrics.SetProviderHealth(p.Name(), healthy)
				s.metrics.SetProviderLatency(p.Name(), latency)
			}
		}(item)
	}

	wg.Wait()
}
