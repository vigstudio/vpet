package game

import (
	"errors"
	"fmt"
	"math"
)

// The tower is the idle ladder: fifty floors, one fight each, progress kept per account so
// it survives burying a pet. Nothing is spent but a daily attempt, which is why only the
// first clear of a floor pays — replaying a cleared floor is practice, not a wage.
const (
	towerFloors    = 50
	towerTriesDay  = 10
	towerLoseHurt  = 5
	towerMilestone = 10 // every tenth floor is a boss and hands over a skill stone
)

var towerBossNames = map[int]string{
	10: "Hộ Pháp Tầng Mười",
	20: "Kiếm Linh Tháp Cổ",
	30: "U Ảnh Tam Thập",
	40: "Chiến Thần Phong Ấn",
	50: "Tháp Chủ Vô Danh",
}

func towerName(floor int) string {
	if towerIsBoss(floor) {
		return towerBossNames[floor]
	}
	return fmt.Sprintf("%s tầng %d", towerNames[(floor-1)%len(towerNames)], floor)
}

var towerNames = []string{"Lính gác tháp", "Vệ binh đá", "Bóng tối lang thang", "Thuật sĩ tháp", "Giáp sĩ hoang"}

func towerIsBoss(floor int) bool { return floor > 0 && floor%towerMilestone == 0 }

// towerFoe builds floor N's opponent from the floor number alone, so everyone meets the
// same fighter there and a preview matches the fight that follows.
func towerFoe(floor int) duelSide {
	rnd := seeded(int64(floor)*7919 + 31)
	parts := map[string]int{}
	for _, k := range partOrder {
		parts[k] = int(rnd()*float64(partLimits[k])) % partLimits[k]
	}
	boss := towerIsBoss(floor)
	mult := 1.0
	name := towerName(floor)
	if boss {
		mult = 1.35
		name = towerBossNames[floor]
	}
	lv := min(maxLv, 2+floor*2)
	region := min((floor-1)/towerMilestone, len(huntMaps)-1)
	s := duelSide{
		Name:    name,
		Sprite:  spriteOf(huntMaps[region].ID, (floor-1)%5),
		Parts:   parts,
		Lv:      lv,
		Speed:   speedBase + float64(lv-1)*speedPerLv,
		Atk:     float64(lv) * 0.3 * mult,
		Def:     float64(lv) * 0.1 * mult,
		HP:      math.Round((70 + float64(lv)*4) * mult),
		SkillLv: map[string]int{},
	}
	for n := min(2+floor/15, maxSkillSlots); len(s.Skills) < n; {
		id := skills[int(rnd()*float64(len(skills)))%len(skills)].ID
		if s.SkillLv[id] > 0 {
			continue
		}
		s.Skills = append(s.Skills, id)
		s.SkillLv[id] = min(1+floor/12, maxSkillLv)
	}
	return s
}

type towerFoeView struct {
	Name  string `json:"name"`
	Lv    int    `json:"lv"`
	Power int    `json:"power"`
	Boss  bool   `json:"boss"`
}

type towerRewardView struct {
	Coin   int `json:"coin"`
	XP     int `json:"xp"`
	Rarity int `json:"rarity"`
	Gem    int `json:"gem"` // skill stones, only on a milestone floor
}

type towerView struct {
	Floor      int              `json:"floor"` // highest floor cleared
	Next       int              `json:"next"`  // floor to attempt, 0 once the tower is done
	Max        int              `json:"max"`
	Left       int              `json:"left"`
	Tries      int              `json:"tries_per_day"`
	Foe        *towerFoeView    `json:"foe,omitempty"`
	Reward     *towerRewardView `json:"reward,omitempty"`
	Milestones []int            `json:"milestones"` // milestone floors already cleared
}

// towerReward is the first-clear payout. Gold goes through payout like every other mode, and
// rewardActivity still gates the daily ceiling, so the tower cannot outrun the economy.
func towerCoinLeft(st *userState, now int64) int {
	if st.CoinDay != dayOf(now) {
		return coinsPerDay
	}
	return max(0, coinsPerDay-st.CoinToday)
}

func towerReward(st *userState, floor int, now int64) towerRewardView {
	v := towerRewardView{
		Coin:   min(st.payout(10+floor*2, now), towerCoinLeft(st, now)),
		XP:     12 + floor*6,
		Rarity: min(floor/12, mythicRarity-1),
	}
	if towerIsBoss(floor) {
		v.Gem = 1
	}
	return v
}

func (st *userState) towerLeft(now int64) int {
	if st.TowerDay != dayOf(now) {
		return towerTriesDay
	}
	return max(0, towerTriesDay-st.TowerToday)
}

func (st *userState) towerView(now int64) towerView {
	v := towerView{Floor: st.TowerFloor, Max: towerFloors, Left: st.towerLeft(now), Tries: towerTriesDay, Milestones: []int{}}
	for floor := towerMilestone; floor <= towerFloors; floor += towerMilestone {
		if floor <= st.TowerFloor {
			v.Milestones = append(v.Milestones, floor)
		}
	}
	if st.TowerFloor >= towerFloors {
		return v
	}
	v.Next = st.TowerFloor + 1
	foe := towerFoe(v.Next)
	v.Foe = &towerFoeView{Name: foe.Name, Lv: foe.Lv, Power: int(math.Round(foe.Atk*3 + foe.Def*3 + foe.HP)), Boss: towerIsBoss(v.Next)}
	reward := towerReward(st, v.Next, now)
	v.Reward = &reward
	return v
}

// tower fights one floor. floor 0 means "the next one"; anything already cleared can be
// replayed for the replay itself, never for loot.
func tower(st *userState, i, floor int, now int64, rnd rng) (*duelResult, string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return nil, "", err
	}
	if floor <= 0 {
		floor = st.TowerFloor + 1
	}
	if floor > towerFloors {
		return nil, "", errors.New("bạn đã chinh phục toàn bộ tháp")
	}
	if floor > st.TowerFloor+1 {
		return nil, "", errors.New("tầng tháp không hợp lệ")
	}
	if st.towerLeft(now) <= 0 {
		return nil, "", errors.New("hết lượt leo tháp hôm nay")
	}
	if err := p.fit(); err != nil {
		return nil, "", err
	}
	if p.FrozenUntil > now {
		return nil, "", errors.New("thú đang nghỉ phép")
	}
	if st.TowerDay != dayOf(now) {
		st.TowerDay, st.TowerToday = dayOf(now), 0
	}
	st.TowerToday++

	foe := towerFoe(floor)
	duel := fight(boostedSideOf("", p, st, now), foe, rnd)
	duel.At = now
	if duel.Winner != 0 {
		// A floor never kills: the tower is meant to be attempted on a whim.
		p.Health = math.Max(1, p.Health-towerLoseHurt)
		return &duel, fmt.Sprintf("Thua %s ở tầng %d, mất %d sức khỏe.", foe.Name, floor, towerLoseHurt), nil
	}
	if floor <= st.TowerFloor {
		return &duel, fmt.Sprintf("Hạ %s ở tầng %d, tầng đã qua nên không có thưởng.", foe.Name, floor), nil
	}
	st.TowerFloor = floor
	reward := towerReward(st, floor, now)
	st.Coin += reward.Coin
	if st.CoinDay != dayOf(now) {
		st.CoinDay, st.CoinToday = dayOf(now), 0
	}
	st.CoinToday += reward.Coin
	pool := lootOf(reward.Rarity)
	id := pool[int(rnd()*float64(len(pool)))%len(pool)]
	st.add(id, 1)
	loot := itemByID[id]
	xp := gainXP(p, reward.XP, rnd)
	// The level-up note is usually empty, and a trailing "%s" that matched nothing left the
	// whole sentence untranslated, so the two shapes are kept as separate sentences.
	note := fmt.Sprintf("Hạ %s, chinh phục tầng %d! +%d vàng và %s %s.", foe.Name, floor, reward.Coin, loot.Emoji, loot.Name)
	if reward.Gem > 0 {
		st.add("ngoc", reward.Gem)
		st.add("sach", 1)
		learned := ""
		if len(p.Skills) < skillSlots(p.Lv) {
			if skill := learnSkill(p, rnd); skill != "" {
				learned = fmt.Sprintf("Mở ô kỹ năng mới: %s %s", skillByID[skill].Emoji, skillByID[skill].Name)
			}
		}
		note = fmt.Sprintf("Hạ %s, chinh phục tầng %d! +%d vàng, %s %s và %d 💠 ngọc kỹ năng. +1 giấy đổi kỹ năng.",
			foe.Name, floor, reward.Coin, loot.Emoji, loot.Name, reward.Gem)
		if learned != "" {
			note += " " + learned
		}
	}
	if xp != "" {
		note += " " + xp
	}
	return &duel, note, nil
}

// swapSkill spends a manual to trade one learned move for a chosen one. The old book rerolled
// both starting skills at random, which threw away training the player had paid gems for.
func swapSkill(st *userState, i int, from, to string) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if st.Inv["sach"] <= 0 {
		return "", errors.New("bạn không có món này")
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	sk := skillByID[to]
	if sk == nil {
		return "", errors.New("không có kỹ năng này")
	}
	at := -1
	for n, id := range p.Skills {
		if id == to {
			return "", errors.New("thú đã có kỹ năng này")
		}
		if id == from {
			at = n
		}
	}
	if at < 0 {
		return "", errors.New("thú không có kỹ năng này")
	}
	// The rank belongs to the slot, not to the skill id that used to sit in it.
	lv := p.skillLv(from)
	p.Skills[at] = to
	if p.SkillLv == nil {
		p.SkillLv = map[string]int{}
	}
	delete(p.SkillLv, from)
	if lv > 1 {
		p.SkillLv[to] = lv
	}
	st.add("sach", -1)
	return fmt.Sprintf("%s đổi sang %s %s, giữ nguyên cấp %d.", p.Name, sk.Emoji, sk.Name, lv), nil
}
