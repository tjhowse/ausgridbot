package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

	if aemoData.Intervals[0].SettlementDate.String() != "2024-01-30 16:35:00 +0000 UTC" {
		t.Fatal("Data didn't deserialise properly")
	}
}
