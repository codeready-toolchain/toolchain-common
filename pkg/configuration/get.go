package configuration

import "time"

func GetBool(value *bool, defaultValue bool) bool {
	if value != nil {
		return *value
	}
	return defaultValue
}

func GetInt(value *int, defaultValue int) int {
	if value != nil {
		return *value
	}
	return defaultValue
}

func GetUint(value *uint, defaultValue uint) uint {
	if value != nil {
		return *value
	}
	return defaultValue
}

func GetInt32(value *int32, defaultValue int32) int32 {
	if value != nil {
		return *value
	}
	return defaultValue
}

func GetString(value *string, defaultValue string) string {
	if value != nil {
		return *value
	}
	return defaultValue
}

// GetDuration parses the given value as a Duration and returns the value.
// The default value is returned if the value is nil or cannot be parsed as a duration.
func GetDuration(value *string, defaultValue time.Duration) time.Duration {
	durationAsString := GetString(value, "invalid value")
	d, err := time.ParseDuration(durationAsString)
	if err != nil {
		return defaultValue
	}
	return d
}

func CopyOf(originalMap map[string]map[string]string) map[string]map[string]string {
	targetMap := make(map[string]map[string]string, len(originalMap))
	for key, value := range originalMap {
		innerMap := make(map[string]string, len(value))
		for k, v := range value {
			innerMap[k] = v
		}
		targetMap[key] = innerMap
	}
	return targetMap
}
