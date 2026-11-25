package core

import (
	"io"
	"os"
)

// ProgressFunc reports transfer progress: read is the number of bytes
// transferred so far and total is the expected size in bytes, or -1 when it is
// not known. It is called after every chunk, so keep it cheap.
type ProgressFunc func(read, total int64)

// NewProgressReader wraps r so that fn is called after each read with the
// cumulative byte count. total is the expected size, or -1 when unknown. Close
// closes r when it is an io.Closer.
func NewProgressReader(r io.Reader, total int64, fn ProgressFunc) io.ReadCloser {
	return &progressReader{r: r, total: total, fn: fn}
}

type progressReader struct {
	r     io.Reader
	total int64
	read  int64
	fn    ProgressFunc
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.read += int64(n)
		if p.fn != nil {
			p.fn(p.read, p.total)
		}
	}
	return n, err
}

func (p *progressReader) Close() error {
	if c, ok := p.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// ReaderSize returns the number of bytes r will yield when it can be told
// (files, bytes.Reader, strings.Reader, bytes.Buffer), otherwise -1.
func ReaderSize(r io.Reader) int64 {
	switch v := r.(type) {
	case nil:
		return -1
	case *os.File:
		if st, err := v.Stat(); err == nil && st.Mode().IsRegular() {
			return st.Size()
		}
	case interface{ Len() int }:
		return int64(v.Len())
	case interface{ Size() int64 }:
		return v.Size()
	}
	return -1
}
