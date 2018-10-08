package advance

import (
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/snugfox/advance/bytesize"
)

type Component interface {
	Print(buf *bytes.Buffer, as *AdvanceState)
}

type Debug struct{}

func (d *Debug) Print(buf *bytes.Buffer, as *AdvanceState) {
	p, err := json.Marshal(as)
	if err != nil {
		panic(err)
	}
	if _, err = buf.Write(p); err != nil {
		panic(err)
	}
	if err = buf.WriteByte(byte('\n')); err != nil {
		panic(err)
	}
}

type FixedWidth struct {
	C     Component
	Width int
}

func (fw *FixedWidth) Print(buf *bytes.Buffer, as *AdvanceState) {
	preLen := buf.Len()
	maxLen := preLen + fw.Width
	fw.C.Print(buf, as)
	postLen := buf.Len()
	if postLen > maxLen {
		buf.Truncate(maxLen)
	} else if postLen < maxLen {
		buf.WriteString(strings.Repeat(" ", maxLen-postLen))
	}
}

type Text struct {
	Text string
}

func (t *Text) Print(buf *bytes.Buffer, as *AdvanceState) {
	buf.WriteString(t.Text)
}

type CycleText struct {
	sync.Mutex

	currText  *list.Element
	Texts     list.List
	Speed     time.Duration
	lastCycle time.Time
}

func (ct *CycleText) Store(text string) *list.Element {
	ct.Lock()
	defer ct.Unlock()

	return ct.Texts.PushBack(text)
}

func (ct *CycleText) Delete(textEl *list.Element) {
	ct.Lock()
	defer ct.Unlock()

	ct.Texts.Remove(textEl)
}

func (ct *CycleText) Print(buf *bytes.Buffer, as *AdvanceState) {
	ct.Lock()
	defer ct.Unlock()

	now := time.Now()
	if ct.currText == nil {
		ct.currText = ct.Texts.Front()
	} else if now.Sub(ct.lastCycle) > ct.Speed {
		ct.lastCycle = now
		ct.currText = ct.currText.Next()
		if ct.currText == nil {
			ct.currText = ct.Texts.Front()
		}
	}
	if ct.currText == nil {
		buf.WriteString("")
	} else {
		buf.WriteString(ct.currText.Value.(string))
	}
}

type ProgressBar struct {
	Width int
	buf   bytes.Buffer
}

func (pb *ProgressBar) Print(buf *bytes.Buffer, as *AdvanceState) {
	pb.buf.Reset()

	// Scale the progress to the available width of the bar
	var fillLen int
	if as.Total == 0 {
		fillLen = 0
	} else if as.Progress >= as.Total {
		fillLen = pb.Width
	} else {
		fillLen = int(int64(pb.Width) * as.Progress / as.Total)
	}
	emptyLen := int(pb.Width - fillLen)

	// Buffer the bar and write it
	pb.buf.WriteRune('[')
	pb.buf.WriteString(strings.Repeat("=", fillLen))
	if emptyLen > 0 {
		pb.buf.WriteRune('>')
		pb.buf.WriteString(strings.Repeat(" ", emptyLen-1))
	}
	pb.buf.WriteRune(']')

	buf.ReadFrom(&pb.buf)
}

type Elapsed struct{}

func (e *Elapsed) Print(buf *bytes.Buffer, as *AdvanceState) {
	buf.WriteString(time.Since(as.Start).Round(time.Second).String())
}

type DataRate struct {
	r        bytesize.Size
	Unit     bytesize.Size
	AutoUnit bool
}

func (r *DataRate) Print(buf *bytes.Buffer, as *AdvanceState) {
	rate := float64(as.DeltaProgress) / as.DeltaUpdate.Seconds()
	if math.IsNaN(r.r.Size()) || math.IsInf(r.r.Size(), 0) {
		r.r = bytesize.Size(rate)
	} else {
		r.r = bytesize.Size((r.r.Size() * 7 / 8) + (rate / 8))
	}
	var s bytesize.Size
	if r.AutoUnit {
		_, s = r.r.AutoBase(true)
	} else {
		s = r.Unit
	}
	buf.WriteString(fmt.Sprint(math.Round(r.r.Base(s))))
	buf.WriteRune(' ')
	buf.WriteString(s.Label())
}
