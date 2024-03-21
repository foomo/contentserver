package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func addAddressFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().String("address", "localhost:8080", "Address to bind to (host:port)")
	_ = v.BindPFlag("address", cmd.Flags().Lookup("address"))
}

func addBasePathFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().String("base-path", "/contentserver", "Base path to export the webserver on")
	_ = v.BindPFlag("base_path", cmd.Flags().Lookup("base_path"))
}

func addPollFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Bool("poll", false, "If true, the address arg will be used to periodically poll the content url")
	_ = v.BindPFlag("poll", cmd.Flags().Lookup("poll"))
}

func addHistoryDirFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().String("history-dir", "/var/lib/contentserver", "Where to put my data")
	_ = v.BindPFlag("history.dir", cmd.Flags().Lookup("history-dir"))
	_ = v.BindEnv("history.dir", "CONTENT_SERVER_HISTORY_DIR")
}

func addHistoryLimitFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Int("history-limit", 2, "Number of history records to keep")
	_ = v.BindPFlag("history.limit", cmd.Flags().Lookup("history-limit"))
}

func addGracefulTimeoutFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Duration("graceful-timeout", 0, "Timeout duration for graceful shutdown")
	_ = v.BindPFlag("graceful.timeout", cmd.Flags().Lookup("graceful-timeout"))
}

func addShutdownTimeoutFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Duration("shutdown-timeout", 0, "Timeout duration for shutdown")
	_ = v.BindPFlag("shutdown.timeout", cmd.Flags().Lookup("shutdown-timeout"))
}

func addOtelEnabledFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Bool("otel-enabled", false, "Enable otel service")
	_ = v.BindPFlag("otel.enabled", cmd.Flags().Lookup("otel-enabled"))
}

func addHealthzEnabledFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Bool("healthz-enabled", false, "Enable healthz service")
	_ = v.BindPFlag("healthz.enabled", cmd.Flags().Lookup("healthz-enabled"))
}

func addPrometheusEnabledFlag(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().Bool("prometheus-enabled", false, "Enable prometheus service")
	_ = v.BindPFlag("prometheus.enabled", cmd.Flags().Lookup("prometheus-enabled"))
}
