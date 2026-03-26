package duration

import (
	"fmt"
	"math"
	"math/big"

	"google.golang.org/protobuf/types/known/durationpb"
)

const TotalPrecision = 64 + 32 + 3 // for seconds and nanos + 3 bits of margin

func f(val float64) *big.Float {
	return big.NewFloat(val).SetPrec(TotalPrecision)
}

var (
	BIMaxInt64 = big.NewInt(math.MaxInt64)
	FThousand  = f(1000)
	FSixty     = f(60)
	FTen       = f(10)

	FNanosecond  = f(1)
	FMicrosecond = f(1).Mul(FThousand, FNanosecond)
	FMillisecond = f(1).Mul(FThousand, FMicrosecond)
	FSecond      = f(1).Mul(FThousand, FMillisecond)
	FMinute      = f(1).Mul(FSixty, FSecond)
	FHour        = f(1).Mul(FSixty, FMinute)
	FDay         = f(1).Mul(f(24), FHour)
)

var unitMap = map[string]*big.Float{
	"ns": FNanosecond,
	"us": FMicrosecond,
	"µs": FMicrosecond, // U+00B5 = micro symbol
	"μs": FMicrosecond, // U+03BC = Greek letter mu
	"ms": FMillisecond,
	"s":  FSecond,
	"m":  FMinute,
	"h":  FHour,
	"d":  FDay,
}

// consumeDigits consumes the leading [0-9]* from s.
func consumeDigits[bytes []byte | string](s bytes) (int, bytes) {
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
	}
	return i, s[i:]
}

// ParseDuration parses a duration string.
// A duration string is a possibly signed sequence of
// decimal numbers, each with optional fraction and a unit suffix,
// such as "300ms", "-1.5h" or "2h45m".
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h", "d".
// Bigger intervals like month and year can not be used out of context as they
// can not be converted to other values without it.
// (eg February < August and 2005 < 2004).
// The result is truncated to nanosecond precision using floor(), not round()
// Empty string results in a 0 duration, because it is a zero-value default
// string, and should logically be equal to zero duration.
// Duration 0 may be without unit.
func ParseDuration(str string) (*durationpb.Duration, error) {
	// [-+]?([0-9]*(\.[0-9]*)?[a-z]+)+
	if str == "" {
		return durationpb.New(0), nil
	}

	original := str
	floatDuration := f(0)
	isNegative := false

	// Consume [-+]?
	if str != "" {
		c := str[0]
		if c == '-' || c == '+' {
			isNegative = c == '-'
			str = str[1:]
		}
	}
	// Special case: if all that is left is "0", this is zero.
	if str == "0" {
		return durationpb.New(0), nil
	}
	if str == "" {
		return nil, fmt.Errorf("empty duration with sign")
	}
	for str != "" {
		// The next character must be [0-9.]
		if str[0] != '.' && (str[0] < '0' || str[0] > '9') {
			return nil, fmt.Errorf("duration part should start with digit or point %q", original)
		}
		// Consume [0-9]*
		unitStart := str
		var intPart, fracPart int
		intPart, str = consumeDigits(str)
		pre := intPart != 0 // whether we consumed anything before a period

		post := false
		if str != "" && str[0] == '.' {
			str = str[1:]
			fracPart, str = consumeDigits(str)
			post = fracPart != 0
		}
		if !pre && !post {
			// no digits (e.g. ".s" or "-.s")
			return nil, fmt.Errorf("duration part should have at least one digit %q", original)
		}
		var unitValueString string
		if post {
			unitValueString = unitStart[:intPart+1+fracPart]
		} else {
			unitValueString = unitStart[:intPart]
		}

		unitValue, _, err := f(0).Parse(unitValueString, 10)
		if err != nil {
			return nil, fmt.Errorf("parsing value %q of %q: %v", unitValueString, original, err)
		}

		// Consume unit.
		i := 0
		for ; i < len(str); i++ {
			c := str[i]
			if c == '.' || ('0' <= c && c <= '9') {
				break
			}
		}
		if i == 0 {
			return nil, fmt.Errorf("missing unit in duration %q", original)
		}
		u := str[:i]
		str = str[i:]
		unit, ok := unitMap[u]
		if !ok {
			return nil, fmt.Errorf("unknown unit %q in duration %q", u, original)
		}
		unitValue.Mul(unitValue, unit)
		floatDuration.Add(floatDuration, unitValue)
	}

	seconds, _ := f(0).Quo(floatDuration, FSecond).Int(big.NewInt(0))
	secFloat := f(0).SetInt(seconds)
	secondsMul := f(0).Mul(secFloat, FSecond)
	nanos := f(0).Sub(floatDuration, secondsMul)

	if seconds.Cmp(BIMaxInt64) > 0 {
		return nil, fmt.Errorf("overflow in duration %q", original)
	}

	secondsInt := seconds.Int64()
	nanosInt, _ := nanos.Int64()

	result := &durationpb.Duration{
		Seconds: secondsInt,
		Nanos:   int32(nanosInt),
	}

	if isNegative {
		result.Seconds = -result.GetSeconds()
		result.Nanos = -result.GetNanos()
	}
	return result, nil
}
