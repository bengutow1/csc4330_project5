// Author: Caden Lund
package points

import (
	"fmt"
	"sync"
	"time"
	// The zone database rides in the binary, so the campus zone loads even on a host that has
	// none, and Campus cannot fail.
	_ "time/tzdata"
)

// campusZone is the campus time zone, named once.
const campusZone = "America/Chicago"

// Campus is the campus time zone. Every day, week and meal-window boundary is drawn in it.
var Campus = sync.OnceValue(func() *time.Location {
	loc, err := time.LoadLocation(campusZone)
	if err != nil {
		// Unreachable: time/tzdata embeds every zone.
		panic(fmt.Sprintf("points: load %s: %v", campusZone, err))
	}
	return loc
})

// DayOf is the campus day t falls on, as that day's midnight in campus time.
func DayOf(t time.Time) time.Time {
	y, m, d := t.In(Campus()).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, Campus())
}

// WeekStart is the Monday 00:00, campus time, that starts t's week.
func WeekStart(t time.Time) time.Time {
	day := DayOf(t)
	// Weekday counts from Sunday, and a week here starts on Monday.
	sinceMonday := (int(day.Weekday()) + 6) % 7
	// AddDate steps in calendar days, so a 23- or 25-hour day still lands on midnight.
	return day.AddDate(0, 0, -sinceMonday)
}

// Window names one of the four meal windows. A player earns at most one cook credit per window.
type Window string

const (
	WindowMorning   Window = "morning"    // 05:00 to 10:59
	WindowMidday    Window = "midday"     // 11:00 to 15:59
	WindowEvening   Window = "evening"    // 16:00 to 21:59
	WindowLateNight Window = "late_night" // 22:00 to 04:59
)

// MealWindow is the meal window t falls in and the campus day that window belongs to. Late night
// runs past midnight and belongs to the day it started, so 00:30 on a Tuesday is Monday's late
// night.
func MealWindow(t time.Time) (window Window, day time.Time) {
	d := DayOf(t)
	hour := t.In(Campus()).Hour()
	switch {
	case hour < 5:
		return WindowLateNight, d.AddDate(0, 0, -1)
	case hour < 11:
		return WindowMorning, d
	case hour < 16:
		return WindowMidday, d
	case hour < 22:
		return WindowEvening, d
	default: // 22:00 to 23:59
		return WindowLateNight, d
	}
}

// Streak is the run of consecutive campus days with a checked plate, ending today, so it is 0
// until today has one. days holds the plates' times or their days, in any order, repeats allowed.
func Streak(days []time.Time, today time.Time) int {
	// DayOf's values share one Location and carry no monotonic reading, so == compares them as
	// days and they work as map keys.
	plated := make(map[time.Time]bool, len(days))
	for _, d := range days {
		plated[DayOf(d)] = true
	}
	n := 0
	for d := DayOf(today); plated[d]; d = d.AddDate(0, 0, -1) {
		n++
	}
	return n
}
