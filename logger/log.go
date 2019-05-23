package logger

import (
	"log"
	"os"

	"go.uber.org/zap"
)

var (
	// Log is the logger instance exposed by this package
	// call Setup() prior to using it
	// want JSON output? Set LOG_JSON env var to 1!
	Log *zap.Logger
)

// SetupLogging configures the logger
func SetupLogging(debug bool, outputPath string) {

	var err error
	if debug {
		zc := zap.NewDevelopmentConfig()
		if os.Getenv("LOG_JSON") == "1" {
			zc.Encoding = "json"
		}
		zc.OutputPaths = append(zc.OutputPaths, outputPath)
		Log, err = zc.Build()
	} else {
		zc := zap.NewProductionConfig()
		if os.Getenv("LOG_JSON") == "1" {
			zc.Encoding = "json"
		}
		zc.OutputPaths = append(zc.OutputPaths, outputPath)
		Log, err = zc.Build()
	}
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
}
