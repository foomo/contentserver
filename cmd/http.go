package cmd

import (
	"context"
	"errors"

	"github.com/foomo/contentserver/pkg/handler"
	"github.com/foomo/contentserver/pkg/repo"
	"github.com/foomo/keel"
	"github.com/foomo/keel/healthz"
	keelhttp "github.com/foomo/keel/net/http"
	"github.com/foomo/keel/net/http/middleware"
	"github.com/foomo/keel/service"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func NewHTTPCommand() *cobra.Command {
	v := newViper()
	cmd := &cobra.Command{
		Use:   "http <url>",
		Short: "Start http server",
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
			svr := keel.NewServer(
				keel.WithHTTPReadmeService(true),
				keel.WithHTTPPrometheusService(servicePrometheusEnabledFlag(v)),
				keel.WithHTTPHealthzService(serviceHealthzEnabledFlag(v)),
				keel.WithPrometheusMeter(servicePrometheusEnabledFlag(v)),
				keel.WithOTLPGRPCTracer(otelEnabledFlag(v)),
				keel.WithGracefulTimeout(gracefulTimeoutFlag(v)),
				keel.WithShutdownTimeout(shutdownTimeoutFlag(v)),
			)

			l := svr.Logger()

			r := repo.New(l,
				args[0],
				repo.NewHistory(l,
					repo.HistoryWithHistoryDir(historyDirFlag(v)),
					repo.HistoryWithHistoryLimit(historyLimitFlag(v)),
				),
				repo.WithHTTPClient(
					keelhttp.NewHTTPClient(
						keelhttp.HTTPClientWithTelemetry(),
					),
				),
				repo.WithPollInterval(pollIntevalFlag(v)),
				repo.WithPoll(pollFlag(v)),
			)

			// start initial update and handle error
			svr.AddReadinessHealthzers(healthz.NewHealthzerFn(func(ctx context.Context) error {
				if !r.Loaded() {
					return errors.New("repo not ready yet")
				}
				return nil
			}))

			svr.AddServices(
				service.NewGoRoutine(l, "repo", func(ctx context.Context, l *zap.Logger) error {
					return r.Start(ctx)
				}),
				service.NewHTTP(l, "http", addressFlag(v),
					handler.NewHTTP(l, r, handler.WithBasePath(basePathFlag(v))),
					middleware.Telemetry(),
					middleware.Logger(),
					middleware.Recover(),
				),
			)

			svr.Run()
			return nil
		},
	}

	flags := cmd.Flags()
	addAddressFlag(flags, v)
	addBasePathFlag(flags, v)
	addPollFlag(flags, v)
	addPollIntervalFlag(flags, v)
	addHistoryDirFlag(flags, v)
	addHistoryLimitFlag(flags, v)
	addGracefulTimeoutFlag(flags, v)
	addShutdownTimeoutFlag(flags, v)
	addOtelEnabledFlag(flags, v)
	addServiceHealthzEnabledFlag(flags, v)
	addServicePrometheusEnabledFlag(flags, v)

	return cmd
}
