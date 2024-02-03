package main

import "time"

type AEMOData struct {
	Intervals []Interval `json:"5MIN"`
}

type JSONTime struct {
	time.Time
}

func (ct *JSONTime) UnmarshalJSON(b []byte) error {
	// Unmarshall "2024-01-30T16:35:00" into a time.Time

	// Annoyingly this timestamp doesn't include any time zone information.
	// We're going to have to learn the timezone based on the region?
	// Bleh, then I need to track start/end of timezones for each region.
	// This sucks.

	date, err := time.Parse("\"2006-01-02T15:04:05\"", string(b))
	if err != nil {
		return err
	}
	ct.Time = date
	return nil
}

type Interval struct {
	SettlementDate          JSONTime `json:"SETTLEMENTDATE"`
	RegionID                string   `json:"REGIONID"`
	Region                  string   `json:"REGION"`
	RRP                     float64  `json:"RRP"`
	TotalDeman              float64  `json:"TOTALDEMAND"`
	PeriodType              string   `json:"PERIODTYPE"`
	NetInterchange          float64  `json:"NETINTERCHANGE"`
	ScheduledGeneration     float64  `json:"SCHEDULEDGENERATION"`
	SemiScheduledGeneration float64  `json:"SEMISCHEDULEDGENERATION"`
}
