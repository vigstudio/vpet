package game

import (
	"math"
	"sort"
)

// The arena board. Challenging someone used to mean typing their username and hoping they
// had a pet, which is why the tab read as a form rather than a fight. The board turns the
// office into a visible ladder: who is up there, how strong their pet is, and whether a
// match is worth picking.
const (
	arenaRows        = 20   // fighters listed on the board
	arenaStartPoints = 1000 // every pet begins on equal footing
	arenaPointFloor  = 100  // losses cannot erase a pet from meaningful progression
	arenaPointBase   = 20   // even matches exchange twenty points
)

func legacyArenaPoints(wins, losses int) int {
	return max(arenaPointFloor, arenaStartPoints+wins*arenaPointBase-losses*8)
}

func arenaPoints(p *pet) int {
	if p.ArenaPoints > 0 {
		return p.ArenaPoints
	}
	return legacyArenaPoints(p.Wins, p.Losses)
}

// arenaPointChange is Elo-like but deliberately readable: an even fight is +20/-20;
// beating a higher-rated pet pays more, farming a lower-rated pet pays less.
func arenaPointChange(mine, theirs int, won bool) int {
	expected := 1 / (1 + math.Pow(10, float64(theirs-mine)/400))
	actual := 0.0
	if won {
		actual = 1
	}
	change := int(math.Round(40 * (actual - expected)))
	if won {
		return max(5, change)
	}
	return min(-5, change)
}

// power is a single readable number for "how hard does this pet hit": level, worn gear, the
// worn title and skill ranks folded together. It started as a comparison aid, but the world
// boss and party hunts now settle on it (expedition.go), so anything that makes a pet
// stronger has to appear here or it is invisible in those two modes.
func power(p *pet) int {
	g := p.gearOf()
	t := titleOf(p)
	n := float64(max(p.Lv, 1))*4 + (g.Atk+t.Atk)*3 + (g.Def+t.Def)*3 + g.HP + t.HP + float64(p.Build.Atk)*2.4 + float64(p.Build.Def)*1.8 + float64(p.Build.HP)*4
	for _, id := range p.Skills {
		n += float64(p.skillLv(id)) * 2
	}
	return int(n)
}

// arenaEntry is one fighter on the board.
type arenaEntry struct {
	User   string `json:"user"`
	Name   string `json:"name"` // display name, username fallback
	Pet    string `json:"pet"`  // the pet they field
	Lv     int    `json:"lv"`
	Power  int    `json:"power"`
	Wins   int    `json:"wins"`
	Losses int    `json:"losses"`
	Points int    `json:"points"`
	Rank   int    `json:"rank"`
	Me     bool   `json:"me"`
	Ready  bool   `json:"ready"` // their pet is alive and fit to fight right now
}

// best picks the user's highest-ranked pet that can fight. Choosing by power hid a pet
// with many arena points whenever the owner also had a stronger but lower-rated pet.
func best(st *userState) *pet {
	better := func(a, b *pet) bool {
		return b == nil || arenaPoints(a) > arenaPoints(b) || arenaPoints(a) == arenaPoints(b) && power(a) > power(b)
	}
	var fit, any *pet
	for _, pt := range st.Pets {
		if better(pt, any) {
			any = pt
		}
		if pt.fit() == nil && better(pt, fit) {
			fit = pt
		}
	}
	if fit != nil {
		return fit
	}
	return any
}

// sortArena ranks by arena points. Power only breaks a tie; it never substitutes for results.
func sortArena(rows []arenaEntry) {
	sort.Slice(rows, func(a, b int) bool {
		if rows[a].Points != rows[b].Points {
			return rows[a].Points > rows[b].Points
		}
		if rows[a].Power != rows[b].Power {
			return rows[a].Power > rows[b].Power
		}
		return rows[a].Name < rows[b].Name
	})
}
