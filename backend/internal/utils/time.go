package utils

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"
)

var daysRegex = regexp.MustCompile(`(\d+)d`)
var fractionalDaysRegex = regexp.MustCompile(`\d*\.\d+d`)

// ParseDurationWithDays parses a duration string that may include "d" for days.
// Converts days to 24 hours and passes the rest to time.ParseDuration.
func ParseDurationWithDays(s string) (time.Duration, error) {
	if fractionalDaysRegex.MatchString(s) {
		return 0, fmt.Errorf("fractional days are not supported")
	}
	var conversionErr error
	s = daysRegex.ReplaceAllStringFunc(s, func(match string) string {
		daysStr := match[:len(match)-1]
		days, err := strconv.ParseUint(daysStr, 10, 64)
		if err != nil || days > uint64(math.MaxInt64/int64(24*time.Hour)) {
			conversionErr = fmt.Errorf("duration overflow")
			return match
		}
		hours := days * 24
		return fmt.Sprintf("%dh", hours)
	})

	if conversionErr != nil {
		return 0, conversionErr
	}
	return time.ParseDuration(s)
}
