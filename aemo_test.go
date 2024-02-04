package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestDeserialiseJson(t *testing.T) {
	// Load testdata.json and parse it as AEMOData
	// Compare the result to the expected result

	f, err := os.Open("data/testdata.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var aemoData AEMOData
	err = json.NewDecoder(f).Decode(&aemoData)
	if err != nil {
		t.Fatal(err)
	}

	if aemoData.Intervals[0].RegionID != "NSW1" {
		t.Fatal("Data didn't deserialise properly")
	}

	if aemoData.Intervals[0].SettlementDate.String() != "2024-01-30 16:35:00 +1000 AEST" {
		t.Fatal("Data didn't deserialise properly", aemoData.Intervals[0].SettlementDate.String())
	}
}

func TestGetAEMOData(t *testing.T) {

	f, err := os.Open("data/testdata.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	testData, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aemo/apps/api/report/5MIN" {
			t.Errorf("Expected to request '/aemo/apps/api/report/5MIN', got: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected to send a POST request, got: %s", r.Method)
		}
		REQBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		// fmt.Printf("Request body: %s", REQBody)
		expectedBody := []byte(`{"timeScale":["30MIN"]}`)
		if !bytes.Equal(REQBody, expectedBody) {
			t.Errorf("Expected to send a POST request with body: %s, got: %s", AEMO_POST_PAYLOAD, r.Body)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))
	defer server.Close()

	aemo := NewAEMO()

	aemoData, err := aemo.GetAEMOData(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if aemoData.Intervals[0].RegionID != "NSW1" {
		t.Fatal("Data didn't deserialise properly")
	}

	if aemoData.Intervals[0].SettlementDate.String() != "2024-01-30 16:35:00 +1000 AEST" {
		t.Fatal("Data didn't deserialise properly")
	}
}

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

func TestGidBotBasicInterval(t *testing.T) {
	cfg := config{}
	cfg.TestMode = true

	gridBot := NewGridBot(cfg, "Queensland")
	if want, got := "Queensland", gridBot.regionString; want != got {
		t.Errorf("Expected %s, got %s", want, got)
	}

	go gridBot.Mainloop()

	peakTime := time.Now().Add(1 * time.Hour)
	peakRRP := float64(301)
	interval := NewForecastInterval(gridBot, peakRRP, peakTime, t)
	gridBot.GetIntervalChannel() <- interval
	// Throw in a cheeky actual interval to make sure it doesn't get tooted
	interval.PeriodType = "ACTUAL"
	interval.RRP = peakRRP * 2
	interval.SettlementDate = JSONTime{time.Now().Add(2 * time.Hour)}
	gridBot.GetIntervalChannel() <- interval
	CommitIntervals(gridBot, t)
	ValidateToot(gridBot, peakRRP, peakTime, FormatExpectedToot(peakRRP, peakTime, "Queensland", 0, PEAK), t)

}
func TestGidBotBasicPeak(t *testing.T) {
	cfg := config{}
	cfg.TestMode = true

	gridBot := NewGridBot(cfg, "Queensland")
	if want, got := "Queensland", gridBot.regionString; want != got {
		t.Errorf("Expected %s, got %s", want, got)
	}

	go gridBot.Mainloop()

	peakTime := time.Now().Add(2 * time.Hour)
	peakRRP := float64(INTERESTING_PEAK_RRP * 3)

	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(-1*time.Hour), t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP, peakTime, t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(1*time.Hour), t)
	CommitIntervals(gridBot, t)
	ValidateToot(gridBot, peakRRP, peakTime, FormatExpectedToot(peakRRP, peakTime, "Queensland", 0, PEAK), t)

	oldPeak := peakRRP
	// Cancel the peak
	peakRRP = float64(INTERESTING_PEAK_RRP - 1)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(-1*time.Hour), t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP, peakTime, t)
	gridBot.GetIntervalChannel() <- NewForecastInterval(gridBot, peakRRP/2, peakTime.Add(1*time.Hour), t)
	CommitIntervals(gridBot, t)
	ValidateToot(gridBot, peakRRP, peakTime, FormatExpectedToot(oldPeak, peakTime, "Queensland", 0, CANCELLED), t)

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

}
