package advance

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"runtime"
	"sync"
	"testing"

	escapes "github.com/snugfox/ansi-escapes"
)

func TestAdvance(t *testing.T) {
	testCases := []struct {
		name   string
		useSet bool
	}{
		{"Set", true},
		{"Add", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			var as AdvanceState
			a := New(&out, 0, &Debug{})
			if tc.useSet {
				a.SetTotal(5)
			} else {
				a.AddTotal(5)
			}
			a.Show()
			out.Reset()
			for i := int64(1); i < 5; i++ {
				if tc.useSet {
					a.Set(i)
				} else {
					a.Add(1)
				}

				// Assert the existence of escape sequences to clear the current line and
				// move the cursor to the left.
				want := escapes.EraseLine + escapes.CursorLeft
				esc := make([]byte, len(want))
				out.Read(esc)
				if !bytes.Equal(esc, []byte(want)) {
					t.Fatalf("Read incorrect escape sequence, want: %q, got: %q", escapes.EraseLine+escapes.CursorLeft, string(esc))
				}

				// Peek and assert that the debug output starts next
				// TODO: Read until '{' and log offending bytes, if any
				r, _, err := out.ReadRune()
				if err != nil {
					t.Fatal(err)
				}
				if r != '{' { // Serialized JSON object begins with the '{' rune
					t.Fatalf("Failed to find serialized JSON object, want: %q, got: %q", '{', r)
				}
				if err := out.UnreadRune(); err != nil {
					t.Fatal(err)
				}

				// Read the debug output (serialized AdvanceState) and assert its
				// validity
				line, err := out.ReadBytes(byte('\n'))
				if err != nil {
					t.Fatal(err)
				}
				if err = json.Unmarshal(line, &as); err != nil {
					t.Fatal(err)
				}
				if as.Progress != i {
					t.Errorf("Incorrect progress, want: %d, got: %d", i, as.Progress)
				}
				if as.Total != 5 {
					t.Errorf("Incorrect total, want: %d, got: %d", 5, as.Total)
				}
				// TODO: Validate other AdvanceState fields
			}
		})
	}
}

func TestAdvanceParallel(t *testing.T) {
	a := New(ioutil.Discard, 0, &Debug{})
	a.Show()
	a.Reset()
	a.SetTotal(100 * int64(runtime.GOMAXPROCS(0)))
	var wg sync.WaitGroup
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		wg.Add(1)
		go func() {
			for i := int64(0); i < 100; i++ {
				a.Add(1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if a.state.Progress != a.state.Total {
		t.Fatalf("Incorrect progress, want: %d, got: %d", a.state.Total, a.state.Progress)
	}
}
