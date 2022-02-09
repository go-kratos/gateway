package client

import (
	"bytes"
	"io"
)

// NopReader is a requset body reader.
// - io.ReadCloser
// - io.Seeker
type NopReader struct {
	rd bytes.Reader
}

// ReadFrom reads data from r until EOF.
func (r *NopReader) ReadFrom(src io.Reader) (int64, error) {
	b, err := io.ReadAll(src)
	if err != nil {
		return 0, err
	}
	r.rd.Reset(b)
	return int64(len(b)), nil
}

// Read implements the io.Reader interface.
func (r *NopReader) Read(b []byte) (n int, err error) {
	return r.rd.Read(b)
}

// Seek implements the io.Seeker interface.
func (r *NopReader) Seek(offset int64, whence int) (int64, error) {
	return r.rd.Seek(offset, whence)
}

// Reset resets the Reader to be reading from b.
func (r *NopReader) Reset(b []byte) {
	r.rd.Reset(b)
}

// Close close the Reader.
func (r *NopReader) Close() error {
	return nil
}
