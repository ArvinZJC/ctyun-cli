package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/config"
)

type EOPRequest struct {
	Query     string
	Body      []byte
	Date      string
	RequestID string
}

func GenerateEOPAuthorization(req EOPRequest, creds config.Credentials) string {
	if creds.AccessKey == "" || creds.SecretKey == "" {
		return ""
	}

	hash := sha256.Sum256(req.Body)
	bodyHash := hex.EncodeToString(hash[:])
	canonical := fmt.Sprintf(
		"ctyun-eop-request-id:%s\neop-date:%s\n\n%s\n%s",
		req.RequestID,
		req.Date,
		req.Query,
		bodyHash,
	)

	dateKey := hmacSHA256(req.Date, creds.SecretKey)
	akKey := hmacSHA256(creds.AccessKey, string(dateKey))
	dayKey := hmacSHA256(req.Date[:8], string(akKey))
	signature := hmacSHA256(canonical, string(dayKey))

	return creds.AccessKey + " Headers=ctyun-eop-request-id;eop-date Signature=" +
		base64.StdEncoding.EncodeToString(signature)
}

func RedactSecrets(input string, secrets []string) string {
	redacted := input
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, secret, "[REDACTED]")
	}
	return redacted
}

func hmacSHA256(message, key string) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return mac.Sum(nil)
}
