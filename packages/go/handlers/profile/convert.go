// Author: Caden Lund
package profile

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	profilev1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/profile/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// standing is where a player stands, as the points rules read their history.
type standing struct {
	total   int // all-time points
	stats   points.Stats
	streak  int // days in a row, ending today
	fryRuns int
}

// toProtoProfile maps the caller's standing onto the wire, with every badge in the concept's order.
func toProtoProfile(me repository.Account, p standing) *profilev1.Profile {
	return &profilev1.Profile{
		Handle: me.Handle,
		Avatar: me.Avatar,
		Points: int64(p.total),
		Rank:   handlers.ToProtoRank(points.RankOf(p.total)),
		Stats:  toProtoStats(p),
		Badges: toProtoBadges(p),
	}
}

// toProtoPlayerCard maps a player's public standing and newest plates onto the wire. It takes a
// repository.Player, which holds no kitchen settings, so none can reach the card.
func toProtoPlayerCard(player repository.Player, p standing, plates []repository.PlateThumb) *profilev1.PlayerCard {
	out := &profilev1.PlayerCard{
		Handle:       player.Handle,
		Avatar:       player.Avatar,
		Points:       int64(p.total),
		Rank:         handlers.ToProtoRank(points.RankOf(p.total)),
		Stats:        toProtoStats(p),
		RecentPlates: make([]*profilev1.Plate, 0, len(plates)),
		Badges:       toProtoBadges(p),
	}
	for _, plate := range plates {
		out.RecentPlates = append(out.RecentPlates, toProtoPlate(plate))
	}
	return out
}

// toProtoStats is the stats row under a player's rank.
func toProtoStats(p standing) *profilev1.Stats {
	// Counts of one player's plates, scans and runs, far inside int32.
	return &profilev1.Stats{
		Cooks:      int32(p.stats.Cooks),
		StreakDays: int32(p.streak),
		FryRuns:    int32(p.fryRuns),
		FryRatio:   points.FryRatio(p.fryRuns, p.stats.Cooks),
		BestScore:  int32(p.stats.BestScore), // 0 to 100, held there by app.scan's check
	}
}

// toProtoBadges is every badge, earned or locked, in the concept's order.
func toProtoBadges(p standing) []*cookv1.Badge {
	badges := points.Badges(p.stats)
	out := make([]*cookv1.Badge, 0, len(badges))
	for _, b := range badges {
		out = append(out, handlers.ToProtoBadge(b))
	}
	return out
}

// toProtoPlates maps a page onto the wire. plates may hold one past the page, which only says
// there is more.
func toProtoPlates(in listPlatesInput, plates []repository.PlateThumb) *profilev1.ListPlatesResponse {
	out := &profilev1.ListPlatesResponse{}
	if len(plates) > in.limit {
		last := plates[in.limit-1]
		out.NextCursor = encodeCursor(repository.PlateKey{At: last.At, PlateID: last.PlateID})
		plates = plates[:in.limit]
	}
	out.Plates = make([]*profilev1.Plate, 0, len(plates))
	for _, p := range plates {
		out.Plates = append(out.Plates, toProtoPlate(p))
	}
	return out
}

// toProtoPlate is one plate on a grid, with its photo.
func toProtoPlate(p repository.PlateThumb) *profilev1.Plate {
	return &profilev1.Plate{
		PlateId:     p.PlateID,
		Photo:       p.Photo,
		RecipeTitle: p.RecipeTitle,
		PlatedAt:    timestamppb.New(p.At),
	}
}
