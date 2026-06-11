package utils

import (
	"time"
)

const (
	DefaultCheckInHour   = 10
	DefaultCheckInMinute = 0
	DefaultCheckOutHour  = 9
	DefaultCheckOutMinute = 30
)

func ParseLocalToUTC(datetimeStr string, timezoneStr string, isCheckIn bool) (time.Time, error) {
	loc, err := time.LoadLocation(timezoneStr)
	if err != nil {
		return time.Time{}, err
	}

	t, err := time.ParseInLocation("2006-01-02T15:04", datetimeStr, loc)
	if err != nil {
		t, err = time.ParseInLocation("2006-01-02", datetimeStr, loc)
		if err != nil {
			return time.Time{}, err
		}

		if isCheckIn {
			t = time.Date(t.Year(), t.Month(), t.Day(), DefaultCheckInHour, DefaultCheckInMinute, 0, 0, loc)
		} else {
			t = time.Date(t.Year(), t.Month(), t.Day(), DefaultCheckOutHour, DefaultCheckOutMinute, 0, 0, loc)
		}
	}

	return t.UTC(), nil
}
