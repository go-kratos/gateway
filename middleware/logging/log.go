package logging

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"

	kratosLog "github.com/go-kratos/kratos/v2/log"
)

var _ kratosLog.Logger = (*fileLogger)(nil)

type fileLogger struct {
	log  *log.Logger
	pool *sync.Pool
}

func NewFileLogger(path string) (kratosLog.Logger, error) {
	logFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if err != nil {
		return nil, err
	}
	return &fileLogger{
		log: log.New(logFile, "", 0),
		pool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}, nil
}

// Log print the kv pairs log.
func (l *fileLogger) Log(level kratosLog.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 {
		return nil
	}
	if (len(keyvals) & 1) == 1 {
		keyvals = append(keyvals, "KEYVALS UNPAIRED")
	}
	buf := l.pool.Get().(*bytes.Buffer)
	buf.WriteString(level.String())
	for i := 0; i < len(keyvals); i += 2 {
		_, _ = fmt.Fprintf(buf, " %s=%v", keyvals[i], keyvals[i+1])
	}
	_ = l.log.Output(4, buf.String()) //nolint:gomnd
	buf.Reset()
	l.pool.Put(buf)
	return nil
}
