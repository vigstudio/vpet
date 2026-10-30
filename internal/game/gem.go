package game

import (
	"errors"
	"fmt"
	"sort"
)

// Crushing and socketing. Production data showed the opposite of inflation: median gold 4,
// median level 1, zero skill stones, and bags full of tier-0 gear nobody will ever wear
// (188 stones, 134 rusty knives, 112 copper rings). That dead stock is the input here, so a
// player who cannot yet reach the forge still has something to build towards.
const (
	shardPerCrush = 1  // shards from one ordinary piece of junk gear
	gemShards     = 6  // shards that fuse into a tier-1 gem
	gemUpCost     = 3  // gems of one tier that fuse into the next
	socketCost    = 25 // gold to seat a gem, a small sink on every attempt
	maxSockets    = 3  // sockets per gear piece
)

// gemKinds are the three ways a gem can pay. Each maps onto a stat the duel already uses, so
// nothing new has to be balanced into combat maths.
var gemKinds = []struct {
	ID    string
	Name  string
	Emoji string
}{
	{"gem_atk", "Hồng Ngọc", "🔴"},
	{"gem_def", "Lam Ngọc", "🔵"},
	{"gem_hp", "Lục Ngọc", "🟢"},
}

// gemTiers: a gem id is kind + tier, so gem_atk_2 is a tier-2 attack gem.
const gemMaxTier = 4

func gemID(kind string, tier int) string { return fmt.Sprintf("%s_%d", kind, tier) }

// gemBonus is what one seated gem adds. Tiers scale steeply enough to be worth the grind but
// stay below a full gear upgrade, so gems supplement the forge rather than replace it.
func gemBonus(kind string, tier int) gearDef {
	step := float64(tier)
	switch kind {
	case "gem_atk":
		return gearDef{Atk: step * 1.5}
	case "gem_def":
		return gearDef{Def: step * 1.2}
	case "gem_hp":
		return gearDef{HP: step * 8}
	}
	return gearDef{}
}

// crushable reports whether this item may be destroyed for shards. Everything below the
// forge-only rungs qualifies: a spare Epic or Golden piece is still spare. Mythic and Divine
// are never crushable, because they cost three pieces each to make and cannot be found.
func crushable(id string) bool {
	if id == "da" {
		return true
	}
	g, ok := gearByID[id]
	if !ok {
		return false
	}
	return tierOf(id) < mythicRarity && g.Slot != ""
}

// shardValue is what one piece of this item pays. A rung is worth three of the rung below,
// matching the forge: fusing three Commons makes a Rare, so crushing that Rare must return
// what went into it. Without that, crushing upward was a loss and nobody did it.
func shardValue(id string) int {
	if id == "da" {
		return 1
	}
	n := shardPerCrush
	for range tierOf(id) {
		n *= 3
	}
	return n
}

// crush turns junk into shards. Worn gear is untouched: only what sits in the bag is at risk.
func crush(st *userState, id string, count int) (string, error) {
	if !crushable(id) {
		return "", errors.New("không đập được: đồ Thần thoại và Thần cấp phải rèn mới có")
	}
	if count < 1 {
		return "", errors.New("số lượng không hợp lệ")
	}
	if st.Inv[id] < count {
		return "", fmt.Errorf("bạn chỉ có %d× %s", st.Inv[id], itemByID[id].Name)
	}
	gain := count * shardValue(id)
	st.add(id, -count)
	st.add("gem_shard", gain)
	st.bump("crush", count)
	return fmt.Sprintf("🔨 Đập %d× %s thành %d mảnh ngọc.", count, itemByID[id].Name, gain), nil
}

// forgeGem spends shards for a tier-1 gem, or three gems for the next tier up.
func forgeGem(st *userState, kind string, tier int) (string, error) {
	valid := false
	for _, k := range gemKinds {
		if k.ID == kind {
			valid = true
		}
	}
	if !valid || tier < 1 || tier > gemMaxTier {
		return "", errors.New("loại ngọc không hợp lệ")
	}
	if tier == 1 {
		if st.Inv["gem_shard"] < gemShards {
			return "", fmt.Errorf("cần %d mảnh ngọc, bạn có %d", gemShards, st.Inv["gem_shard"])
		}
		st.add("gem_shard", -gemShards)
		st.add(gemID(kind, 1), 1)
		return fmt.Sprintf("💎 Ghép %d mảnh thành %s cấp 1.", gemShards, gemName(kind)), nil
	}
	lower := gemID(kind, tier-1)
	if st.Inv[lower] < gemUpCost {
		return "", fmt.Errorf("cần %d× %s cấp %d, bạn có %d", gemUpCost, gemName(kind), tier-1, st.Inv[lower])
	}
	st.add(lower, -gemUpCost)
	st.add(gemID(kind, tier), 1)
	return fmt.Sprintf("💎 Nâng %s lên cấp %d.", gemName(kind), tier), nil
}

func gemName(kind string) string {
	for _, k := range gemKinds {
		if k.ID == kind {
			return k.Name
		}
	}
	return kind
}

// socket seats a gem in a worn piece. Sockets belong to the pet's gear slot rather than the
// item id, so swapping gear does not silently move gems onto something else.
func socket(st *userState, i int, slot, gem string, now int64) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	worn := p.Gear[slot]
	if worn == "" {
		return "", errors.New("ô này chưa có trang bị để khảm")
	}
	if st.Inv[gem] <= 0 {
		return "", errors.New("bạn không có viên ngọc này")
	}
	if _, _, ok := parseGem(gem); !ok {
		return "", errors.New("món này không phải ngọc")
	}
	if p.Sockets == nil {
		p.Sockets = map[string][]string{}
	}
	if len(p.Sockets[slot]) >= maxSockets {
		return "", fmt.Errorf("ô %s đã đủ %d lỗ khảm", slot, maxSockets)
	}
	if st.Coin < socketCost {
		return "", fmt.Errorf("cần %d vàng tiền công khảm, bạn còn %d", socketCost, st.Coin)
	}
	st.Coin -= socketCost
	st.add(gem, -1)
	p.Sockets[slot] = append(p.Sockets[slot], gem)
	return fmt.Sprintf("✨ Khảm %s vào %s.", itemByID[gem].Name, itemByID[worn].Name), nil
}

// unsocket pries a gem back out. The gem survives; the fee does not, so swapping builds costs
// something each time.
func unsocket(st *userState, i int, slot string, at int) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	gems := p.Sockets[slot]
	if at < 0 || at >= len(gems) {
		return "", errors.New("lỗ khảm không tồn tại")
	}
	if st.Coin < socketCost {
		return "", fmt.Errorf("cần %d vàng để gỡ ngọc, bạn còn %d", socketCost, st.Coin)
	}
	gem := gems[at]
	st.Coin -= socketCost
	st.add(gem, 1)
	p.Sockets[slot] = append(gems[:at:at], gems[at+1:]...)
	if len(p.Sockets[slot]) == 0 {
		delete(p.Sockets, slot)
	}
	return fmt.Sprintf("Đã gỡ %s khỏi ô %s.", itemByID[gem].Name, slot), nil
}

// parseGem splits a gem id back into its kind and tier.
func parseGem(id string) (string, int, bool) {
	for _, k := range gemKinds {
		for tier := 1; tier <= gemMaxTier; tier++ {
			if gemID(k.ID, tier) == id {
				return k.ID, tier, true
			}
		}
	}
	return "", 0, false
}

// gemsOf totals every seated gem on a pet. Gear that was unequipped leaves its sockets behind,
// so only slots still holding gear contribute.
func (p *pet) gemsOf() gearDef {
	var t gearDef
	for slot, gems := range p.Sockets {
		if p.Gear[slot] == "" {
			continue
		}
		for _, id := range gems {
			kind, tier, ok := parseGem(id)
			if !ok {
				continue
			}
			b := gemBonus(kind, tier)
			t.Atk += b.Atk
			t.Def += b.Def
			t.HP += b.HP
		}
	}
	return t
}

// gemItems registers every gem and the shard in the catalog, so icons, the bag and the market
// all treat them like any other item.
func gemItems() []itemDef {
	out := []itemDef{{ID: "gem_shard", Name: "Mảnh ngọc", Emoji: "🔸", Kind: "gem",
		Desc: fmt.Sprintf("Đập trang bị thừa mà có. Ghép %d mảnh thành một viên ngọc", gemShards)}}
	for _, k := range gemKinds {
		for tier := 1; tier <= gemMaxTier; tier++ {
			b := gemBonus(k.ID, tier)
			stat := ""
			switch k.ID {
			case "gem_atk":
				stat = fmt.Sprintf("+%.1f sát thương", b.Atk)
			case "gem_def":
				stat = fmt.Sprintf("+%.1f giáp", b.Def)
			case "gem_hp":
				stat = fmt.Sprintf("+%.0f máu đấu", b.HP)
			}
			out = append(out, itemDef{ID: gemID(k.ID, tier), Name: fmt.Sprintf("%s cấp %d", k.Name, tier),
				Emoji: k.Emoji, Kind: "gem", Rarity: min(tier, 3),
				Desc: fmt.Sprintf("Khảm vào trang bị: %s", stat)})
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	return out
}

// socketView is what the panel draws for one gear slot.
type socketView struct {
	Slot string   `json:"slot"`
	Item string   `json:"item"`
	Gems []string `json:"gems"`
	Free int      `json:"free"`
}

func (p *pet) socketViews() []socketView {
	out := []socketView{}
	for _, s := range gearSlots {
		if p.Gear[s.ID] == "" {
			continue
		}
		gems := append([]string{}, p.Sockets[s.ID]...)
		out = append(out, socketView{Slot: s.ID, Item: p.Gear[s.ID], Gems: gems, Free: maxSockets - len(gems)})
	}
	return out
}
