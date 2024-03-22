package cmd

import (
	"time"

	"github.com/spf13/cobra"
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

func addAddressFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().String("address", ":8080", "Address to bind to (host:port)")
	_ = v.BindPFlag("address", cmd.Flags().Lookup("address"))
	_ = v.BindEnv("address", "CONTENT_SERVER_ADDRESS")
}

func basePathFlag(v *viper.Viper) string {
	return v.GetString("base_path")
}

func addBasePathFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().String("base-path", "/contentserver", "Base path to export the webserver on")
	_ = v.BindPFlag("base_path", cmd.Flags().Lookup("base_path"))
	_ = v.BindEnv("base_path", "CONTENT_SERVER_BASE_PATH")
}

func pollFlag(v *viper.Viper) bool {
	return v.GetBool("poll")
}

func addPollFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Bool("poll", false, "If true, the address arg will be used to periodically poll the content url")
	_ = v.BindPFlag("poll", cmd.Flags().Lookup("poll"))
	_ = v.BindEnv("poll", "CONTENT_SERVER_POLL")
}

func historyDirFlag(v *viper.Viper) string {
	return v.GetString("history.dir")
}

func addHistoryDirFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().String("history-dir", "/var/lib/contentserver", "Where to put my data")
	_ = v.BindPFlag("history.dir", cmd.Flags().Lookup("history-dir"))
	_ = v.BindEnv("history.dir", "CONTENT_SERVER_HISTORY_DIR")
}

func historyLimitFlag(v *viper.Viper) string {
	return v.GetString("history.limit")
}

func addHistoryLimitFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Int("history-limit", 2, "Number of history records to keep")
	_ = v.BindPFlag("history.limit", cmd.Flags().Lookup("history-limit"))
	_ = v.BindEnv("history.limit", "CONTENT_SERVER_HISTORY_LIMIT")
}

func gracefulTimeoutFlag(v *viper.Viper) time.Duration {
	return v.GetDuration("graceful_timeout")
}

func addGracefulTimeoutFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Duration("graceful-timeout", 0, "Timeout duration for graceful shutdown")
	_ = v.BindPFlag("graceful_timeout", cmd.Flags().Lookup("graceful-timeout"))
	_ = v.BindEnv("graceful_timeout", "CONTENT_SERVER_GRACEFUL_TIMEOUT")
}

func shutdownTimeoutFlag(v *viper.Viper) time.Duration {
	return v.GetDuration("shutdown_timeout")
}

func addShutdownTimeoutFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Duration("shutdown-timeout", 0, "Timeout duration for shutdown")
	_ = v.BindPFlag("shutdown_timeout", cmd.Flags().Lookup("shutdown-timeout"))
	_ = v.BindEnv("shutdown_timeout", "CONTENT_SERVER_SHUTDOWN_TIMEOUT")
}

func serviceHealthzEnabledFlag(v *viper.Viper) bool {
	return v.GetBool("service.healthz.enabled")
}

func addServiceHealthzEnabledFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Bool("service-healthz-enabled", false, "Enable healthz service")
	_ = v.BindPFlag("service.healthz.enabled", cmd.Flags().Lookup("service-healthz-enabled"))
}

func servicePrometheusEnabledFlag(v *viper.Viper) bool {
	return v.GetBool("service.prometheus.enabled")
}

func addServicePrometheusEnabledFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Bool("service-prometheus-enabled", false, "Enable prometheus service")
	_ = v.BindPFlag("service.prometheus.enabled", cmd.Flags().Lookup("service-prometheus-enabled"))
}

func otelEnabledFlag(v *viper.Viper) bool {
	return v.GetBool("otel.enabled")
}

func addOtelEnabledFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Bool("otel-enabled", false, "Enable otel service")
	_ = v.BindPFlag("otel.enabled", cmd.Flags().Lookup("otel-enabled"))
	_ = v.BindEnv("otel.enabled", "OTEL_ENABLED")
}
