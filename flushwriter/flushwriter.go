package flushwriter

import (
	"io"
	"net/http"
)

// FlushWriter is a writer that will flush on every write.
type FlushWriter struct {
	flusher http.Flusher
	writer  io.Writer
}

// New returns a new FlushWriter.
func New(writer io.Writer) FlushWriter {
	fw := FlushWriter{writer: writer}

	if f, ok := writer.(http.Flusher); ok {
		fw.flusher = f
	}

	return fw
}

// Write will write p []byte to FlushWriter.writer and if it has
// a flusher will flush the data.
func (fw *FlushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.writer.Write(p)
	if fw.flusher != nil {
		fw.flusher.Flush()
	}
	return
}
