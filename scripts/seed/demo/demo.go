// Author: Caden Lund
// Package demo writes the demo week the seed command lays down: fifteen players, their scans this
// week and last, their fry runs and a clear Fridge of the Week. Everything goes through the
// repository, so every constraint a real player meets applies to the demo too.
package demo

import (
	"context"
	"fmt"
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
			for _, s := range p.scans {
				at := s.when.at(weekStart)
				if !at.Before(now) {
					continue
				}
				err = writeScan(ctx, tx, account.AccountID, s.fridge, at)
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
		}
		return nil
	})
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

// writeScan stores one of the demo's fridges as a scan made at at. The score comes from
// packages/go/scoring, as a real scan's does, never from the story.
func writeScan(ctx context.Context, tx *repository.Repository, accountID int64, f fridge, at time.Time) error {
	listed := make([]scoring.Item, 0, len(f.items))
	for _, it := range f.items {
		listed = append(listed, scoring.Item{Name: it.Name, Category: scoring.Category(it.Category)})
	}
	made, err := tx.CreateScan(ctx, accountID, scoring.Fridge(listed), f.roast, f.items)
	if err != nil {
		return err
	}
	moved, err := tx.BackdateScan(ctx, accountID, made.ScanID, at)
	if err != nil {
		return err
	}
	if !moved {
		return fmt.Errorf("backdate scan %d: %w", made.ScanID, repository.ErrNotFound)
	}
	return nil
}
