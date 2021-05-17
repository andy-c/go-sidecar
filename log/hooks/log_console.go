/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/13
  @version:v1
**/
package hooks

import (
	"github.com/sirupsen/logrus"
	"os"
	"sync"
)

func NewLogPrinterHook() *LogPrinterHook {
	return &LogPrinterHook{
		lock:   new(sync.Mutex),
		writer: os.Stdout,
		levels: []logrus.Level{
			logrus.DebugLevel,
			logrus.InfoLevel,
		},
	}
}

func NewLogPrinterForErrorHook() *LogPrinterHook {
	return &LogPrinterHook{
		lock:   new(sync.Mutex),
		writer: os.Stderr,
		levels: []logrus.Level{
			logrus.WarnLevel,
			logrus.ErrorLevel,
			logrus.FatalLevel,
			logrus.PanicLevel,
		},
	}
}

type LogPrinterHook struct {
	levels []logrus.Level
	lock   *sync.Mutex
	writer *os.File
}

func (hook *LogPrinterHook) Fire(entry *logrus.Entry) error {
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

func (hook *LogPrinterHook) Levels() []logrus.Level {
	return hook.levels
}

