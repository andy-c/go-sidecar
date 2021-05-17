/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/12
  @version:v1
**/
package log

import (
	"github.com/sirupsen/logrus"
	"go-sidecar/config"
	"go-sidecar/log/hooks"
	"os"
	"strings"
	"time"
)

func InitLogger(){
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(getLogLevel(config.Config.Log.Level))
	logrus.SetReportCaller(true)
	if config.Config.Log.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			ForceColors: true,
			TimestampFormat: time.RFC3339,
		})
	}
	logrus.AddHook(hooks.NewGlobalField(config.Config.Log.Name))

	if config.Config.Log.Path != "" {
		logrus.AddHook(hooks.NewLogWriterHook(config.Config.Log.Path))
		logrus.AddHook(hooks.NewLogWriterForErrorHook(config.Config.Log.Path))
	} else {
		logrus.AddHook(hooks.NewLogPrinterHook())
		logrus.AddHook(hooks.NewLogPrinterForErrorHook())
	}
}

func getLogLevel(l string) logrus.Level {
	level, err := logrus.ParseLevel(strings.ToLower(l))
	if err == nil {
		return level
	}
	return logrus.InfoLevel
}