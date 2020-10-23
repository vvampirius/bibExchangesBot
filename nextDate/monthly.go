package nextDate

import (
	"log"
	"time"
)

func NearMonthlyDate(srcTime time.Time, dayOfMonth int, timeOfDay string, beforeWeekend bool) time.Time {
	timeOfDayTime, err := time.ParseInLocation("15:04:05", timeOfDay, time.FixedZone(`Europe/Minsk`, 3*60*60))
	if err != nil {
		log.Println(timeOfDay, err.Error())
		timeOfDayTime = time.Now()
	}

	thisMonth := time.Date(srcTime.Year(), srcTime.Month(), srcTime.Day(), timeOfDayTime.Hour(), timeOfDayTime.Minute(),
		timeOfDayTime.Second(), 0, timeOfDayTime.Location())
	thisMonth = LastDayOfMonth(thisMonth, dayOfMonth)
	if beforeWeekend { thisMonth = DayBeforeWeekend(thisMonth) }
	if srcTime.Before(thisMonth) { return thisMonth }

	nextMonth := time.Date(srcTime.Year(), srcTime.Month(), srcTime.Day(), timeOfDayTime.Hour(), timeOfDayTime.Minute(),
		timeOfDayTime.Second(), 0, timeOfDayTime.Location()).AddDate(0, 1, 0)
	nextMonth = LastDayOfMonth(nextMonth, dayOfMonth)
	if beforeWeekend { nextMonth = DayBeforeWeekend(nextMonth) }
	return nextMonth
}
