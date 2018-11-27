package log

import (
	"fmt"
	"strings"
	"time"
)

// Level logging level enum
type Level int

const (
	// LevelError an error - as bad as it gets
	LevelError Level = 0
	// LevelRecord put this to the logs in any case
	LevelRecord Level = 1
	// LevelWarning not that bad
	LevelWarning Level = 2
	// LevelNotice almost on debug level
	LevelNotice Level = 3
	// LevelDebug we are debugging
	LevelDebug Level = 4
)

// SelectedLevel selected log level
var SelectedLevel = LevelDebug

var prefices = map[Level]string{
	LevelRecord:  "record  : ",
	LevelError:   "error   : ",
	LevelWarning: "warning : ",
	LevelNotice:  "notice  : ",
	LevelDebug:   "debug   : ",
}

func log(msg string, level Level) string {
	if level <= SelectedLevel {
		prefix := time.Now().Format(time.RFC3339Nano) + " " + prefices[level]
		lines := strings.Split(msg, "\n")
		for i := 0; i < len(lines); i++ {
			fmt.Println(level, prefix+lines[i])
		}
	}
	return msg
}

func logThings(msgs []interface{}, level Level) string {
	r := ""
	for _, msg := range msgs {
		r += "\n" + fmt.Sprint(msg)
	}
	r = strings.Trim(r, "\n")
	return log(r, level)
}

// Debug write debug messages to the log
func Debug(msgs ...interface{}) string {
	return logThings(msgs, LevelDebug)
}

// Notice write notice messages to the log
func Notice(msgs ...interface{}) string {
	return logThings(msgs, LevelNotice)
}

// Warning write warning messages to the log
func Warning(msgs ...interface{}) string {
	return logThings(msgs, LevelWarning)
}

// Record write record messages to the log
func Record(msgs ...interface{}) string {
	return logThings(msgs, LevelRecord)
}

// Error write error messages to the log
func Error(msgs ...interface{}) string {
	return logThings(msgs, LevelError)
}
