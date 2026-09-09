package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateTimestampsExportedAtStartup(t *testing.T) {
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	for _, name := range []string{
		"contentserver_last_successful_update_timestamp_seconds",
		"contentserver_last_failed_update_timestamp_seconds",
	} {
		found := false

		for _, family := range families {
			if family.GetName() != name {
				continue
			}

			found = true

			require.Len(t, family.GetMetric(), 1)
			metric := family.GetMetric()[0]
			require.NotNil(t, metric.Gauge)
			assert.Zero(t, metric.GetGauge().GetValue())
			assert.Empty(t, metric.GetLabel())
		}

		assert.True(t, found, "metric %s must be exported before its first event", name)
	}
}
