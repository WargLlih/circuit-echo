package logwriter

import (
	"io"
	"time"
)

type log struct {
	msgs chan string
}

var instance *log = nil

func Logger() io.Writer {
	if instance == nil {
		instance = &log{
			msgs: make(chan string, 1024),
		}
	}
	return instance
}

func (log) Write(data []byte) (int, error) {
	select {
	case instance.msgs <- string(data):
	case <-time.After(100 * time.Millisecond):
		return 0, nil
	}

	return 0, nil
}

func Messages() <-chan string {
	l := Logger().(*log)
	return l.msgs
}
