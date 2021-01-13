package sample

import "time"

// Sample represent one measurement.
type Sample struct {
	ResponseDuration time.Duration
	Status           int
	Timestamp        time.Time
}

// NewSample returns a pointer to a new sample.
func NewSample(statusCode int, timestamp time.Time) *Sample {
	// TODO Should we directly convert the timestamp into a string with the proper format?
	return &Sample{
		ResponseDuration: time.Since(timestamp),
		Status:           statusCode,
		Timestamp:        timestamp,
	}
}
