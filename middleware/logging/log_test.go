package logging

import (
	"testing"

	kratosLog "github.com/go-kratos/kratos/v2/log"
)

func TestFileLog(t *testing.T) {
	path := "./test.log"
	log, err := NewFileLogger(path)
	if err != nil {
		t.Error(err)
	}
	log.Log(kratosLog.LevelDebug, "test", "kratos")
}
