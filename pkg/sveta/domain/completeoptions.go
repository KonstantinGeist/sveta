package domain

var DefaultCompleteOptions = CompleteOptions{}

type CompleteOptions struct {
	// JSONMode makes sure the output will be a syntactically valid JSON (grammar-restricted completion)
	JSONMode bool
	// Temperature specifies
	Temperature float64
}

func (c CompleteOptions) WithJSONMode(value bool) CompleteOptions {
	c.JSONMode = value
	return c
}

func (c CompleteOptions) WithTemperature(value float64) CompleteOptions {
	c.Temperature = value
	return c
}

func (c CompleteOptions) TemperatureOrDefault(defaultValue float64) float64 {
	if c.Temperature == 0.0 {
		return defaultValue
	}
	return c.Temperature
}
