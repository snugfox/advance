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
	dispLock sync.Mutex
	w        io.Writer
	buf      bytes.Buffer

	active bool

	state *AdvanceState

	// Sloppy counters
	slopLock                sync.RWMutex
	nextProgress, nextTotal int64

	components []Component
	cIndexes   []int
}

func New(w io.Writer, refreshInterval time.Duration, components ...Component) *Advance {
	if len(components) == 0 {
		panic("Must have at least one component")
	}

	a := Advance{
		w:          w,
		components: components,
		cIndexes:   make([]int, len(components)),
		state:      newAdvanceState(refreshInterval),
	}
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
		c.Print(&a.buf, a.state)
		if i != len(a.components)-1 {
			a.buf.WriteRune(' ')
		}
	}

	return a.w.Write(a.buf.Bytes())
}

func (a *Advance) requestUpdate(force bool) bool {
	if a.state.requestUpdate(a.nextProgress, a.nextTotal, force) {
		a.dispLock.Lock()
		defer a.dispLock.Unlock()

		a.print()
		return true
	}
	return false
}

func (a *Advance) Reset() {
	a.slopLock.Lock()
	defer a.slopLock.Unlock()

	a.state.reset()
	a.nextProgress = 0
	a.nextTotal = 0
}

func (a *Advance) Show() {
	a.dispLock.Lock()
	defer a.dispLock.Unlock()

	if !a.active {
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
			_, err = a.print()
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

	a.slopLock.RLock()
	defer a.slopLock.RUnlock()

	atomic.StoreInt64(&a.nextProgress, p)
	a.requestUpdate(false)
}

func (a *Advance) Add(p int64) {
	if p < 0 {
		panic("Progress must be positive")
	}

	a.slopLock.RLock()
	defer a.slopLock.RUnlock()

	atomic.AddInt64(&a.nextProgress, p)
	a.requestUpdate(false)
}

func (a *Advance) SetTotal(total int64) {
	if total < 0 {
		panic("Total must be positive")
	}

	a.slopLock.RLock()
	defer a.slopLock.RUnlock()

	atomic.StoreInt64(&a.nextTotal, total)
	a.requestUpdate(false)
}

func (a *Advance) AddTotal(total int64) {
	if total < 0 {
		panic("Total must be positive")
	}

	a.slopLock.RLock()
	defer a.slopLock.RUnlock()

	atomic.AddInt64(&a.nextTotal, total)
	a.requestUpdate(false)
}
