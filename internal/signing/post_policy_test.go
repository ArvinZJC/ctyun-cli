/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package signing

import "testing"

// TestPOSTPolicyV2SignsExactEncodedPolicy guards against decoding or reserializing the signed text.
func TestPOSTPolicyV2SignsExactEncodedPolicy(t *testing.T) {
	if got := SignPOSTPolicyV2("policy", "secret"); got != "DQYKsMXInaBeQ2OuQ0nMglptC1c=" {
		t.Fatal(got)
	}
}
