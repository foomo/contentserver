package cmd

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCommandsRejectInvalidLogLevelsBeforeStorage(t *testing.T) {
	for _, command := range []struct {
		name string
		new  func() *cobra.Command
	}{
		{"http", NewHTTPCommand},
		{"socket", NewSocketCommand},
	} {
		t.Run(command.name, func(t *testing.T) {
			t.Setenv("LOG_LEVEL_MISSING_NODE", "")
			t.Setenv("LOG_LEVEL_RESOLVED", "")

			cmd := command.new()
			require.Equal(t, "ERROR", cmd.Flags().Lookup("log-level-missing-node").DefValue)
			require.Equal(t, "INFO", cmd.Flags().Lookup("log-level-resolved").DefValue)

			for _, flag := range []struct {
				name string
				env  string
			}{
				{"log-level-missing-node", "LOG_LEVEL_MISSING_NODE"},
				{"log-level-resolved", "LOG_LEVEL_RESOLVED"},
			} {
				for _, source := range []string{"flag", "environment"} {
					t.Run(flag.name+"_"+source, func(t *testing.T) {
						cmd := command.new()
						cmd.SetOut(io.Discard)
						cmd.SetErr(io.Discard)

						args := []string{"--storage-type=invalid-storage-canary", "http://example.invalid/repo.json"}
						if source == "flag" {
							args = append(args, "--"+flag.name+"=FATAL")
						} else {
							t.Setenv(flag.env, "FATAL")
						}

						cmd.SetArgs(args)

						err := cmd.Execute()
						require.ErrorContains(t, err, "invalid --"+flag.name+" value")
						require.NotContains(t, err.Error(), "storage")
					})
				}
			}
		})
	}
}
