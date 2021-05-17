/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/13
  @version:v1
**/
package hooks

import (
	"github.com/sirupsen/logrus"
)

func NewGlobalField(appName string) *Field{
	return &Field{
		Name:appName,
	}
}

type Field struct {
	Name string
}

func (f *Field) Fire(entry *logrus.Entry) error{
	entry.Data["app_name"] = f.Name
	if _, ok := entry.Data["mod"]; !ok {
		entry.Data["mod"] = "log"
	}
	return nil
}

func (f *Field) Levels() []logrus.Level {
	return logrus.AllLevels
}
