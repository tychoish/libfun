package libfun

import (
	"bytes"
	"io"
	"strings"

	"github.com/tychoish/fun/intish"
)

type SizeReportingWriter struct {
	out io.Writer
	len intish.Atomic[int]
}

func NewSizeReportingWriter(base io.Writer) *SizeReportingWriter {
	return &SizeReportingWriter{out: base}
}

func (w *SizeReportingWriter) Write(in []byte) (out int, err error) {
	out, err = w.out.Write(in)
	w.len.Add(out)
	return
}

func (w *SizeReportingWriter) Size() int { return w.len.Load() }

type TrimWhitespaceWriter struct {
	out io.Writer
}

func NewTrimWhitespaceWriter(wr io.Writer) *TrimWhitespaceWriter {
	return &TrimWhitespaceWriter{out: wr}
}

func (b *TrimWhitespaceWriter) Write(in []byte) (int, error) {
	_, _ = b.out.Write(bytes.TrimSpace(in))
	return len(in), nil
}

func (b *TrimWhitespaceWriter) WriteString(in string) (int, error) {
	_, _ = b.out.Write([]byte(strings.TrimSpace(in)))
	return len(in), nil
}
