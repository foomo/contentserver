package server

import (
	"github.com/foomo/contentserver/server/repo"
)

// our data

type Stats struct {
	requests int64
}

func NewStats() *Stats {
	stats := new(Stats)
	stats.requests = 0
	return stats
}

var stats *Stats = NewStats()
var contentRepo *repo.Repo

func countRequest() {
	stats.requests++
}

func numRequests() int64 {
	return stats.requests
}
