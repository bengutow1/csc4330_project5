// Author: Caden Lund
package points_test

import (
	"testing"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/points"
)

// campus is a wall-clock time on campus. Inside the hour that daylight saving repeats the wall
// clock names two instants, so those cases are written in UTC instead.
func campus(y int, mo time.Month, d, h, mi int) time.Time {
	return time.Date(y, mo, d, h, mi, 0, 0, points.Campus())
}

// day is a campus day the way DayOf names it: its midnight on campus.
func day(y int, mo time.Month, d int) time.Time {
	return campus(y, mo, d, 0, 0)
}

// utc is an instant written in UTC, the way the database hands timestamps back.
func utc(y int, mo time.Month, d, h, mi int) time.Time {
	return time.Date(y, mo, d, h, mi, 0, 0, time.UTC)
}

// TestCampusIsChicagoWithItsDaylightSaving: a fixed offset would move every boundary by an hour
// for half the year.
func TestCampusIsChicagoWithItsDaylightSaving(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		at     time.Time
		zone   string
		offset int
	}{
		{name: "october is daylight time", at: utc(2026, 10, 21, 17, 0), zone: "CDT", offset: -5 * 60 * 60},
		{name: "december is standard time", at: utc(2026, 12, 21, 18, 0), zone: "CST", offset: -6 * 60 * 60},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			zone, offset := c.at.In(points.Campus()).Zone()

			if zone != c.zone || offset != c.offset {
				t.Errorf("zone = %s %d, want %s %d", zone, offset, c.zone, c.offset)
			}
		})
	}
}

// TestDayOfIsTheCampusCalendarDay: a day turns over at campus midnight whatever zone a time
// arrives in, so a plate at 23:30 in Chicago is not counted on the next UTC day.
func TestDayOfIsTheCampusCalendarDay(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		at   time.Time
		want time.Time
	}{
		{name: "23:59 is still today", at: campus(2026, 10, 20, 23, 59), want: day(2026, 10, 20)},
		{name: "00:00 is the next day", at: campus(2026, 10, 21, 0, 0), want: day(2026, 10, 21)},
		{name: "noon", at: campus(2026, 10, 21, 12, 0), want: day(2026, 10, 21)},
		{name: "utc past midnight is still the campus evening", at: utc(2026, 10, 21, 4, 30), want: day(2026, 10, 20)},
		{name: "campus midnight written in utc", at: utc(2026, 10, 21, 5, 0), want: day(2026, 10, 21)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := points.DayOf(c.at)

			if !got.Equal(c.want) {
				t.Errorf("DayOf(%v) = %v, want %v", c.at, got, c.want)
			}
		})
	}
}

// TestDaysTurnOverAtCampusMidnightAcrossDaylightSaving pins the instants themselves, in UTC:
// November 1, 2026 runs 25 hours and March 14, 2027 runs 23, so no day is the last plus 24 hours.
func TestDaysTurnOverAtCampusMidnightAcrossDaylightSaving(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		at   time.Time
		want time.Time
	}{
		{name: "november 1 starts on daylight time", at: utc(2026, 11, 1, 5, 0), want: utc(2026, 11, 1, 5, 0)},
		{name: "01:30 the first time", at: utc(2026, 11, 1, 6, 30), want: utc(2026, 11, 1, 5, 0)},
		{name: "01:30 again an hour later", at: utc(2026, 11, 1, 7, 30), want: utc(2026, 11, 1, 5, 0)},
		{name: "23:59 that night, the 25th hour", at: utc(2026, 11, 2, 5, 59), want: utc(2026, 11, 1, 5, 0)},
		{name: "november 2 starts on standard time", at: utc(2026, 11, 2, 6, 0), want: utc(2026, 11, 2, 6, 0)},
		{name: "march 14 starts on standard time", at: utc(2027, 3, 14, 7, 59), want: utc(2027, 3, 14, 6, 0)},
		{name: "03:00 straight after 01:59", at: utc(2027, 3, 14, 8, 0), want: utc(2027, 3, 14, 6, 0)},
		{name: "23:59 that night, the 23rd hour", at: utc(2027, 3, 15, 4, 59), want: utc(2027, 3, 14, 6, 0)},
		{name: "march 15 starts on daylight time", at: utc(2027, 3, 15, 5, 0), want: utc(2027, 3, 15, 5, 0)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := points.DayOf(c.at)

			if !got.Equal(c.want) {
				t.Errorf("DayOf(%v) = %v, want %v", c.at, got.UTC(), c.want)
			}
		})
	}
}

// TestTheWeekStartsMondayAtCampusMidnight is the board's Monday reset: Sunday 23:59 still counts
// toward the week that is ending, Monday 00:00 starts the next, and the week holding the 25-hour
// day is not seven times 24 hours long.
func TestTheWeekStartsMondayAtCampusMidnight(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		at   time.Time
		want time.Time
	}{
		{name: "monday 00:00 starts its week", at: campus(2026, 10, 19, 0, 0), want: day(2026, 10, 19)},
		{name: "midweek", at: campus(2026, 10, 21, 12, 0), want: day(2026, 10, 19)},
		{name: "sunday 23:59 is the end of the week", at: campus(2026, 10, 25, 23, 59), want: day(2026, 10, 19)},
		{name: "monday 00:00 starts the next week", at: campus(2026, 10, 26, 0, 0), want: day(2026, 10, 26)},
		{name: "utc says monday before campus does", at: utc(2026, 10, 26, 4, 30), want: day(2026, 10, 19)},
		{name: "sunday 23:59 after clocks fall back", at: utc(2026, 11, 2, 5, 59), want: utc(2026, 10, 26, 5, 0)},
		{name: "monday 00:00 after clocks fall back", at: utc(2026, 11, 2, 6, 0), want: utc(2026, 11, 2, 6, 0)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := points.WeekStart(c.at)

			if !got.Equal(c.want) {
				t.Errorf("WeekStart(%v) = %v, want %v", c.at, got, c.want)
			}
		})
	}
}

// TestMealWindowsFollowTheCampusClock pins every window edge in the contracts and the day each
// window belongs to. Late night runs past midnight and stays with the day it started, so 23:59
// and 00:00 are one window and one cook credit.
func TestMealWindowsFollowTheCampusClock(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		at         time.Time
		wantWindow points.Window
		wantDay    time.Time
	}{
		{
			name: "04:59 is still last night", at: campus(2026, 10, 21, 4, 59),
			wantWindow: points.WindowLateNight, wantDay: day(2026, 10, 20),
		},
		{
			name:       "the last nanosecond before 05:00",
			at:         time.Date(2026, 10, 21, 4, 59, 59, 999_999_999, points.Campus()),
			wantWindow: points.WindowLateNight, wantDay: day(2026, 10, 20),
		},
		{
			name: "05:00 starts the morning", at: campus(2026, 10, 21, 5, 0),
			wantWindow: points.WindowMorning, wantDay: day(2026, 10, 21),
		},
		{
			name: "10:59 is still morning", at: campus(2026, 10, 21, 10, 59),
			wantWindow: points.WindowMorning, wantDay: day(2026, 10, 21),
		},
		{
			name: "11:00 starts midday", at: campus(2026, 10, 21, 11, 0),
			wantWindow: points.WindowMidday, wantDay: day(2026, 10, 21),
		},
		{
			name: "15:59 is still midday", at: campus(2026, 10, 21, 15, 59),
			wantWindow: points.WindowMidday, wantDay: day(2026, 10, 21),
		},
		{
			name: "16:00 starts the evening", at: campus(2026, 10, 21, 16, 0),
			wantWindow: points.WindowEvening, wantDay: day(2026, 10, 21),
		},
		{
			name: "21:59 is still evening", at: campus(2026, 10, 21, 21, 59),
			wantWindow: points.WindowEvening, wantDay: day(2026, 10, 21),
		},
		{
			name: "22:00 starts late night", at: campus(2026, 10, 21, 22, 0),
			wantWindow: points.WindowLateNight, wantDay: day(2026, 10, 21),
		},
		{
			name: "23:59 is late night", at: campus(2026, 10, 21, 23, 59),
			wantWindow: points.WindowLateNight, wantDay: day(2026, 10, 21),
		},
		{
			name: "00:00 is the same late night", at: campus(2026, 10, 22, 0, 0),
			wantWindow: points.WindowLateNight, wantDay: day(2026, 10, 21),
		},
		{
			name: "a utc time falls in its campus window", at: utc(2026, 10, 21, 15, 30),
			wantWindow: points.WindowMorning, wantDay: day(2026, 10, 21),
		},
		{
			name: "01:30 the first time clocks fall back", at: utc(2026, 11, 1, 6, 30),
			wantWindow: points.WindowLateNight, wantDay: day(2026, 10, 31),
		},
		{
			name: "01:30 again an hour later", at: utc(2026, 11, 1, 7, 30),
			wantWindow: points.WindowLateNight, wantDay: day(2026, 10, 31),
		},
		{
			name: "04:59 on standard time", at: utc(2026, 11, 1, 10, 59),
			wantWindow: points.WindowLateNight, wantDay: day(2026, 10, 31),
		},
		{
			name: "05:00 on standard time", at: utc(2026, 11, 1, 11, 0),
			wantWindow: points.WindowMorning, wantDay: day(2026, 11, 1),
		},
		{
			name: "04:59 the day clocks spring forward", at: utc(2027, 3, 14, 9, 59),
			wantWindow: points.WindowLateNight, wantDay: day(2027, 3, 13),
		},
		{
			name: "05:00 the day clocks spring forward", at: utc(2027, 3, 14, 10, 0),
			wantWindow: points.WindowMorning, wantDay: day(2027, 3, 14),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			gotWindow, gotDay := points.MealWindow(c.at)

			if gotWindow != c.wantWindow {
				t.Errorf("window = %q, want %q", gotWindow, c.wantWindow)
			}
			if !gotDay.Equal(c.wantDay) {
				t.Errorf("day = %v, want %v", gotDay, c.wantDay)
			}
		})
	}
}

// TestStreakCountsDaysInARowEndingToday: the streak bonus and the profile both read it, so
// repeats, an unsorted history, a gap and a missing today must each count as the receipt says.
func TestStreakCountsDaysInARowEndingToday(t *testing.T) {
	t.Parallel()
	today := campus(2026, 10, 21, 18, 0)
	cases := []struct {
		name  string
		days  []time.Time
		today time.Time
		want  int
	}{
		{name: "no plates", days: nil, today: today, want: 0},
		{name: "a plate today", days: []time.Time{campus(2026, 10, 21, 12, 0)}, today: today, want: 1},
		{
			name:  "today and yesterday",
			days:  []time.Time{campus(2026, 10, 21, 8, 0), campus(2026, 10, 20, 19, 0)},
			today: today, want: 2,
		},
		{
			name:  "nothing yet today",
			days:  []time.Time{campus(2026, 10, 20, 12, 0), campus(2026, 10, 19, 12, 0)},
			today: today, want: 0,
		},
		{
			name: "a missed day ends it",
			days: []time.Time{
				campus(2026, 10, 21, 12, 0), campus(2026, 10, 20, 12, 0), campus(2026, 10, 18, 12, 0),
			},
			today: today, want: 2,
		},
		{
			name: "repeats in any order",
			days: []time.Time{
				campus(2026, 10, 20, 9, 0), campus(2026, 10, 21, 20, 0), campus(2026, 10, 19, 12, 0),
				campus(2026, 10, 21, 8, 0), campus(2026, 10, 20, 22, 0),
			},
			today: today, want: 3,
		},
		{
			name: "days given as DayOf days", days: []time.Time{day(2026, 10, 21), day(2026, 10, 20)},
			today: today, want: 2,
		},
		{
			name:  "a plate after today is not part of it",
			days:  []time.Time{campus(2026, 10, 22, 12, 0), campus(2026, 10, 21, 12, 0)},
			today: today, want: 1,
		},
		{
			name:  "23:59 and 00:00 are two days",
			days:  []time.Time{campus(2026, 10, 20, 23, 59), campus(2026, 10, 21, 0, 0)},
			today: today, want: 2,
		},
		{
			// 03:00 UTC on the 21st is 22:00 on the 20th on campus.
			name:  "utc timestamps count on their campus day",
			days:  []time.Time{utc(2026, 10, 21, 3, 0), utc(2026, 10, 21, 15, 0)},
			today: today, want: 2,
		},
		{
			name: "across the 25-hour day",
			days: []time.Time{
				campus(2026, 10, 31, 12, 0), campus(2026, 11, 1, 12, 0), campus(2026, 11, 2, 12, 0),
			},
			today: campus(2026, 11, 2, 20, 0), want: 3,
		},
		{
			name: "across the 23-hour day",
			days: []time.Time{
				campus(2027, 3, 13, 12, 0), campus(2027, 3, 14, 12, 0), campus(2027, 3, 15, 12, 0),
			},
			today: campus(2027, 3, 15, 20, 0), want: 3,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := points.Streak(c.days, c.today)

			if got != c.want {
				t.Errorf("Streak = %d, want %d", got, c.want)
			}
		})
	}
}

// TestAStreakItselfHasNoCap: only the bonus stops at seven days; the receipt still shows the
// whole run.
func TestAStreakItselfHasNoCap(t *testing.T) {
	t.Parallel()
	var days []time.Time
	for i := range 30 {
		days = append(days, campus(2026, 10, 21-i, 12, 0))
	}

	got := points.Streak(days, campus(2026, 10, 21, 18, 0))

	if got != 30 {
		t.Errorf("Streak = %d, want 30", got)
	}
}
