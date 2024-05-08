package logging

import (
	gomodlog "github.com/cyverse-de/go-mod/logging"
	"github.com/cyverse/qms/config"
	"github.com/sirupsen/logrus"
)

func GetLogger() *logrus.Entry {
	return gomodlog.Log.WithFields(logrus.Fields{"service": config.ServiceName})
}

func SetupLogging(level string) {
	gomodlog.SetupLogging(level)
}
