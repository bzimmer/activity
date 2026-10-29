package garmin

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/martinlindhe/unit"
)

// Fault is an error returned by the Garmin API
type Fault struct { //nolint:errname // convention
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	// RetryAfter is the wait a 429 asked for, zero when it gave none
	RetryAfter time.Duration `json:"-"`
}

func (f *Fault) Error() string {
	return f.Message
}

// HTTPStatusCode returns the HTTP status code of the response which produced this Fault.
func (f *Fault) HTTPStatusCode() int {
	return f.StatusCode
}

// SetDefaults populates StatusCode and Message from the HTTP response when the body does not supply them.
func (f *Fault) SetDefaults(code int, message string) {
	if f.StatusCode == 0 {
		f.StatusCode = code
	}
	if f.Message == "" {
		f.Message = message
	}
}

// Time is one of Garmin's zoneless "2006-01-02 15:04:05" timestamps,
// parsed as UTC; zero when null or empty.
type Time struct {
	time.Time
}

// UnmarshalJSON parses Garmin's timestamp format
func (t *Time) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		t.Time = time.Time{}
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	v, err := time.Parse(time.DateTime, s)
	if err != nil {
		return err
	}
	t.Time = v
	return nil
}

// ActivityType is the kind of activity
type ActivityType struct {
	TypeID  int    `json:"typeId"`
	TypeKey string `json:"typeKey"`
}

// ActivitySummary is one entry of the activity list
type ActivitySummary struct {
	ActivityID   int64        `json:"activityId"`
	ActivityName string       `json:"activityName"`
	ActivityType ActivityType `json:"activityType"`
	// StartTimeGMT is the start in UTC
	StartTimeGMT Time `json:"startTimeGMT"`
	// StartTimeLocal is the wall clock time where the activity started,
	// carried in a UTC time.Time
	StartTimeLocal Time        `json:"startTimeLocal"`
	Distance       unit.Length `json:"distance" units:"m"`
	ElevationGain  unit.Length `json:"elevationGain" units:"m"`
	// Duration is the elapsed time in seconds
	Duration float64 `json:"duration"`
}
