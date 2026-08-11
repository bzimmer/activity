package hammerhead

import (
	"time"

	"github.com/martinlindhe/unit"
)

// Fault is an error returned by the Hammerhead API
type Fault struct { //nolint:errname // convention
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
}

func (f *Fault) Error() string {
	return f.Message
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

// ActivityType is the type of activity
type ActivityType string

const (
	ActivityTypeRide         ActivityType = "RIDE"
	ActivityTypeEBike        ActivityType = "EBIKE"
	ActivityTypeMountainBike ActivityType = "MOUNTAIN_BIKE"
	ActivityTypeGravel       ActivityType = "GRAVEL"
	ActivityTypeEMountain    ActivityType = "EMOUNTAIN_BIKE"
	ActivityTypeVelomobile   ActivityType = "VELOMOBILE"
)

// ActivitySummary is a summary of an activity returned in list responses
type ActivitySummary struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	CreatedAt time.Time   `json:"createdAt"`
	Duration  int         `json:"duration"`
	Distance  unit.Length `json:"distance" units:"m"`
}

// Activity is a full activity with additional fields
type Activity struct {
	ActivitySummary
	ActivityType ActivityType `json:"activityType"`
	Description  string       `json:"description"`
	Polyline     string       `json:"polyline"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

// ActivitiesPage is the paginated response for listing activities
type ActivitiesPage struct {
	TotalItems  int                `json:"totalItems"`
	TotalPages  int                `json:"totalPages"`
	PerPage     int                `json:"perPage"`
	CurrentPage int                `json:"currentPage"`
	Data        []*ActivitySummary `json:"data"`
}
