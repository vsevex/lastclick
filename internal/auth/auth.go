package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

const initDataMaxAge = 5 * time.Minute

// ValidateInitData validates Telegram Mini App initData according to
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
func ValidateInitData(initData, botToken string) error {
	vals, err := url.ParseQuery(initData)
	if err != nil {
		return fmt.Errorf("parse init data: %w", err)
	}

	receivedHash := vals.Get("hash")
	if receivedHash == "" {
		return fmt.Errorf("missing hash")
	}

	authDate := vals.Get("auth_date")
	if authDate == "" {
		return fmt.Errorf("missing auth_date")
	}

	var ts int64
	if _, err := fmt.Sscanf(authDate, "%d", &ts); err != nil {
		return fmt.Errorf("invalid auth_date: %w", err)
	}

	if time.Since(time.Unix(ts, 0)) > initDataMaxAge {
		return fmt.Errorf("init data expired")
	}

	dataCheckString := buildDataCheckString(vals)

	secretKey := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	computed := hex.EncodeToString(hmacSHA256(secretKey, []byte(dataCheckString)))

	if computed != receivedHash {
		return fmt.Errorf("hash mismatch")
	}

	return nil
}

func buildDataCheckString(vals url.Values) string {
	keys := make([]string, 0, len(vals))
	for k := range vals {
		if k == "hash" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + vals.Get(k)
	}
	return strings.Join(parts, "\n")
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
