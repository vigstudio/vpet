package game

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// Hunting, levels, gear and skill ranks. Pets explore a map, bring back loot by rarity and
// earn XP; gear sits in three slots and feeds straight into duel maths.
const (
	huntCooldown  = 10 * time.Minute
	maxSkillSlots = 4
	maxSkillLv    = 5
	maxLv         = 500
	huntHunger    = 10 // hunger spent per trip
	minHuntHealth = 20
)

// rarity 0 trắng, 1 xanh, 2 tím, 3 hoàng kim, 4 thần thoại (chỉ rèn ra được)
var rarityNames = []string{"Thường", "Hiếm", "Sử thi", "Hoàng kim", "Thần thoại", "Thần cấp", "Tối Thượng"}

// mythicRarity is forge-only: no map, bag, shop or blind box ever hands one out. Everything
// at or above this rung is earned at the forge, which keeps the top of the ladder a goal
// rather than a lucky drop.
const mythicRarity = 4

// divineRarity sits one rung above mythic: three mythic pieces fuse into it. It is the only
// reason to keep playing once a full mythic set is worn.
const divineRarity = 5
const supremeRarity = 6

type gearDef struct {
	// json tags: the client shows the worn totals on the paper doll
	// json tags: the client shows the worn totals on the paper doll
	Slot string  `json:"slot"` // vukhi, giap, bua
	Atk  float64 `json:"atk,omitempty"`
	Def  float64 `json:"def,omitempty"`
	HP   float64 `json:"hp,omitempty"`
}

// huntItems are found while hunting only (Weight 0 keeps them out of food bags).
var huntItems = []itemDef{
	{ID: "dao", Name: "Dao gỉ", Emoji: "🔪", Kind: "gear", Desc: "Vũ khí: +2 sát thương", Rarity: 0},
	{ID: "kiem", Name: "Kiếm sắt", Emoji: "⚔️", Kind: "gear", Desc: "Vũ khí: +5 sát thương", Rarity: 1},
	{ID: "dinhba", Name: "Đinh ba biển", Emoji: "🔱", Kind: "gear", Desc: "Vũ khí: +9 sát thương", Rarity: 2},
	{ID: "gaythan", Name: "Gậy thần", Emoji: "🪄", Kind: "gear", Desc: "Vũ khí: +14 sát thương", Rarity: 3},
	{ID: "aovai", Name: "Áo vải", Emoji: "🥋", Kind: "gear", Desc: "Giáp: giảm 1 sát thương mỗi đòn", Rarity: 0},
	{ID: "giapda", Name: "Giáp da", Emoji: "🦬", Kind: "gear", Desc: "Giáp: giảm 3 sát thương mỗi đòn", Rarity: 1},
	{ID: "giapthep", Name: "Giáp thép", Emoji: "🦺", Kind: "gear", Desc: "Giáp: giảm 5 sát thương mỗi đòn", Rarity: 2},
	{ID: "longbao", Name: "Long bào", Emoji: "👘", Kind: "gear", Desc: "Giáp: giảm 8 sát thương mỗi đòn", Rarity: 3},
	{ID: "nonre", Name: "Nón rơm", Emoji: "👒", Kind: "gear", Desc: "Mũ: +1 giáp, +3 máu", Rarity: 0},
	{ID: "muda", Name: "Mũ da", Emoji: "🪖", Kind: "gear", Desc: "Mũ: +2 giáp, +6 máu", Rarity: 1},
	{ID: "muthep", Name: "Mũ thép", Emoji: "⛑️", Kind: "gear", Desc: "Mũ: +3 giáp, +12 máu", Rarity: 2},
	{ID: "vuongmien", Name: "Vương miện", Emoji: "👑", Kind: "gear", Desc: "Mũ: +4 giáp, +22 máu", Rarity: 3},
	{ID: "gangvai", Name: "Găng vải", Emoji: "🧤", Kind: "gear", Desc: "Găng: +1 sát thương", Rarity: 0},
	{ID: "gangda", Name: "Găng da", Emoji: "🥊", Kind: "gear", Desc: "Găng: +2 sát thương, +1 giáp", Rarity: 1},
	{ID: "gangthep", Name: "Găng thép", Emoji: "🦾", Kind: "gear", Desc: "Găng: +4 sát thương, +2 giáp", Rarity: 2},
	{ID: "gangrong", Name: "Găng vảy rồng", Emoji: "🐲", Kind: "gear", Desc: "Găng: +7 sát thương, +3 giáp", Rarity: 3},
	{ID: "depco", Name: "Dép cỏ", Emoji: "🩴", Kind: "gear", Desc: "Giày: +3 máu", Rarity: 0},
	{ID: "giayda", Name: "Giày da", Emoji: "👞", Kind: "gear", Desc: "Giày: +1 giáp, +6 máu", Rarity: 1},
	{ID: "giaythep", Name: "Giày thép", Emoji: "🥾", Kind: "gear", Desc: "Giày: +2 giáp, +12 máu", Rarity: 2},
	{ID: "hiabay", Name: "Hia bay", Emoji: "🪽", Kind: "gear", Desc: "Giày: +3 giáp, +20 máu", Rarity: 3},
	{ID: "nhandong", Name: "Nhẫn đồng", Emoji: "⭕", Kind: "gear", Desc: "Nhẫn: +6 máu", Rarity: 0},
	{ID: "chuoingoc", Name: "Chuỗi ngọc", Emoji: "📿", Kind: "gear", Desc: "Nhẫn: +14 máu", Rarity: 1},
	{ID: "nhanbac", Name: "Nhẫn bạc", Emoji: "💍", Kind: "gear", Desc: "Nhẫn: +2 sát thương, +18 máu", Rarity: 2},
	{ID: "nhanrong", Name: "Nhẫn rồng", Emoji: "🔱", Kind: "gear", Desc: "Nhẫn: +4 sát thương, +30 máu", Rarity: 3},
	// shields and cloaks: the two slots added in 1.14, same five-rung ladder
	{ID: "motnap", Name: "Mộc nắp vung", Emoji: "🥘", Kind: "gear", Desc: "Khiên: giảm 2 sát thương", Rarity: 0},
	{ID: "khiengo", Name: "Khiên gỗ", Emoji: "🪵", Kind: "gear", Desc: "Khiên: giảm 3 sát thương, +5 máu", Rarity: 1},
	{ID: "khiensat", Name: "Khiên sắt", Emoji: "🔰", Kind: "gear", Desc: "Khiên: giảm 5 sát thương, +10 máu", Rarity: 2},
	{ID: "khienbac", Name: "Khiên bạc", Emoji: "🛡️", Kind: "gear", Desc: "Khiên: giảm 7 sát thương, +18 máu", Rarity: 3},
	{ID: "aotoi", Name: "Áo tơi", Emoji: "🍃", Kind: "gear", Desc: "Áo khoác: giảm 1 sát thương, +2 máu", Rarity: 0},
	{ID: "aogam", Name: "Áo gấm", Emoji: "🧥", Kind: "gear", Desc: "Áo khoác: +1 sát thương, +1 giáp, +6 máu", Rarity: 1},
	{ID: "aolong", Name: "Áo lông thú", Emoji: "🧶", Kind: "gear", Desc: "Áo khoác: +2 sát thương, +2 giáp, +12 máu", Rarity: 2},
	{ID: "aosao", Name: "Áo sao băng", Emoji: "🌟", Kind: "gear", Desc: "Áo khoác: +3 sát thương, +3 giáp, +20 máu", Rarity: 3},
	// mythic rung: only the forge makes these, never a drop
	{ID: "kiemrong", Name: "Kiếm Long Ngâm", Emoji: "🐉", Kind: "gear", Desc: "Vũ khí: +21 sát thương", Rarity: 4},
	{ID: "giaprong", Name: "Giáp vảy rồng", Emoji: "🐲", Kind: "gear", Desc: "Giáp: giảm 12 sát thương mỗi đòn", Rarity: 4},
	{ID: "muthan", Name: "Mũ miện thần", Emoji: "✨", Kind: "gear", Desc: "Mũ: +6 giáp, +34 máu", Rarity: 4},
	{ID: "gangthan", Name: "Găng thần lực", Emoji: "💥", Kind: "gear", Desc: "Găng: +11 sát thương, +5 giáp", Rarity: 4},
	{ID: "hiathan", Name: "Hia thiên lý", Emoji: "🌠", Kind: "gear", Desc: "Giày: +5 giáp, +32 máu", Rarity: 4},
	{ID: "nhanthan", Name: "Nhẫn thần", Emoji: "💫", Kind: "gear", Desc: "Nhẫn: +7 sát thương, +46 máu", Rarity: 4},
	{ID: "khienthan", Name: "Khiên thần", Emoji: "🛑", Kind: "gear", Desc: "Khiên: giảm 10 sát thương, +30 máu", Rarity: 4},
	{ID: "aothan", Name: "Thiên y", Emoji: "👼", Kind: "gear", Desc: "Áo khoác: +5 sát thương, +5 giáp, +32 máu", Rarity: 4},
	// Thần cấp: forged from three mythic pieces, the sixth rung on every ladder.
	{ID: "kiemthanlong", Name: "Kiếm Thần Long", Emoji: "🗡️", Kind: "gear", Desc: "Vũ khí: +30 sát thương", Rarity: 5},
	{ID: "giapthanlong", Name: "Giáp Thần Long", Emoji: "🛡️", Kind: "gear", Desc: "Giáp: giảm 17 sát thương mỗi đòn", Rarity: 5},
	{ID: "muthiende", Name: "Mũ Thiên Đế", Emoji: "🔆", Kind: "gear", Desc: "Mũ: +9 giáp, +48 máu", Rarity: 5},
	{ID: "gangthienloi", Name: "Găng Thiên Lôi", Emoji: "⚡", Kind: "gear", Desc: "Găng: +16 sát thương, +7 giáp", Rarity: 5},
	{ID: "hiathienvuong", Name: "Hia Thiên Vương", Emoji: "🌟", Kind: "gear", Desc: "Giày: +7 giáp, +46 máu", Rarity: 5},
	{ID: "nhanthienmenh", Name: "Nhẫn Thiên Mệnh", Emoji: "🔮", Kind: "gear", Desc: "Nhẫn: +10 sát thương, +64 máu", Rarity: 5},
	{ID: "khienthienha", Name: "Khiên Thiên Hà", Emoji: "🌌", Kind: "gear", Desc: "Khiên: giảm 14 sát thương, +42 máu", Rarity: 5},
	{ID: "aothienmenh", Name: "Thiên Mệnh Bào", Emoji: "🎆", Kind: "gear", Desc: "Áo khoác: +7 sát thương, +7 giáp, +46 máu", Rarity: 5},
	// Supreme gear and bound souls never enter ordinary loot pools.
	{ID: "kiemhongmong", Name: "Kiếm Hồng Mông", Emoji: "🗡️", Kind: "gear", Desc: "sát thương +42", Rarity: 6},
	{ID: "giaphongmong", Name: "Giáp Hồng Mông", Emoji: "🛡️", Kind: "gear", Desc: "giáp +23.8", Rarity: 6},
	{ID: "muhongmong", Name: "Mũ Hồng Mông", Emoji: "👑", Kind: "gear", Desc: "giáp +12.6, máu +67.2", Rarity: 6},
	{ID: "ganghongmong", Name: "Găng Hồng Mông", Emoji: "🧤", Kind: "gear", Desc: "sát thương +22.4, giáp +9.8", Rarity: 6},
	{ID: "hiahongmong", Name: "Hia Hồng Mông", Emoji: "🌠", Kind: "gear", Desc: "giáp +9.8, máu +64.4", Rarity: 6},
	{ID: "nhanhongmong", Name: "Nhẫn Hồng Mông", Emoji: "💍", Kind: "gear", Desc: "sát thương +14, máu +89.6", Rarity: 6},
	{ID: "khienhongmong", Name: "Khiên Hồng Mông", Emoji: "🔰", Kind: "gear", Desc: "giáp +19.6, máu +58.8", Rarity: 6},
	{ID: "aohongmong", Name: "Áo Hồng Mông", Emoji: "🧥", Kind: "gear", Desc: "sát thương +9.8, giáp +9.8, máu +64.4", Rarity: 6},
	{ID: "tinhhon", Name: "Tinh Hồn", Emoji: "🔆", Kind: "material", Desc: "Vật liệu khóa: cần 3 để rèn Tối Thượng. Boss bị hạ: 15% nếu góp ít nhất 60% mức bình quân; thắng lãnh thổ bậc 3: 5%. Không mua bán hay trao đổi", Rarity: 6},
	{ID: "ngoc", Name: "Ngọc kỹ năng", Emoji: "💠", Kind: "stone", Desc: "Nâng 1 cấp cho một kỹ năng tỷ thí", Rarity: 2},
	{ID: "da", Name: "Đá thường", Emoji: "🪨", Kind: "junk", Desc: "Nhặt được khi đi săn. Bán không ai mua", Rarity: 0},
	{ID: "thaomoc", Name: "Thảo mộc", Emoji: "🌿", Kind: "med", Desc: "Hồi 12 sức khỏe", Health: 12, Rarity: 0},
	{ID: "mathu", Name: "Mật thú rừng", Emoji: "🍯", Kind: "food", Desc: "Ngọt lịm, no và vui", Hunger: 35, Happy: 18, Effect: "ngon", Minutes: 15, Rarity: 1},
}

type huntMap struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Emoji string   `json:"emoji"`
	Desc  string   `json:"desc"`
	MinLv int      `json:"min_lv"`
	XP    int      `json:"xp"`
	Risk  float64  `json:"risk"` // chance of coming back hurt
	Odds  [4]int   `json:"odds"` // drop weight per rarity
	X     int      `json:"x"`    // where the region sits on the 100x100 world canvas
	Y     int      `json:"y"`
	Finds []string `json:"-"` // extra item ids this map can yield
}

// maps go from the back garden to a volcano; later maps need levels but drop far better loot.
var huntMaps = []huntMap{
	{ID: "vuon", Name: "Vườn sau nhà", Emoji: "🌳", Desc: "An toàn, hợp thú mới lớn", MinLv: 1, XP: 8, Risk: 0.04, Odds: [4]int{92, 7, 1, 0}, X: 18, Y: 74},
	{ID: "rung", Name: "Rừng rậm", Emoji: "🌲", Desc: "Nhiều thảo mộc và thú hoang", MinLv: 5, XP: 16, Risk: 0.12, Odds: [4]int{80, 17, 3, 0}, X: 40, Y: 56},
	{ID: "hang", Name: "Hang tối", Emoji: "🕳️", Desc: "Tối om, dễ sứt đầu mẻ trán", MinLv: 12, XP: 28, Risk: 0.2, Odds: [4]int{66, 27, 6, 1}, X: 62, Y: 70},
	{ID: "nui", Name: "Núi tuyết", Emoji: "🏔️", Desc: "Lạnh cắt da, đồ xịn trên đỉnh", MinLv: 22, XP: 44, Risk: 0.27, Odds: [4]int{50, 35, 13, 2}, X: 72, Y: 32},
	{ID: "nuilua", Name: "Núi lửa", Emoji: "🌋", Desc: "Nguy hiểm nhất, báu vật hoàng kim", MinLv: 35, XP: 65, Risk: 0.34, Odds: [4]int{32, 42, 22, 4}, X: 88, Y: 62},
}

var mapByID = func() map[string]*huntMap {
	m := make(map[string]*huntMap, len(huntMaps))
	for i := range huntMaps {
		m[huntMaps[i].ID] = &huntMaps[i]
	}
	return m
}()

// xpNeed is the XP to go from lv to lv+1.
// softCapLv is where the old fifty-level ladder used to end. Everything at or below it keeps
// exactly the cost it always had, so nobody's stored progress is revalued; past it the curve
// turns steep, which is what stops level 500 from arriving in a weekend.
const softCapLv = 50

func xpNeed(lv int) int {
	n := 20 + lv*12
	if lv > softCapLv {
		over := lv - softCapLv
		n += over * over * 3
	}
	return n
}

// skillSlots grows with level, so a veteran pet fights with more moves. The ceiling is
// reached well before the level cap, so the extra levels are a long tail rather than a
// second power curve.
func skillSlots(lv int) int {
	n := 2 + lv/10
	return min(n, maxSkillSlots)
}

type medal struct {
	Emoji string `json:"emoji"`
	Name  string `json:"name"`
	Desc  string `json:"desc"`
}

// medals are derived from the pet's record, so there is nothing extra to store.
func medals(p *pet) []medal {
	out := []medal{}
	add := func(ok bool, emoji, name, desc string) {
		if ok {
			out = append(out, medal{emoji, name, desc})
		}
	}
	add(p.Lv >= 5, "🥉", "Tân binh", "Đạt cấp 5")
	add(p.Lv >= 15, "🥈", "Lão luyện", "Đạt cấp 15")
	add(p.Lv >= 30, "🥇", "Cao thủ", "Đạt cấp 30")
	add(p.Lv >= maxLv, "💎", "Huyền thoại", "Đạt cấp tối đa")
	add(p.Hunts >= 20, "🗺️", "Lữ khách", "Đi săn 20 lần")
	add(p.Hunts >= 100, "🏹", "Thợ săn", "Đi săn 100 lần")
	add(p.Wild >= 25, "🐾", "Thợ diệt thú", "Hạ 25 thú hoang")
	add(p.Wins >= 10, "⚔️", "Chiến binh", "Thắng 10 trận tỷ thí")
	add(p.Wins >= 50, "🏆", "Vô địch", "Thắng 50 trận tỷ thí")
	add(p.Gold > 0, "👑", "Tay săn hoàng kim", "Nhặt được đồ hoàng kim")
	return out
}

// gearOf totals the equipped bonuses of a pet, set bonus included, so the paper doll shows
// exactly the numbers the duel will use.
func (p *pet) gearOf() gearDef {
	var t gearDef
	for slot, id := range p.Gear {
		g := enhancedGear(id, p.GearLevel[slot])
		t.Atk += g.Atk
		t.Def += g.Def
		t.HP += g.HP
	}
	if b := setOf(p.Gear); b != nil {
		t.Atk += b.Atk
		t.Def += b.Def
		t.HP += b.HP
	}
	g := p.gemsOf()
	t.Atk += g.Atk
	t.Def += g.Def
	t.HP += g.HP
	return t
}

func (p *pet) skillLv(id string) int {
	if lv := p.SkillLv[id]; lv > 0 {
		return min(lv, maxSkillLv)
	}
	return 1
}

// gainXP adds XP and levels the pet up, learning a new skill whenever a slot opens.
func gainXP(p *pet, xp int, rnd rng) string {
	if p.Lv < 1 {
		p.Lv = 1
	}
	p.XP += xp
	var notes []string
	for p.Lv < maxLv && p.XP >= xpNeed(p.Lv) {
		p.XP -= xpNeed(p.Lv)
		p.Lv++
		notes = append(notes, fmt.Sprintf("🎉 Lên cấp %d!", p.Lv))
		if len(p.Skills) < skillSlots(p.Lv) {
			if id := learnSkill(p, rnd); id != "" {
				notes = append(notes, fmt.Sprintf("Mở ô kỹ năng mới: %s %s", skillByID[id].Emoji, skillByID[id].Name))
			}
		}
	}
	if p.Lv >= maxLv {
		p.XP = 0
	}
	return strings.Join(notes, " ")
}

// learnSkill teaches one move the pet does not have yet.
func learnSkill(p *pet, rnd rng) string {
	var pool []string
	for _, s := range skills {
		has := false
		for _, id := range p.Skills {
			if id == s.ID {
				has = true
			}
		}
		if !has {
			pool = append(pool, s.ID)
		}
	}
	if len(pool) == 0 {
		return ""
	}
	id := pool[int(rnd()*float64(len(pool)))%len(pool)]
	p.Skills = append(p.Skills, id)
	return id
}

// rollRarity picks a rarity band for one find.
func rollRarity(m *huntMap, rnd rng) int {
	total := 0
	for _, w := range m.Odds {
		total += w
	}
	n := rnd() * float64(total)
	for r, w := range m.Odds {
		n -= float64(w)
		if w > 0 && n < 0 {
			return r
		}
	}
	return 0
}

// lootOf lists the items a map can yield at one rarity.
func lootOf(rarity int) []string {
	var ids []string
	if rarity >= mythicRarity {
		return nil // the mythic rung is forged, never found
	}
	for _, it := range huntItems {
		if it.Rarity == rarity && it.Kind != "junk" {
			ids = append(ids, it.ID)
		}
	}
	if rarity == 0 {
		ids = append(ids, "da", "com", "tui")
	}
	return ids
}

type huntFind struct {
	Item   string `json:"item"`
	Rarity int    `json:"rarity"`
	Amount int    `json:"amount,omitempty"`
}

type huntResult struct {
	Map   string     `json:"map"`
	Finds []huntFind `json:"finds"`
	XP    int        `json:"xp"`
	Coin  int        `json:"coin"` // gold brought home from the trip
	Hurt  float64    `json:"hurt"`
	Note  string     `json:"note"`
}

// hunt sends pet i to a map: it costs hunger and time, may hurt the pet, and brings back
// one to three finds plus XP. Nothing changes when it returns an error.
func hunt(st *userState, i int, mapID string, now int64, rnd rng) (*huntResult, error) {
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
		return nil, errors.New(p.Name + " đang bệnh, chữa khỏi rồi hãy đi săn")
	case p.Health < minHuntHealth:
		return nil, errors.New(p.Name + " yếu quá, cần hồi sức trước")
	case p.Hunger < huntHunger:
		return nil, errors.New(p.Name + " đói quá, cho ăn rồi hãy đi")
	case p.Lv < m.MinLv:
		return nil, fmt.Errorf("%s cần cấp %d mới vào được %s", p.Name, m.MinLv, m.Name)
	case p.LastHunt > 0 && now-p.LastHunt < huntCooldown.Milliseconds():
		left := (huntCooldown.Milliseconds() - (now - p.LastHunt)) / 60000
		return nil, fmt.Errorf("%s còn mệt, nghỉ thêm %d phút nữa", p.Name, left+1)
	}

	p.LastHunt = now
	p.Hunts++
	p.Hunger = clamp(p.Hunger - huntHunger)
	p.Happy = clamp(p.Happy + 6)

	res := &huntResult{Map: m.ID, XP: m.XP + int(rnd()*float64(m.XP)/2)}
	n := 1
	if rnd() < 0.25 {
		n = 2
	}
	if rnd() < 0.04 {
		n = 3
	}
	for k := 0; k < n; k++ {
		r := rollRarity(m, rnd)
		pool := lootOf(r)
		if len(pool) == 0 {
			continue
		}
		id := pool[int(rnd()*float64(len(pool)))%len(pool)]
		st.add(id, 1)
		res.Finds = append(res.Finds, huntFind{Item: id, Rarity: r})
		if r >= 3 {
			p.Gold++
		}
	}
	if rnd() < m.Risk {
		// the nastier the map, the harder the wound: only the late maps can actually kill
		res.Hurt = math.Round(5 + rnd()*12 + m.Risk*40)
		p.hurt(res.Hurt, "bị thương khi đi săn ở "+m.Name)
		p.setEffect("dau", 10, now)
	}
	if p.Dead {
		p.DiedAt = now
		res.Note = "💀 " + p.Name + " đã bỏ mạng ở " + m.Name + "…"
		return res, nil
	}
	// the trip pays: deeper maps pay more, so the shop and the blind box stay reachable
	res.Coin = st.payout(m.XP/2+int(rnd()*float64(m.XP)), now)
	st.Coin += res.Coin
	st.bump("hunt", 1)
	res.Note = gainXP(p, res.XP, rnd)
	return res, nil
}

// equip puts a gear item from the bag into its slot; whatever was there goes back to the bag.

// unequipSlot takes the gear out of one slot.

// skillUp spends a skill stone to raise one of the pet's moves.
func skillUp(st *userState, i int, skill string, now int64) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if st.Inv["ngoc"] <= 0 {
		return "", errors.New("bạn không có 💠 Ngọc kỹ năng")
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	has := false
	for _, id := range p.Skills {
		if id == skill {
			has = true
		}
	}
	sk := skillByID[skill]
	if !has || sk == nil {
		return "", errors.New("thú không có kỹ năng này")
	}
	if p.skillLv(skill) >= maxSkillLv {
		return "", fmt.Errorf("%s %s đã đạt cấp tối đa", sk.Emoji, sk.Name)
	}
	if p.SkillLv == nil {
		p.SkillLv = map[string]int{}
	}
	st.add("ngoc", -1)
	p.SkillLv[skill] = p.skillLv(skill) + 1
	return fmt.Sprintf("%s %s lên cấp %d!", sk.Emoji, sk.Name, p.SkillLv[skill]), nil
}
