package cmd

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestRepositoryLogLevelFlags(t *testing.T) {
	for _, flag := range []struct {
		name         string
		env          string
		defaultLevel zapcore.Level
		add          func(*pflag.FlagSet, *viper.Viper)
		get          func(*viper.Viper) (zapcore.Level, error)
	}{
		{"log-level-missing-node", "LOG_LEVEL_MISSING_NODE", zapcore.ErrorLevel, addLogLevelMissingNodeFlag, logLevelMissingNodeFlag},
		{"log-level-resolved", "LOG_LEVEL_RESOLVED", zapcore.InfoLevel, addLogLevelResolvedFlag, logLevelResolvedFlag},
	} {
		t.Run(flag.name, func(t *testing.T) {
			t.Setenv(flag.env, "")

			v := viper.New()
			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flag.add(flags, v)
			level, err := flag.get(v)
			require.NoError(t, err)
			require.Equal(t, flag.defaultLevel, level)

			for _, tc := range []struct {
				value string
				level zapcore.Level
			}{
				{"DEBUG", zapcore.DebugLevel}, {"debug", zapcore.DebugLevel}, {"DeBuG", zapcore.DebugLevel},
				{"INFO", zapcore.InfoLevel}, {"info", zapcore.InfoLevel}, {"InFo", zapcore.InfoLevel},
				{"WARN", zapcore.WarnLevel}, {"warn", zapcore.WarnLevel}, {"WaRn", zapcore.WarnLevel},
				{"ERROR", zapcore.ErrorLevel}, {"error", zapcore.ErrorLevel}, {"ErRoR", zapcore.ErrorLevel},
			} {
				t.Run(tc.value, func(t *testing.T) {
					require.NoError(t, flags.Set(flag.name, tc.value))
					level, err := flag.get(v)
					require.NoError(t, err)
					require.Equal(t, tc.level, level)
				})
			}

			for _, value := range []string{"", "invalid", "WARNING", "DPANIC", "PANIC", "FATAL", "-1", " INFO "} {
				t.Run("invalid_"+value, func(t *testing.T) {
					require.NoError(t, flags.Set(flag.name, value))
					_, err := flag.get(v)
					require.ErrorContains(t, err, "invalid --"+flag.name+" value")
					require.ErrorContains(t, err, "expected DEBUG, INFO, WARN or ERROR")
				})
			}
		})

		t.Run(flag.name+"_environment_precedence", func(t *testing.T) {
			t.Setenv(flag.env, "WaRn")

			v := viper.New()
			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flag.add(flags, v)
			level, err := flag.get(v)
			require.NoError(t, err)
			require.Equal(t, zapcore.WarnLevel, level)

			t.Setenv(flag.env, "FATAL")
			_, err = flag.get(v)
			require.Error(t, err)

			require.NoError(t, flags.Set(flag.name, flag.defaultLevel.CapitalString()))
			level, err = flag.get(v)
			require.NoError(t, err)
			require.Equal(t, flag.defaultLevel, level)
		})
	}
}
