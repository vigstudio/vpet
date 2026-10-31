package game

import (
	"errors"
	"fmt"
	"math"
)

const (
	chapterStages = 6
	replayDropCap = 0.15
)

var difficultyNames = []string{"Dễ", "Vừa", "Khó"}
var stageNames = map[string][]string{
	"vuon":   {"Bờ ruộng", "Vườn hoa", "Bụi tre", "Giếng cổ", "Cánh đồng", "Vua vườn"},
	"rung":   {"Bìa rừng", "Suối xanh", "Rừng gai", "Tổ sói", "Cây đại thụ", "Chúa rừng"},
	"hang":   {"Cửa hang", "Động dơi", "Mạch đá", "Ổ nhện", "Giếng sâu", "Vua hang tối"},
	"nui":    {"Chân núi", "Đèo gió", "Hồ băng", "Vách tuyết", "Đỉnh trắng", "Bạo chúa băng"},
	"nuilua": {"Tro tàn", "Sông lửa", "Hỏa ngục", "Thành dung nham", "Ổ rồng", "Long vương"},
}

type campaignRewardPreview struct {
	Coin       int     `json:"coin"`
	XP         int     `json:"xp"`
	Rarity     int     `json:"rarity"`
	Shards     int     `json:"shards"`
	Replay     bool    `json:"replay"`
	DropChance float64 `json:"drop_chance,omitempty"`
}

type stageView struct {
	ID      string                   `json:"id"`
	Map     string                   `json:"map"`
	Name    string                   `json:"name"`
	Stage   int                      `json:"stage"`
	MinLv   int                      `json:"min_lv"`
	Stars   [3]int                   `json:"stars"`
	Open    bool                     `json:"open"`
	Boss    bool                     `json:"boss"`
	Rewards [3]campaignRewardPreview `json:"rewards"`
}
type campaignResult struct {
	wildResult
	Stage       int  `json:"stage"`
	Difficulty  int  `json:"difficulty"`
	Stars       int  `json:"stars"`
	First       bool `json:"first"`
	ReplayBonus bool `json:"replay_bonus,omitempty"`
}

type replayReward struct {
	item   string
	amount int
	weight int
}

func replayRewardPool(difficulty int, boss bool) []replayReward {
	tier := difficulty
	if boss {
		tier++
	}
	switch min(tier, 3) {
	case 0:
		return []replayReward{{"tui", 1, 40}, {"gem_shard", 1, 35}, {"tui_y", 1, 20}, {"tui_vukhi", 1, 5}}
	case 1:
		return []replayReward{{"gem_shard", 2, 35}, {"tui_y", 1, 30}, {"tui_vukhi", 1, 25}, {"nuoc_tang_luc", 1, 10}}
	case 2:
		return []replayReward{{"gem_shard", 3, 35}, {"tui_vukhi", 1, 30}, {"tra_sam", 1, 20}, {"tui_vang", 1, 15}}
	default:
		return []replayReward{{"gem_shard", 5, 30}, {"tui_vang", 1, 30}, {"ruou_than", 1, 20}, {"tui_vukhi", 2, 20}}
	}
}

func replayDropChance(difficulty int, boss bool) float64 {
	chance := []float64{0.08, 0.10, 0.12}[min(max(difficulty, 0), 2)]
	if boss {
		chance += 0.03
	}
	return math.Min(chance, replayDropCap)
}

func grantReplayReward(st *userState, difficulty int, boss bool, rnd rng) (huntFind, bool) {
	if rnd() >= replayDropChance(difficulty, boss) {
		return huntFind{}, false
	}
	pool := replayRewardPool(difficulty, boss)
	total := 0
	for _, reward := range pool {
		total += reward.weight
	}
	roll := int(rnd() * float64(total))
	picked := pool[len(pool)-1]
	for _, reward := range pool {
		roll -= reward.weight
		if roll < 0 {
			picked = reward
			break
		}
	}
	st.add(picked.item, picked.amount)
	rarity := 0
	if item := itemByID[picked.item]; item != nil {
		rarity = item.Rarity
	}
	return huntFind{Item: picked.item, Rarity: rarity, Amount: picked.amount}, true
}

func stageKey(mapID string, stage int) string { return fmt.Sprintf("%s_%d", mapID, stage) }

func campaignReward(st *userState, m *huntMap, stage, difficulty int, cleared, boss bool, now int64) campaignRewardPreview {
	scale := []float64{1, 2, 3.6}[difficulty]
	if cleared {
		return campaignRewardPreview{Coin: st.payout(max(1, int(float64(3+stage)*scale)), now), XP: max(1, int(float64(m.XP/3+stage)*scale)), Replay: true, DropChance: replayDropChance(difficulty, boss)}
	}
	rarity := min(difficulty+booli(boss), mythicRarity-1)
	return campaignRewardPreview{Coin: st.payout(int(float64(12+stage*4)*scale), now), XP: int(float64(m.XP+stage*4) * scale), Rarity: rarity, Shards: booli(difficulty == len(difficultyNames)-1) * 2}
}

func booli(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (p *pet) campaignViews(st *userState, now int64) []stageView {
	out := []stageView{}
	for chapter, m := range huntMaps {
		for i := 1; i <= chapterStages; i++ {
			key := stageKey(m.ID, i)
			open := max(p.Lv, 1) >= m.MinLv
			if i > 1 {
				open = open && p.Campaign[stageKey(m.ID, i-1)][0] > 0
			} else if chapter > 0 {
				open = open && p.Campaign[stageKey(huntMaps[chapter-1].ID, chapterStages)][0] > 0
			}
			view := stageView{ID: key, Map: m.ID, Stage: i, Name: stageNames[m.ID][i-1], MinLv: m.MinLv, Stars: p.Campaign[key], Open: open, Boss: i == chapterStages}
			for difficulty := range difficultyNames {
				view.Rewards[difficulty] = campaignReward(st, &huntMaps[chapter], i, difficulty, view.Stars[difficulty] > 0, view.Boss, now)
			}
			out = append(out, view)
		}
	}
	return out
}

func campaign(st *userState, i int, mapID string, stage, difficulty int, now int64, rnd rng, boostBPS ...int) (*campaignResult, error) {
	p, err := st.petAt(i)
	if err != nil {
		return nil, err
	}
	m := mapByID[mapID]
	if m == nil || stage < 1 || stage > chapterStages || difficulty < 0 || difficulty >= len(difficultyNames) {
		return nil, errors.New("ải hoặc độ khó không hợp lệ")
	}
	key := stageKey(mapID, stage)
	var entry stageView
	for _, v := range p.campaignViews(st, now) {
		if v.ID == key {
			entry = v
			break
		}
	}
	if !entry.Open {
		return nil, errors.New("chưa mở ải: hoàn thành ải trước và đạt cấp yêu cầu")
	}
	if difficulty > 0 && p.Campaign[key][difficulty-1] == 0 {
		return nil, errors.New("thắng độ khó trước rồi mới thử độ khó này")
	}
	if err := p.fit(); err != nil {
		return nil, err
	}
	if p.FrozenUntil > now {
		return nil, errors.New("thú đang nghỉ phép")
	}
	if p.Hunger < wildHunger {
		return nil, errors.New("cần 5 độ no để vào ải")
	}
	cost := energyStage
	if stage == chapterStages {
		cost = energyBoss
	}
	if err := p.spendEnergy(cost, now); err != nil {
		return nil, err
	}
	// Stage foes used to scale from the chapter's minimum level, a constant. A level 20 pet
	// replaying the garden therefore met level 1 opponents and won with no gear at all, which
	// is why investment felt pointless. The foe now tracks the pet, so a stage stays a test of
	// how well the pet is built rather than how far past the chapter it has levelled.
	petLv := max(p.Lv, 1)
	foe := wildSide(m, petLv, seeded(int64(stage*97+difficulty*173)))
	foe.Lv = min(maxLv, max(m.MinLv, petLv)+(stage-1)*2+difficulty*3)
	// Easy stays beatable while under-geared; Hard assumes gear, points and gems.
	mult := []float64{0.45, 0.85, 1.25}[difficulty] * (1 + float64(stage-1)*.1)
	if stage == chapterStages {
		mult *= 1.3
		foe.Name = stageNames[mapID][stage-1]
	}
	foe.HP = math.Round((70 + float64(foe.Lv)*4.5) * mult)
	foe.Atk = (float64(foe.Lv)*0.26 + float64(difficulty*2)) * []float64{0.65, 1.0, 1.25}[difficulty]
	foe.Def = float64(foe.Lv)*0.08 + float64(difficulty)*1.0
	foe.Speed = speedBase + float64(foe.Lv-1)*speedPerLv + float64(difficulty)*2
	foe.Sprite = spriteOf(mapID, (stage-1)%5)
	p.Hunger = clamp(p.Hunger - wildHunger)
	attacker := sideOf("", p)
	if len(boostBPS) > 0 {
		attacker = boostedSideWithBPS(attacker, boostBPS[0])
	}
	duel := fight(attacker, foe, rnd)
	duel.At = now
	result := &campaignResult{wildResult: wildResult{Map: mapID, Duel: duel, Won: duel.Winner == 0, Finds: []huntFind{}}, Stage: stage, Difficulty: difficulty}
	if !result.Won {
		result.Hurt = math.Round(7 + float64(stage+difficulty)*1.5)
		p.hurt(result.Hurt, "thua ải "+entry.Name)
		if p.Dead {
			p.DiedAt = now
		}
		result.Note = fmt.Sprintf("Thua %s. Hồi sức, đổi build rồi thử lại; chưa nhận thưởng.", entry.Name)
		return result, nil
	}
	st.bump("stage", 1)
	result.Stars = 1
	last := duel.Turns[len(duel.Turns)-1]
	if last.HP[0] >= duel.A.HP*.35 {
		result.Stars = 2
	}
	if last.HP[0] >= duel.A.HP*.7 && len(duel.Turns) <= 18 {
		result.Stars = 3
	}
	if p.Campaign == nil {
		p.Campaign = map[string][3]int{}
	}
	progress := p.Campaign[key]
	result.First = progress[difficulty] == 0
	progress[difficulty] = max(progress[difficulty], result.Stars)
	p.Campaign[key] = progress
	p.Wild++
	if result.First {
		// Rewards have to argue for the harder setting. Easy used to pay almost the same as
		// Hard, so nobody had a reason to take the risk; each step up now multiplies gold and
		// experience and raises the quality of the drop.
		scale := []float64{1, 2, 3.6}[difficulty]
		result.XP = int(float64(m.XP+stage*4) * scale)
		result.Coin = st.payout(int(float64(12+stage*4)*scale), now)
		st.Coin += result.Coin
		// Drop quality follows the difficulty, and the chapter boss pays one rung better
		// still, so the last stage on Hard is the best thing in the chapter.
		rarity := difficulty
		if stage == chapterStages {
			rarity++
		}
		rarity = min(rarity, mythicRarity-1) // mythic and divine stay forge-only
		pool := lootOf(rarity)
		id := pool[int(rnd()*float64(len(pool)))%len(pool)]
		st.add(id, 1)
		result.Finds = append(result.Finds, huntFind{Item: id, Rarity: rarity})
		// Hard also hands over forge material, which is the only way to climb the ladder.
		if difficulty == len(difficultyNames)-1 {
			st.add("gem_shard", 2)
			result.Finds = append(result.Finds, huntFind{Item: "gem_shard", Rarity: 0})
		}
		result.Note = fmt.Sprintf("%s: %d sao! Thưởng lần đầu +%d vàng. %s", entry.Name, result.Stars, result.Coin, gainXP(p, result.XP, rnd))
	} else {
		// A replay used to pay nothing at all, which made the whole campaign dead content once
		// cleared. It now pays a small wage that still scales with difficulty, worth far less
		// than the first clear so farming Easy never beats progressing.
		scale := []float64{1, 2, 3.6}[difficulty]
		result.XP = max(1, int(float64(m.XP/3+stage)*scale))
		result.Coin = st.payout(max(1, int(float64(3+stage)*scale)), now)
		st.Coin += result.Coin
		if bonus, ok := grantReplayReward(st, difficulty, stage == chapterStages, rnd); ok {
			result.Finds = append(result.Finds, bonus)
			result.ReplayBonus = true
		}
		result.Note = fmt.Sprintf("%s: %d sao! Đánh lại +%d vàng. %s", entry.Name, result.Stars, result.Coin, gainXP(p, result.XP, rnd))
	}
	return result, nil
}

// Every level grants base combat growth automatically and three build points to allocate.
// Existing pets derive entitlement from their level, so upgrades need no one-off migration.
type buildDef struct {
	Atk int `json:"atk"`
	Def int `json:"def"`
	HP  int `json:"hp"`
}

// buildPoints is the lifetime entitlement for a level. Three per level up to the old cap
// keeps every existing pet exactly as it was; past it each level grants two. One per level
// made the long tail feel like no reward at all against an XP curve that grows with the
// square of the level, so the late game pays double while staying well under 3/level.
func buildPoints(lv int) int {
	lv = max(lv, 1)
	return (min(lv, softCapLv)-1)*3 + max(0, lv-softCapLv)*2
}

func (p *pet) buildLeft() int {
	return max(0, buildPoints(p.Lv)+p.BuildBonus-p.Build.Atk-p.Build.Def-p.Build.HP)
}
func allocate(st *userState, i int, stat string, amount int) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if p.Dead {
		return "", errors.New("hồi sinh thú trước khi cộng điểm")
	}
	if amount < 1 || amount > p.buildLeft() {
		return "", errors.New("không đủ điểm tiềm năng")
	}
	switch stat {
	case "atk":
		p.Build.Atk += amount
	case "def":
		p.Build.Def += amount
	case "hp":
		p.Build.HP += amount
	default:
		return "", errors.New("chọn Công, Thủ hoặc Sinh lực")
	}
	return fmt.Sprintf("Đã cộng %d điểm; còn %d điểm tiềm năng.", amount, p.buildLeft()), nil
}
