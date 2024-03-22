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
				keel.WithHTTPPrometheusService(v.GetBool("service.prometheus.enabled")),
				keel.WithHTTPHealthzService(v.GetBool("service.healthz.enabled")),
				keel.WithPrometheusMeter(v.GetBool("service.prometheus.enabled")),
				keel.WithOTLPGRPCTracer(v.GetBool("otel.enabled")),
				keel.WithGracefulTimeout(v.GetDuration("graceful.timeout")),
				keel.WithShutdownTimeout(v.GetDuration("shutdown.timeout")),
			)

			l := svr.Logger()

			l.Error("test")

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

			// start initial update and handle error
			svr.AddReadinessHealthzers(healthz.NewHealthzerFn(func(ctx context.Context) error {
				if !r.Loaded() {
					return errors.New("repo not ready yet")
				}
				return nil
			}))

			svr.AddServices(
				service.NewHTTP(l, "http", v.GetString("address"),
					handler.NewHTTP(l, r, handler.WithPath(v.GetString("path"))),
					middleware.Telemetry(),
					middleware.Logger(),
					middleware.Recover(),
				),
				service.NewGoRoutine(l, "repo", func(ctx context.Context, l *zap.Logger) error {
					return r.Start(ctx)
				}),
			)

			svr.Run()
			return nil
		},
	}

	addAddressFlag(cmd, v)
	addBasePathFlag(cmd, v)
	addPollFlag(cmd, v)
	addHistoryDirFlag(cmd, v)
	addHistoryLimitFlag(cmd, v)
	addGracefulTimeoutFlag(cmd, v)
	addShutdownTimeoutFlag(cmd, v)
	addOtelEnabledFlag(cmd, v)
	addServiceHealthzEnabledFlag(cmd, v)
	addServicePrometheusEnabledFlag(cmd, v)

	return cmd
}
