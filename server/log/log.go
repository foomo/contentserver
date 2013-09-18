package log

import (
	"fmt"
	"strings"
	"time"
)

var logLevel int = LOG_LEVEL_DEBUG
var numErrors int64 = 0

const (
	LOG_LEVEL_ERROR        = 0
	LOG_LEVEL_NAME_ERROR   = "error"
	LOG_LEVEL_RECORD       = 1
	LOG_LEVEL_NAME_RECORD  = "record"
	LOG_LEVEL_WARNING      = 2
	LOG_LEVEL_NAME_WARNING = "warning"
	LOG_LEVEL_NOTICE       = 3
	LOG_LEVEL_NAME_NOTICE  = "notice"
	LOG_LEVEL_DEBUG        = 4
	LOG_LEVEL_NAME_DEBUG   = "debug"
)

var prefices = map[int]string{
	LOG_LEVEL_RECORD:  "record  : ",
	LOG_LEVEL_ERROR:   "error   : ",
	LOG_LEVEL_WARNING: "warning : ",
	LOG_LEVEL_NOTICE:  "notice  : ",
	LOG_LEVEL_DEBUG:   "debug   : ",
}

var logLevelMap = map[string]int{
	LOG_LEVEL_NAME_ERROR:   LOG_LEVEL_ERROR,
	LOG_LEVEL_NAME_RECORD:  LOG_LEVEL_RECORD,
	LOG_LEVEL_NAME_WARNING: LOG_LEVEL_WARNING,
	LOG_LEVEL_NAME_NOTICE:  LOG_LEVEL_NOTICE,
	LOG_LEVEL_NAME_DEBUG:   LOG_LEVEL_DEBUG,
}

func GetLogLevelByName(name string) int {
	if level, ok := logLevelMap[name]; ok {
		return level
	} else {
		return LOG_LEVEL_RECORD
	}

}

func log(msg string, level int) string {
	if level <= logLevel {
		prefix := time.Now().Format(time.RFC3339Nano) + " " + prefices[level]
		lines := strings.Split(msg, "\n")
		for i := 0; i < len(lines); i++ {
			fmt.Println(prefix + lines[i])
		}
	}
	return msg
}

func SetLogLevel(level int) bool {
	if level > LOG_LEVEL_ERROR && level <= LOG_LEVEL_DEBUG {
		logLevel = level
		return true
	} else {
		return false
	}
}

func Debug(msg string) string {
	return log(msg, LOG_LEVEL_DEBUG)
}

func Notice(msg string) string {
	return log(msg, LOG_LEVEL_NOTICE)
}

func Warning(msg string) string {
	return log(msg, LOG_LEVEL_WARNING)
}

func Record(msg string) string {
	return log(msg, LOG_LEVEL_RECORD)
}

func Error(msg string) string {
	numErrors++
	return log(fmt.Sprintf("(%d) ", numErrors)+msg, LOG_LEVEL_ERROR)
}
