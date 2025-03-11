package proxy

import (
	"io"
	"net/http"
)

type bodyCopier func(io.Writer, io.Reader) (int64, error)

func copyNoBuffering(w http.ResponseWriter) bodyCopier {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return io.Copy
	}
	return func(dst io.Writer, src io.Reader) (int64, error) {
		return copyBufferWithCallback(dst, src, []byte{0}, func(_ int) {
			flusher.Flush()
		})
	}
}

// Same header as nginx: X-Accel-Buffering: no
func isNoBufferingResponse(resp *http.Response) bool {
	return resp.Header.Get("X-Accel-Buffering") == "no"
}

// This is a modified version of io.CopyBuffer that supports a callback for each written byte.
func copyBufferWithCallback(dst io.Writer, src io.Reader, buf []byte, onWrite func(int)) (int64, error) {
	if len(buf) == 0 {
		panic("empty buffer in copyBufferWithCallback")
	}

	var totalWritten int64
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			written, writeErr := dst.Write(buf[:n])
			totalWritten += int64(written)

			// Trigger the callback with the written chunk size
			if onWrite != nil {
				onWrite(written)
			}

			if writeErr != nil {
				return totalWritten, writeErr
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return totalWritten, readErr
		}
	}

	return totalWritten, nil
}
