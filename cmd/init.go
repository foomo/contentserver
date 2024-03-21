package cmd

import (
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.EnvKeyReplacer(strings.NewReplacer(".", "_"))
}

// initConfig reads in config file and ENV variables if set.
func initLogger() {
	var err error
	c := zap.NewProductionConfig()
	c.Level, err = zap.ParseAtomicLevel(logLevel)
	if err != nil {
		panic(err)
	}
	logger, err = c.Build()
	if err != nil {
		panic(err)
	}
}
