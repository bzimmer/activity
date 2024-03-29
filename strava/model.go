package strava

import (
	"time"

	"github.com/martinlindhe/unit"
	"github.com/twpayne/go-geom"

	"github.com/bzimmer/activity"
)

// Error is an error from the Strava API
type Error struct {
	Resource string `json:"resource"`
	Field    string `json:"field"`
	Code     string `json:"code"`
}

// Fault contains errors
type Fault struct { //nolint:errname // convention
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Errors  []*Error `json:"errors"`
}

func (f *Fault) Error() string {
	return f.Message
}

// Coordinates are a [lat, lng] pair
type Coordinates []float64

// StreamMetadata for the stream
type StreamMetadata struct {
	OriginalSize int    `json:"original_size"`
	Resolution   string `json:"resolution"`
	SeriesType   string `json:"series_type"`
}

// Stream of data from an activity
type Stream struct {
	StreamMetadata
	Data []float64 `json:"data"`
}

// CoordinateStream of data from an activity
type CoordinateStream struct {
	StreamMetadata
	Data []Coordinates `json:"data"`
}

// SpeedStream of data from an activity
type SpeedStream struct {
	StreamMetadata
	Data []unit.Speed `json:"data" units:"mps"`
}

// LengthStream of data from an activity
type LengthStream struct {
	StreamMetadata
	Data []unit.Length `json:"data" units:"m"`
}

// BoolStream of data from an activity
type BoolStream struct {
	StreamMetadata
	Data []bool `json:"data"`
}

// Streams of data available for an activity, not all activities will have all streams
type Streams struct {
	ActivityID  int64             `json:"activity_id"`
	LatLng      *CoordinateStream `json:"latlng,omitempty"`
	Elevation   *LengthStream     `json:"altitude,omitempty"`
	Time        *Stream           `json:"time,omitempty"`
	Distance    *LengthStream     `json:"distance,omitempty"`
	Velocity    *SpeedStream      `json:"velocity_smooth,omitempty"`
	HeartRate   *Stream           `json:"heartrate,omitempty"`
	Cadence     *Stream           `json:"cadence,omitempty"`
	Watts       *Stream           `json:"watts,omitempty"`
	Temperature *Stream           `json:"temp,omitempty"`
	Moving      *BoolStream       `json:"moving,omitempty"`
	Grade       *Stream           `json:"grade_smooth,omitempty"`
}

// Gear represents gear used by the athlete
type Gear struct {
	ID            string      `json:"id"`
	Primary       bool        `json:"primary"`
	Name          string      `json:"name"`
	ResourceState int         `json:"resource_state"`
	Distance      unit.Length `json:"distance" units:"m"`
	AthleteID     int         `json:"athlete_id"`
}

// Totals for stats
type Totals struct {
	Distance         unit.Length   `json:"distance" units:"m"`
	AchievementCount int           `json:"achievement_count"`
	Count            int           `json:"count"`
	ElapsedTime      unit.Duration `json:"elapsed_time" units:"s"`
	ElevationGain    unit.Length   `json:"elevation_gain" units:"m"`
	MovingTime       unit.Duration `json:"moving_time" units:"s"`
}

// Stats for the athlete
type Stats struct {
	RecentRunTotals           *Totals     `json:"recent_run_totals"`
	AllRunTotals              *Totals     `json:"all_run_totals"`
	RecentSwimTotals          *Totals     `json:"recent_swim_totals"`
	BiggestRideDistance       unit.Length `json:"biggest_ride_distance" units:"m"`
	YTDSwimTotals             *Totals     `json:"ytd_swim_totals"`
	AllSwimTotals             *Totals     `json:"all_swim_totals"`
	RecentRideTotals          *Totals     `json:"recent_ride_totals"`
	BiggestClimbElevationGain unit.Length `json:"biggest_climb_elevation_gain" units:"m"`
	YTDRideTotals             *Totals     `json:"ytd_ride_totals"`
	AllRideTotals             *Totals     `json:"all_ride_totals"`
	YTDRunTotals              *Totals     `json:"ytd_run_totals"`
}

// Club in which an athlete can be a member
type Club struct {
	Admin           bool   `json:"admin"`
	City            string `json:"city"`
	Country         string `json:"country"`
	CoverPhoto      string `json:"cover_photo"`
	CoverPhotoSmall string `json:"cover_photo_small"`
	Featured        bool   `json:"featured"`
	ID              int    `json:"id"`
	MemberCount     int    `json:"member_count"`
	Membership      string `json:"membership"`
	Name            string `json:"name"`
	Owner           bool   `json:"owner"`
	Private         bool   `json:"private"`
	Profile         string `json:"profile"`
	ProfileMedium   string `json:"profile_medium"`
	ResourceState   int    `json:"resource_state"`
	SportType       string `json:"sport_type"`
	State           string `json:"state"`
	URL             string `json:"url"`
	Verified        bool   `json:"verified"`
}

// Athlete represents a Strava athlete
type Athlete struct {
	ID                    int       `json:"id"`
	Username              string    `json:"username"`
	ResourceState         int       `json:"resource_state"`
	Firstname             string    `json:"firstname"`
	Lastname              string    `json:"lastname"`
	City                  string    `json:"city"`
	State                 string    `json:"state"`
	Country               string    `json:"country"`
	Sex                   string    `json:"sex"`
	Premium               bool      `json:"premium"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	BadgeTypeID           int       `json:"badge_type_id"`
	ProfileMedium         string    `json:"profile_medium"`
	Profile               string    `json:"profile"`
	Friend                any       `json:"friend"`
	Follower              any       `json:"follower"`
	FollowerCount         int       `json:"follower_count"`
	FriendCount           int       `json:"friend_count"`
	MutualFriendCount     int       `json:"mutual_friend_count"`
	AthleteType           int       `json:"athlete_type"`
	DatePreference        string    `json:"date_preference"`
	MeasurementPreference string    `json:"measurement_preference"`
	Clubs                 []*Club   `json:"clubs"`
	FTP                   float64   `json:"ftp"`
	Weight                unit.Mass `json:"weight" units:"kg"`
	Bikes                 []*Gear   `json:"bikes"`
	Shoes                 []*Gear   `json:"shoes"`
}

// Map of the activity or route
type Map struct {
	ID              string `json:"id"`
	Polyline        string `json:"polyline"`
	ResourceState   int    `json:"resource_state"`
	SummaryPolyline string `json:"summary_polyline"`
}

// LineString of the map
// This function first checks for a Polyline and then a SummaryPolyline, returning the first non-nil LineString
func (m *Map) LineString() (*geom.LineString, error) {
	return polylineToLineString(m.Polyline, m.SummaryPolyline)
}

// Lap data
type Lap struct {
	ID                 int64         `json:"id"`
	ResourceState      int           `json:"resource_state"`
	Name               string        `json:"name"`
	Activity           *Activity     `json:"activity"`
	Athlete            *Athlete      `json:"athlete"`
	ElapsedTime        unit.Duration `json:"elapsed_time" units:"s"`
	MovingTime         unit.Duration `json:"moving_time" units:"s"`
	StartDate          time.Time     `json:"start_date"`
	StartDateLocal     time.Time     `json:"start_date_local"`
	Distance           unit.Length   `json:"distance" units:"m"`
	StartIndex         int           `json:"start_index"`
	EndIndex           int           `json:"end_index"`
	TotalElevationGain unit.Length   `json:"total_elevation_gain" units:"m"`
	AverageSpeed       unit.Speed    `json:"average_speed" units:"mps"`
	MaxSpeed           unit.Speed    `json:"max_speed" units:"mps"`
	AverageCadence     float64       `json:"average_cadence"`
	DeviceWatts        bool          `json:"device_watts"`
	AverageWatts       float64       `json:"average_watts"`
	LapIndex           int           `json:"lap_index"`
	Split              int           `json:"split"`
}

// PREffort for the segment
type PREffort struct {
	Distance       unit.Length   `json:"distance" units:"m"`
	StartDateLocal time.Time     `json:"start_date_local"`
	ActivityID     int           `json:"activity_id"`
	ElapsedTime    unit.Duration `json:"elapsed_time" units:"s"`
	IsKOM          bool          `json:"is_kom"`
	ID             int           `json:"id"`
	StartDate      time.Time     `json:"start_date"`
}

// SegmentStats for the segment
type SegmentStats struct {
	PRElapsedTime unit.Duration `json:"pr_elapsed_time" units:"s"`
	PRDate        time.Time     `json:"pr_date"`
	EffortCount   int           `json:"effort_count"`
	PRActivityID  int           `json:"pr_activity_id"`
}

// Segment .
type Segment struct {
	ID                  int           `json:"id"`
	ResourceState       int           `json:"resource_state"`
	Name                string        `json:"name"`
	ActivityType        string        `json:"activity_type"`
	Distance            unit.Length   `json:"distance" units:"m"`
	AverageGrade        float64       `json:"average_grade"`
	MaximumGrade        float64       `json:"maximum_grade"`
	ElevationHigh       unit.Length   `json:"elevation_high" units:"m"`
	ElevationLow        unit.Length   `json:"elevation_low" units:"m"`
	StartLatlng         Coordinates   `json:"start_latlng"`
	EndLatlng           Coordinates   `json:"end_latlng"`
	ClimbCategory       int           `json:"climb_category"`
	City                string        `json:"city"`
	State               string        `json:"state"`
	Country             string        `json:"country"`
	Private             bool          `json:"private"`
	Hazardous           bool          `json:"hazardous"`
	Starred             bool          `json:"starred"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
	ElevationGain       unit.Length   `json:"total_elevation_gain" units:"m"`
	Map                 *Map          `json:"map"`
	EffortCount         int           `json:"effort_count"`
	AthleteCount        int           `json:"athlete_count"`
	StarCount           int           `json:"star_count"`
	PREffort            *PREffort     `json:"athlete_pr_effort"`
	AthleteSegmentStats *SegmentStats `json:"athlete_segment_stats"`
}

// MetaActivity .
type MetaActivity struct {
	ID            int64 `json:"id"`
	ResourceState int   `json:"resource_state"`
}

// Achievement .
type Achievement struct {
	Rank   int    `json:"rank"`
	Type   string `json:"type"`
	TypeID int    `json:"type_id"`
}

// SegmentEffort .
type SegmentEffort struct {
	ID             int64          `json:"id"`
	ResourceState  int            `json:"resource_state"`
	Name           string         `json:"name"`
	Activity       *MetaActivity  `json:"activity"`
	Athlete        *Athlete       `json:"athlete"`
	ElapsedTime    unit.Duration  `json:"elapsed_time" units:"s"`
	MovingTime     unit.Duration  `json:"moving_time" units:"s"`
	StartDate      time.Time      `json:"start_date"`
	StartDateLocal time.Time      `json:"start_date_local"`
	Distance       unit.Length    `json:"distance" units:"m"`
	StartIndex     int            `json:"start_index"`
	EndIndex       int            `json:"end_index"`
	AverageCadence float64        `json:"average_cadence"`
	DeviceWatts    bool           `json:"device_watts"`
	AverageWatts   float64        `json:"average_watts"`
	Segment        *Segment       `json:"segment"`
	KOMRank        int            `json:"kom_rank"`
	PRRank         int            `json:"pr_rank"`
	Achievements   []*Achievement `json:"achievements"`
	Hidden         bool           `json:"hidden"`
}

// SplitsMetric .
type SplitsMetric struct {
	Distance            unit.Length   `json:"distance" units:"m"`
	ElapsedTime         unit.Duration `json:"elapsed_time" units:"s"`
	ElevationDifference unit.Length   `json:"elevation_difference" units:"m"`
	MovingTime          unit.Duration `json:"moving_time" units:"s"`
	Split               int           `json:"split"`
	AverageSpeed        float64       `json:"average_speed"`
	PaceZone            int           `json:"pace_zone"`
}

// HighlightedKudosers .
type HighlightedKudosers struct {
	DestinationURL string `json:"destination_url"`
	DisplayName    string `json:"display_name"`
	AvatarURL      string `json:"avatar_url"`
	ShowName       bool   `json:"show_name"`
}

// Photo metadata for activity and post photos
type Photo struct {
	ID             int64             `json:"id"`
	UniqueID       string            `json:"unique_id"`
	AthleteID      int               `json:"athlete_id"`
	ActivityID     int64             `json:"activity_id"`
	ActivityName   string            `json:"activity_name"`
	PostID         int64             `json:"post_id"`
	ResourceState  int               `json:"resource_state"`
	Ref            string            `json:"ref"`
	UID            string            `json:"uid"`
	Caption        string            `json:"caption"`
	Type           string            `json:"type"`
	Source         int               `json:"source"`
	UploadedAt     time.Time         `json:"uploaded_at"`
	CreatedAt      time.Time         `json:"created_at"`
	CreatedAtLocal time.Time         `json:"created_at_local"`
	URLs           map[string]string `json:"urls"`
	Sizes          map[string][]int  `json:"sizes"`
	DefaultPhoto   bool              `json:"default_photo"`
	Location       []float64         `json:"location"`
}

// Photos for an activity
type Photos struct {
	Primary struct {
		ID       int64             `json:"id"`
		UniqueID string            `json:"unique_id"`
		URLs     map[string]string `json:"urls"`
		Source   int               `json:"source"`
	} `json:"primary"`
	UsePrimaryPhoto bool `json:"use_primary_photo"`
	Count           int  `json:"count"`
}

// UpdatableActivity represents an activity with updatable attributes
type UpdatableActivity struct {
	ID          int64   `json:"id"`
	Commute     *bool   `json:"commute,omitempty"`
	Trainer     *bool   `json:"trainer,omitempty"`
	Hidden      *bool   `json:"hide_from_home,omitempty"`
	Description *string `json:"description,omitempty"`
	Name        *string `json:"name,omitempty"`
	SportType   *string `json:"sport_type,omitempty"`
	GearID      *string `json:"gear_id,omitempty"`
}

// Activity represents an activity
type Activity struct {
	ID                       int64                  `json:"id"`
	ResourceState            int                    `json:"resource_state"`
	ExternalID               string                 `json:"external_id"`
	UploadID                 int64                  `json:"upload_id"`
	Athlete                  *Athlete               `json:"athlete"`
	Name                     string                 `json:"name"`
	Hidden                   bool                   `json:"hide_from_home"`
	Distance                 unit.Length            `json:"distance" units:"m"`
	MovingTime               unit.Duration          `json:"moving_time" units:"s"`
	ElapsedTime              unit.Duration          `json:"elapsed_time" units:"s"`
	ElevationGain            unit.Length            `json:"total_elevation_gain" units:"m"`
	Type                     string                 `json:"type"`
	SportType                string                 `json:"sport_type"`
	StartDate                time.Time              `json:"start_date"`
	StartDateLocal           time.Time              `json:"start_date_local"`
	Timezone                 string                 `json:"timezone"`
	UTCOffset                float64                `json:"utc_offset"`
	StartLatlng              Coordinates            `json:"start_latlng"`
	EndLatlng                Coordinates            `json:"end_latlng"`
	LocationCity             string                 `json:"location_city"`
	LocationState            string                 `json:"location_state"`
	LocationCountry          string                 `json:"location_country"`
	AchievementCount         int                    `json:"achievement_count"`
	KudosCount               int                    `json:"kudos_count"`
	CommentCount             int                    `json:"comment_count"`
	AthleteCount             int                    `json:"athlete_count"`
	PhotoCount               int                    `json:"photo_count"`
	Map                      *Map                   `json:"map"`
	Trainer                  bool                   `json:"trainer"`
	Commute                  bool                   `json:"commute"`
	Manual                   bool                   `json:"manual"`
	Private                  bool                   `json:"private"`
	Flagged                  bool                   `json:"flagged"`
	GearID                   string                 `json:"gear_id"`
	FromAcceptedTag          bool                   `json:"from_accepted_tag"`
	AverageSpeed             unit.Speed             `json:"average_speed" units:"mps"`
	MaxSpeed                 unit.Speed             `json:"max_speed" units:"mps"`
	AverageCadence           float64                `json:"average_cadence"`
	AverageTemperature       float64                `json:"average_temp" units:"C"`
	AverageWatts             float64                `json:"average_watts"`
	WeightedAverageWatts     int                    `json:"weighted_average_watts"`
	Kilojoules               float64                `json:"kilojoules"`
	DeviceWatts              bool                   `json:"device_watts"`
	HasHeartrate             bool                   `json:"has_heartrate"`
	MaxWatts                 int                    `json:"max_watts"`
	ElevationHigh            unit.Length            `json:"elev_high" units:"m"`
	ElevationLow             unit.Length            `json:"elev_low" units:"m"`
	PRCount                  int                    `json:"pr_count"`
	TotalPhotoCount          int                    `json:"total_photo_count"`
	HasKudoed                bool                   `json:"has_kudoed"`
	WorkoutType              int                    `json:"workout_type"`
	SufferScore              float64                `json:"suffer_score"`
	Description              string                 `json:"description"`
	PrivateNote              string                 `json:"private_note"`
	Calories                 float64                `json:"calories"`
	SegmentEfforts           []*SegmentEffort       `json:"segment_efforts,omitempty"`
	SplitsMetric             []*SplitsMetric        `json:"splits_metric,omitempty"`
	Laps                     []*Lap                 `json:"laps,omitempty"`
	Gear                     *Gear                  `json:"gear,omitempty"`
	Photos                   *Photos                `json:"photos,omitempty"`
	HighlightedKudosers      []*HighlightedKudosers `json:"highlighted_kudosers,omitempty"`
	DeviceName               string                 `json:"device_name"`
	EmbedToken               string                 `json:"embed_token"`
	SegmentLeaderboardOptOut bool                   `json:"segment_leaderboard_opt_out"`
	LeaderboardOptOut        bool                   `json:"leaderboard_opt_out"`
	PerceivedExertion        float64                `json:"perceived_exertion"`
	PreferPerceivedExertion  bool                   `json:"prefer_perceived_exertion"`
	Streams                  *Streams               `json:"streams,omitempty"`
}

// Route is a planned activity
type Route struct {
	Private             bool          `json:"private"`
	Distance            unit.Length   `json:"distance" units:"m"`
	Athlete             *Athlete      `json:"athlete"`
	Description         string        `json:"description"`
	CreatedAt           time.Time     `json:"created_at"`
	ElevationGain       unit.Length   `json:"elevation_gain" units:"m"`
	Type                int           `json:"type"`
	EstimatedMovingTime unit.Duration `json:"estimated_moving_time" units:"s"`
	Segments            []*Segment    `json:"segments"`
	Starred             bool          `json:"starred"`
	UpdatedAt           time.Time     `json:"updated_at"`
	SubType             int           `json:"sub_type"`
	IDString            string        `json:"id_str"`
	Name                string        `json:"name"`
	ID                  int64         `json:"id"`
	Map                 *Map          `json:"map"`
	Timestamp           int           `json:"timestamp"`
}

type TrainingDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type FitnessProfile struct {
	Fitness        float64 `json:"fitness"`
	Impulse        int     `json:"impulse"`
	RelativeEffort int     `json:"relative_effort"`
	Fatigue        float64 `json:"fatigue"`
	Form           float64 `json:"form"`
}

type ActivityProfile struct {
	ID             int64 `json:"id"`
	Impulse        int   `json:"impulse"`
	RelativeEffort int   `json:"relative_effort"`
}

type TrainingLoad struct {
	TrainingDate   *TrainingDate      `json:"date"`
	FitnessProfile *FitnessProfile    `json:"fitness_profile"`
	Activities     []*ActivityProfile `json:"activities"`
}

// ActivityResult is the result of querying for a stream of activities
type ActivityResult struct {
	Activity *Activity
	Err      error
}

// Upload is the state representation of an uploaded activity
type Upload struct {
	ID         int64  `json:"id"`
	IDString   string `json:"id_str"`
	ExternalID string `json:"external_id"`
	Error      string `json:"error"`
	Status     string `json:"status"`
	ActivityID int64  `json:"activity_id"`
}

func (u *Upload) Identifier() activity.UploadID {
	return activity.UploadID(u.ID)
}

func (u *Upload) Done() bool {
	return u.ActivityID > 0 || u.Error != ""
}

// UploadResult is the result of polling for upload status
type UploadResult struct {
	Upload *Upload `json:"upload"`
	Err    error   `json:"error"`
}
