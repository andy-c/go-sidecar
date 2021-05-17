/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/13
  @version:v1
**/
package hooks

import (
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

func NewLogWriterHook(p string) *LogWriterHook {
	return &LogWriterHook{
		lock: new(sync.Mutex),
		levels: []logrus.Level{
			logrus.DebugLevel,
			logrus.InfoLevel,
		},
		writer: &lumberjack.Logger{
			Filename:   p+"info.log",
			MaxSize:    500, // megabytes
			MaxBackups: 3,
			MaxAge:     30, //days
		},
	}
}

func NewLogWriterForErrorHook(p string) *LogWriterHook {
	return &LogWriterHook{
		lock: new(sync.Mutex),
		levels: []logrus.Level{
			logrus.WarnLevel,
			logrus.ErrorLevel,
			logrus.FatalLevel,
			logrus.PanicLevel,
		},
		writer: &lumberjack.Logger{
			Filename:   p + "error.log",
			MaxSize:    500, // megabytes
			MaxBackups: 3,
			MaxAge:     30, //days
		},
	}
}

type LogWriterHook struct {
	levels []logrus.Level
	writer io.Writer
	lock   *sync.Mutex
}

func (hook *LogWriterHook) Fire(entry *logrus.Entry) error {
	hook.lock.Lock()
	defer hook.lock.Unlock()

	msg, err := entry.String()
	if err != nil {
		return err
	} else {
		hook.writer.Write([]byte(msg))
	}

	return nil
}

func (hook *LogWriterHook) Levels() []logrus.Level {
	return hook.levels
}
