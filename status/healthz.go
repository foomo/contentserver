package status

import (
	"fmt"
	"net/http"

	"github.com/foomo/contentserver/log"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func RunHealthzHandlerListener(address string, serviceName string) {
	log.Notice(fmt.Sprintf("starting healthz handler on '%s'" + address))
	log.Error(http.ListenAndServe(address, HealthzHandler(serviceName)))
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
			log.Error("failed to write healthz status: ", err)
		}
	}))

	return h
}
