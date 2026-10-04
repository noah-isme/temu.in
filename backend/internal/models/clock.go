package models

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"gorm.io/gorm/schema"
)

// clockSerializer maps a time.Time struct field onto a Postgres `time`
// column. The pgx driver returns `time` values as strings ("15:04:05"), which
// database/sql cannot scan into time.Time, so without this every read of a
// booking row fails. Values are written as wall-clock strings (FormatClock).
type clockSerializer struct{}

func init() {
	schema.RegisterSerializer("clock", clockSerializer{})
}

var clockLayouts = []string{"15:04:05.999999", "15:04:05", "15:04"}

const (
	clockWallLayout   = "15:04:05.999999"
	clockWallHMLayout = "15:04"
	// clockEndOfDay is the text Postgres uses for midnight at the end of the
	// day. It is a valid `time` value (and a valid end_time under the
	// start < end CHECK), but time.Parse rejects hour 24.
	clockEndOfDay   = "24:00:00"
	clockEndOfDayHM = "24:00"
)

// EndOfDayClock is how a "24:00:00" `time` value is represented: midnight on
// the day after the zero date. It equals time.Date(0, 1, 1, 23, 0, 0, 0,
// UTC).Add(time.Hour), so duration arithmetic from a start time lands on it.
var EndOfDayClock = time.Date(0, 1, 2, 0, 0, 0, 0, time.UTC)

// isEndOfDayClock reports whether t's wall clock is the end-of-day sentinel.
func isEndOfDayClock(t time.Time) bool {
	return t.Year() == 0 && t.Month() == time.January && t.Day() == 2 &&
		t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
}

// FormatClock renders t's wall clock (ignoring its zone and date) as the text
// bound to a Postgres `time` column: "15:04:05.999999", or "24:00:00" for
// EndOfDayClock. It is the single formatter for every `time` write and bind,
// so a non-UTC time.Time can never shift the wall clock.
func FormatClock(t time.Time) string {
	if isEndOfDayClock(t) {
		return clockEndOfDay
	}
	return t.Format(clockWallLayout)
}

// FormatClockHM renders t's wall clock as "15:04" for API responses and
// notifications, or "24:00" for EndOfDayClock.
func FormatClockHM(t time.Time) string {
	if isEndOfDayClock(t) {
		return clockEndOfDayHM
	}
	return t.Format(clockWallHMLayout)
}

// ParseClock parses a Postgres `time` text value ("15:04", "15:04:05" or with
// fractional seconds) into a time.Time on the zero date (0000-01-01 UTC).
// "24:00", "24:00:00" and "24:00:00.000000" yield EndOfDayClock.
func ParseClock(value string) (time.Time, error) {
	switch value {
	case "24:00", "24:00:00", "24:00:00.0", "24:00:00.000000":
		return EndOfDayClock, nil
	}
	for _, layout := range clockLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("parse clock value %q", value)
}

func (clockSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	var parsed time.Time
	switch v := dbValue.(type) {
	case nil:
		// Leave the zero value.
	case time.Time:
		parsed = time.Date(0, 1, 1, v.Hour(), v.Minute(), v.Second(), v.Nanosecond(), time.UTC)
	case string:
		t, err := ParseClock(v)
		if err != nil {
			return err
		}
		parsed = t
	case []byte:
		t, err := ParseClock(string(v))
		if err != nil {
			return err
		}
		parsed = t
	default:
		return fmt.Errorf("unsupported clock value type %T", dbValue)
	}
	return field.Set(ctx, dst, parsed)
}

func (clockSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	switch v := fieldValue.(type) {
	case time.Time:
		return FormatClock(v), nil
	case *time.Time:
		if v == nil {
			return nil, nil
		}
		return FormatClock(*v), nil
	default:
		return nil, fmt.Errorf("unsupported clock field type %T", fieldValue)
	}
}
