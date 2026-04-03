package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var daysRegex = regexp.MustCompile(`(\d+)d`)

// ParseDurationWithDays parses a duration string that may include "d" for days.
// Converts days to 24 hours and passes the rest to time.ParseDuration.
func ParseDurationWithDays(s string) (time.Duration, error) {
	s = daysRegex.ReplaceAllStringFunc(s, func(match string) string {
		daysStr := match[:len(match)-1]
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return match
		}
		hours := days * 24
		return fmt.Sprintf("%dh", hours)
	})

	return time.ParseDuration(s)
}
