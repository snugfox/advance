package advance

import (
	"time"
)

type AdvanceState struct {
	// Timings
	Start                  time.Time
	LastUpdate, NextUpdate time.Time
	DeltaUpdate            time.Duration

	// Progress and total
	Progress, Total           int64 // Current
	LastProgress, LastTotal   int64 // Previous
	DeltaProgress, DeltaTotal int64 // Delta
}

func (as *AdvanceState) update(progress, total int64, refreshInterval time.Duration) {
	now := time.Now()

	// Update the state, only allowing one thread in the critical section.
	// Since the critical section is is likely significantly shorter than the
	// refresh interval, failure the acquire the lock will simply return
	// (non-blocking).
	as.DeltaProgress = progress - as.Progress
	as.LastProgress = as.Progress
	as.Progress = progress
	as.DeltaTotal = total - as.Total
	as.LastTotal = as.Total
	as.Total = total

	as.DeltaUpdate = now.Sub(as.LastUpdate)
	as.LastUpdate = now
	as.NextUpdate = now.Add(refreshInterval)
}

func (as *AdvanceState) reset(refreshInterval time.Duration) {
	as.Progress = 0
	as.LastProgress = 0
	as.DeltaProgress = 0
	as.Total = 0
	as.LastTotal = 0
	as.DeltaTotal = 0

	as.Start = time.Now()
	as.LastUpdate = as.Start
	as.DeltaUpdate = 0
	as.NextUpdate = as.Start.Add(refreshInterval)
}
