package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const AEMO_HOST = "https://aemo.com.au"
const AEMO_URL = "/aemo/apps/api/report/5MIN"

// It would probably be overkill to make this an actual structure...
const AEMO_POST_PAYLOAD = `{"timeScale":["30MIN"]}`

type AEMO struct {
	client *RLHTTPClient
}

func NewAEMO() *AEMO {
	limiter := rate.NewLimiter(rate.Every(1*time.Second), 1)
	a := &AEMO{}
	a.client = &RLHTTPClient{
		client: &http.Client{
			Transport: &http.Transport{},
		},
		Ratelimiter: limiter,
	}
	return a
}

func (aemo *AEMO) GetAEMOData(HostUrl string) (AEMOData, error) {
	// Fetch AEMO data from the AEMO API

	// Send a POST request to AEMO_URL with AEMO_POST_PAYLOAD
	if HostUrl == "" {
		HostUrl = AEMO_HOST
	}
	REQBody := strings.NewReader(AEMO_POST_PAYLOAD)
	POSTReq, err := http.NewRequest("POST", HostUrl+AEMO_URL, REQBody)
	if err != nil {
		return AEMOData{}, err
	}
	POSTReq.Header.Set("Content-Type", "application/json")

	POSTResp, err := aemo.client.Do(POSTReq)

	if err != nil {
		return AEMOData{}, err
	}

	// Check the response status code
	if POSTResp.StatusCode != 200 {
		return AEMOData{}, fmt.Errorf("got status code %d", POSTResp.StatusCode)
	}
	// Read the response body
	RESPBody, err := io.ReadAll(POSTResp.Body)
	if err != nil {
		return AEMOData{}, err
	}

	if POSTResp.Header.Get("Content-Encoding") == "gzip" {
		var r io.Reader
		if r, err = gzip.NewReader(bytes.NewReader(RESPBody)); err != nil {
			return AEMOData{}, err
		}
		if RESPBody, err = io.ReadAll(r); err != nil {
			return AEMOData{}, err
		}
	}

	// Parse the data into an AEMOData structure
	var decoded AEMOData
	if err = json.Unmarshal(RESPBody, &decoded); err != nil {
		return AEMOData{}, err
	}

	// Return the AEMOData structure
	return decoded, nil
}
