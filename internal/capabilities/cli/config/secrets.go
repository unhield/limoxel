package config

import (
	"strings"
)

// MaskedValueConstant represents the universal redaction placeholder.
const MaskedValueConstant = "***REDACTED***"

// secretKeywords contains lowercase substrings indicating sensitive configuration keys.
var secretKeywords = []string{
	"token",
	"secret",
	"password",
	"passwd",
	"auth",
	"credential",
	"api_key",
	"apikey",
	"private_key",
	"access_key",
	"client_secret",
}

// IsSecretKey checks whether a configuration key is classified as sensitive.
func IsSecretKey(key string) bool {
	clean := strings.ToLower(strings.TrimSpace(key))
	for _, kw := range secretKeywords {
		if strings.Contains(clean, kw) {
			return true
		}
	}
	return false
}

// MaskValue returns the masked placeholder if the key is a secret, otherwise returns raw value.
func MaskValue(key string, val any) any {
	if val == nil {
		return nil
	}
	if IsSecretKey(key) {
		return MaskedValueConstant
	}
	return val
}

// RedactMap deeply sanitizes a map by masking any sensitive keys.
func RedactMap(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	result := make(map[string]any, len(data))
	for k, v := range data {
		if IsSecretKey(k) {
			result[k] = MaskedValueConstant
			continue
		}
		if subMap, ok := v.(map[string]any); ok {
			result[k] = RedactMap(subMap)
		} else {
			result[k] = v
		}
	}
	return result
}
