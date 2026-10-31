package game

import (
	"errors"
	"fmt"
	"math"
)

// Wild encounters on the world map: walking through a region runs into a wild pet and the
// fight is resolved by the same simulation as a duel, so the replay animation is identical.
const wildHunger = 5

// partOrder keeps generated parts deterministic for a given rng (map iteration is not).
var partOrder = []string{"head", "top", "eyes", "mouth", "body", "pattern", "legs", "tail", "c1", "c2", "c3"}

var wildNames = map[string][]string{
	"vuon":   {"Sâu Rau", "Bướm Cải", "Ốc Sên", "Chuồn Kim", "Gà Con Lạc"},
	"rung":   {"Sói Xám", "Khỉ Đuôi Dài", "Nhím Gai", "Heo Rừng", "Rắn Lục"},
	"hang":   {"Dơi Mù", "Nhện Đá", "Bọ Cạp", "Ma Trơi", "Giun Khổng Lồ"},
	"nui":    {"Gấu Tuyết", "Đại Bàng Băng", "Yeti Con", "Sói Tuyết", "Linh Dương Đá"},
	"nuilua": {"Kỳ Nhông Lửa", "Rồng Con", "Quỷ Dung Nham", "Phượng Hoàng Lửa", "Golem Đá Nung"},
}

// spriteOf maps a region and one of its wild names to the monster art strip, which lists
// five creatures per region in huntMaps order (see webapp/assets/make-world.py). 1-based.
func spriteOf(mapID string, pick int) int {
	for i, m := range huntMaps {
		if m.ID == mapID {
			return i*5 + pick + 1
		}
	}
	return 0
}

// wildSide builds the opponent for one encounter: level follows the region and the player.
func wildSide(m *huntMap, petLv int, rnd rng) duelSide {
	lv := max(m.MinLv, petLv-1) + int(rnd()*3)
	lv = min(max(lv, 1), maxLv)
	parts := map[string]int{}
	for _, k := range partOrder {
		parts[k] = int(rnd()*float64(partLimits[k])) % partLimits[k]
	}
	names := wildNames[m.ID]
	pick := int(rnd()*float64(len(names))) % len(names)
	s := duelSide{
		Name:    names[pick] + " hoang",
		Sprite:  spriteOf(m.ID, pick),
		Parts:   parts,
		Lv:      lv,
		Speed:   speedBase + float64(lv-1)*speedPerLv,
		Atk:     float64(lv) * 0.35,
		Def:     float64(lv) * 0.12,
		HP:      math.Round(70 + float64(lv)*2.5),
		SkillLv: map[string]int{},
	}
	n := min(2+lv/12, maxSkillSlots)
	for len(s.Skills) < n {
		id := skills[int(rnd()*float64(len(skills)))%len(skills)].ID
		dup := false
		for _, have := range s.Skills {
			if have == id {
				dup = true
			}
		}
		if !dup {
			s.Skills = append(s.Skills, id)
			s.SkillLv[id] = min(1+lv/12, maxSkillLv)
		}
	}
	return s
}

type wildResult struct {
	Map   string     `json:"map"`
	Coin  int        `json:"coin"` // gold taken off the wild pet
	Duel  duelResult `json:"duel"`
	Won   bool       `json:"won"`
	XP    int        `json:"xp"`
	Finds []huntFind `json:"finds"`
	Hurt  float64    `json:"hurt"`
	Note  string     `json:"note"`
}

// encounter fights a wild pet in one region. Energy and hunger pace attempts; in dangerous
// regions, a lost fight can kill the pet. Nothing changes when it returns an error.
func encounter(st *userState, i int, mapID string, now int64, rnd rng, boostBPS ...int) (*wildResult, error) {
	p, err := st.petAt(i)
	if err != nil {
		return nil, err
	}
	m := mapByID[mapID]
	if m == nil {
		return nil, errors.New("không có vùng đất này")
	}
	switch {
	case p.Dead:
		return nil, errors.New(p.Name + " đã chết rồi")
	case p.Sick:
		return nil, errors.New(p.Name + " đang bệnh, chữa khỏi rồi hãy đi")
	case p.Health < minHuntHealth:
		return nil, errors.New(p.Name + " yếu quá, cần hồi sức trước")
	case p.Hunger < wildHunger:
		return nil, errors.New(p.Name + " đói quá, cho ăn rồi hãy đi")
	case max(p.Lv, 1) < m.MinLv:
		return nil, fmt.Errorf("%s cần cấp %d mới vào được %s", p.Name, m.MinLv, m.Name)
	}
	if err := p.spendEnergy(energyWild, now); err != nil {
		return nil, err
	}

	p.LastWild = now
	p.Hunger = clamp(p.Hunger - wildHunger)

	foe := wildSide(m, max(p.Lv, 1), rnd)
	attacker := sideOf("", p)
	if len(boostBPS) > 0 {
		attacker = boostedSideWithBPS(attacker, boostBPS[0])
	}
	res := &wildResult{Map: m.ID, Duel: fight(attacker, foe, rnd)}
	res.Duel.At = now
	res.Won = res.Duel.Winner == 0
	base := m.XP/2 + foe.Lv

	if !res.Won {
		res.XP = max(base/4, 1)
		res.Hurt = math.Round(5 + rnd()*8 + m.Risk*25)
		p.hurt(res.Hurt, "thua thú hoang ở "+m.Name)
		p.setEffect("dau", 10, now)
		if p.Dead {
			p.DiedAt = now
			res.Note = "💀 " + p.Name + " đã bỏ mạng trước " + foe.Name + "…"
			return res, nil
		}
		res.Note = fmt.Sprintf("%s thua %s, mất %.0f sức khỏe. %s", p.Name, foe.Name, res.Hurt, gainXP(p, res.XP, rnd))
		return res, nil
	}

	p.Wild++
	st.bump("wild", 1)
	p.Happy = clamp(p.Happy + 4)
	res.XP = base
	r := rollRarity(m, rnd)
	if pool := lootOf(r); len(pool) > 0 {
		id := pool[int(rnd()*float64(len(pool)))%len(pool)]
		st.add(id, 1)
		res.Finds = append(res.Finds, huntFind{Item: id, Rarity: r})
		if r >= 3 {
			p.Gold++
		}
	}
	res.Coin = st.payout(foe.Lv+int(rnd()*8), now)
	st.Coin += res.Coin
	res.Note = fmt.Sprintf("%s hạ gục %s, nhặt %d 🪙! %s", p.Name, foe.Name, res.Coin, gainXP(p, res.XP, rnd))
	return res, nil
}
