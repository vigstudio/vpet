package game

import (
	"errors"
	"fmt"
	"time"
)

// Energy replaces the flat 45-second wait on adventures. A pet banks a pool that refills on
// its own, so a player who has been away can run several stages back to back instead of
// sitting through one timer per fight. The pool grows with level, which makes levelling feel
// like it buys playtime rather than only bigger numbers.
const (
	energyTick    = 6 * time.Minute // real time for one point to come back
	energyBase    = 20              // pool at level 1
	energyPerLv   = 2               // every level widens the pool
	energyStage   = 3               // one campaign stage
	energyBoss    = 5               // the chapter boss at stage 6
	energyWild    = 2               // free roam encounter
	maxEnergyCard = 30              // ceiling the permanent card may add
)

// maxEnergy is the pool size for this pet, including any permanent card bonuses.
func (p *pet) maxEnergy() int {
	return energyBase + (max(p.Lv, 1)-1)*energyPerLv + min(p.EnergyMax, maxEnergyCard)
}

// energyAt refills the pool from the clock without writing: every full tick since the last
// settle is one point. Pets created before 1.16 start full rather than empty.
func (p *pet) energyAt(now int64) int {
	cap := p.maxEnergy()
	if p.EnergyAt == 0 {
		return cap
	}
	gained := int((now - p.EnergyAt) / energyTick.Milliseconds())
	return min(cap, p.Energy+max(gained, 0))
}

// settleEnergy folds the regenerated points into stored state. The timestamp keeps the
// remainder of the current tick, so switching tabs never rounds progress away.
func (p *pet) settleEnergy(now int64) {
	cap := p.maxEnergy()
	if p.EnergyAt == 0 {
		p.Energy, p.EnergyAt = cap, now
		return
	}
	ticks := (now - p.EnergyAt) / energyTick.Milliseconds()
	if ticks > 0 {
		p.Energy = min(cap, p.Energy+int(ticks))
		p.EnergyAt += ticks * energyTick.Milliseconds()
	}
	if p.Energy >= cap {
		// a full pool has no partial tick to carry, so the next spend starts a clean timer
		p.Energy, p.EnergyAt = cap, now
	}
}

// spendEnergy takes points for one run. It returns an error naming the wait, so the UI can
// explain the cost instead of greying a button out silently.
func (p *pet) spendEnergy(cost int, now int64) error {
	have := p.energyAt(now)
	if have < cost {
		at := p.EnergyAt
		if at == 0 {
			at = now
		}
		// only the remainder of the current tick is already served
		served := (now - at) % energyTick.Milliseconds()
		wait := time.Duration(int64(cost-have)*energyTick.Milliseconds()-served) * time.Millisecond
		return fmt.Errorf("cần %d năng lượng, còn %d. Đủ sau khoảng %d phút", cost, have, int(wait.Minutes())+1)
	}
	p.settleEnergy(now)
	p.Energy -= cost
	return nil
}

// energyView is what the panel draws: the bar, the cost table and the next-point countdown.
type energyView struct {
	Now   int   `json:"now"`
	Max   int   `json:"max"`
	Tick  int   `json:"tick_seconds"`
	Next  int64 `json:"next"` // server time the next point lands, 0 when the pool is full
	Stage int   `json:"stage_cost"`
	Boss  int   `json:"boss_cost"`
	Wild  int   `json:"wild_cost"`
}

func (p *pet) energyView(now int64) energyView {
	v := energyView{Now: p.energyAt(now), Max: p.maxEnergy(), Tick: int(energyTick.Seconds()),
		Stage: energyStage, Boss: energyBoss, Wild: energyWild}
	if v.Now < v.Max {
		at := p.EnergyAt
		if at == 0 {
			at = now
		}
		// the stored timestamp only advances on settle, so walk it forward to the pending tick
		ticks := (now - at) / energyTick.Milliseconds()
		v.Next = at + (ticks+1)*energyTick.Milliseconds()
	}
	return v
}

// drinkEnergy restores points from a potion. Nothing is spent when the pool is already full.
func drinkEnergy(st *userState, i int, id string, amount int, now int64) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	p.settleEnergy(now)
	if p.Energy >= p.maxEnergy() {
		return "", errors.New(p.Name + " đang đầy năng lượng")
	}
	st.add(id, -1)
	before := p.Energy
	p.Energy = min(p.maxEnergy(), p.Energy+amount)
	return fmt.Sprintf("⚡ %s hồi %d năng lượng (%d/%d).", p.Name, p.Energy-before, p.Energy, p.maxEnergy()), nil
}

// growEnergy spends the permanent card. The ceiling stops one lucky player from banking an
// endless pool and skipping the pacing the system exists to create.
func growEnergy(st *userState, i int, id string, amount int, now int64) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	if p.EnergyMax >= maxEnergyCard {
		return "", fmt.Errorf("%s đã đạt giới hạn +%d năng lượng tối đa", p.Name, maxEnergyCard)
	}
	p.settleEnergy(now)
	st.add(id, -1)
	p.EnergyMax = min(maxEnergyCard, p.EnergyMax+amount)
	p.Energy = min(p.maxEnergy(), p.Energy+amount)
	return fmt.Sprintf("🔋 %s tăng %d năng lượng tối đa (%d/%d).", p.Name, amount, p.Energy, p.maxEnergy()), nil
}
