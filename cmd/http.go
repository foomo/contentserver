package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	// TODO: When keel is updated, set it in the correct place
	service.DefaultHTTPPProfAddr = ":6060"

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
				keel.WithHTTPPProfService(servicePProfEnabledFlag(v)),
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
						keelhttp.HTTPClientWithTimeout(repositoryTimeoutFlag(v)),
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

			svr.AddClosers(func(ctx context.Context) error {
				return history.Close()
			})

			svr.AddServices(
				service.NewGoRoutine(l.Named("go.repo"), "repo", func(ctx context.Context, l *zap.Logger) error {
					return r.Start(ctx)
				}),
				service.NewHTTP(l.Named("svc.http"), "http", addressFlag(v),
					handler.NewHTTP(l.Named("inst.handler"), r, handler.WithBasePath(basePathFlag(v))),
					middleware.Telemetry(),
					middleware.Logger(),
					middleware.GZip(middleware.GZipWithLevel(gzipLevelFlag(v))),
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
	addServicePProfEnabledFlag(flags, v)
	addStorageTypeFlag(flags, v)
	addStorageBlobBucketFlag(flags, v)
	addStorageBlobPrefixFlag(flags, v)
	addRepositoryTimeoutFlag(flags, v)
	addGzipLevelFlag(flags, v)

	return cmd
}

// supportedBlobSchemes lists the URL schemes supported by blob storage
var supportedBlobSchemes = []string{"gs://", "s3://", "azblob://"}

// createStorage creates a storage backend based on the configuration
func createStorage(ctx context.Context, v *viper.Viper, l *zap.Logger) (repo.Storage, error) {
	storageType := storageTypeFlag(v)
	blobBucket := storageBlobBucketFlag(v)
	blobPrefix := storageBlobPrefixFlag(v)

	// Warn about ignored blob config
	if storageType != "blob" && (blobBucket != "" || blobPrefix != "") {
		l.Warn("blob storage flags are set but storage-type is not 'blob'; blob config will be ignored",
			zap.String("storage-type", storageType),
			zap.String("blob-bucket", blobBucket),
			zap.String("blob-prefix", blobPrefix),
		)
	}

	l.Info("creating storage", zap.String("type", storageType))

	switch storageType {
	case "blob":
		if blobBucket == "" {
			return nil, fmt.Errorf("blob bucket URL is required when storage-type is 'blob' (supported schemes: gs://, s3://, azblob://)")
		}
		if !isValidBlobScheme(blobBucket) {
			return nil, fmt.Errorf("unsupported blob storage URL scheme in %q; supported schemes: gs://, s3://, azblob://", blobBucket)
		}
		l.Info("using blob storage",
			zap.String("bucket", blobBucket),
			zap.String("prefix", blobPrefix),
			zap.String("provider", detectBlobProvider(blobBucket)),
		)
		return repo.NewBlobStorage(ctx, blobBucket, blobPrefix)
	case "filesystem", "":
		dir := historyDirFlag(v)
		l.Info("using filesystem storage", zap.String("dir", dir))
		return repo.NewFilesystemStorage(dir)
	default:
		return nil, fmt.Errorf("unknown storage type: %s (supported: filesystem, blob)", storageType)
	}
}

// isValidBlobScheme checks if the bucket URL has a supported scheme
func isValidBlobScheme(bucketURL string) bool {
	for _, scheme := range supportedBlobSchemes {
		if strings.HasPrefix(bucketURL, scheme) {
			return true
		}
	}
	return false
}

// detectBlobProvider returns a human-readable provider name from the URL scheme
func detectBlobProvider(bucketURL string) string {
	switch {
	case strings.HasPrefix(bucketURL, "gs://"):
		return "Google Cloud Storage"
	case strings.HasPrefix(bucketURL, "s3://"):
		return "AWS S3"
	case strings.HasPrefix(bucketURL, "azblob://"):
		return "Azure Blob Storage"
	default:
		return "unknown"
	}
}
