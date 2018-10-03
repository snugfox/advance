package advance

import (
	"sync"
	"time"
)

type AdvanceState struct {
	resetLock sync.RWMutex
	nbLock    chan struct{}

	// Timings
	Start                  time.Time
	LastUpdate, NextUpdate time.Time
	DeltaUpdate            time.Duration
	refreshInterval        time.Duration

	// Progress and total
	Progress, Total           int64 // Current
	LastProgress, LastTotal   int64 // Previous
	DeltaProgress, DeltaTotal int64 // Delta
}

func newAdvanceState(refreshInterval time.Duration) *AdvanceState {
	as := AdvanceState{
		nbLock:          make(chan struct{}, 1),
		refreshInterval: refreshInterval,
	}
	as.nbLock <- struct{}{} // Unlocked by default
	as.Start = time.Now()
	as.LastUpdate = as.Start
	as.NextUpdate = as.Start.Add(refreshInterval)
	return &as
}

func (as *AdvanceState) requestUpdate(progress, total int64, force bool) bool {
	// Only process an update if the refresh duration has elapsed since the
	// previous refresh.
	now := time.Now()
	if force || now.After(as.NextUpdate) || now.Equal(as.NextUpdate) {
		if force {
			as.resetLock.Lock()
			defer as.resetLock.Unlock()
		} else {
			as.resetLock.RLock()
			defer as.resetLock.RUnlock()
		}

		// Update the state, only allowing one thread in the critical section.
		// Since the critical section is is likely significantly shorter than the
		// refresh interval, failure the acquire the lock will simply return
		// (non-blocking).
		select {
		case <-as.nbLock:
			as.DeltaProgress = progress - as.Progress
			as.LastProgress = as.Progress
			as.Progress = progress
			as.DeltaTotal = total - as.Total
			as.LastTotal = as.Total
			as.Total = total

			as.DeltaUpdate = now.Sub(as.LastUpdate)
			as.LastUpdate = now
			as.NextUpdate = now.Add(as.refreshInterval)

			as.nbLock <- struct{}{}
			return true
		default:
			break
		}
		return false
	}
	return false
}

func (as *AdvanceState) reset() {
	as.resetLock.Lock()
	defer as.resetLock.Unlock()

	// Reinitialize the state if no other thread is already (non-blocking)
	select {
	case <-as.nbLock:
		as.Progress = 0
		as.LastProgress = 0
		as.DeltaProgress = 0
		as.Total = 0
		as.LastTotal = 0
		as.DeltaTotal = 0

		as.Start = time.Now()
		as.LastUpdate = as.Start
		as.DeltaUpdate = 0
		as.NextUpdate = as.Start.Add(as.refreshInterval)

		as.nbLock <- struct{}{}
	default:
		break
	}
}
