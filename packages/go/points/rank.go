// Author: Caden Lund
package points

// Rank is the title a player's all-time points earn, and how far they are toward the next one.
type Rank struct {
	Name     string  // Fry Cook, Line Cook, Sous Chef, Head Chef or Executive Chef
	Next     string  // the next title, or "" at Executive Chef
	ToNext   int     // the points still needed for Next, or 0 at the top
	Progress float64 // the share of the way from this title to the next, 0 up to 1; 1 at the top
}

// ranks are the titles in order, each with the all-time points it starts at.
var ranks = [...]struct {
	name  string
	floor int
}{
	{name: "Fry Cook", floor: 0},
	{name: "Line Cook", floor: 500},
	{name: "Sous Chef", floor: 1500},
	{name: "Head Chef", floor: 4000},
	{name: "Executive Chef", floor: 10000},
}

// RankOf is the rank a player's all-time points total earns.
func RankOf(total int) Rank {
	i := 0
	for i+1 < len(ranks) && total >= ranks[i+1].floor {
		i++
	}
	if i == len(ranks)-1 {
		return Rank{Name: ranks[i].name, Progress: 1}
	}
	at, next := ranks[i], ranks[i+1]
	return Rank{
		Name:     at.name,
		Next:     next.name,
		ToNext:   next.floor - total,
		Progress: float64(total-at.floor) / float64(next.floor-at.floor),
	}
}
