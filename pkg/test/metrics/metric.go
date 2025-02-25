package metrics

import (
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"
	promclientgo "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func AssertHistogramBucketEquals(t *testing.T, expected, bucket int, h prometheus.Histogram, msgAndArgs ...interface{}) {
	metric := promclientgo.Metric{}
	err := h.Write(&metric)
	require.NoError(t, err)
	for _, buck := range metric.GetHistogram().GetBucket() {
		if buck.GetUpperBound() == float64(bucket) {
			assert.Equal(t, uint64(expected), buck.GetCumulativeCount(), msgAndArgs...) // nolint:gosec // (expected value should be always non-negative)
			return
		}
	}
	assert.Fail(t, fmt.Sprintf("the bucket with the upper limit '%d' wasn't found, actual: %v", bucket, metric.GetHistogram().GetBucket()), msgAndArgs...)
}

func AssertAllHistogramBucketsAreEmpty(t *testing.T, h prometheus.Histogram, msgAndArgs ...interface{}) {
	metric := promclientgo.Metric{}
	err := h.Write(&metric)
	require.NoError(t, err)
	for _, buck := range metric.GetHistogram().GetBucket() {
		assert.Empty(t, buck.GetCumulativeCount(), msgAndArgs...)
	}
}
