// Author: Caden Lund
package scan

import (
	scanv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/scan/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/scoring"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// Shared mappings live in handlers because Combo also renders and scores scans.
func toProtoVerdict(s repository.Scan) (*scanv1.Verdict, error) { return handlers.ToProtoVerdict(s) }
func categoryWord(c scanv1.Category) string                     { return handlers.CategoryWord(c) }
func scoringItems(items []repository.ScanItem) ([]scoring.Item, error) {
	return handlers.ScoringItems(items)
}
func roastLevel(level string) (vision.RoastLevel, error) { return handlers.RoastLevel(level) }
