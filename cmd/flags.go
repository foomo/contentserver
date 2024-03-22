package cmd

import (
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func logLevelFlag(v *viper.Viper) string {
	return v.GetString("log.level")
}

func addLogLevelFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.String("log-level", "info", "log level")
	_ = v.BindPFlag("log.level", flags.Lookup("log-level"))
	_ = v.BindEnv("log.level", "LOG_LEVEL")
}

func logFormatFlag(v *viper.Viper) string {
	return v.GetString("log.format")
}

func addLogFormatFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.String("log-format", "json", "log format")
	_ = v.BindPFlag("log.format", flags.Lookup("log-format"))
	_ = v.BindEnv("log.format", "LOG_FORMAT")
}

func addressFlag(v *viper.Viper) string {
	return v.GetString("address")
}

func addAddressFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.String("address", ":8080", "Address to bind to (host:port)")
	_ = v.BindPFlag("address", flags.Lookup("address"))
	_ = v.BindEnv("address", "CONTENT_SERVER_ADDRESS")
}

func basePathFlag(v *viper.Viper) string {
	return v.GetString("base_path")
}

func addBasePathFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.String("base-path", "/contentserver", "Base path to export the webserver on")
	_ = v.BindPFlag("base_path", flags.Lookup("base_path"))
	_ = v.BindEnv("base_path", "CONTENT_SERVER_BASE_PATH")
}

func pollFlag(v *viper.Viper) bool {
	return v.GetBool("poll.enabled")
}

func addPollFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.Bool("poll", false, "If true, the address arg will be used to periodically poll the content url")
	_ = v.BindPFlag("poll.enabled", flags.Lookup("poll"))
	_ = v.BindEnv("poll.enabled", "CONTENT_SERVER_POLL")
}

func pollIntevalFlag(v *viper.Viper) time.Duration {
	return v.GetDuration("poll.interval")
}

func addPollIntervalFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.Duration("poll-interval", time.Minute, "Specifies the poll interval")
	_ = v.BindPFlag("poll.interval", flags.Lookup("poll-interval"))
	_ = v.BindEnv("poll.interval", "CONTENT_SERVER_POLL_INTERVAL")
}

func historyDirFlag(v *viper.Viper) string {
	return v.GetString("history.dir")
}

func addHistoryDirFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.String("history-dir", "/var/lib/contentserver", "Where to put my data")
	_ = v.BindPFlag("history.dir", flags.Lookup("history-dir"))
	_ = v.BindEnv("history.dir", "CONTENT_SERVER_HISTORY_DIR")
}

func historyLimitFlag(v *viper.Viper) int {
	return v.GetInt("history.limit")
}

func addHistoryLimitFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.Int("history-limit", 2, "Number of history records to keep")
	_ = v.BindPFlag("history.limit", flags.Lookup("history-limit"))
	_ = v.BindEnv("history.limit", "CONTENT_SERVER_HISTORY_LIMIT")
}

func gracefulTimeoutFlag(v *viper.Viper) time.Duration {
	return v.GetDuration("graceful_timeout")
}

func addGracefulTimeoutFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.Duration("graceful-timeout", 0, "Timeout duration for graceful shutdown")
	_ = v.BindPFlag("graceful_timeout", flags.Lookup("graceful-timeout"))
	_ = v.BindEnv("graceful_timeout", "CONTENT_SERVER_GRACEFUL_TIMEOUT")
}

func shutdownTimeoutFlag(v *viper.Viper) time.Duration {
	return v.GetDuration("shutdown_timeout")
}

func addShutdownTimeoutFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.Duration("shutdown-timeout", 0, "Timeout duration for shutdown")
	_ = v.BindPFlag("shutdown_timeout", flags.Lookup("shutdown-timeout"))
	_ = v.BindEnv("shutdown_timeout", "CONTENT_SERVER_SHUTDOWN_TIMEOUT")
}

func serviceHealthzEnabledFlag(v *viper.Viper) bool {
	return v.GetBool("service.healthz.enabled")
}

func addServiceHealthzEnabledFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.Bool("service-healthz-enabled", false, "Enable healthz service")
	_ = v.BindPFlag("service.healthz.enabled", flags.Lookup("service-healthz-enabled"))
}

func servicePrometheusEnabledFlag(v *viper.Viper) bool {
	return v.GetBool("service.prometheus.enabled")
}

func addServicePrometheusEnabledFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.Bool("service-prometheus-enabled", false, "Enable prometheus service")
	_ = v.BindPFlag("service.prometheus.enabled", flags.Lookup("service-prometheus-enabled"))
}

func otelEnabledFlag(v *viper.Viper) bool {
	return v.GetBool("otel.enabled")
}

func addOtelEnabledFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.Bool("otel-enabled", false, "Enable otel service")
	_ = v.BindPFlag("otel.enabled", flags.Lookup("otel-enabled"))
	_ = v.BindEnv("otel.enabled", "OTEL_ENABLED")
}
