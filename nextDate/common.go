package nextDate

import "time"

func LastDayOfMonth(srcTime time.Time, dayOfMonth int) time.Time {
	if dayOfMonth > 28 {
		var date time.Time
		for i := dayOfMonth; i >= 28; i = i - 1 {
			date = time.Date(srcTime.Year(), srcTime.Month(), i, srcTime.Hour(), srcTime.Minute(),
				srcTime.Second(), srcTime.Nanosecond(), srcTime.Location())
			if date.Day() == i { break }
		}
		return date
	}
	return time.Date(srcTime.Year(), srcTime.Month(), dayOfMonth, srcTime.Hour(), srcTime.Minute(), srcTime.Second(),
		srcTime.Nanosecond(), srcTime.Location())
}


func DayBeforeWeekend(srcTime time.Time) time.Time {
	date := srcTime
	for {
		weekday := date.Weekday()
		if weekday > 0 && weekday < 6 { break }
		date = date.AddDate(0, 0, -1)
	}
	return date
}
