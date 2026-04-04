package orchestrator

import (
	"fmt"
	"time"
)

type Heartbeat struct {
	Interval time.Duration
	LastCheck time.Time
	TaskQueue []string // Placeholder for persistent task queue
}

func NewHeartbeat(interval time.Duration) *Heartbeat {
	return &Heartbeat{
		Interval: interval,
		LastCheck: time.Now(),
	}
}

func (h *Heartbeat) Start(callback func()) {
	fmt.Printf("💓 Heartbeat started with interval %v\n", h.Interval)
	ticker := time.NewTicker(h.Interval)
	go func() {
		for range ticker.C {
			h.LastCheck = time.Now()
			// Callback can handle the vision logic
			callback()
		}
	}()
}

// PerformAnalysis is a helper for proactive desktop checks (Paperclip/Claw style)
func (h *Heartbeat) PerformAnalysis(analyzer func() (string, error)) {
	result, err := analyzer()
	if err == nil && result != "" {
		fmt.Printf("💓 Proactive Analysis Result: %s\n", result)
	}
}
