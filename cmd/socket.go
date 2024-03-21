package cmd

import (
	"context"
	"net"

	"github.com/foomo/contentserver/pkg/handler"
	"github.com/foomo/contentserver/pkg/repo"
	keelhttp "github.com/foomo/keel/net/http"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func NewSocketCommand() *cobra.Command {
	v := viper.New()
	cmd := &cobra.Command{
		Use:   "socket <url>",
		Short: "Start socket server",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			var comps []string
			if len(args) == 0 {
				comps = cobra.AppendActiveHelp(comps, "You must specify the URL for the repository you are adding")
			} else {
				comps = cobra.AppendActiveHelp(comps, "This command does not take any more arguments")
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			l := logger

			r := repo.New(l,
				args[0],
				repo.NewHistory(l,
					repo.HistoryWithVarDir(v.GetString("history.dir")),
					repo.HistoryWithMax(v.GetInt("history.limit")),
				),
				repo.WithHTTPClient(
					keelhttp.NewHTTPClient(
						keelhttp.HTTPClientWithTelemetry(),
					),
				),
				repo.WithPollForUpdates(v.GetBool("poll")),
			)

			// create socket server
			handle := handler.NewSocket(l, r)

			// listen on socket
			ln, err := net.Listen("tcp", v.GetString("address"))
			if err != nil {
				return err
			}

			// start repo
			up := make(chan bool, 1)
			r.OnStart(func() {
				up <- true
			})
			go r.Start(context.Background()) //nolint:errcheck
			<-up

			l.Info("started listening", zap.String("address", v.GetString("address")))

			for {
				// this blocks until connection or error
				conn, err := ln.Accept()
				if err != nil {
					l.Error("runSocketServer: could not accept connection", zap.Error(err))
					continue
				}

				// a goroutine handles conn so that the loop can accept other connections
				go func() {
					l.Debug("accepted connection", zap.String("source", conn.RemoteAddr().String()))
					handle.Serve(conn)
					if err := conn.Close(); err != nil {
						l.Warn("failed to close connection", zap.Error(err))
					}
				}()
			}
		},
	}

	addAddressFlag(cmd, v)
	addPollFlag(cmd, v)
	addHistoryDirFlag(cmd, v)
	addHistoryLimitFlag(cmd, v)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewSocketCommand())
}
