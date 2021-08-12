package configuration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetBool(t *testing.T) {
	t.Run("when value is nil use default value", func(t *testing.T) {
		// when
		res := GetBool(nil, true)

		// then
		assert.True(t, res)

		t.Run("different default value", func(t *testing.T) {
			// when
			res := GetBool(nil, false)

			// then
			assert.False(t, res)
		})
	})

	t.Run("use value when provided", func(t *testing.T) {
		// given
		v := false
		// when
		res := GetBool(&v, true)
		// then
		assert.False(t, res)
	})
}

func TestGetInt(t *testing.T) {
	t.Run("when value is nil use default value", func(t *testing.T) {
		// when
		res := GetInt(nil, 5)
		// then
		assert.Equal(t, 5, res)
	})
	t.Run("use value when provided", func(t *testing.T) {
		// given
		v := 10
		// when
		res := GetInt(&v, 5)
		// then
		assert.Equal(t, 10, res)
	})
}

func TestGetInt32(t *testing.T) {
	t.Run("when value is nil use default value", func(t *testing.T) {
		// when
		res := GetInt32(nil, 5)
		// then
		assert.Equal(t, int32(5), res)
	})
	t.Run("use value when provided", func(t *testing.T) {
		// given
		v := int32(10)
		// when
		res := GetInt32(&v, 5)
		// then
		assert.Equal(t, int32(10), res)
	})
}

func TestGetString(t *testing.T) {
	t.Run("when value is nil use default value", func(t *testing.T) {
		// when
		res := GetString(nil, "defValue")
		// then
		assert.Equal(t, "defValue", res)
	})
	t.Run("use value when provided", func(t *testing.T) {
		// given
		v := "providedValue"
		// when
		res := GetString(&v, "defValue")
		// then
		assert.Equal(t, "providedValue", res)

	})
}

func TestGetDuration(t *testing.T) {
	t.Run("use default value when provided value is nil", func(t *testing.T) {
		// when
		res := GetDuration(nil, 5*time.Minute)
		// then
		assert.Equal(t, 5*time.Minute, res)
	})
	t.Run("use default value when provided value is invalid", func(t *testing.T) {
		// given
		v := "invalid"
		// when
		res := GetDuration(&v, 5*time.Minute)
		// then
		assert.Equal(t, 5*time.Minute, res)
	})
	t.Run("use value when provided", func(t *testing.T) {
		// given
		v := "10m"
		// when
		res := GetDuration(&v, 5*time.Minute)
		// then
		assert.Equal(t, 10*time.Minute, res)
	})
}

func TestCopyOfMap(t *testing.T) {
	// given
	originalMap := map[string]map[string]string{
		"Canada": {
			"Toronto":   "Ontario",
			"Vancouver": "British Columbia",
			"Winniped":  "Manitoba",
		},
		"USA": {
			"New York":      "New York",
			"Chicago":       "Illinois",
			"San Francisco": "California",
			"Seattla":       "Washington",
		},
	}

	// when
	res := CopyOf(originalMap)

	// then
	assert.Equal(t, res, originalMap)
}
