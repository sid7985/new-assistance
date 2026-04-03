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
			callback()
		}
	}()
}
