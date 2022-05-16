package logging

import (
	gomodlog "github.com/cyverse-de/go-mod/logging"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	echolog "github.com/spirosoik/echo-logrus"
)

var Log = gomodlog.Log

func SetServiceName(serviceName string) {
	Log = Log.WithFields(logrus.Fields{"service": serviceName})
}

// GetEchoLogger returns an echo.Logger based off of the logrus logger passed
// in.
func GetEchoLogger(logger *logrus.Entry) echo.Logger {
	return echolog.NewLoggerMiddleware(logger)
}
