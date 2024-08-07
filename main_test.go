// Simple HTTP benchmark tool
//
// @authors Minigun Maintainers
// @copyright 2020 Wayfair, LLC -- All rights reserved.
package main

import "testing"

// TODO: write more tests!

func TestRandomBytes(t *testing.T) {

	result := randomBytes(512)

	if len(result) != 512 {
		t.Errorf("randomBytes(512) expected to return 512 length result, got %d",
			len(result))
	}
}

func TestHeadersParsing(t *testing.T) {
	var sendHTTPHeaders httpHeaders

	sendHTTPHeaders.Set("Host: hostname.local")
	sendHTTPHeaders.Set("H1  : V1,V2  ")
	sendHTTPHeaders.Set("Authorization: id:hash")
	sendHTTPHeaders.Set("Keep-Alive: timeout=2, max=100")
	sendHTTPHeaders.Set("Content-Type: application/xml")
	sendHTTPHeaders.Set("Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8   ")

	expected := httpHeaders{
		"Authorization": "id:hash",
		"H1":            "V1,V2",
		"Host":          "hostname.local",
		"Keep-Alive":    "timeout=2, max=100",
		"Content-Type":  "application/xml",
		"Accept":        "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	}

	for k, v := range expected {
		if sendHTTPHeaders[k] != v {
			t.Errorf("Expected header: %q => %q, got header:  %q => %q", k, v, k, sendHTTPHeaders[k])
		}
	}
}

func TestHumanizeDurationSeconds(t *testing.T) {
	requestTime := humanizeDurationSeconds(6000)
	if requestTime != "6000.00s" {
		t.Errorf("Wrong requestTime: %q", requestTime)
	}

	requestTime = humanizeDurationSeconds(0.05)
	if requestTime != "50.00ms" {
		t.Errorf("Wrong requestTime: %q", requestTime)
	}

	requestTime = humanizeDurationSeconds(99999999)
	if requestTime != "99999999.00s" {
		t.Errorf("Wrong requestTime: %q", requestTime)
	}
}
