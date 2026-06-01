package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var utc = time.UTC

type Expression struct {
	minute field
	hour   field
	day    field
	month  field
	week   field
}

type field struct {
	all    bool
	values map[int]struct{}
}

func ValidateCron(expr string) error {
	_, err := ParseCron(expr)
	return err
}

func ParseCron(expr string) (*Expression, error) {
	parts := strings.Fields(strings.TrimSpace(expr))
	if len(parts) != 5 {
		return nil, fmt.Errorf("cron schedule must have 5 fields")
	}

	minute, err := parseField(parts[0], 0, 59, false)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field: %w", err)
	}
	hour, err := parseField(parts[1], 0, 23, false)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field: %w", err)
	}
	day, err := parseField(parts[2], 1, 31, false)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-month field: %w", err)
	}
	month, err := parseField(parts[3], 1, 12, false)
	if err != nil {
		return nil, fmt.Errorf("invalid month field: %w", err)
	}
	week, err := parseField(parts[4], 0, 7, true)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-week field: %w", err)
	}

	return &Expression{
		minute: minute,
		hour:   hour,
		day:    day,
		month:  month,
		week:   week,
	}, nil
}

func (e *Expression) PreviousOrSame(now time.Time) (time.Time, bool) {
	current := now.In(utc).Truncate(time.Minute)
	for i := 0; i <= 366*24*60; i++ {
		if e.matches(current) {
			return current, true
		}
		current = current.Add(-time.Minute)
	}
	return time.Time{}, false
}

func (e *Expression) matches(ts time.Time) bool {
	if !e.minute.has(ts.Minute()) || !e.hour.has(ts.Hour()) || !e.month.has(int(ts.Month())) {
		return false
	}

	dayMatch := e.day.has(ts.Day())
	weekMatch := e.week.has(int(ts.Weekday()))

	switch {
	case e.day.all && e.week.all:
		return dayMatch && weekMatch
	case e.day.all:
		return weekMatch
	case e.week.all:
		return dayMatch
	default:
		return dayMatch || weekMatch
	}
}

func (f field) has(v int) bool {
	if f.all {
		return true
	}
	_, ok := f.values[v]
	return ok
}

func parseField(raw string, min, max int, sundayAlias bool) (field, error) {
	if raw == "*" {
		return field{all: true}, nil
	}

	values := make(map[int]struct{})
	for _, token := range strings.Split(raw, ",") {
		if token == "" {
			return field{}, fmt.Errorf("empty token")
		}
		if err := addToken(values, token, min, max, sundayAlias); err != nil {
			return field{}, err
		}
	}

	return field{values: values}, nil
}

func addToken(values map[int]struct{}, token string, min, max int, sundayAlias bool) error {
	base := token
	step := 1
	if strings.Contains(token, "/") {
		parts := strings.Split(token, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid step token %q", token)
		}
		base = parts[0]
		parsed, err := strconv.Atoi(parts[1])
		if err != nil || parsed <= 0 {
			return fmt.Errorf("invalid step token %q", token)
		}
		step = parsed
	}

	start := min
	end := max
	switch {
	case base == "*":
	case strings.Contains(base, "-"):
		rangeParts := strings.Split(base, "-")
		if len(rangeParts) != 2 {
			return fmt.Errorf("invalid range %q", token)
		}
		var err error
		start, err = parseValue(rangeParts[0], min, max, sundayAlias)
		if err != nil {
			return err
		}
		end, err = parseValue(rangeParts[1], min, max, sundayAlias)
		if err != nil {
			return err
		}
	default:
		value, err := parseValue(base, min, max, sundayAlias)
		if err != nil {
			return err
		}
		start = value
		end = value
	}

	if start > end {
		return fmt.Errorf("invalid range %q", token)
	}
	for value := start; value <= end; value += step {
		values[value] = struct{}{}
	}
	return nil
}

func parseValue(raw string, min, max int, sundayAlias bool) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid value %q", raw)
	}
	if sundayAlias && value == 7 {
		value = 0
	}
	if value < min || value > max || (sundayAlias && value == 7) {
		return 0, fmt.Errorf("value %q out of range", raw)
	}
	return value, nil
}
