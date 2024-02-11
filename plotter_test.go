package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestGetPlot(t *testing.T) {
	// TestGetPlot tests the GetPlot function

	labels := []time.Time{
		time.Date(2024, 1, 30, 16, 35, 0, 0, time.UTC),
		time.Date(2024, 1, 30, 16, 40, 0, 0, time.UTC),
		time.Date(2024, 1, 30, 16, 45, 0, 0, time.UTC),
		time.Date(2024, 1, 30, 16, 50, 0, 0, time.UTC),
		time.Date(2024, 1, 30, 16, 55, 0, 0, time.UTC),
	}
	values := []float64{5, 3, 7, 8, 6}

	// Open a io.Writer to a file to capture the output
	file, err := os.Create("test.png")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	GetPlot(labels, values, file)
}

func TestPlotData(t *testing.T) {
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

	labels := make([]time.Time, 0)
	values := make([]float64, 0)

	for i, interval := range aemoData.Intervals {
		if aemoData.Intervals[i].RegionID != "QLD1" {
			continue
		}
		if aemoData.Intervals[i].PeriodType != "FORECAST" {
			continue
		}
		// Discard time points outside 17:00 to 21:00
		if !(interval.SettlementDate.Hour() >= 17 && interval.SettlementDate.Hour() <= 21) {
			continue
		}
		if len(labels) > 8 {
			continue
		}
		labels = append(labels, interval.SettlementDate.Time)
		values = append(values, interval.RRP)
	}

	file, err := os.Create("test.png")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	GetPlot(labels, values, file)
}
