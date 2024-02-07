package main

import (
	"fmt"
	"time"
)

type AEMOData struct {
	Intervals []Interval `json:"5MIN"`
}

type JSONTime struct {
	time.Time
}

type RegionID string

func (ct *JSONTime) UnmarshalJSON(b []byte) error {
	// Unmarshall "2024-01-30T16:35:00" into a time.Time

	// Annoyingly this timestamp doesn't include any time zone information.
	// We're going to have to learn the timezone based on the region?
	// Bleh, then I need to track start/end of DST for each region.
	// This sucks.

	// For now assume everything's in Brisbane time, I.E. UTC+10
	brisbaneLocation, err := time.LoadLocation("Australia/Brisbane")
	if err != nil {
		return err
	}

	date, err := time.ParseInLocation("\"2006-01-02T15:04:05\"", string(b), brisbaneLocation)
	if err != nil {
		return err
	}
	ct.Time = date
	return nil
}

type Interval struct {
	SettlementDate          JSONTime `json:"SETTLEMENTDATE"`
	RegionID                RegionID `json:"REGIONID"`
	Region                  string   `json:"REGION"`
	RRP                     float64  `json:"RRP"`
	TotalDemand             float64  `json:"TOTALDEMAND"`
	PeriodType              string   `json:"PERIODTYPE"`
	NetInterchange          float64  `json:"NETINTERCHANGE"`
	ScheduledGeneration     float64  `json:"SCHEDULEDGENERATION"`
	SemiScheduledGeneration float64  `json:"SEMISCHEDULEDGENERATION"`
}

func (i *Interval) Validate() error {
	// Validate the Interval struct

	if i.PeriodType != "FORECAST" &&
		i.PeriodType != "ACTUAL" {
		return fmt.Errorf("PeriodType must be 'FORECAST' or 'ACTUAL'")
	}

	if i.RegionID != "NSW1" &&
		i.RegionID != "QLD1" &&
		i.RegionID != "SA1" &&
		i.RegionID != "TAS1" &&
		i.RegionID != "VIC1" {
		return fmt.Errorf("RegionID must be 'NSW1', 'QLD1', 'SA1', 'TAS1', or 'VIC1'")
	}

	if i.Region != string(i.RegionID) {
		return fmt.Errorf("RegionID must match Region")
	}

	return nil
}

type GridBotCfg struct {
	// Required fields.
	RegionID             RegionID `json:"RegionID"`
	MastodonClientID     string   `json:"MastodonClientID"`
	MastodonClientSecret string   `json:"MastodonClientSecret"`
	MastodonUserEmail    string   `json:"MastodonUserEmail"`
	MastodonUserPassword string   `json:"MastodonUserPassword"`
	TestMode             bool
	MastodonURL          string
}

type GridBotCfgs struct {
	Credentials []GridBotCfg `json:"Credentials"`
}
