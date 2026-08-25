package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Duration is a custom type for time.Duration that supports JSON unmarshaling from string
type Duration time.Duration

// UnmarshalJSON implements json.Unmarshaler interface
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration format %q: %w", s, err)
	}

	*d = Duration(duration)
	return nil
}

// MarshalJSON implements json.Marshaler interface
func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(*d).String())
}

// ToDuration converts custom Duration to time.Duration
func (d *Duration) ToDuration() time.Duration {
	return time.Duration(*d)
}

// String implements flag.Value interface (and fmt.Stringer)
func (d *Duration) String() string {
	return time.Duration(*d).String()
}

// Set implements flag.Value interface
// Accepts "5" (seconds) or "5s" (Go duration format)
func (d *Duration) Set(value string) error {
	if seconds, err := strconv.Atoi(value); err == nil {
		*d = Duration(time.Duration(seconds) * time.Second)
		return nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: expected seconds (e.g. \"5\") or Go duration (e.g. \"5s\", \"100ms\")", value)
	}

	*d = Duration(duration)
	return nil
}
