// Author: Caden Lund
package handlers

import (
	"fmt"
	"time"

	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// What the points rules decide, put on the wire for every service that shows it: the plate's
// receipt and the profile both show a rank and badges, read from the same counts.

// StatsOf is a player's counts as the badges read them. The ledger holds everything but the
// finished weeks they topped the Hall of Fries, which the caller counts from the fry runs.
func StatsOf(s repository.PlayerStats, hallOfFriesWeeks int) points.Stats {
	return points.Stats{
		Scans:                 s.Scans,
		BestScore:             s.BestScore,
		Cooks:                 s.Cooks,
		CooksUnder35:          s.CooksUnder35,
		CooksWithThreeOrFewer: s.CooksWithThreeOrFewer,
		ComboCooks:            s.ComboCooks,
		LongestStreak:         s.LongestStreak,
		LateNightPlates:       s.LateNightPlates,
		MicrowaveOnlyCooks:    s.MicrowaveOnlyCooks,
		BiggestWeeklyRise:     s.BiggestWeeklyRise,
		HallOfFriesWeeks:      hallOfFriesWeeks,
	}
}

// CampusDays reads the ledger's cook days ("2026-10-21") as campus midnights, which is what the
// points rules count days in.
func CampusDays(days []string) ([]time.Time, error) {
	out := make([]time.Time, 0, len(days))
	for _, d := range days {
		t, err := time.ParseInLocation(time.DateOnly, d, points.Campus())
		if err != nil {
			return nil, fmt.Errorf("read cook day %q: %w", d, err)
		}
		out = append(out, t)
	}
	return out, nil
}

// ToProtoRank maps a rank to the wire.
func ToProtoRank(r points.Rank) *cookv1.Rank {
	return &cookv1.Rank{
		Name:     r.Name,
		Next:     r.Next,
		ToNext:   int32(r.ToNext), // at most the 10000 to the top title
		Progress: r.Progress,
	}
}

// ToProtoBadge maps one badge to the wire.
func ToProtoBadge(b points.Badge) *cookv1.Badge {
	return &cookv1.Badge{Name: b.Name, HowToEarn: b.HowToEarn, Earned: b.Earned}
}
