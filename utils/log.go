package utils

import (
	"fmt"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const MainProcess = "main"
const SubProcess = "cntr"

var processFlag = MainProcess

var PrintToConsole = fmt.Printf

var Info = logrus.Info
var Infof = logrus.Infof
var Warn = logrus.Warn
var Warnf = logrus.Warnf

type LogFormatter struct{}

func (f *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	logMessage := fmt.Sprintf("%s [%s] [%s:%d] %s: %s\n",
		entry.Time.Format("2006-01-02 15:04:05.000"),
		processFlag,
		filepath.Base(entry.Caller.File),
		entry.Caller.Line,
		entry.Level,
		entry.Message,
	)
	return []byte(logMessage), nil
}

func SetSubProcessFlag() {
	processFlag = SubProcess
}
