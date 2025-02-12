package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func AssertMetricsCounterEquals(t *testing.T, expected int, c prometheus.Counter) {
	assert.InDelta(t, float64(expected), promtestutil.ToFloat64(c), 0.01)
}

func AssertCounterEqualsInt(t *testing.T, expected int, c prometheus.Counter) {
	assert.Equal(t, expected, GetCounterInt(c))
}

func GetCounterInt(c prometheus.Counter) int {
	return int(promtestutil.ToFloat64(c))
}

func AssertCounterGreaterOrEqualsInt(t *testing.T, threshold int, c prometheus.Counter) {
	assert.GreaterOrEqual(t, GetCounterInt(c), threshold)
}

func AssertMetricsGaugeEquals(t *testing.T, expected int, g prometheus.Gauge, msgAndArgs ...interface{}) {
	assert.InDelta(t, float64(expected), promtestutil.ToFloat64(g), 0.01, msgAndArgs...)
}
