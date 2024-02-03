package main

import (
	"encoding/json"
	"os"
	"testing"
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

	if aemoData.Intervals[0].SettlementDate.String() != "2024-01-30 16:35:00 +0000 UTC" {
		t.Fatal("Data didn't deserialise properly")
	}
}
