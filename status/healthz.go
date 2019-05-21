package status

import (
	"encoding/json"
	"fmt"
	"github.com/foomo/contentserver/log"
	"net/http"
)

func RunHealthzHandlerListener(address string, serviceName string) {
	log.Notice(fmt.Sprintf("starting healthz handler on '%s'" + address))
	log.Error(http.ListenAndServe(address, HealthzHandler(serviceName)))
}

func HealthzHandler(serviceName string) http.Handler {
	data := map[string]string{
		"service": serviceName,
	}
	status, _ := json.Marshal(data)
	h := http.NewServeMux()
	h.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write(status)
	}))

	return h
}
