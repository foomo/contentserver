package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/foomo/contentserver/pkg/handler"
	"github.com/foomo/contentserver/pkg/repo"
	"github.com/foomo/keel"
	"github.com/foomo/keel/healthz"
	keelhttp "github.com/foomo/keel/net/http"
	"github.com/foomo/keel/net/http/middleware"
	"github.com/foomo/keel/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
				keel.WithHTTPPrometheusService(servicePrometheusEnabledFlag(v)),
				keel.WithHTTPHealthzService(serviceHealthzEnabledFlag(v)),
				keel.WithPrometheusMeter(servicePrometheusEnabledFlag(v)),
				keel.WithGracefulPeriod(gracefulPeriodFlag(v)),
				keel.WithOTLPGRPCTracer(otelEnabledFlag(v)),
				keel.WithHTTPPProfService(true),
			)

			l := svr.Logger()

			// Create storage based on configuration
			storage, err := createStorage(cmd.Context(), v, l)
			if err != nil {
				return fmt.Errorf("failed to create storage: %w", err)
			}

			history, err := repo.NewHistory(l.Named("inst.history"),
				repo.HistoryWithStorage(storage),
				repo.HistoryWithHistoryDir(historyDirFlag(v)),
				repo.HistoryWithHistoryLimit(historyLimitFlag(v)),
			)
			if err != nil {
				return fmt.Errorf("failed to create history: %w", err)
			}

			r := repo.New(l.Named("inst.repo"),
				args[0],
				history,
				repo.WithHTTPClient(
					keelhttp.NewHTTPClient(
						keelhttp.HTTPClientWithTelemetry(),
					),
				),
				repo.WithPollInterval(pollIntevalFlag(v)),
				repo.WithPoll(pollFlag(v)),
			)

			isLoadedHealtherFn := healthz.NewHealthzerFn(func(ctx context.Context) error {
				if !r.Loaded() {
					return errors.New("repo not loaded yet")
				}
				return nil
			})
			// start initial update and handle error
			svr.AddStartupHealthzers(isLoadedHealtherFn)
			svr.AddReadinessHealthzers(isLoadedHealtherFn)

			svr.AddServices(
				service.NewGoRoutine(l.Named("go.repo"), "repo", func(ctx context.Context, l *zap.Logger) error {
					return r.Start(ctx)
				}),
				service.NewHTTP(l.Named("svc.http"), "http", addressFlag(v),
					handler.NewHTTP(l.Named("inst.handler"), r, handler.WithBasePath(basePathFlag(v))),
					middleware.Telemetry(),
					middleware.Logger(),
					middleware.GZip(),
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
	addShutdownTimeoutFlag(flags, v)
	addOtelEnabledFlag(flags, v)
	addServiceHealthzEnabledFlag(flags, v)
	addServicePrometheusEnabledFlag(flags, v)
	addStorageTypeFlag(flags, v)
	addStorageGCSBucketFlag(flags, v)
	addStorageGCSPrefixFlag(flags, v)

	return cmd
}

// createStorage creates a storage backend based on the configuration
func createStorage(ctx context.Context, v *viper.Viper, l *zap.Logger) (repo.Storage, error) {
	storageType := storageTypeFlag(v)
	gcsBucket := storageGCSBucketFlag(v)
	gcsPrefix := storageGCSPrefixFlag(v)

	// Warn about ignored GCS config
	if storageType != "gcs" && (gcsBucket != "" || gcsPrefix != "") {
		l.Warn("GCS configuration flags are set but storage-type is not 'gcs'; GCS config will be ignored",
			zap.String("storage-type", storageType),
			zap.String("gcs-bucket", gcsBucket),
			zap.String("gcs-prefix", gcsPrefix),
		)
	}

	l.Info("creating storage", zap.String("type", storageType))

	switch storageType {
	case "gcs":
		if gcsBucket == "" {
			return nil, fmt.Errorf("GCS bucket URL is required for gcs storage type")
		}
		l.Info("using GCS storage", zap.String("bucket", gcsBucket), zap.String("prefix", gcsPrefix))
		return repo.NewBlobStorage(ctx, gcsBucket, gcsPrefix)
	case "filesystem", "":
		dir := historyDirFlag(v)
		l.Info("using filesystem storage", zap.String("dir", dir))
		return repo.NewFilesystemStorage(dir)
	default:
		return nil, fmt.Errorf("unknown storage type: %s", storageType)
	}
}
