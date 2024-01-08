package common

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	values map[string]any
}

// LoadConfig allows to customize parameters instead of hard-coding them. Always use this function instead of
// hard-coding constants.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := make(map[string]any)
	err = yaml.Unmarshal(data, &values)
	if err != nil {
		return nil, err
	}
	return &Config{values: values}, nil
}

// GetString returns a string-typed parameter. If nothing is found, or if the value cannot be parsed as a string,
// returns an empty value.
func (c *Config) GetString(key string) string {
	value, ok := c.values[key]
	if !ok {
		return ""
	}
	str, ok := value.(string)
	if !ok {
		return ""
	}
	return str
}

// GetStringOrDefault returns a string-typed parameter. If nothing is found, or if the value cannot be parsed as a string,
// returns `defaultValue`.
func (c *Config) GetStringOrDefault(key, defaultValue string) string {
	value := c.GetString(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetIntOrDefault returns an integer-typed parameter. If nothing is found, or if the value cannot be parsed as an integer,
// returns `defaultValue`.
func (c *Config) GetIntOrDefault(key string, defaultValue int) int {
	value, ok := c.values[key]
	if !ok {
		return defaultValue
	}
	intValue, ok := value.(int)
	if !ok {
		return defaultValue
	}
	return intValue
}

// GetFloatOrDefault returns a float-typed parameter. If nothing is found, or if the value cannot be parsed as a float,
// returns `defaultValue`.
func (c *Config) GetFloatOrDefault(key string, defaultValue float64) float64 {
	value, ok := c.values[key]
	if !ok {
		return defaultValue
	}
	floatValue, ok := value.(float64)
	if !ok {
		return defaultValue
	}
	return floatValue
}

// GetDurationOrDefault returns a duration-typed parameter. If nothing is found, or if the value cannot be parsed as a duration
// (i.e. an integer which specifies milliseconds), returns `defaultValue`.
func (c *Config) GetDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	intValue := c.GetIntOrDefault(key, -1)
	if intValue < 0 {
		return defaultValue
	}
	return time.Duration(intValue) * time.Millisecond
}
