package duration

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	NMicros = int32(1000)
	NMillis = NMicros * 1000
	SMinute = int64(60)
	SHour   = SMinute * 60
	SDay    = SHour * 24
)

// FormatDuration formats duration as NdNhNmNsNmsNusNns ommitting parts that are
// zeroes. If duration is 0, it will return 0s.
func FormatDuration(d *durationpb.Duration) string {
	if d == nil {
		return ""
	}
	if d.GetSeconds() == 0 && d.GetNanos() == 0 {
		return "0s"
	}
	isNegative := d.GetSeconds() < 0 || d.GetNanos() < 0

	// Extract the components from the duration
	seconds := d.GetSeconds()
	nanoseconds := d.GetNanos()
	if seconds < 0 {
		seconds = -seconds
	}
	if nanoseconds < 0 {
		nanoseconds = -nanoseconds
	}

	// Compute each time unit
	days := seconds / SDay
	seconds %= SDay

	hours := seconds / SHour
	seconds %= SHour

	minutes := seconds / SMinute
	seconds %= SMinute

	milliseconds := nanoseconds / NMillis
	nanoseconds %= NMillis

	microseconds := nanoseconds / NMicros
	nanoseconds %= NMicros

	// Build the result string
	var parts []string
	if isNegative {
		parts = append(parts, "-")
	}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	if milliseconds > 0 {
		parts = append(parts, fmt.Sprintf("%dms", milliseconds))
	}
	if microseconds > 0 {
		parts = append(parts, fmt.Sprintf("%dus", microseconds))
	}
	if nanoseconds > 0 {
		parts = append(parts, fmt.Sprintf("%dns", nanoseconds))
	}

	return strings.Join(parts, "")
}
