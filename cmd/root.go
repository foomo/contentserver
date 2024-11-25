package cmd

import (
	"strings"

	"github.com/foomo/keel/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// NewRootCommand represents the base command when called without any subcommands
func NewRootCommand() *cobra.Command {
	v := newViper()
	cmd := &cobra.Command{
		Use:   "contentserver",
		Short: "Serves content tree structures very quickly",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			zap.ReplaceGlobals(log.NewLogger(
				logLevelFlag(v),
				logFormatFlag(v),
			))
		},
	}

	addLogLevelFlag(cmd.PersistentFlags(), v)
	addLogFormatFlag(cmd.PersistentFlags(), v)

	cmd.AddCommand(NewHTTPCommand())
	cmd.AddCommand(NewSocketCommand())
	cmd.AddCommand(NewVersionCommand())

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		log.Logger().Fatal("failed to run command", zap.Error(err))
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.EnvKeyReplacer(strings.NewReplacer(".", "_"))
}

func newViper() *viper.Viper {
	v := viper.New()
	v.AutomaticEnv()
	return v
}
