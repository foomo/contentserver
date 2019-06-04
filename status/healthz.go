package status

import (
	"fmt"
	"net/http"

	. "github.com/foomo/contentserver/logger"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func RunHealthzHandlerListener(address string, serviceName string) {
	Log.Info(fmt.Sprintf("starting healthz handler on '%s'" + address))
	Log.Error("healthz server failed", zap.Error(http.ListenAndServe(address, HealthzHandler(serviceName))))
}

func HealthzHandler(serviceName string) http.Handler {
	var (
		data = map[string]string{
			"service": serviceName,
		}
		status, _ = json.Marshal(data)
		h         = http.NewServeMux()
	)
	h.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, err := w.Write(status)
		if err != nil {
			Log.Error("failed to write healthz status", zap.Error(err))
		}
	}))

	return h
}
