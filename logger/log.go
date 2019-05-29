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

	var (
		zc  zap.Config
		err error
	)

	if debug {
		zc = zap.NewDevelopmentConfig()
		zc.OutputPaths = append(zc.OutputPaths, outputPath)
	} else {
		zc = zap.NewProductionConfig()
		zc.OutputPaths = append(zc.OutputPaths, outputPath)
	}

	if os.Getenv("LOG_JSON") == "1" {
		zc.Encoding = "json"
	} else {
		zc.Encoding = "console"
	}

	Log, err = zc.Build()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
}
