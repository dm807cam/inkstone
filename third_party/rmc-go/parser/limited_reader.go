package parser

import (
	"io"
)

// LimitedBufReader wraps an io.Reader and limits how much can be read
type LimitedBufReader struct {
	reader    io.Reader
	remaining int64
}

// NewLimitedBufReader creates a new limited reader
func NewLimitedBufReader(r io.Reader, limit int64) *LimitedBufReader {
	return &LimitedBufReader{
		reader:    r,
		remaining: limit,
	}
}

// Read implements io.Reader
func (l *LimitedBufReader) Read(p []byte) (n int, err error) {
	if l.remaining <= 0 {
		return 0, io.EOF
	}

	if int64(len(p)) > l.remaining {
		p = p[0:l.remaining]
	}

	n, err = l.reader.Read(p)
	l.remaining -= int64(n)
	return
}

// Remaining returns how many bytes are left
func (l *LimitedBufReader) Remaining() int64 {
	return l.remaining
}

// Skip skips the remaining bytes using chunked reading to avoid large allocations
func (l *LimitedBufReader) Skip() error {
	if l.remaining <= 0 {
		return nil
	}

	// Use chunked reading to prevent OOM on large blocks
	const chunkSize = 8192 // 8KB chunks
	buf := make([]byte, chunkSize)

	for l.remaining > 0 {
		readSize := l.remaining
		if readSize > chunkSize {
			readSize = chunkSize
		}

		n, err := io.ReadFull(l.reader, buf[:readSize])
		l.remaining -= int64(n)
		if err != nil {
			return err
		}
	}

	return nil
}
