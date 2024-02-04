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
}

func FeedForecastInterval(gridBot *GridBot, intervalRRP float64, intervalTime time.Time, t *testing.T) {
	interval := Interval{
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

	gridBot.GetIntervalChannel() <- interval
}

func FormatExpectedToot(intervalRRP float64, intervalTime time.Time, region string, peak bool) string {
	if peak {
		return fmt.Sprintf(PEAK_TOOT_FORMAT, "Queensland", intervalRRP/1000, intervalTime.Format("15:04"))
	} else {
		return fmt.Sprintf(PEAK_CANCELLED_TOOT_FORMAT, "Queensland", intervalRRP/1000, intervalTime.Format("15:04"))
	}
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
	FeedForecastInterval(gridBot, peakRRP, peakTime, t)
	CommitIntervals(gridBot, t)
	ValidateToot(gridBot, peakRRP, peakTime, FormatExpectedToot(peakRRP, peakTime, "Queensland", true), t)

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

	FeedForecastInterval(gridBot, peakRRP/2, peakTime.Add(-1*time.Hour), t)
	FeedForecastInterval(gridBot, peakRRP, peakTime, t)
	FeedForecastInterval(gridBot, peakRRP/2, peakTime.Add(1*time.Hour), t)
	CommitIntervals(gridBot, t)
	ValidateToot(gridBot, peakRRP, peakTime, FormatExpectedToot(peakRRP, peakTime, "Queensland", true), t)

	gridBot.lastToot = ""
	oldPeak := peakRRP
	// Cancel the peak
	peakRRP = float64(INTERESTING_PEAK_RRP - 1)
	FeedForecastInterval(gridBot, peakRRP/2, peakTime.Add(-1*time.Hour), t)
	FeedForecastInterval(gridBot, peakRRP, peakTime, t)
	FeedForecastInterval(gridBot, peakRRP/2, peakTime.Add(1*time.Hour), t)
	CommitIntervals(gridBot, t)
	ValidateToot(gridBot, peakRRP, peakTime, FormatExpectedToot(oldPeak, peakTime, "Queensland", false), t)

}
