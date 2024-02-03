package main

import (
	"net/http"
	"strings"

	"golang.org/x/time/rate"
)

const AEMO_URL = "https://aemo.com.au/aemo/apps/api/report/5MIN"

// It would probably be overkill to make this an actual structure...
const AEMO_POST_PAYLOAD = "{\"timeScale\":[\"30MIN\"]}"

type AEMO struct {
	client *RLHTTPClient
}

func NewAEMO() *AEMO {
	var limiter *rate.Limiter
	a := AEMO{}
	a.client = &RLHTTPClient{
		client: &http.Client{
			Transport: &http.Transport{},
		},
		Ratelimiter: limiter,
	}
	return &a
}

func (aemo *AEMO) GetAEMOData() (AEMOData, error) {
	// Fetch AEMO data from the AEMO API

	// Send a POST request to AEMO_URL with AEMO_POST_PAYLOAD

	body := strings.NewReader(AEMO_POST_PAYLOAD)
	POSTReq, err := http.NewRequest("POST", AEMO_URL, body)
	if err != nil {
		return AEMOData{}, err
	}

	POSTResp, err := aemo.client.Do(POSTReq)

	if err != nil {
		return AEMOData{}, err
	}

	// Check the response status code
	if POSTResp.StatusCode != 200 {
		return AEMOData{}, HTTPError{POSTResp.StatusCode}
	}

	// Parse the data into an AEMOData structure
	// Return the AEMOData structure

	return AEMOData{}, nil
}
