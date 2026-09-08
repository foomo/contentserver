package handler

import (
	"bytes"
	"errors"

	"github.com/foomo/contentserver/requests"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func decodeRequest(data []byte, request any) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("request must not be null")
	}

	return json.Unmarshal(data, request)
}

func validateNodesRequest(request *requests.Nodes) error {
	for name, node := range request.Nodes {
		if name == "" {
			continue
		}

		if node == nil {
			return errors.New("request node must not be nil")
		}

		if node.ID != "" && request.Env == nil {
			return errors.New("request.Env must not be nil")
		}
	}

	return nil
}

func invalidNodes(nodes map[string]*requests.Node) bool {
	for name, node := range nodes {
		if name == "" || node == nil || node.ID == "" {
			return true
		}
	}

	return false
}
