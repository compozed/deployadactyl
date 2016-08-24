package interfaces

// FlushWriter interface.
type FlushWriter interface {
	Write(p []byte) (n int, err error)
}
