package game

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// Eight equipment slots and the forge. Gear comes in seven tiers per slot; three of the same
// piece can be fused into the next tier up, which is the only way to create mythic pieces.
// Wearing several pieces of the same tier pays a set bonus, so a matched outfit beats a
// mixed outfit may trade the set bonus for stronger individual pieces.
const (
	fuseBase        = 40 // the smith charges this for the bottom rung and doubles each step up
	setMin          = 3  // pieces of one tier before the set bonus wakes up
	maxEnhanceLevel = 10
)

// Lower recipes stay unchanged. Supreme costs ~4x the extrapolated 1,280 fee:
// a long-term gold sink, with bound souls separately gating playtime.
func fuseCountOf(id string) int {
	if tierOf(id) == divineRarity {
		return 4
	}
	return 3
}
func fuseSoulsOf(id string) int {
	if tierOf(id) == divineRarity {
		return 3
	}
	return 0
}
func fuseCostOf(id string) int {
	if tierOf(id) == divineRarity {
		return 5000
	}
	return fuseBase * (1 << tierOf(id))
}

func enhanceCost(id string, next int) (coin, shards int) {
	tier := tierOf(id) + 1
	return 8 * tier * next, (tier*next + 1) / 2
}

func enhancedStat(base float64, level int) float64 {
	lv := float64(min(max(level, 0), maxEnhanceLevel))
	if base <= 0 {
		return 0
	}
	step := math.Max(0.2, math.Round(base*0.06*10)/10)
	return math.Round(step*lv*10) / 10
}

func enhancedGear(id string, level int) gearDef {
	g := gearByID[id]
	return gearDef{Slot: g.Slot, Atk: g.Atk + enhancedStat(g.Atk, level), Def: g.Def + enhancedStat(g.Def, level), HP: g.HP + enhancedStat(g.HP, level)}
}

var gearSlots = []struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
}{
	{"vukhi", "Vũ khí", "🗡️"},
	{"giap", "Giáp", "🛡️"},
	{"mu", "Mũ", "⛑️"},
	{"gang", "Găng", "🧤"},
	{"giay", "Giày", "🥾"},
	{"nhan", "Nhẫn", "💍"},
	{"khien", "Khiên", "🔰"},
	{"aokhoac", "Áo khoác", "🧥"},
}

// gearByID: Slot decides where it goes, the stats add straight into a duel.
var gearByID = map[string]gearDef{
	// weapons
	"dao": {Slot: "vukhi", Atk: 2}, "kiem": {Slot: "vukhi", Atk: 5},
	"dinhba": {Slot: "vukhi", Atk: 9}, "gaythan": {Slot: "vukhi", Atk: 14},
	"kiemrong": {Slot: "vukhi", Atk: 21},
	// body armour
	"aovai": {Slot: "giap", Def: 1}, "giapda": {Slot: "giap", Def: 3},
	"giapthep": {Slot: "giap", Def: 5}, "longbao": {Slot: "giap", Def: 8},
	"giaprong": {Slot: "giap", Def: 12},
	// helmets
	"nonre": {Slot: "mu", Def: 1, HP: 3}, "muda": {Slot: "mu", Def: 2, HP: 6},
	"muthep": {Slot: "mu", Def: 3, HP: 12}, "vuongmien": {Slot: "mu", Def: 4, HP: 22},
	"muthan": {Slot: "mu", Def: 6, HP: 34},
	// gloves
	"gangvai": {Slot: "gang", Atk: 1}, "gangda": {Slot: "gang", Atk: 2, Def: 1},
	"gangthep": {Slot: "gang", Atk: 4, Def: 2}, "gangrong": {Slot: "gang", Atk: 7, Def: 3},
	"gangthan": {Slot: "gang", Atk: 11, Def: 5},
	// boots
	"depco": {Slot: "giay", HP: 3}, "giayda": {Slot: "giay", Def: 1, HP: 6},
	"giaythep": {Slot: "giay", Def: 2, HP: 12}, "hiabay": {Slot: "giay", Def: 3, HP: 20},
	"hiathan": {Slot: "giay", Def: 5, HP: 32},
	// rings and trinkets
	"nhandong": {Slot: "nhan", HP: 6}, "chuoingoc": {Slot: "nhan", HP: 14},
	"nhanbac": {Slot: "nhan", Atk: 2, HP: 18}, "nhanrong": {Slot: "nhan", Atk: 4, HP: 30},
	"nhanthan": {Slot: "nhan", Atk: 7, HP: 46},
	// shields: the defensive slot, trades attack for staying power
	"motnap": {Slot: "khien", Def: 2}, "khiengo": {Slot: "khien", Def: 3, HP: 5},
	"khiensat": {Slot: "khien", Def: 5, HP: 10}, "khienbac": {Slot: "khien", Def: 7, HP: 18},
	"khienthan": {Slot: "khien", Def: 10, HP: 30},
	// cloaks: light but broad
	"aotoi": {Slot: "aokhoac", Def: 1, HP: 2}, "aogam": {Slot: "aokhoac", Atk: 1, Def: 1, HP: 6},
	"aolong": {Slot: "aokhoac", Atk: 2, Def: 2, HP: 12}, "aosao": {Slot: "aokhoac", Atk: 3, Def: 3, HP: 20},
	"aothan": {Slot: "aokhoac", Atk: 5, Def: 5, HP: 32},
	// divine: forged from three mythic pieces, roughly 1.4x the mythic line
	"kiemthanlong": {Slot: "vukhi", Atk: 30}, "giapthanlong": {Slot: "giap", Def: 17},
	"muthiende": {Slot: "mu", Def: 9, HP: 48}, "gangthienloi": {Slot: "gang", Atk: 16, Def: 7},
	"hiathienvuong": {Slot: "giay", Def: 7, HP: 46}, "nhanthienmenh": {Slot: "nhan", Atk: 10, HP: 64},
	"khienthienha": {Slot: "khien", Def: 14, HP: 42}, "aothienmenh": {Slot: "aokhoac", Atk: 7, Def: 7, HP: 46},
	// Supreme: exactly 1.4x Divine, preserving slot profiles.
	"kiemhongmong":  {Slot: "vukhi", Atk: 42},
	"giaphongmong":  {Slot: "giap", Def: 23.8},
	"muhongmong":    {Slot: "mu", Def: 12.6, HP: 67.2},
	"ganghongmong":  {Slot: "gang", Atk: 22.4, Def: 9.8},
	"hiahongmong":   {Slot: "giay", Def: 9.8, HP: 64.4},
	"nhanhongmong":  {Slot: "nhan", Atk: 14, HP: 89.6},
	"khienhongmong": {Slot: "khien", Def: 19.6, HP: 58.8},
	"aohongmong":    {Slot: "aokhoac", Atk: 9.8, Def: 9.8, HP: 64.4},
}

// forgeTiers lists each ladder from scrap to Supreme; fusing walks up one step.
var forgeTiers = map[string][]string{
	"vukhi":   {"dao", "kiem", "dinhba", "gaythan", "kiemrong", "kiemthanlong", "kiemhongmong"},
	"giap":    {"aovai", "giapda", "giapthep", "longbao", "giaprong", "giapthanlong", "giaphongmong"},
	"mu":      {"nonre", "muda", "muthep", "vuongmien", "muthan", "muthiende", "muhongmong"},
	"gang":    {"gangvai", "gangda", "gangthep", "gangrong", "gangthan", "gangthienloi", "ganghongmong"},
	"giay":    {"depco", "giayda", "giaythep", "hiabay", "hiathan", "hiathienvuong", "hiahongmong"},
	"nhan":    {"nhandong", "chuoingoc", "nhanbac", "nhanrong", "nhanthan", "nhanthienmenh", "nhanhongmong"},
	"khien":   {"motnap", "khiengo", "khiensat", "khienbac", "khienthan", "khienthienha", "khienhongmong"},
	"aokhoac": {"aotoi", "aogam", "aolong", "aosao", "aothan", "aothienmenh", "aohongmong"},
}

// tierOf is the rung this piece sits on, 0 for scrap. Unknown ids answer 0.
func tierOf(id string) int {
	for _, tiers := range forgeTiers {
		for i, t := range tiers {
			if t == id {
				return i
			}
		}
	}
	return 0
}

// nextTier returns what this item fuses into, or "" when it is already the best.
func nextTier(id string) string {
	tiers := forgeTiers[gearByID[id].Slot]
	for i, t := range tiers {
		if t == id && i+1 < len(tiers) {
			return tiers[i+1]
		}
	}
	return ""
}

// setBonusDef rewards a matched outfit. Pieces of the same tier count together: three of a
// tier wakes the bonus and every further piece adds to it, so a late decision is "keep the
// set" against "take the single better drop".
type setBonusDef struct {
	Tier  int     `json:"tier"`
	Name  string  `json:"name"`
	Parts int     `json:"parts"`
	Atk   float64 `json:"atk,omitempty"`
	Def   float64 `json:"def,omitempty"`
	HP    float64 `json:"hp,omitempty"`
}

// setNames label the ladders; the bonus scales with how high the matched tier is.
var setNames = []string{"Bộ Tân binh", "Bộ Lữ hành", "Bộ Chiến binh", "Bộ Hoàng kim", "Bộ Thần thoại", "Bộ Thần cấp", "Bộ Tối Thượng"}

// setOf reports the best matched-tier bonus a pet is wearing, or nil when nothing matches.
func setOf(gear map[string]string) *setBonusDef {
	count := map[int]int{}
	for slot, id := range gear {
		if gearByID[id].Slot != slot {
			continue // stale data: an item that no longer belongs in this slot
		}
		count[tierOf(id)]++
	}
	var best *setBonusDef
	for tier, n := range count {
		if n < setMin {
			continue
		}
		step := float64(tier+1) * float64(n-setMin+1)
		b := setBonusDef{Tier: tier, Name: setNames[min(tier, len(setNames)-1)], Parts: n,
			Atk: step * 0.8, Def: step * 0.6, HP: step * 5}
		if best == nil || b.Tier > best.Tier || (b.Tier == best.Tier && b.Parts > best.Parts) {
			c := b
			best = &c
		}
	}
	return best
}

type forgeRecipe struct {
	From         string `json:"from"`
	To           string `json:"to"`
	Need         int    `json:"need"`
	Have         int    `json:"have"`
	Cost         int    `json:"cost"`
	Ready        bool   `json:"ready"`
	Material     string `json:"material,omitempty"`
	MaterialNeed int    `json:"material_need,omitempty"`
	MaterialHave int    `json:"material_have,omitempty"`
}

// forgeList is what the smith offers right now: every ladder step the user owns a copy of.
func (st *userState) forgeList() []forgeRecipe {
	out := []forgeRecipe{}
	for id := range gearByID {
		to := nextTier(id)
		if to == "" || st.Inv[id] <= 0 {
			continue
		}
		have, cost, need, souls := st.Inv[id], fuseCostOf(id), fuseCountOf(id), fuseSoulsOf(id)
		recipe := forgeRecipe{From: id, To: to, Need: need, Have: have, Cost: cost,
			Ready: have >= need && st.Coin >= cost && st.Inv["tinhhon"] >= souls}
		if souls > 0 {
			recipe.Material, recipe.MaterialNeed, recipe.MaterialHave = "tinhhon", souls, st.Inv["tinhhon"]
		}
		out = append(out, recipe)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].From < out[b].From })
	return out
}

// fuseAll consumes every affordable recipe in ladder order. Outputs from one rung may
// immediately feed the next rung; worn gear is never part of the inventory pool.
func fuseAll(st *userState) (string, error) {
	before, total := st.Coin, 0
	for tier := 0; tier < len(forgeTiers[gearSlots[0].ID])-1; tier++ {
		for _, slot := range gearSlots {
			id := forgeTiers[slot.ID][tier]
			for st.Inv[id] >= fuseCountOf(id) && st.Coin >= fuseCostOf(id) && st.Inv["tinhhon"] >= fuseSoulsOf(id) {
				if _, err := fuse(st, id); err != nil {
					return "", err
				}
				total++
			}
		}
	}
	if total == 0 {
		return "", errors.New("chưa đủ trang bị, vàng hoặc Tinh Hồn để ghép")
	}
	return fmt.Sprintf("🔨 Ghép nhanh %d lần, dùng %d vàng.", total, before-st.Coin), nil
}

func fuse(st *userState, id string) (string, error) {
	to := nextTier(id)
	if to == "" {
		if _, ok := gearByID[id]; !ok {
			return "", errors.New("món này không rèn được")
		}
		return "", errors.New("món này đã ở bậc cao nhất")
	}
	need, souls := fuseCountOf(id), fuseSoulsOf(id)
	if st.Inv[id] < need {
		return "", fmt.Errorf("cần %d× %s, bạn có %d", need, itemByID[id].Name, st.Inv[id])
	}
	if st.Inv["tinhhon"] < souls {
		return "", fmt.Errorf("cần %d× %s, bạn có %d", souls, itemByID["tinhhon"].Name, st.Inv["tinhhon"])
	}
	cost := fuseCostOf(id)
	if st.Coin < cost {
		return "", fmt.Errorf("cần %d vàng tiền công, bạn còn %d", cost, st.Coin)
	}
	st.Coin -= cost
	st.add(id, -need)
	if souls > 0 {
		st.add("tinhhon", -souls)
	}
	st.add(to, 1)
	st.bump("forge", 1)
	return fmt.Sprintf("🔨 Rèn %d× %s thành %s %s!", need, itemByID[id].Name, itemByID[to].Emoji, itemByID[to].Name), nil
}

func (st *userState) enhancedCount(id string, level int) int {
	return st.Enhanced[id][level]
}

func (st *userState) addGear(id string, level, count int) {
	if level <= 0 {
		st.add(id, count)
		return
	}
	if st.Enhanced == nil {
		st.Enhanced = map[string]map[int]int{}
	}
	if st.Enhanced[id] == nil {
		st.Enhanced[id] = map[int]int{}
	}
	st.Enhanced[id][level] += count
	if st.Enhanced[id][level] <= 0 {
		delete(st.Enhanced[id], level)
	}
	if len(st.Enhanced[id]) == 0 {
		delete(st.Enhanced, id)
	}
}

func returnSocketGems(st *userState, p *pet, slot string) {
	for _, gem := range p.Sockets[slot] {
		if _, _, ok := parseGem(gem); ok {
			st.add(gem, 1)
		}
	}
	delete(p.Sockets, slot)
}

func takeBagGear(st *userState, id string, level int) bool {
	if level <= 0 {
		if st.Inv[id] <= 0 {
			return false
		}
		st.add(id, -1)
		return true
	}
	if st.enhancedCount(id, level) <= 0 {
		return false
	}
	st.addGear(id, level, -1)
	return true
}

func returnWornGear(st *userState, p *pet, slot string) {
	if id := p.Gear[slot]; id != "" {
		st.addGear(id, p.GearLevel[slot], 1)
	}
	delete(p.Gear, slot)
	delete(p.GearLevel, slot)
}

func equip(st *userState, i int, id string, level ...int) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	g, ok := gearByID[id]
	if !ok {
		return "", errors.New("món này không đeo được")
	}
	lv := 0
	if len(level) > 0 {
		lv = level[0]
	}
	if lv < 0 || lv > maxEnhanceLevel {
		return "", errors.New("cấp cường hóa không hợp lệ")
	}
	if (lv == 0 && st.Inv[id] <= 0) || (lv > 0 && st.enhancedCount(id, lv) <= 0) {
		return "", errors.New("bạn không có món này")
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	if p.Gear == nil {
		p.Gear = map[string]string{}
	}
	if p.GearLevel == nil {
		p.GearLevel = map[string]int{}
	}
	if !takeBagGear(st, id, lv) {
		return "", errors.New("bạn không có món này")
	}
	returnSocketGems(st, p, g.Slot)
	returnWornGear(st, p, g.Slot)
	p.Gear[g.Slot] = id
	if lv > 0 {
		p.GearLevel[g.Slot] = lv
	}
	msg := p.Name + " trang bị " + itemByID[id].Emoji + " " + itemByID[id].Name
	if lv > 0 {
		msg += fmt.Sprintf(" +%d", lv)
	}
	if b := setOf(p.Gear); b != nil {
		msg += fmt.Sprintf(" · ✨ %s (%d món)", b.Name, b.Parts)
	}
	return msg, nil
}

// equipBest compares every reachable tier-count state, preserving set bonuses without a
// combinatorial scan of every outfit. Items already worn and enhanced bag items participate.
func equipBest(st *userState, i int) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	original, originalLevels := map[string]string{}, map[string]int{}
	for slot, id := range p.Gear {
		original[slot], originalLevels[slot] = id, p.GearLevel[slot]
	}
	type choice struct {
		id    string
		level int
	}
	type outfit struct {
		score  float64
		gear   map[string]string
		levels map[string]int
	}
	states := map[[supremeRarity + 1]uint8]outfit{{}: {gear: map[string]string{}, levels: map[string]int{}}}
	for _, slot := range gearSlots {
		candidates := []choice{}
		for _, id := range forgeTiers[slot.ID] {
			if st.Inv[id] > 0 {
				candidates = append(candidates, choice{id: id})
			}
			for level, count := range st.Enhanced[id] {
				if count > 0 {
					candidates = append(candidates, choice{id: id, level: level})
				}
			}
			if original[slot.ID] == id {
				candidates = append(candidates, choice{id: id, level: originalLevels[slot.ID]})
			}
		}
		if len(candidates) == 0 {
			candidates = []choice{{}}
		}
		next := map[[supremeRarity + 1]uint8]outfit{}
		for counts, current := range states {
			for _, selected := range candidates {
				c, score := counts, current.score
				if selected.id != "" {
					c[tierOf(selected.id)]++
					g := enhancedGear(selected.id, selected.level)
					score += g.Atk*3 + g.Def*3 + g.HP
				}
				if seen, ok := next[c]; ok && seen.score >= score {
					continue
				}
				gear, levels := map[string]string{}, map[string]int{}
				for wornSlot, worn := range current.gear {
					gear[wornSlot], levels[wornSlot] = worn, current.levels[wornSlot]
				}
				if selected.id != "" {
					gear[slot.ID], levels[slot.ID] = selected.id, selected.level
				}
				next[c] = outfit{score: score, gear: gear, levels: levels}
			}
		}
		states = next
	}
	oldPower, bestPower := power(p), power(p)
	best := outfit{gear: original, levels: originalLevels}
	for _, candidate := range states {
		p.Gear, p.GearLevel = candidate.gear, candidate.levels
		if n := power(p); n > bestPower {
			bestPower, best = n, candidate
		}
	}
	p.Gear, p.GearLevel = original, originalLevels
	if bestPower <= oldPower {
		return fmt.Sprintf("%s đang mặc bộ mạnh nhất · lực chiến %d.", p.Name, bestPower), nil
	}
	for _, slot := range gearSlots {
		if original[slot.ID] != best.gear[slot.ID] || originalLevels[slot.ID] != best.levels[slot.ID] {
			returnSocketGems(st, p, slot.ID)
		}
		returnWornGear(st, p, slot.ID)
	}
	for slot, id := range best.gear {
		level := best.levels[slot]
		if !takeBagGear(st, id, level) {
			return "", errors.New("không thể lấy bộ trang bị đã chọn")
		}
		p.Gear[slot] = id
		if level > 0 {
			p.GearLevel[slot] = level
		}
	}
	msg := fmt.Sprintf("⚔️ %s tự mặc bộ mạnh nhất · lực chiến %d → %d", p.Name, oldPower, bestPower)
	if b := setOf(best.gear); b != nil {
		msg += fmt.Sprintf(" · ✨ %s (%d món)", b.Name, b.Parts)
	}
	return msg, nil
}

func unequipSlot(st *userState, i int, slot string) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	id := p.Gear[slot]
	if id == "" {
		return "", errors.New("khung này đang trống")
	}
	level := p.GearLevel[slot]
	returnSocketGems(st, p, slot)
	returnWornGear(st, p, slot)
	if level > 0 {
		return fmt.Sprintf("Đã tháo %s +%d", itemByID[id].Name, level), nil
	}
	return "Đã tháo " + itemByID[id].Name, nil
}

func enhance(st *userState, i int, slot string) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	id := p.Gear[slot]
	if id == "" {
		return "", errors.New("khung này đang trống")
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	current := p.GearLevel[slot]
	if current >= maxEnhanceLevel {
		return "", fmt.Errorf("%s đã đạt +%d", itemByID[id].Name, maxEnhanceLevel)
	}
	next := current + 1
	coin, shards := enhanceCost(id, next)
	if st.Coin < coin {
		return "", fmt.Errorf("cần %d vàng, bạn còn %d", coin, st.Coin)
	}
	if st.Inv["gem_shard"] < shards {
		return "", fmt.Errorf("cần %d mảnh ngọc, bạn có %d", shards, st.Inv["gem_shard"])
	}
	st.Coin -= coin
	st.add("gem_shard", -shards)
	if p.GearLevel == nil {
		p.GearLevel = map[string]int{}
	}
	p.GearLevel[slot] = next
	return fmt.Sprintf("✨ %s đã lên +%d.", itemByID[id].Name, next), nil
}

func restoreEnhanced(st *userState, id string, level int) (string, error) {
	if _, ok := gearByID[id]; !ok || level < 1 || level > maxEnhanceLevel || st.enhancedCount(id, level) <= 0 {
		return "", errors.New("không có trang bị cường hóa này")
	}
	st.addGear(id, level, -1)
	st.add(id, 1)
	return fmt.Sprintf("Đã hoàn nguyên %s +%d về +0.", itemByID[id].Name, level), nil
}

type enhancedStackView struct {
	Item  string `json:"item"`
	Level int    `json:"level"`
	Count int    `json:"count"`
}

type enhanceView struct {
	Slot       string  `json:"slot"`
	Item       string  `json:"item"`
	Level      int     `json:"level"`
	Max        int     `json:"max"`
	Coin       int     `json:"coin"`
	Shards     int     `json:"shards"`
	Current    gearDef `json:"current"`
	Next       gearDef `json:"next"`
	Affordable bool    `json:"affordable"`
}

func (p *pet) enhanceViews(st *userState) []enhanceView {
	out := []enhanceView{}
	for _, slot := range gearSlots {
		id := p.Gear[slot.ID]
		if id == "" {
			continue
		}
		level := p.GearLevel[slot.ID]
		view := enhanceView{Slot: slot.ID, Item: id, Level: level, Max: maxEnhanceLevel, Current: enhancedGear(id, level), Next: enhancedGear(id, min(level+1, maxEnhanceLevel))}
		if level < maxEnhanceLevel {
			view.Coin, view.Shards = enhanceCost(id, level+1)
			view.Affordable = st.Coin >= view.Coin && st.Inv["gem_shard"] >= view.Shards
		}
		out = append(out, view)
	}
	return out
}

func (st *userState) enhancedViews() []enhancedStackView {
	out := []enhancedStackView{}
	for id, levels := range st.Enhanced {
		for level, count := range levels {
			if count > 0 {
				out = append(out, enhancedStackView{Item: id, Level: level, Count: count})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Item == out[j].Item {
			return out[i].Level < out[j].Level
		}
		return out[i].Item < out[j].Item
	})
	return out
}
