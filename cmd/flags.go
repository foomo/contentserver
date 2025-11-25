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
	_ = v.BindPFlag("base_path", flags.Lookup("base-path"))
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

func gracefulPeriodFlag(v *viper.Viper) time.Duration {
	return v.GetDuration("graceful.period")
}

func addShutdownTimeoutFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.Duration("graceful-period", 0, "Graceful shutdown period")
	_ = v.BindPFlag("graceful.period", flags.Lookup("graceful-period"))
	_ = v.BindEnv("graceful.period", "CONTENT_SERVER_GRACEFULE_PERIOD")
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

func servicePProfEnabledFlag(v *viper.Viper) bool {
	return v.GetBool("service.pprof.enabled")
}

func addServicePProfEnabledFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.Bool("service-pprof-enabled", false, "Enable pprof service")
	_ = v.BindPFlag("service.pprof.enabled", flags.Lookup("service-pprof-enabled"))
}

func otelEnabledFlag(v *viper.Viper) bool {
	return v.GetBool("otel.enabled")
}

func addOtelEnabledFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.Bool("otel-enabled", false, "Enable otel service")
	_ = v.BindPFlag("otel.enabled", flags.Lookup("otel-enabled"))
	_ = v.BindEnv("otel.enabled", "OTEL_ENABLED")
}

func storageTypeFlag(v *viper.Viper) string {
	return v.GetString("storage.type")
}

func addStorageTypeFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.String("storage-type", "filesystem", "Storage backend type: filesystem or gcs")
	_ = v.BindPFlag("storage.type", flags.Lookup("storage-type"))
	_ = v.BindEnv("storage.type", "CONTENT_SERVER_STORAGE_TYPE")
}

func storageGCSBucketFlag(v *viper.Viper) string {
	return v.GetString("storage.gcs.bucket")
}

func addStorageGCSBucketFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.String("storage-gcs-bucket", "", "GCS bucket URL (e.g., gs://my-bucket)")
	_ = v.BindPFlag("storage.gcs.bucket", flags.Lookup("storage-gcs-bucket"))
	_ = v.BindEnv("storage.gcs.bucket", "CONTENT_SERVER_STORAGE_GCS_BUCKET")
}

func storageGCSPrefixFlag(v *viper.Viper) string {
	return v.GetString("storage.gcs.prefix")
}

func addStorageGCSPrefixFlag(flags *pflag.FlagSet, v *viper.Viper) {
	flags.String("storage-gcs-prefix", "", "GCS object prefix (e.g., contentserver/)")
	_ = v.BindPFlag("storage.gcs.prefix", flags.Lookup("storage-gcs-prefix"))
	_ = v.BindEnv("storage.gcs.prefix", "CONTENT_SERVER_STORAGE_GCS_PREFIX")
}
