package utils

import (
	"strconv"
	"strings"
	"time"
)

// ParseSize parses size string (e.g., "1KB", "1.5MB", "2GB") into bytes
func ParseSize(s string) (int64, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return 0, nil
	}
	
	multiplier := int64(1)
	if strings.HasSuffix(s, "KB") {
		multiplier = 1024
		s = s[:len(s)-2]
	} else if strings.HasSuffix(s, "MB") {
		multiplier = 1024 * 1024
		s = s[:len(s)-2]
	} else if strings.HasSuffix(s, "GB") {
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-2]
	} else if strings.HasSuffix(s, "B") {
		s = s[:len(s)-1]
	}
	
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	
	return int64(value * float64(multiplier)), nil
}

// ParseAge parses age string (e.g., "1h", "2d", "1w", "1m") into duration
func ParseAge(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, nil
	}
	
	multiplier := time.Hour
	if strings.HasSuffix(s, "h") {
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "d") {
		multiplier = time.Hour * 24
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "w") {
		multiplier = time.Hour * 24 * 7
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "m") {
		multiplier = time.Hour * 24 * 30
		s = s[:len(s)-1]
	}
	
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	
	return time.Duration(float64(multiplier) * value), nil
}

// SplitCommaList splits comma-separated string into slice of strings
func SplitCommaList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
} 