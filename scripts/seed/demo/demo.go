// Author: Caden Lund
// Package demo writes the demo week the seed command lays down: fifteen players, their scans this
// week and last, their plates and the points those earned, their fry runs, and a clear Fridge of
// the Week. Everything goes through the repository, so every constraint a real player meets
// applies to the demo too.
package demo

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/scoring"
)

// Report is what one run wrote. A run on a database that already holds the demo writes nothing.
type Report struct {
	AlreadySeeded bool
	Players       int
	Scans         int
	FryRuns       int
	Plates        int
}

// Seed writes the demo week through repo in one transaction, timed around now: this week starts
// Monday 00:00 campus time, and a moment of it still ahead of now is left out. A database that
// already holds the demo (its first player exists) is left as it is, so a second run changes
// nothing, and a run that fails writes nothing at all.
func Seed(ctx context.Context, repo *repository.Repository, now time.Time) (Report, error) {
	var report Report
	err := repo.InTx(ctx, func(tx *repository.Repository) error {
		// 1. A database that holds the demo already is left alone
		seeded, err := tx.HandleTaken(ctx, players[0].handle)
		if err != nil {
			return err
		}
		if seeded {
			report.AlreadySeeded = true
			return nil
		}

		// 2. Each player, then their week
		weekStart := points.WeekStart(now)
		for _, p := range players {
			account, err := tx.CreateAccount(ctx, p.handle, p.avatar)
			if err != nil {
				return fmt.Errorf("create %s: %w", p.handle, err)
			}
			report.Players++
			// The player's scans in story order. One still ahead of now stays zero; its plates come
			// later still, so they are ahead of now too.
			made := make([]madeScan, len(p.scans))
			for i, s := range p.scans {
				at := s.when.at(weekStart)
				if !at.Before(now) {
					continue
				}
				made[i], err = writeScan(ctx, tx, account.AccountID, s.fridge, at)
				if err != nil {
					return fmt.Errorf("scan for %s: %w", p.handle, err)
				}
				report.Scans++
			}
			for _, r := range p.runs {
				at := r.when.at(weekStart)
				if !at.Before(now) {
					continue
				}
				window, day := points.MealWindow(at)
				_, err = tx.LogFryRun(ctx, account.AccountID, repository.FryRun{
					SpotID:     r.spot.id,
					SpotName:   r.spot.name,
					MealDay:    day,
					MealWindow: string(window),
					At:         at,
				})
				if err != nil {
					return fmt.Errorf("fry run for %s: %w", p.handle, err)
				}
				report.FryRuns++
			}
			for _, pl := range p.plates {
				at := pl.when.at(weekStart)
				if !at.Before(now) {
					continue
				}
				err = writePlate(ctx, tx, account.AccountID, &made[pl.scan], p.scans[pl.scan].fridge, at, report.Plates)
				if err != nil {
					return fmt.Errorf("plate for %s: %w", p.handle, err)
				}
				report.Plates++
			}
		}
		return nil
	})
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

// madeScan is a scan the seed stored: what a plate from it needs to know.
type madeScan struct {
	scanID       int64
	itemsVersion int
	score        int
	recipeID     int64 // its dish's recipe, once a plate has needed it
}

// writeScan stores one of the demo's fridges as a scan made at at. The score comes from
// packages/go/scoring, as a real scan's does, never from the story.
func writeScan(ctx context.Context, tx *repository.Repository, accountID int64, f fridge, at time.Time) (madeScan, error) {
	listed := make([]scoring.Item, 0, len(f.items))
	for _, it := range f.items {
		listed = append(listed, scoring.Item{Name: it.Name, Category: scoring.Category(it.Category)})
	}
	made, err := tx.CreateScan(ctx, accountID, scoring.Fridge(listed), f.roast, f.items)
	if err != nil {
		return madeScan{}, err
	}
	moved, err := tx.BackdateScan(ctx, accountID, made.ScanID, at)
	if err != nil {
		return madeScan{}, err
	}
	if !moved {
		return madeScan{}, fmt.Errorf("backdate scan %d: %w", made.ScanID, repository.ErrNotFound)
	}
	return madeScan{scanID: made.ScanID, itemsVersion: made.ItemsVersion, score: made.Score}, nil
}

// writePlate cooks the dish from one of the player's scans at at, with the nth photo, and pays it
// the way handlers/cook pays a plate, from the rules in packages/go/points: one cook credit, the
// bonus a bad fridge earns, and a streak on the day's first plate.
func writePlate(
	ctx context.Context, tx *repository.Repository, accountID int64, from *madeScan, f fridge, at time.Time, n int,
) error {
	// 1. The scan's one-dish set, made the first time a plate needs it
	if from.scanID == 0 {
		return fmt.Errorf("the plate at %v comes before its scan, which the story must not do", at)
	}
	if from.recipeID == 0 {
		if f.dish == nil {
			return fmt.Errorf("scan %d has no dish to plate", from.scanID)
		}
		_, err := tx.CreateRecipes(ctx, accountID, from.scanID, from.itemsVersion, []repository.NewRecipe{*f.dish})
		if err != nil {
			return err
		}
		recipes, err := tx.ListRecipes(ctx, accountID, from.scanID, from.itemsVersion, repository.RecipeFilter{})
		if err != nil {
			return err
		}
		if len(recipes) != 1 {
			return fmt.Errorf("scan %d holds %d recipes, want its one dish", from.scanID, len(recipes))
		}
		from.recipeID = recipes[0].RecipeID
	}

	// 2. The plate, with a photo of its own
	photo, err := platePhoto(n)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(photo)
	plateID, err := tx.CreatePlate(ctx, accountID, repository.NewPlate{
		RecipeID: from.recipeID, Photo: photo, PhotoSHA256: hash[:], At: at,
	})
	if err != nil {
		return err
	}

	// 3. The awards. The streak counts this plate's day, and only the day's first plate pays it
	cooked, err := tx.CookDays(ctx, accountID)
	if err != nil {
		return err
	}
	days := make([]time.Time, 0, len(cooked)+1)
	for _, d := range cooked {
		day, err := time.ParseInLocation(time.DateOnly, d, points.Campus())
		if err != nil {
			return fmt.Errorf("read cook day %q: %w", d, err)
		}
		days = append(days, day)
	}
	window, day := points.MealWindow(at)
	today := day.Format(time.DateOnly)
	lines := points.PlateAward(points.Plate{
		Score:      from.score,
		FirstOfDay: !slices.Contains(cooked, today),
		Streak:     points.Streak(append(days, day), day),
	})
	awards := make([]repository.Award, 0, len(lines))
	for _, l := range lines {
		awards = append(awards, repository.Award{Reason: string(l.Reason), Points: l.Points})
	}
	return tx.AwardPoints(ctx, accountID, plateID, today+" "+string(window), at, awards)
}
