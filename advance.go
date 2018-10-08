package advance

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"
	"time"

	escapes "github.com/snugfox/ansi-escapes"
)

type Advance struct {
	w        io.Writer
	buf      bytes.Buffer
	active   bool
	dispLock sync.Mutex

	state     AdvanceState
	stateLock *tryMutex

	// Sloppy counters
	nextProgress, nextTotal int64
	refreshInterval         time.Duration

	components []Component
	cIndexes   []int
}

func New(w io.Writer, refreshInterval time.Duration, components ...Component) *Advance {
	if len(components) == 0 {
		panic("Must have at least one component")
	}

	a := Advance{
		w:               w,
		components:      components,
		cIndexes:        make([]int, len(components)),
		stateLock:       newTryMutex(),
		refreshInterval: refreshInterval,
	}
	a.state.reset(refreshInterval)
	a.buf.WriteString(escapes.EraseLine + escapes.CursorLeft)
	a.cIndexes[0] = a.buf.Len()
	return &a
}

func (a *Advance) clear() (n int, err error) {
	return a.w.Write(a.buf.Bytes()[:a.cIndexes[0]])
}

func (a *Advance) print() (n int, err error) {
	a.buf.Truncate(a.cIndexes[0])
	for i, c := range a.components {
		a.cIndexes[i] = a.buf.Len()
		c.Print(&a.buf, &a.state)
		if i != len(a.components)-1 {
			a.buf.WriteRune(' ')
		}
	}

	return a.w.Write(a.buf.Bytes())
}

func (a *Advance) requestUpdate(force bool) bool {
	a.dispLock.Lock()
	defer a.dispLock.Unlock()

	if !a.active {
		return false
	}

	now := time.Now()
	ready := false
	if force {
		a.stateLock.Lock()
		defer a.stateLock.Unlock()
		ready = true
	} else {
		if a.stateLock.TryLock() {
			defer a.stateLock.Unlock()
			ready = (now.After(a.state.NextUpdate) || now.Equal(a.state.NextUpdate))
		}
	}

	if ready {
		nextProgress := atomic.LoadInt64(&a.nextProgress)
		nextTotal := atomic.LoadInt64(&a.nextTotal)
		a.state.update(nextProgress, nextTotal, a.refreshInterval)
		a.print()
	}

	return ready
}

func (a *Advance) Reset() {
	a.stateLock.Lock()
	a.state.reset(a.refreshInterval)
	a.stateLock.Unlock()

	atomic.StoreInt64(&a.nextProgress, 0)
	atomic.StoreInt64(&a.nextTotal, 0)
}

func (a *Advance) Show() {
	a.dispLock.Lock()
	defer a.dispLock.Unlock()

	if !a.active {
		a.stateLock.Lock()
		defer a.stateLock.Unlock()

		a.active = true
		a.print()
	}
}

func (a *Advance) Hide() {
	a.dispLock.Lock()
	defer a.dispLock.Unlock()

	if a.active {
		a.active = false
		a.clear()
	}
}

func (a *Advance) Write(p []byte) (n int, err error) {
	a.dispLock.Lock()
	defer a.dispLock.Unlock()

	if a.active {
		a.clear()
		n, err = a.w.Write(p)
		if err != nil {
			return
		}
		if p[len(p)-1] == '\n' {
			a.stateLock.Lock()
			_, err = a.print()
			a.stateLock.Unlock()
			if err != nil {
				return
			}
		}
		return
	}
	return a.w.Write(p)
}

func (a *Advance) Set(p int64) {
	if p < 0 {
		panic("Progress must be positive")
	}

	atomic.StoreInt64(&a.nextProgress, p)
	a.requestUpdate(false)
}

func (a *Advance) Add(p int64) {
	if p < 0 {
		panic("Progress must be positive")
	}

	atomic.AddInt64(&a.nextProgress, p)
	a.requestUpdate(false)
}

func (a *Advance) SetTotal(total int64) {
	if total < 0 {
		panic("Total must be positive")
	}

	atomic.StoreInt64(&a.nextTotal, total)
	a.requestUpdate(false)
}

func (a *Advance) AddTotal(total int64) {
	if total < 0 {
		panic("Total must be positive")
	}

	atomic.AddInt64(&a.nextTotal, total)
	a.requestUpdate(false)
}
