package advance

import (
	"io"
)

type ReadWrapper struct {
	R io.Reader
	A *Advance
}

func (rw *ReadWrapper) Read(p []byte) (n int, err error) {
	n, err = rw.R.Read(p)
	rw.A.Add(int64(n))
	return
}

type WriteWrapper struct {
	W io.Writer
	A *Advance
}

func (ww *WriteWrapper) Write(p []byte) (n int, err error) {
	n, err = ww.W.Write(p)
	ww.A.Add(int64(n))
	return
}
