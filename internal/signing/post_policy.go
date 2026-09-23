/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package signing

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
)

// SignPOSTPolicyV2 signs the exact base64 policy text with the storage secret key.
// SHA-1 is required by the storage POST V2 protocol, independently of EOP signing.
func SignPOSTPolicyV2(policy, secret string) string {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write([]byte(policy))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
