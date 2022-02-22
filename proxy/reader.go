package proxy

import (
	"bytes"
	"io"
)

// BodyReader is a requset body reader.
// - io.ReadCloser
// - io.Seeker
type BodyReader struct {
	rd bytes.Reader
}

// ReadFrom reads data from r until EOF.
func (r *BodyReader) ReadFrom(src io.Reader) (int64, error) {
	b, err := io.ReadAll(src)
	if err != nil {
		return 0, err
	}
	r.rd.Reset(b)
	return int64(len(b)), nil
}

// Read implements the io.Reader interface.
func (r *BodyReader) Read(b []byte) (n int, err error) {
	return r.rd.Read(b)
}

// Seek implements the io.Seeker interface.
func (r *BodyReader) Seek(offset int64, whence int) (int64, error) {
	return r.rd.Seek(offset, whence)
}

// Reset resets the Reader to be reading from b.
func (r *BodyReader) Reset(b []byte) {
	r.rd.Reset(b)
}

// Close close the Reader.
func (r *BodyReader) Close() error {
	return nil
}
