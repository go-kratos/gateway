package streammeta

import (
	"fmt"
	"io"
	"net/http"

	configv1 "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

func init() {
	middleware.RegisterV2("streamrecorder", New)
}

func New(*configv1.Middleware) (middleware.MiddlewareV2, error) {
	return &MetaStreamRecorder{}, nil
}

type MetaStreamRecorder struct{}

var _ middleware.MiddlewareV2 = (*MetaStreamRecorder)(nil)

func NewMetaStreamRecorder() *MetaStreamRecorder {
	return &MetaStreamRecorder{}
}

type streamRecorderKey struct{}
type StreamRecorder struct {
	Request  []*middleware.MetaStreamChunk
	Response []*middleware.MetaStreamChunk
}

func InitStreamRecorder(reqOpts *middleware.RequestOptions, recorder *StreamRecorder) {
	reqOpts.Values.Set(streamRecorderKey{}, recorder)
}

func GetStreamRecorder(reqOpts *middleware.RequestOptions) (*StreamRecorder, bool) {
	recorder, ok := reqOpts.Values.Get(streamRecorderKey{})
	if ok {
		return recorder.(*StreamRecorder), true
	}
	return nil, false
}

func (s *MetaStreamRecorder) Process(next http.RoundTripper) http.RoundTripper {
	return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		reqOpts, ok := middleware.FromRequestContext(req.Context())
		if !ok {
			// not stream request
			return next.RoundTrip(req)
		}
		streamCtx, ok := middleware.GetMetaStreamContext(reqOpts)
		if !ok {
			// not stream request
			return next.RoundTrip(req)
		}

		recorder := &StreamRecorder{
			Request:  make([]*middleware.MetaStreamChunk, 0),
			Response: make([]*middleware.MetaStreamChunk, 0),
		}
		InitStreamRecorder(reqOpts, recorder)
		streamCtx.OnChunk = append(streamCtx.OnChunk, func(req *http.Request, reply *http.Response, chunk *middleware.MetaStreamChunk) {
			switch chunk.Tag {
			case middleware.TagRequest:
				recorder.Request = append(recorder.Request, chunk)
			case middleware.TagResponse:
				recorder.Response = append(recorder.Response, chunk)
			}
		})
		return next.RoundTrip(req)
	})
}

func (s *MetaStreamRecorder) Close() error {
	return nil
}

var _ io.ReadSeeker = (*streamReaderSeeker)(nil)

type streamReaderSeeker struct {
	inner        []*middleware.MetaStreamChunk
	currentChunk int   // Current chunk index
	chunkOffset  int64 // Offset within current chunk
	totalPos     int64 // Total position across all chunks
}

func (s *streamReaderSeeker) Read(p []byte) (int, error) {
	if len(s.inner) == 0 {
		return 0, io.EOF
	}

	totalRead := 0
	for totalRead < len(p) && s.currentChunk < len(s.inner) {
		chunk := s.inner[s.currentChunk]
		remaining := int64(len(chunk.Data)) - s.chunkOffset

		if remaining <= 0 {
			// Move to next chunk
			s.currentChunk++
			s.chunkOffset = 0
			continue
		}

		// Calculate how much to read from current chunk
		toRead := int64(len(p) - totalRead)
		if toRead > remaining {
			toRead = remaining
		}

		// Copy data from current chunk
		copy(p[totalRead:], chunk.Data[s.chunkOffset:s.chunkOffset+toRead])
		totalRead += int(toRead)
		s.chunkOffset += toRead
		s.totalPos += toRead
	}

	if totalRead == 0 && s.currentChunk >= len(s.inner) {
		return 0, io.EOF
	}

	return totalRead, nil
}

func (s *streamReaderSeeker) Seek(offset int64, whence int) (int64, error) {
	var newPos int64

	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = s.totalPos + offset
	case io.SeekEnd:
		// Calculate total size
		totalSize := int64(0)
		for _, chunk := range s.inner {
			totalSize += int64(len(chunk.Data))
		}
		newPos = totalSize + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newPos < 0 {
		return 0, fmt.Errorf("negative position")
	}

	// Find the chunk and offset for the new position
	pos := int64(0)
	for i, chunk := range s.inner {
		chunkSize := int64(len(chunk.Data))
		if pos+chunkSize > newPos {
			// Position is within this chunk
			s.currentChunk = i
			s.chunkOffset = newPos - pos
			s.totalPos = newPos
			return s.totalPos, nil
		}
		pos += chunkSize
	}

	// Position is at or beyond the end
	s.currentChunk = len(s.inner)
	s.chunkOffset = 0
	s.totalPos = newPos
	return s.totalPos, nil
}
