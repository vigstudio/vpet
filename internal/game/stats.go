package game

import "math"

// One place that answers "how strong is this pet". The panel used to show only worn-gear
// totals, so a player could not see what levels, build points and gems added up to. Every
// number here comes from the same call the duel makes, so the sheet cannot drift from combat.

// Speed decides who gets extra turns. It starts at a flat base and grows with level, build
// points in attack and worn gloves and boots, which are the pieces a player associates with
// moving quickly.
const (
	speedBase    = 10
	speedPerLv   = 0.35
	speedPerAtk  = 0.25 // one point of the attack build
	speedGearAtk = 0.6  // per point of attack on gloves and boots
	extraTurnCap = 0.45 // the fastest a pet can ever act twice
)

// speedOf is the pet's action speed. Gloves and boots pay double because they are the slots
// a player reads as "quick", which keeps the stat legible without a new gear type.
func (p *pet) speedOf() float64 {
	s := speedBase + float64(max(p.Lv, 1)-1)*speedPerLv + float64(p.Build.Atk)*speedPerAtk
	for _, slot := range []string{"gang", "giay"} {
		if id := p.Gear[slot]; id != "" {
			s += gearByID[id].Atk * speedGearAtk
		}
	}
	return math.Round(s*10) / 10
}

// extraTurnChance is how often the faster side acts twice in a row. A tie gives nobody a
// bonus, and the cap stops a speed build from removing the opponent's turns entirely.
func extraTurnChance(mine, theirs float64) float64 {
	if mine <= theirs {
		return 0
	}
	return math.Min(extraTurnCap, (mine-theirs)/40)
}

// statSheet is the full picture the panel draws.
type statSheet struct {
	Atk      float64 `json:"atk"`       // damage added to every hit
	Def      float64 `json:"def"`       // damage subtracted from every hit taken
	HP       float64 `json:"hp"`        // battle HP this pet enters a fight with
	Speed    float64 `json:"speed"`     // action speed
	Power    int     `json:"power"`     // the single comparison number used on the arena board
	BaseAtk  float64 `json:"base_atk"`  // the plain bite before gear
	GearAtk  float64 `json:"gear_atk"`  // from worn gear, set bonus and gems
	BuildAtk float64 `json:"build_atk"` // from allocated points
	TitleAtk float64 `json:"title_atk"` // from the worn title
	GearDef  float64 `json:"gear_def"`
	BuildDef float64 `json:"build_def"`
	LvDef    float64 `json:"lv_def"`
	TitleDef float64 `json:"title_def"`
	GearHP   float64 `json:"gear_hp"`
	BuildHP  float64 `json:"build_hp"`
	LvHP     float64 `json:"lv_hp"`
	TitleHP  float64 `json:"title_hp"`
	HealthHP float64 `json:"health_hp"` // battle HP that comes from current health
}

// statsOf mirrors sideOf exactly, then splits each total into where it came from so the
// panel can explain a number instead of only printing it.
func (p *pet) statsOf() statSheet {
	side := sideOf("", p)
	g := p.gearOf()
	ti := titleOf(p)
	lv := float64(max(p.Lv, 1))
	return statSheet{
		Atk: side.Atk, Def: side.Def, HP: side.HP,
		Speed: p.speedOf(), Power: power(p),
		BaseAtk:  8.5, // the average plain bite, shown so a new pet is not all zeroes
		GearAtk:  g.Atk,
		BuildAtk: float64(p.Build.Atk) * .8,
		TitleAtk: ti.Atk,
		GearDef:  g.Def,
		BuildDef: float64(p.Build.Def) * .6,
		LvDef:    (lv - 1) * .15,
		TitleDef: ti.Def,
		GearHP:   g.HP,
		BuildHP:  float64(p.Build.HP) * 4,
		LvHP:     (lv - 1) * 2,
		TitleHP:  ti.HP,
		HealthHP: math.Round(40 + p.Health*0.6),
	}
}
