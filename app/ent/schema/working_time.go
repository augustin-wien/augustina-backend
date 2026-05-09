package schema

// WorkingTime represents structured opening hours for a Location.
// It supports per-day ranges, an everyday range, whole-week flag and
// full-day markers so the UI can present options like "from/to by day",
// "everyday from/to", "whole week", or "full day".
type WorkingTime struct {
    // Mode is a hint for the UI: "by_day", "everyday", "whole_week" or "custom".
    Mode string `json:"mode,omitempty"`

    // WeekDays maps weekday keys (mon, tue, wed, thu, fri, sat, sun)
    // to an array of time ranges for that day.
    WeekDays map[string][]TimeRange `json:"week_days,omitempty"`

    // Everyday, if set, applies the same TimeRange(s) to every day.
    Everyday []TimeRange `json:"everyday,omitempty"`

    // WholeWeek indicates the location is open the whole week (24/7).
    WholeWeek bool `json:"whole_week,omitempty"`
}

// TimeRange represents an opening period. Use FullDay to indicate the
// whole day is open (no from/to required). Times are expected as
// strings in HH:MM (24h) format.
type TimeRange struct {
    From    string `json:"from,omitempty"`
    To      string `json:"to,omitempty"`
    FullDay bool   `json:"full_day,omitempty"`
}
