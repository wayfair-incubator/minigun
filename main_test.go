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
