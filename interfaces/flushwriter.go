package interfaces

type FlushWriter interface {
	Write(p []byte) (n int, err error)
}
