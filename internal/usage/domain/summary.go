package domain

type SummaryInterval string

const (
	SummaryDaily   SummaryInterval = "day"
	SummaryWeekly  SummaryInterval = "week"
	SummaryMonthly SummaryInterval = "month"
)

type UsagePoint struct {
	Bucket        string `json:"bucket"`
	EventCount    int64  `json:"event_count"`
	CustomerCount int64  `json:"customer_count"`
	MeterCount    int64  `json:"meter_count"`
	ValueMicros   int64  `json:"value_micros"`
}

type UsageSummary struct {
	From     string          `json:"from"`
	To       string          `json:"to"`
	Interval SummaryInterval `json:"interval"`
	Points   []UsagePoint    `json:"points"`
}
