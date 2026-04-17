package strava

type Athlete struct {
	ID        int64  `json:"id"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	City      string `json:"city"`
	Country   string `json:"country"`
	Sex       string `json:"sex"`
	Premium   bool   `json:"premium"`
	CreatedAt string `json:"created_at"`
}

type ActivitySummary struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	Type               string  `json:"type"`
	SportType          string  `json:"sport_type"`
	Distance           float64 `json:"distance"`
	MovingTime         int     `json:"moving_time"`
	ElapsedTime        int     `json:"elapsed_time"`
	TotalElevationGain float64 `json:"total_elevation_gain"`
	StartDate          string  `json:"start_date"`
	StartDateLocal     string  `json:"start_date_local"`
	Timezone           string  `json:"timezone"`
	AverageSpeed       float64 `json:"average_speed"`
	MaxSpeed           float64 `json:"max_speed"`
	AverageHeartrate   float64 `json:"average_heartrate"`
	MaxHeartrate       float64 `json:"max_heartrate"`
	Kilojoules         float64 `json:"kilojoules"`
	AverageWatts       float64 `json:"average_watts"`
	MaxWatts           float64 `json:"max_watts"`
	WeightedAvgWatts   int     `json:"weighted_average_watts"`
	SufferScore        float64 `json:"suffer_score"`
	Kudos              int     `json:"kudos_count"`
	PRCount            int     `json:"pr_count"`
	AchievementCount   int     `json:"achievement_count"`
	MapPolyline        string  `json:"map_polyline,omitempty"`
}

type ActivityDetail struct {
	ActivitySummary
	Description   string  `json:"description"`
	Calories      float64 `json:"calories"`
	DeviceName    string  `json:"device_name"`
	EmbedToken    string  `json:"embed_token"`
	SegmentEfforts []SegmentEffort `json:"segment_efforts"`
	Laps          []Lap   `json:"laps"`
	BestEfforts   []BestEffort `json:"best_efforts"`
}

type SegmentEffort struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Distance     float64 `json:"distance"`
	MovingTime   int     `json:"moving_time"`
	ElapsedTime  int     `json:"elapsed_time"`
	PRRank       int     `json:"pr_rank"`
	KOMRank      int     `json:"kom_rank"`
	AverageWatts float64 `json:"average_watts"`
}

type Lap struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	ElapsedTime      int     `json:"elapsed_time"`
	MovingTime       int     `json:"moving_time"`
	Distance         float64 `json:"distance"`
	AverageSpeed     float64 `json:"average_speed"`
	MaxSpeed         float64 `json:"max_speed"`
	AverageHeartrate float64 `json:"average_heartrate"`
	MaxHeartrate     float64 `json:"max_heartrate"`
	AverageWatts     float64 `json:"average_watts"`
	LapIndex         int     `json:"lap_index"`
}

type BestEffort struct {
	Name        string `json:"name"`
	ElapsedTime int    `json:"elapsed_time"`
	MovingTime  int    `json:"moving_time"`
	PRRank      int    `json:"pr_rank"`
}

type AthleteStats struct {
	RecentRunTotals    ActivityTotals `json:"recent_run_totals"`
	RecentRideTotals   ActivityTotals `json:"recent_ride_totals"`
	RecentSwimTotals   ActivityTotals `json:"recent_swim_totals"`
	YTDRunTotals       ActivityTotals `json:"ytd_run_totals"`
	YTDRideTotals      ActivityTotals `json:"ytd_ride_totals"`
	YTDSwimTotals      ActivityTotals `json:"ytd_swim_totals"`
	AllRunTotals       ActivityTotals `json:"all_run_totals"`
	AllRideTotals      ActivityTotals `json:"all_ride_totals"`
	AllSwimTotals      ActivityTotals `json:"all_swim_totals"`
}

type ActivityTotals struct {
	Count            int     `json:"count"`
	Distance         float64 `json:"distance"`
	MovingTime       int     `json:"moving_time"`
	ElapsedTime      int     `json:"elapsed_time"`
	ElevationGain    float64 `json:"elevation_gain"`
	AchievementCount int     `json:"achievement_count"`
}

type HeartRateZones struct {
	CustomZones bool    `json:"custom_zones"`
	Zones       []Zone  `json:"zones"`
}

type Zone struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type AthleteZones struct {
	HeartRate HeartRateZones `json:"heart_rate"`
	Power     PowerZones     `json:"power"`
}

type PowerZones struct {
	CustomZones bool   `json:"custom_zones"`
	Zones       []Zone `json:"zones"`
}
