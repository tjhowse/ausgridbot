package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

func ValidateToot(gridBot *GridBot, intervalRRP float64, intervalTime time.Time, expectedToot string, t *testing.T) {

	if want, got := intervalRRP, gridBot.lastTootedPeakRRP; !FloatEquals(want, got) {
		t.Errorf("Expected %f, got %f", want, got)
	}

	if want, got := intervalTime, gridBot.lastTootedPeakTime; !want.Equal(got) {
		t.Errorf("Expected %s, got %s", want, got)
	}

	if want, got := expectedToot, gridBot.lastToot; want != got {
		t.Errorf("Expected %s, got %s", want, got)
	}
	gridBot.lastToot = ""

	time.Sleep(10 * time.Millisecond)
}

func NewForecastInterval(gridBot *GridBot, intervalRRP float64, intervalTime time.Time, t *testing.T) Interval {
	return Interval{
		SettlementDate:          JSONTime{intervalTime},
		RegionID:                "QLD1",
		Region:                  "QLD1",
		RRP:                     intervalRRP,
		TotalDemand:             0,
		PeriodType:              "FORECAST",
		NetInterchange:          0,
		ScheduledGeneration:     0,
		SemiScheduledGeneration: 0,
	}
}

type peakType int

const (
	PEAK = iota
	DOWNGRADE
	CANCELLED
)

func FormatExpectedToot(intervalRRP float64, intervalTime time.Time, region string, oldPeakIntervalRRP float64, peak peakType) string {
	switch peak {
	case PEAK:
		return fmt.Sprintf(PEAK_TOOT_FORMAT, "Queensland", intervalRRP/1000, intervalTime.Format("15:04"))
	case DOWNGRADE:
		return fmt.Sprintf(PEAK_DOWNGRADE_TOOT_FORMAT, "Queensland", oldPeakIntervalRRP/1000, intervalRRP/1000, intervalTime.Format("15:04"))
	case CANCELLED:
		return fmt.Sprintf(PEAK_CANCELLED_TOOT_FORMAT, "Queensland", intervalRRP/1000, intervalTime.Format("15:04"))
	}
	return ""
}

func CommitIntervals(gridBot *GridBot, t *testing.T) {
	close(gridBot.GetIntervalChannel())
	done := make(chan bool)
	go func() {
		for {
			if gridBot.lastToot != "" {
				done <- true
			}
		}
	}()

	select {
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for gridBot to process interval")
	case <-done:
		break
	}
}

func TestGridBotBadRegionid(t *testing.T) {
	cfg := GridBotCfg{}
	cfg.TestMode = true
	cfg.RegionID = "NT1"

	if _, err := NewGridBot(cfg); err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestGridBotBasicInterval(t *testing.T) {
	cfg := GridBotCfg{}
	cfg.TestMode = true
	cfg.RegionID = "QLD1"

	var gridBot *GridBot
	var err error

	if gridBot, err = NewGridBot(cfg); err != nil {
		t.Fatal(err)
	}
	if want, got := "Queensland", gridBot.regionString; want != got {
		t.Errorf("Expected %s, got %s", want, got)
	}

	go gridBot.Mainloop()

	peakTime := time.Now().Add(1 * time.Hour)
	peakRRP := float64(INTERESTING_PEAK_RRP + 1)
	interval := NewForecastInterval(gridBot, peakRRP, peakTime, t)
	gridBot.GetIntervalChannel() <- interval
	// Throw in a cheeky actual interval to make sure it doesn't get tooted
	interval.PeriodType = "ACTUAL"
	interval.RRP = peakRRP * 2
	interval.SettlementDate = JSONTime{time.Now().Add(2 * time.Hour)}
	gridBot.GetIntervalChannel() <- interval

	if want, got := 1, len(gridBot.forecasts); want != got {
		t.Errorf("Expected %d, got %d", want, got)
	}
	if gridBot.forecastsStale {
		t.Errorf("Expected forecastsStale to be false, got true")
	}
	CommitIntervals(gridBot, t)
	ValidateToot(gridBot, peakRRP, peakTime, FormatExpectedToot(peakRRP, peakTime, "Queensland", 0, PEAK), t)
	if want, got := 1, len(gridBot.forecasts); want != got {
		t.Errorf("Expected %d, got %d", want, got)
	}
	if !gridBot.forecastsStale {
		t.Errorf("Expected forecastsStale to be true, got false")
	}
}
func TestGridBotBasicPeak(t *testing.T) {
	cfg := GridBotCfg{}
	cfg.TestMode = true
	cfg.RegionID = "QLD1"

	var gridBot *GridBot
	var err error

	if gridBot, err = NewGridBot(cfg); err != nil {
		t.Fatal(err)
	}
	if want, got := "Queensland", gridBot.regionString; want != got {
		t.Errorf("Expected %s, got %s", want, got)
	}

	go gridBot.Mainloop()

	peakTime := time.Now().Add(2 * time.Hour)
	peakRRP := float64(INTERESTING_PEAK_RRP * 3)

	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(-1*time.Hour), t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP, peakTime, t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(1*time.Hour), t)
	if want, got := 3, len(gridBot.forecasts); want != got {
		t.Errorf("Expected %d, got %d", want, got)
	}
	if gridBot.forecastsStale {
		t.Errorf("Expected forecastsStale to be false, got true")
	}
	CommitIntervals(gridBot, t)
	ValidateToot(gridBot, peakRRP, peakTime, FormatExpectedToot(peakRRP, peakTime, "Queensland", 0, PEAK), t)
	if !gridBot.forecastsStale {
		t.Errorf("Expected forecastsStale to be true, got false")
	}

	oldPeak := peakRRP
	// Cancel the peak
	peakRRP = float64(INTERESTING_PEAK_RRP - 1)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(-1*time.Hour), t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP, peakTime, t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(1*time.Hour), t)
	if gridBot.forecastsStale {
		t.Errorf("Expected forecastsStale to be false, got true")
	}
	CommitIntervals(gridBot, t)
	ValidateToot(gridBot, peakRRP, peakTime, FormatExpectedToot(oldPeak, peakTime, "Queensland", 0, CANCELLED), t)
	if want, got := 3, len(gridBot.forecasts); want != got {
		t.Errorf("Expected %d, got %d", want, got)
	}

	// Restore the peak
	peakRRP = float64(INTERESTING_PEAK_RRP * 3)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(-1*time.Hour), t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP, peakTime, t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(1*time.Hour), t)
	CommitIntervals(gridBot, t)
	ValidateToot(gridBot, peakRRP, peakTime, FormatExpectedToot(peakRRP, peakTime, "Queensland", 0, PEAK), t)

	// Lower peak
	peakRRP = float64(INTERESTING_PEAK_RRP * 2)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(-1*time.Hour), t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP, peakTime, t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(1*time.Hour), t)
	CommitIntervals(gridBot, t)
	ValidateToot(gridBot, peakRRP, peakTime, FormatExpectedToot(peakRRP, peakTime, "Queensland", oldPeak, DOWNGRADE), t)

	// Marginally larger peak, should be ignored.
	peakRRP = float64(INTERESTING_PEAK_RRP*2 + 10)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(-1*time.Hour), t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP, peakTime, t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(1*time.Hour), t)

	// Can't use the ValidateToot helper here, since it waits for the lastToot to be != "", but that's
	// exactly what we want in this case. Just close off the channel and wait a second.
	close(gridBot.GetIntervalChannel())
	time.Sleep(1 * time.Second)
	if want, got := "", gridBot.lastToot; want != got {
		t.Errorf("Expected no toot, got %s", got)
	}
}

func TestBuildBasicGridBot(t *testing.T) {
	cfg := config{}
	cfg.GridBotCredentials = `[
		{
			"RegionID": "QLD1",
			"MastodonClientID": "clientid",
			"MastodonClientSecret": "clientsecret",
			"MastodonUserEmail": "useremail",
			"MastodonUserPassword": "userpassword"
		}
	]`
	cfg.TestMode = true
	cfg.MastodonURL = "https://mastodon.example.com"

	if gridBots, err := BuildGridBots(cfg); err != nil {
		t.Fatal(err)
	} else {
		if len(gridBots) != 1 {
			t.Fatal("Expected 1 gridBot, got", len(gridBots))
		}
		if want, got := "Queensland", gridBots["QLD1"].regionString; want != got {
			t.Fatal("Expected regionString to be ", want, "got", got)
		}
		if !gridBots["QLD1"].cfg.TestMode {
			t.Fatal("Expected TestMode to be true, got", gridBots["QLD1"].cfg.TestMode)
		}
	}
}

func TestBuildBasicGridBots(t *testing.T) {
	cfg := config{}
	cfg.GridBotCredentials = `[
		{
			"RegionID": "QLD1",
			"MastodonClientID": "qldclientid",
			"MastodonClientSecret": "qldclientsecret",
			"MastodonUserEmail": "qlduseremail",
			"MastodonUserPassword": "qlduserpassword"
		},
		{
			"RegionID": "NSW1",
			"MastodonClientID": "nswclientid",
			"MastodonClientSecret": "nswclientsecret",
			"MastodonUserEmail": "nswuseremail",
			"MastodonUserPassword": "nswuserpassword"
		}
	]`
	cfg.TestMode = true
	cfg.MastodonURL = "https://mastodon.example.com"

	if gridBots, err := BuildGridBots(cfg); err != nil {
		t.Fatal(err)
	} else {
		if len(gridBots) != 2 {
			t.Fatal("Expected 1 gridBot, got", len(gridBots))
		}
		if want, got := "Queensland", gridBots["QLD1"].regionString; want != got {
			t.Fatal("Expected regionString to be ", want, "got", got)
		}
		if want, got := "qldclientid", gridBots["QLD1"].cfg.MastodonClientID; want != got {
			t.Fatal("Expected MastodonClientID to be ", want, "got", got)
		}
		if want, got := "qldclientsecret", gridBots["QLD1"].cfg.MastodonClientSecret; want != got {
			t.Fatal("Expected MastodonClientSecret to be ", want, "got", got)
		}
		if want, got := "qlduseremail", gridBots["QLD1"].cfg.MastodonUserEmail; want != got {
			t.Fatal("Expected MastodonUserEmail to be ", want, "got", got)
		}
		if want, got := "qlduserpassword", gridBots["QLD1"].cfg.MastodonUserPassword; want != got {
			t.Fatal("Expected MastodonUserPassword to be ", want, "got", got)
		}

		if !gridBots["QLD1"].cfg.TestMode {
			t.Fatal("Expected TestMode to be true, got", gridBots["QLD1"].cfg.TestMode)
		}

		if want, got := "New South Wales", gridBots["NSW1"].regionString; want != got {
			t.Fatal("Expected regionString to be ", want, "got", got)
		}
		if want, got := "nswclientid", gridBots["NSW1"].cfg.MastodonClientID; want != got {
			t.Fatal("Expected MastodonClientID to be ", want, "got", got)
		}
		if want, got := "nswclientsecret", gridBots["NSW1"].cfg.MastodonClientSecret; want != got {
			t.Fatal("Expected MastodonClientSecret to be ", want, "got", got)
		}
		if want, got := "nswuseremail", gridBots["NSW1"].cfg.MastodonUserEmail; want != got {
			t.Fatal("Expected MastodonUserEmail to be ", want, "got", got)
		}
		if want, got := "nswuserpassword", gridBots["NSW1"].cfg.MastodonUserPassword; want != got {
			t.Fatal("Expected MastodonUserPassword to be ", want, "got", got)
		}
		if !gridBots["NSW1"].cfg.TestMode {
			t.Fatal("Expected TestMode to be true, got", gridBots["NSW1"].cfg.TestMode)
		}

	}
}

func TestPlotting(t *testing.T) {
	var err error
	f, err := os.Open("data/exampledata.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var aemoData AEMOData
	err = json.NewDecoder(f).Decode(&aemoData)
	if err != nil {
		t.Fatal(err)
	}

	// Find the last non-forecast time in the intervals.
	var startTime time.Time
	for _, interval := range aemoData.Intervals {
		if interval.RegionID != "QLD1" {
			continue
		}
		if interval.PeriodType == "FORECAST" {
			break
		}
		startTime = interval.SettlementDate.Time
	}

	// Round offsetFromNowToDataStartTime to the nearest half-hour
	offsetFromNowToDataStartTime := time.Since(startTime).Round(30 * time.Minute)

	// Re-timebase the example data such that the start time is now
	for i := range aemoData.Intervals {
		aemoData.Intervals[i].SettlementDate.Time = aemoData.Intervals[i].SettlementDate.Time.Add(offsetFromNowToDataStartTime)
	}

	cfg := GridBotCfg{}
	cfg.TestMode = true
	cfg.RegionID = "QLD1"

	var gridBot *GridBot

	if gridBot, err = NewGridBot(cfg); err != nil {
		t.Fatal(err)
	}
	go gridBot.Mainloop()

	for _, d := range aemoData.Intervals {
		gridBot.GetIntervalChannel() <- d
	}
	close(gridBot.GetIntervalChannel())

	file, err := os.Create("test2.png")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	gridBot.generatePlot(file)
}
