// Author: Caden Lund
package points

// Stats is what the badges read from a player's history; the repository counts it. Every field
// must only ever grow (a count, a best, a longest), so a badge once earned stays earned.
type Stats struct {
	Scans                 int // fridges scanned
	BestScore             int // the highest score any scan earned
	Cooks                 int // checked plates
	CooksUnder35          int // checked plates of recipes from a scan that scored under 35
	CooksWithThreeOrFewer int // checked plates of recipes that use three ingredients or fewer
	ComboCooks            int // checked plates cooked in a combo
	LongestStreak         int // the most campus days in a row with a checked plate
	LateNightPlates       int // checked plates between midnight and 4 am, campus time
	MicrowaveOnlyCooks    int // checked plates of recipes that need nothing but a microwave
	BiggestWeeklyRise     int // the biggest score rise from one scan to a later one, within 7 days
	HallOfFriesWeeks      int // finished weeks that the player ended on top of the Hall of Fries
}

// Badge is a sticker for one feat. Badges are derived from Stats on every read and never stored.
type Badge struct {
	Name      string
	HowToEarn string // the line a locked badge shows
	Earned    bool
}

// badges are the concept's eleven, in its order.
var badges = [...]struct {
	name      string
	howToEarn string
	earned    func(Stats) bool
}{
	{
		name:      "First Roast",
		howToEarn: "Scan your first fridge",
		earned:    func(s Stats) bool { return s.Scans >= 1 },
	},
	{
		name:      "Certified Chef",
		howToEarn: "Score 85 or higher on a scan",
		earned:    func(s Stats) bool { return s.BestScore >= 85 },
	},
	{
		name:      "On the Board",
		howToEarn: "Earn your first points",
		// Only cooking earns points, and every checked plate earns the cook credit.
		earned: func(s Stats) bool { return s.Cooks >= 1 },
	},
	{
		name:      "Underdog",
		howToEarn: "Cook from a fridge that scored under 35",
		earned:    func(s Stats) bool { return s.CooksUnder35 >= 1 },
	},
	{
		name:      "Three's a Meal",
		howToEarn: "Cook a recipe with three ingredients or fewer",
		earned:    func(s Stats) bool { return s.CooksWithThreeOrFewer >= 1 },
	},
	{
		name:      "Combo Meal",
		howToEarn: "Plate a meal in Combo mode",
		earned:    func(s Stats) bool { return s.ComboCooks >= 1 },
	},
	{
		name:      "Clean Plate Club",
		howToEarn: "Cook seven days in a row",
		earned:    func(s Stats) bool { return s.LongestStreak >= 7 },
	},
	{
		name:      "Late Night Legend",
		howToEarn: "Plate a meal between midnight and 4 am",
		earned:    func(s Stats) bool { return s.LateNightPlates >= 1 },
	},
	{
		name:      "Glow Up",
		howToEarn: "Raise your fridge score by 40 or more within a week",
		earned:    func(s Stats) bool { return s.BiggestWeeklyRise >= 40 },
	},
	{
		name:      "Microwave Maestro",
		howToEarn: "Cook five microwave-only recipes",
		earned:    func(s Stats) bool { return s.MicrowaveOnlyCooks >= 5 },
	},
	{
		name:      "Bag Secured",
		howToEarn: "Top the Hall of Fries for a week",
		earned:    func(s Stats) bool { return s.HallOfFriesWeeks >= 1 },
	},
}

// Badges is every badge, earned or locked, in the concept's order.
func Badges(s Stats) []Badge {
	out := make([]Badge, len(badges))
	for i, b := range badges {
		out[i] = Badge{Name: b.name, HowToEarn: b.howToEarn, Earned: b.earned(s)}
	}
	return out
}
