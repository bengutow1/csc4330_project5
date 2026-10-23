// Author: Caden Lund
package cook

import (
	"fmt"

	cookv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/cook/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/handlers"
	"github.com/bengutow1/csc4330_project5/packages/go/points"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

// difficulties is the closed set in one table: the word app.recipe stores and the wire's enum. A
// word missing here fails the rpc with a logged internal error rather than reaching the app as
// unspecified.
var difficulties = map[string]cookv1.Difficulty{
	repository.DifficultyEasy:   cookv1.Difficulty_DIFFICULTY_EASY,
	repository.DifficultyMedium: cookv1.Difficulty_DIFFICULTY_MEDIUM,
	repository.DifficultyHard:   cookv1.Difficulty_DIFFICULTY_HARD,
}

// toProtoShelves splits a set into its two shelves, keeping its order on each: what the player
// can make now, and what lacks one thing.
func toProtoShelves(set []repository.Recipe) (*cookv1.ListRecipesResponse, error) {
	resp := &cookv1.ListRecipesResponse{}
	for _, r := range set {
		recipe, err := toProtoRecipe(r)
		if err != nil {
			return nil, err
		}
		if len(r.Missing) == 0 {
			resp.MakeItNow = append(resp.MakeItNow, recipe)
		} else {
			resp.OneItemAway = append(resp.OneItemAway, recipe)
		}
	}
	return resp, nil
}

// toProtoRecipe maps a stored recipe to the wire, with what plating it would pay.
func toProtoRecipe(r repository.Recipe) (*cookv1.Recipe, error) {
	difficulty, ok := difficulties[r.Difficulty]
	if !ok {
		return nil, fmt.Errorf("cook: no wire difficulty for %q", r.Difficulty)
	}
	equipment, err := handlers.WireEquipment(r.Equipment)
	if err != nil {
		return nil, err
	}
	steps := make([]*cookv1.CookingStep, len(r.Steps))
	for i, instruction := range r.Steps {
		steps[i] = &cookv1.CookingStep{Instruction: instruction}
		if i < len(r.StepTimers) && r.StepTimers[i] > 0 {
			seconds := int32(r.StepTimers[i]) // bounded to four hours by app.recipe
			steps[i].TimerSeconds = &seconds
		}
	}
	return &cookv1.Recipe{
		RecipeId:      r.RecipeID,
		Title:         r.Title,
		Minutes:       int32(r.Minutes), // 1 to 240, held there by app.recipe's check
		Difficulty:    difficulty,
		Equipment:     equipment,
		Uses:          r.Uses,
		Missing:       r.Missing,
		Steps:         r.Steps,
		CookingSteps:  steps,
		PointsPreview: pointsPreview(r.ScanScore),
	}, nil
}

// pointsPreview is what plating a recipe from a scan with this score pays before any streak: the
// cook, and the bonus a bad fridge earns. The points rules decide it, so the card and the receipt
// cannot disagree.
func pointsPreview(score int) int32 {
	total := 0
	for _, line := range points.PlateAward(points.Plate{Score: score}) {
		total += line.Points
	}
	return int32(total) // at most 200 before a streak or a combo
}

// visionItems is a stored list as the model reads it.
func visionItems(items []repository.ScanItem) []vision.Item {
	out := make([]vision.Item, 0, len(items))
	for _, item := range items {
		out = append(out, vision.Item{Name: item.Name, Category: item.Category, Quantity: item.Quantity})
	}
	return out
}

// pointReasons is the closed set in one table: the points rules' reason and the wire's enum. A
// reason missing here fails the plate with a logged internal error rather than reaching the app
// as unspecified.
var pointReasons = map[points.Reason]cookv1.PointReason{
	points.ReasonCook:           cookv1.PointReason_POINT_REASON_COOK,
	points.ReasonScrappyBonus:   cookv1.PointReason_POINT_REASON_SCRAPPY_BONUS,
	points.ReasonAgainstAllOdds: cookv1.PointReason_POINT_REASON_AGAINST_ALL_ODDS,
	points.ReasonStreak:         cookv1.PointReason_POINT_REASON_STREAK,
	points.ReasonComboCook:      cookv1.PointReason_POINT_REASON_COMBO_COOK,
}

// paidPlate is what one plate paid, and where it left the player.
type paidPlate struct {
	lines                   []points.Line
	streak                  int
	before, after           repository.PlayerStats
	totalBefore, totalAfter int
}

// toAwards is a plate's lines as the ledger stores them; the points rules spell each reason as
// app.point_event's check does.
func toAwards(lines []points.Line) []repository.Award {
	out := make([]repository.Award, 0, len(lines))
	for _, line := range lines {
		out = append(out, repository.Award{Reason: string(line.Reason), Points: line.Points})
	}
	return out
}

// toProtoReceipt is the receipt for a plate that passed: its lines in receipt order, their
// total, the streak, the badges this plate earned, and the rank before and after.
func toProtoReceipt(message string, p paidPlate) (*cookv1.Receipt, error) {
	r := &cookv1.Receipt{
		Passed:     true,
		Message:    message,
		StreakDays: int32(p.streak), // a count of days, far under int32
		RankBefore: handlers.ToProtoRank(points.RankOf(p.totalBefore)),
		RankAfter:  handlers.ToProtoRank(points.RankOf(p.totalAfter)),
		// A plate never changes the weeks a player topped the Hall of Fries, so neither side counts
		// them: no badge that reads them can be new on a receipt.
		NewBadges: newBadges(handlers.StatsOf(p.before, 0), handlers.StatsOf(p.after, 0)),
	}
	for _, line := range p.lines {
		reason, ok := pointReasons[line.Reason]
		if !ok {
			return nil, fmt.Errorf("cook: no wire reason for %q", line.Reason)
		}
		earned := int32(line.Points) // at most 100 a line
		r.Lines = append(r.Lines, &cookv1.AwardLine{Reason: reason, Points: earned})
		r.Total += earned
	}
	return r, nil
}

// newBadges are the badges after earns that before did not, in the concept's order.
func newBadges(before, after points.Stats) []*cookv1.Badge {
	had := points.Badges(before)
	var out []*cookv1.Badge
	for i, b := range points.Badges(after) {
		if b.Earned && !had[i].Earned {
			out = append(out, handlers.ToProtoBadge(b))
		}
	}
	return out
}
