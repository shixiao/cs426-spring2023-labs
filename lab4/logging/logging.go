package logging

import (
	"flag"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	logLevel = flag.String("log-level", "INFO", "Overrides the logging level, default INFO means INFO or higher (WARN, ERROR) are printed. Override to DEBUG or TRACE to get more logs")
	logFile  = flag.String("log-file", "", "If set, log to a file instead")
)

func InitLogging() {
	switch strings.ToLower(*logLevel) {
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		DisableColors: false,
	})
	if *logFile != "" {
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logrus.WithField("file", file).Fatalf("failed to open log file: %q", err)
		}
		logrus.SetOutput(file)
	}
}
