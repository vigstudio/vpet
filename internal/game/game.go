package game

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

// Game rules. State changes stay on userState so they can be tested without HTTP or
// PostgreSQL; engine.go handles loading, locking and saving.
const (
	hungerPerHour = 1.5 // full to empty in ~66 hours, so a pet fed on Friday evening survives the weekend
	happyPerHour  = 1.0
	starveDamage  = 5.0 // health lost per hour while the hunger bar is empty
	sickDamage    = 2.5
	sadDamage     = 1.0
	regenPerHour  = 1.0
	sickAfterLow  = 4.0 // hours below 15% hunger before the pet catches a cold
	maxGraves     = 20
	dropCooldown  = 30 * time.Minute
	dropsPerDay   = 6
	dropChance    = 0.4
	startBags     = 5
	maxAir        = 3 // bags waiting for pickup; further drops go straight into the bag
	offerTTL      = 72 * time.Hour
	maxOffers     = 10
	maxNameRunes  = 20
	maxSlots      = 4 // pets one user can keep at once
	vacation      = 72 * time.Hour
)

var vnZone = time.FixedZone("ICT", 7*3600)

type rng func() float64

type itemDef struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Emoji   string  `json:"emoji"`
	Kind    string  `json:"kind"` // bag, food, med, revive, card, acc, gear, stone, junk
	Desc    string  `json:"desc"`
	Hunger  int     `json:"hunger"`
	Happy   int     `json:"happy"`
	Health  int     `json:"health"`
	Effect  string  `json:"effect,omitempty"`
	Minutes int     `json:"-"`
	Sick    float64 `json:"-"`      // chance of falling ill after eating
	Poison  float64 `json:"-"`      // chance of poisoning, otherwise the pet gets hyper
	Weight  int     `json:"-"`      // drop weight inside a food bag
	Rarity  int     `json:"rarity"` // 0 thường, 1 hiếm, 2 sử thi, 3 hoàng kim
}

var catalog = []itemDef{
	{ID: "tui", Name: "Túi thức ăn", Emoji: "🎁", Kind: "bag", Desc: "Mở ra được 1–2 món ngẫu nhiên"},
	{ID: "tui_y", Name: "Túi quân y", Emoji: "🧰", Kind: "bag", Desc: "Thuốc men và thảo mộc", Rarity: 1},
	{ID: "tui_vukhi", Name: "Hòm vũ khí", Emoji: "📦", Kind: "bag", Desc: "Một món trang bị ngẫu nhiên", Rarity: 1},
	{ID: "tui_vang", Name: "Rương vàng", Emoji: "🧳", Kind: "bag", Desc: "Vàng và một món hiếm", Rarity: 2},
	{ID: "com", Name: "Cơm nắm", Emoji: "🍙", Kind: "food", Desc: "No bụng, lành tính", Hunger: 25, Happy: 2, Weight: 22},
	{ID: "banhmi", Name: "Bánh mì", Emoji: "🥖", Kind: "food", Desc: "Chắc dạ", Hunger: 30, Weight: 18},
	{ID: "pho", Name: "Phở bò", Emoji: "🍜", Kind: "food", Desc: "Ngon! Thú nhảy cẫng lên", Hunger: 40, Happy: 15, Effect: "ngon", Minutes: 15, Weight: 8},
	{ID: "sushi", Name: "Sushi", Emoji: "🍣", Kind: "food", Desc: "Món sang, rất vui", Hunger: 30, Happy: 20, Effect: "ngon", Minutes: 15, Weight: 5},
	{ID: "banhkem", Name: "Bánh kem", Emoji: "🎂", Kind: "food", Desc: "Ngọt ngào, vui hết cỡ", Hunger: 20, Happy: 25, Effect: "ngon", Minutes: 20, Weight: 4},
	{ID: "trasua", Name: "Trà sữa", Emoji: "🧋", Kind: "food", Desc: "Vui vẻ nhưng ít no", Hunger: 15, Happy: 20, Effect: "ngon", Minutes: 10, Weight: 7},
	{ID: "ot", Name: "Ớt hiểm", Emoji: "🌶️", Kind: "food", Desc: "Cay! Thú chạy vòng vòng. Ăn cay liên tục hại sức khỏe", Hunger: 5, Happy: -5, Health: -3, Effect: "cay", Minutes: 20, Weight: 7},
	{ID: "micay", Name: "Mì cay 7 cấp", Emoji: "🍲", Kind: "food", Desc: "No nhưng cay xé lưỡi", Hunger: 30, Happy: 5, Health: -5, Effect: "cay", Minutes: 30, Weight: 6},
	{ID: "saurieng", Name: "Sầu riêng", Emoji: "🍈", Kind: "food", Desc: "Thơm với đứa này, hôi với đứa kia", Hunger: 30, Happy: 10, Effect: "hoi", Minutes: 30, Weight: 6},
	{ID: "mamtom", Name: "Mắm tôm", Emoji: "🦐", Kind: "food", Desc: "Hôi nồng nặc, ruồi bu", Hunger: 20, Effect: "hoi", Minutes: 40, Weight: 6},
	{ID: "trungthoi", Name: "Trứng thối", Emoji: "🥚", Kind: "food", Desc: "Hôi, dễ đau bụng", Hunger: 15, Happy: -10, Effect: "hoi", Minutes: 30, Sick: 0.5, Weight: 5},
	{ID: "nam", Name: "Nấm lạ", Emoji: "🍄", Kind: "food", Desc: "Thử vận may: phê hoặc ngộ độc", Hunger: 10, Poison: 0.35, Weight: 4},
	{ID: "caphe", Name: "Cà phê sữa đá", Emoji: "☕", Kind: "food", Desc: "Tỉnh như sáo, chạy nhảy liên tục", Hunger: 5, Happy: 10, Effect: "hyper", Minutes: 30, Weight: 7},
	{ID: "thuoc", Name: "Thuốc cảm", Emoji: "💊", Kind: "med", Desc: "Chữa bệnh. Không bệnh mà uống thì hại", Health: 15, Happy: -5, Weight: 7},
	{ID: "vitamin", Name: "Vitamin", Emoji: "🧪", Kind: "med", Desc: "Hồi sức khỏe", Health: 25, Happy: 5, Weight: 5},
	{ID: "banggac", Name: "Băng gạc", Emoji: "🩹", Kind: "med", Desc: "Hồi 18 sức khỏe, dùng ngay sau khi đi săn về", Health: 18, Weight: 6},
	{ID: "thuocbo", Name: "Thuốc bổ", Emoji: "🍶", Kind: "med", Desc: "Hồi 40 sức khỏe và 15 vui", Health: 40, Happy: 15, Weight: 3, Rarity: 1},
	{ID: "canhtien", Name: "Canh tiên", Emoji: "🍲", Kind: "med", Desc: "Hồi đầy sức khỏe, hết bệnh", Health: 100, Happy: 25, Weight: 1, Rarity: 2},
	{ID: "nuocthanh", Name: "Nước thánh", Emoji: "⚗️", Kind: "med", Desc: "Hồi 70 sức khỏe, xoá mọi hiệu ứng xấu", Health: 70, Weight: 2, Rarity: 2},
	{ID: "long", Name: "Lông phượng hoàng", Emoji: "🪶", Kind: "revive", Desc: "Cực hiếm: hồi sinh thú đã chết", Weight: 1},
	{ID: "the_skin", Name: "Thẻ biến hình", Emoji: "🎴", Kind: "card", Desc: "Đổi giao diện và tên một con thú, giữ nguyên chỉ số", Weight: 3},
	{ID: "the_slot", Name: "Thẻ thêm chuồng", Emoji: "🏠", Kind: "card", Desc: "Nuôi thêm được một con thú (tối đa 4)", Weight: 2},
	{ID: "the_nghi", Name: "Thẻ nghỉ phép", Emoji: "🏖️", Kind: "card", Desc: "Thú không đói, không buồn trong 72 giờ (dùng khi nghỉ lễ)", Weight: 2},
	{ID: "nonla", Name: "Nón lá", Emoji: "👒", Kind: "acc", Desc: "Phụ kiện đội đầu. Đeo lại món khác thì món cũ về túi", Weight: 2},
	{ID: "kinh", Name: "Kính râm", Emoji: "🕶️", Kind: "acc", Desc: "Phụ kiện: ngầu hết nấc", Weight: 2},
	{ID: "no", Name: "Nơ đỏ", Emoji: "🎀", Kind: "acc", Desc: "Phụ kiện: dễ thương", Weight: 2},
	{ID: "khan", Name: "Khăn quàng", Emoji: "🧣", Kind: "acc", Desc: "Phụ kiện: ấm áp", Weight: 2},
	{ID: "sach", Name: "Bí kíp võ công", Emoji: "📜", Kind: "card", Desc: "Chọn một kỹ năng để đổi sang kỹ năng khác, giữ nguyên cấp đã luyện", Weight: 3},
	{ID: "the_ten", Name: "Thẻ đổi tên", Emoji: "🏷️", Kind: "card", Desc: "Đặt lại tên cho một con thú, giữ nguyên hình dáng và chỉ số", Weight: 3},
	{ID: "the_quen", Name: "Thẻ tẩy kỹ năng", Emoji: "🫗", Kind: "card", Desc: "Xoá cấp đã nâng của mọi kỹ năng, hoàn lại một nửa số ngọc đã dùng", Weight: 2, Rarity: 1},
	{ID: "phieu_hoan_diem", Name: "Phiếu hoàn điểm", Emoji: "♻️", Kind: "card", Desc: "Trả toàn bộ điểm Công/Thủ/Sinh lực về quỹ để cộng lại", Rarity: 3},
	{ID: "nuoc_tang_luc", Name: "Nước tăng lực", Emoji: "⚡", Kind: "energy", Desc: "Hồi 5 năng lượng phiêu lưu", Weight: 5},
	{ID: "tra_sam", Name: "Trà sâm", Emoji: "🍵", Kind: "energy", Desc: "Hồi 12 năng lượng phiêu lưu", Weight: 2, Rarity: 1},
	{ID: "ruou_than", Name: "Rượu thần", Emoji: "🍶", Kind: "energy", Desc: "Hồi đầy năng lượng phiêu lưu", Rarity: 2},
	{ID: "the_nang_luong", Name: "Thẻ nguyên khí", Emoji: "🔋", Kind: "card", Desc: "Tăng vĩnh viễn 5 năng lượng tối đa (tối đa +30)", Rarity: 3},
	{ID: "tinh_the_tiem_nang", Name: "Tinh thể tiềm năng", Emoji: "🔮", Kind: "card", Desc: "Cực hiếm: nhận ngẫu nhiên 1 hoặc 2 điểm tiềm năng vĩnh viễn", Rarity: 3},
	{ID: "banhtet", Name: "Bánh tét", Emoji: "🍥", Kind: "food", Desc: "No lâu, ăn Tết", Hunger: 45, Happy: 8, Weight: 6},
	{ID: "chegiai", Name: "Chè giải nhiệt", Emoji: "🍧", Kind: "food", Desc: "Mát ruột, hết cay", Hunger: 12, Happy: 12, Effect: "ngon", Minutes: 10, Weight: 6},
	{ID: "nuocmia", Name: "Nước mía", Emoji: "🥤", Kind: "food", Desc: "Ngọt mát, vui vẻ", Hunger: 10, Happy: 14, Weight: 7},
	{ID: "chaogiai", Name: "Cháo giải cảm", Emoji: "🥣", Kind: "med", Desc: "Hồi 30 sức khỏe, dịu bụng", Health: 30, Happy: 5, Weight: 5},
	{ID: "tinhdau", Name: "Tinh dầu tràm", Emoji: "🧴", Kind: "med", Desc: "Hồi 22 sức khỏe và xoá hiệu ứng hôi", Health: 22, Weight: 5},
	{ID: "sammy", Name: "Sâm Ngọc Linh", Emoji: "🌱", Kind: "med", Desc: "Hồi 55 sức khỏe và 20 vui", Health: 55, Happy: 20, Weight: 2, Rarity: 1},
	{ID: "mu_bao", Name: "Mũ bảo hiểm", Emoji: "🪖", Kind: "acc", Desc: "Phụ kiện: an toàn là bạn", Weight: 2},
	{ID: "vongtay", Name: "Vòng tay cỏ", Emoji: "🌾", Kind: "acc", Desc: "Phụ kiện: tết từ cỏ may", Weight: 2},
}

type actionDef struct {
	Happy, Health, Hunger float64
	Cooldown              int64 // seconds before a kind action gives stats again
	Effect                string
	Minutes               int
	Msg                   string
}

// actions are the right-click interactions. Kind ones only pay off after their cooldown so
// spamming does nothing; rough ones always land.
var actions = map[string]actionDef{
	"vuot": {Happy: 8, Cooldown: 20, Msg: "Rừ rừ… thích quá!"},
	"gai":  {Happy: 10, Cooldown: 30, Msg: "Gãi đúng chỗ ngứa!"},
	"cu":   {Happy: 12, Hunger: -1, Cooldown: 30, Effect: "cuoi", Minutes: 1, Msg: "Khúc khích… nhột quá!"},
	"om":   {Happy: 15, Cooldown: 60, Effect: "ngon", Minutes: 5, Msg: "Ấm áp quá ♥"},
	"choi": {Happy: 18, Hunger: -6, Cooldown: 120, Msg: "Bắt bóng nè!"},
	"tam":  {Happy: 3, Cooldown: 120, Msg: "Thơm tho sạch sẽ!"},
	"ngu":  {Health: 5, Cooldown: 600, Effect: "ngu", Minutes: 30, Msg: "Zzz…"},
	"bung": {Happy: -4, Effect: "dau", Minutes: 1, Msg: "Ui! Búng trán đau đó!"},
	"dam":  {Happy: -15, Health: -6, Effect: "dau", Minutes: 10, Msg: "Á! Sao đánh tớ?!"},
}

var itemByID = func() map[string]*itemDef {
	catalog = append(catalog, huntItems...)  // hunt-only loot lives in hunt.go
	catalog = append(catalog, gemItems()...) // gems and shards live in gem.go
	m := make(map[string]*itemDef, len(catalog))
	for i := range catalog {
		m[catalog[i].ID] = &catalog[i]
	}
	return m
}()

// partLimits mirrors PARTS/COLORS in webapp/src/sprite.cjs: number of choices per part.
var partLimits = map[string]int{
	"head": 6, "body": 6, "legs": 6, "tail": 6, "top": 8, "eyes": 7, "mouth": 6, "pattern": 5,
	"c1": 12, "c2": 12, "c3": 12,
}

type pet struct {
	Build       buildDef            `json:"build"`
	BuildBonus  int                 `json:"build_bonus,omitempty"`
	Energy      int                 `json:"energy"`
	EnergyAt    int64               `json:"energy_at,omitempty"`
	EnergyMax   int                 `json:"energy_max,omitempty"`
	Sockets     map[string][]string `json:"sockets,omitempty"`
	GearLevel   map[string]int      `json:"gear_level,omitempty"`
	Campaign    map[string][3]int   `json:"campaign,omitempty"`
	Name        string              `json:"name"`
	Parts       map[string]int      `json:"parts"`
	Born        int64               `json:"born"`
	Hunger      float64             `json:"hunger"`
	Happy       float64             `json:"happy"`
	Health      float64             `json:"health"`
	Sick        bool                `json:"sick"`
	LowHours    float64             `json:"low_hours,omitempty"`
	Effect      string              `json:"effect,omitempty"`
	EffectUntil int64               `json:"effect_until,omitempty"`
	LastTick    int64               `json:"last_tick"`
	Dead        bool                `json:"dead"`
	Cause       string              `json:"cause,omitempty"`
	DiedAt      int64               `json:"died_at,omitempty"`
	Harm        string              `json:"harm,omitempty"`
	Likes       string              `json:"likes,omitempty"` // hidden from the client until discovered
	Hates       string              `json:"hates,omitempty"`
	Known       map[string]string   `json:"known,omitempty"`
	Fed         int                 `json:"fed"`
	Acc         string              `json:"acc,omitempty"`
	FrozenUntil int64               `json:"frozen_until,omitempty"`
	CareUntil   int64               `json:"care_until,omitempty"`
	Touch       map[string]int64    `json:"touch,omitempty"`
	Skills      []string            `json:"skills,omitempty"`
	Wins        int                 `json:"wins,omitempty"`
	Losses      int                 `json:"losses,omitempty"`
	ArenaPoints int                 `json:"arena_points,omitempty"`
	Lv          int                 `json:"lv,omitempty"`
	XP          int                 `json:"xp,omitempty"`
	Gear        map[string]string   `json:"gear,omitempty"` // slot -> item id
	Title       string              `json:"title,omitempty"`
	SkillLv     map[string]int      `json:"skill_lv,omitempty"`
	Hunts       int                 `json:"hunts,omitempty"`
	Punish      bool                `json:"punish,omitempty"` // owner is marked cruel: harsher decay
	Wild        int                 `json:"wild,omitempty"`   // wild pets beaten on the world map
	LastWild    int64               `json:"last_wild,omitempty"`
	Gold        int                 `json:"gold,omitempty"` // golden finds, for the medal
	LastHunt    int64               `json:"last_hunt,omitempty"`
	WarnHungry  bool                `json:"warn_hungry,omitempty"`
	WarnSick    bool                `json:"warn_sick,omitempty"`
}

type grave struct {
	Snapshot *pet           `json:"snapshot,omitempty"`
	Name     string         `json:"name"`
	Parts    map[string]int `json:"parts"`
	Born     int64          `json:"born"`
	Died     int64          `json:"died"`
	Cause    string         `json:"cause"`
}

type userState struct {
	Pet               *pet                   `json:"pet,omitempty"` // 1.0/1.1 data, moved into Pets by normalize
	Pets              []*pet                 `json:"pets,omitempty"`
	Slots             int                    `json:"slots,omitempty"`
	Gift              int                    `json:"gift,omitempty"`
	Inv               map[string]int         `json:"inv"`
	Enhanced          map[string]map[int]int `json:"enhanced,omitempty"`
	Graves            []grave                `json:"graves,omitempty"`
	LastDrop          int64                  `json:"last_drop,omitempty"`
	DropDay           string                 `json:"drop_day,omitempty"`
	Drops             int                    `json:"drops,omitempty"`
	WildDay           string                 `json:"wild_day,omitempty"`
	WildToday         int                    `json:"wild_today,omitempty"`
	WildAt            int64                  `json:"wild_at,omitempty"`
	ReplayDay         string                 `json:"replay_day,omitempty"`
	ReplayToday       int                    `json:"replay_today,omitempty"`
	Air               int                    `json:"air,omitempty"` // bags on their way by plane, claimed by clicking them
	In                []string               `json:"in,omitempty"`
	Out               []string               `json:"out,omitempty"`
	Muted             bool                   `json:"muted,omitempty"`
	DIn               []string               `json:"din,omitempty"` // duel challenges received / sent
	DOut              []string               `json:"dout,omitempty"`
	Duels             []duelResult           `json:"duels,omitempty"`
	DuelDay           string                 `json:"duel_day,omitempty"`
	DuelWins          int                    `json:"duel_wins,omitempty"`
	Coin              int                    `json:"coin,omitempty"` // 🪙 earned by gameplay activity
	CoinDay           string                 `json:"coin_day,omitempty"`
	CoinToday         int                    `json:"coin_today,omitempty"`
	BossCoinDay       string                 `json:"boss_coin_day,omitempty"`
	BossCoinToday     int                    `json:"boss_coin_today,omitempty"`
	LastCoin          int64                  `json:"last_coin,omitempty"`
	ActivityAt        map[string]int64       `json:"activity_at,omitempty"`
	ShopDay           string                 `json:"shop_day,omitempty"`
	Bought            map[string]int         `json:"bought,omitempty"` // bought today, caps the daily stock
	Sell              []string               `json:"sell,omitempty"`   // market listing ids this user has up
	Buy               []string               `json:"buy,omitempty"`    // prepaid buy-order ids this user has up
	Cruel             int64                  `json:"cruel,omitempty"`  // the cruelty mark runs out at this time
	Kills             int                    `json:"kills,omitempty"`
	Counts            map[string]int         `json:"counts,omitempty"`
	QuestDay          string                 `json:"quest_day,omitempty"`
	QuestBase         map[string]int         `json:"quest_base,omitempty"`
	QuestDone         map[string]bool        `json:"quest_done,omitempty"`
	CheckDay          string                 `json:"check_day,omitempty"` // last claimed attendance date
	Streak            int                    `json:"streak,omitempty"`
	BestStreak        int                    `json:"best_streak,omitempty"`
	WelcomeGift       bool                   `json:"welcome_gift_3000,omitempty"`
	TowerFloor        int                    `json:"tower_floor,omitempty"` // highest floor cleared, kept for the account
	TowerToday        int                    `json:"tower_today,omitempty"`
	TowerDay          string                 `json:"tower_day,omitempty"`
	SeasonBoostBPS    int                    `json:"season_boost_bps,omitempty"`
	SeasonBoostUntil  int64                  `json:"season_boost_until,omitempty"`
	SeasonBoostSource string                 `json:"season_boost_source,omitempty"`
}

type challenge struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	Pet     int64  `json:"pet"` // challenger's pet, by birth time
	Created int64  `json:"created"`
}

type offer struct {
	ID      string         `json:"id"`
	From    string         `json:"from"`
	To      string         `json:"to"`
	Give    map[string]int `json:"give"`
	Want    map[string]int `json:"want"`
	Created int64          `json:"created"`
}

func newUserState() *userState {
	st := &userState{Inv: map[string]int{"tui": startBags}}
	st.normalize()
	return st
}

// normalize upgrades stored state: the single pet becomes a list, and everyone gets the 1.2
// welcome cards once.
func (st *userState) normalize() {
	if st.Inv == nil {
		st.Inv = map[string]int{}
	}
	if st.Enhanced == nil {
		st.Enhanced = map[string]map[int]int{}
	}
	if st.Pet != nil {
		st.Pets = append([]*pet{st.Pet}, st.Pets...)
		st.Pet = nil
	}
	if st.Slots < 1 {
		st.Slots = 1
	}
	if st.Gift < 1 {
		st.Gift = 1
		st.add("the_skin", 1)
		st.add("the_slot", 1)
	}
	st.retireMissingItems()
	for _, p := range st.Pets {
		if p.Gear == nil {
			p.Gear = map[string]string{}
		}
		if p.GearLevel == nil {
			p.GearLevel = map[string]int{}
		}
		if p.Sockets == nil {
			p.Sockets = map[string][]string{}
		}
		if p.Known == nil {
			p.Known = map[string]string{}
		}
		if len(p.Skills) == 0 {
			p.Skills = rollSkills(seeded(p.Born)) // pets from before 1.2 get stable skills
		}
		if p.Lv < 1 {
			p.Lv = 1 // pets from before 1.5 start hunting at level 1
		}
		if p.ArenaPoints < 1 {
			p.ArenaPoints = legacyArenaPoints(p.Wins, p.Losses)
		}
	}
}

// retired maps ids that no longer exist onto their closest live equivalent. 1.9.0 removed
// the charm slot and renamed the ring tiers but left the items sitting in people's bags,
// where nothing could draw them: the bag only renders what the catalog describes, so the
// goods were invisible and unusable rather than lost. Converting on load gives them back.
var retired = map[string]string{
	"buamay": "nhandong", // the charm slot went away; its stat line matches the copper ring
	"nhan":   "nhanbac",  // the plain ring became a tier ladder in 1.9.0
}

// retireMissingItems converts obsolete ids, including anything equipped into a slot that no
// longer exists, so a pet never wears a piece the game cannot describe.
func (st *userState) retireMissingItems() {
	for old, now := range retired {
		if n := st.Inv[old]; n > 0 {
			delete(st.Inv, old)
			st.add(now, n)
		}
	}
	for _, p := range st.Pets {
		for slot, id := range p.Gear {
			if now, ok := retired[id]; ok {
				id = now
				p.Gear[slot] = now
			}
			g, known := gearByID[id]
			if !known {
				delete(p.Gear, slot)
				continue
			}
			// a renamed piece can belong to a different slot than the one it sat in
			if g.Slot != slot {
				delete(p.Gear, slot)
				if p.Gear[g.Slot] == "" {
					p.Gear[g.Slot] = id
				} else {
					st.add(id, 1)
				}
			}
		}
	}
}

// seeded is a tiny deterministic generator for upgrading old data.
func seeded(seed int64) rng {
	x := uint64(seed)*2862933555777941757 + 3037000493
	return func() float64 {
		x = x*6364136223846793005 + 1442695040888963407
		return float64(x>>11) / (1 << 53)
	}
}

func (st *userState) petAt(i int) (*pet, error) {
	if i < 0 || i >= len(st.Pets) {
		return nil, errors.New("không có thú này")
	}
	return st.Pets[i], nil
}

func (st *userState) tickAll(now int64) {
	st.markPets(now)
	for _, p := range st.Pets {
		p.tick(now)
	}
}

// markPets copies the cruelty mark onto each pet so tick needs no extra argument.
func (st *userState) markPets(now int64) {
	cruel := st.isCruel(now)
	for _, p := range st.Pets {
		p.Punish = cruel
	}
}

func clamp(v float64) float64 { return math.Max(0, math.Min(100, v)) }

func (p *pet) hurt(amount float64, cause string) {
	p.Health -= amount
	if cause != "" {
		p.Harm = cause
	}
	if p.Health <= 0 {
		p.Health, p.Dead, p.Cause = 0, true, p.Harm
	}
}

// tick advances the pet to now in 15-minute steps. Returns true when the pet died in this call.
func (p *pet) tick(now int64) bool {
	if p.Dead || now <= p.LastTick {
		return false
	}
	protectedUntil := max(p.FrozenUntil, p.CareUntil)
	if protectedUntil > p.LastTick {
		// Vacation cards and paid hospital care both stop unattended decay.
		p.LastTick = min(now, protectedUntil)
		if now <= protectedUntil {
			if p.EffectUntil <= now {
				p.Effect, p.EffectUntil = "", 0
			}
			return false
		}
	}
	start := p.LastTick
	hours := math.Min(float64(now-start)/3.6e6, 24*30)
	elapsed := 0.0
	for hours > 0 && !p.Dead {
		dt := math.Min(hours, 0.25)
		hours -= dt
		elapsed += dt
		// a cruel owner's pets go hungry and glum faster
		hRate, jRate := hungerPerHour, happyPerHour
		if p.Punish {
			hRate, jRate = hRate*cruelHunger, jRate*cruelHappy
		}
		p.Hunger = math.Max(0, p.Hunger-hRate*dt)
		p.Happy = math.Max(0, p.Happy-jRate*dt)
		if p.Hunger < 15 {
			p.LowHours += dt
			if p.LowHours >= sickAfterLow && !p.Sick {
				p.Sick, p.Harm = true, "cảm vì đói lâu ngày"
			}
		} else {
			p.LowHours = 0
		}
		switch {
		case p.Hunger <= 0:
			p.hurt((starveDamage+boolf(p.Sick)*sickDamage)*dt, "chết đói")
		case p.Sick:
			p.hurt(sickDamage*dt, "bệnh nặng không được chữa")
		case p.Happy <= 0:
			p.hurt(sadDamage*dt, "buồn quá không ai chơi cùng")
		case p.Hunger > 50 && p.Happy > 30:
			p.Health = math.Min(100, p.Health+regenPerHour*dt)
		}
	}
	p.LastTick = now
	if p.EffectUntil <= now {
		p.Effect, p.EffectUntil = "", 0
	}
	if p.Dead {
		p.DiedAt = start + int64(elapsed*3.6e6)
		return true
	}
	return false
}

func boolf(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func (p *pet) setEffect(effect string, minutes int, now int64) {
	p.Effect, p.EffectUntil = effect, now+int64(minutes)*60000
}

func validateParts(in map[string]int) (map[string]int, error) {
	out := make(map[string]int, len(partLimits))
	for k, v := range in {
		limit, ok := partLimits[k]
		if !ok || v < 0 || v >= limit {
			return nil, fmt.Errorf("bộ phận không hợp lệ: %s", k)
		}
		out[k] = v
	}
	for k := range partLimits {
		out[k] += 0
	}
	return out, nil
}

func foodIDs() []string {
	var ids []string
	for _, it := range catalog {
		if it.Kind == "food" {
			ids = append(ids, it.ID)
		}
	}
	return ids
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxNameRunes || strings.ContainsAny(name, "\r\n") {
		return "", fmt.Errorf("tên thú dài 1–%d ký tự", maxNameRunes)
	}
	return name, nil
}

// recoverPetItems returns every item attached to a pet. Burial is irreversible, so gear,
// socketed gems and the cosmetic accessory must cross back into the inventory first.
func recoverPetItems(st *userState, p *pet) int {
	recovered := 0
	for slot, id := range p.Gear {
		if _, ok := gearByID[id]; ok {
			st.addGear(id, p.GearLevel[slot], 1)
			recovered++
		}
		delete(p.Gear, slot)
		delete(p.GearLevel, slot)
	}
	for slot, gems := range p.Sockets {
		for _, id := range gems {
			if _, _, ok := parseGem(id); ok {
				st.add(id, 1)
				recovered++
			}
		}
		delete(p.Sockets, slot)
	}
	if p.Acc != "" {
		if itemByID[p.Acc] != nil {
			st.add(p.Acc, 1)
			recovered++
		}
		p.Acc = ""
	}
	return recovered
}

// bury moves dead pet i to the graveyard and frees its slot.
func bury(st *userState, i int) error {
	p, err := st.petAt(i)
	if err != nil {
		return err
	}
	if !p.Dead {
		return errors.New(p.Name + " vẫn còn sống")
	}
	snapshot, err := copyPet(p)
	if err != nil {
		return err
	}
	recoverPetItems(st, p)
	st.Graves = append([]grave{{Snapshot: snapshot, Name: p.Name, Parts: p.Parts, Born: p.Born, Died: p.DiedAt, Cause: p.Cause}}, st.Graves...)
	if len(st.Graves) > maxGraves {
		st.Graves = st.Graves[:maxGraves]
	}
	st.Pets = append(st.Pets[:i], st.Pets[i+1:]...)
	return nil
}

// createPet hatches a new pet into a free slot. When all slots are taken, a dead pet is buried first.
func createPet(st *userState, name string, parts map[string]int, now int64, rnd rng) error {
	if st.isCruel(now) {
		return fmt.Errorf("bạn đang mang nhãn Kẻ tàn bạo, còn %d ngày nữa mới được nuôi thú mới", st.cruelLeft(now))
	}
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	valid, err := validateParts(parts)
	if err != nil {
		return err
	}
	if len(st.Pets) >= st.Slots {
		dead := -1
		for i, p := range st.Pets {
			if p.Dead {
				dead = i
				break
			}
		}
		if dead < 0 {
			return errors.New("hết chuồng rồi. Dùng 🏠 Thẻ thêm chuồng để nuôi thêm")
		}
		if err := bury(st, dead); err != nil {
			return err
		}
	}
	foods := foodIDs()
	likes := foods[int(rnd()*float64(len(foods)))%len(foods)]
	hates := foods[int(rnd()*float64(len(foods)))%len(foods)]
	if hates == likes {
		hates = ""
	}
	st.Pets = append(st.Pets, &pet{Name: name, Parts: valid, Born: now, LastTick: now, Hunger: 70, Happy: 70, Health: 100,
		Likes: likes, Hates: hates, Known: map[string]string{}, Skills: rollSkills(rnd), Lv: 1, ArenaPoints: arenaStartPoints})
	return nil
}

// reskin spends a transformation card to change pet i's look and name; stats stay.
func reskin(st *userState, i int, name string, parts map[string]int) error {
	p, err := st.petAt(i)
	if err != nil {
		return err
	}
	if st.Inv["the_skin"] <= 0 {
		return errors.New("bạn không có 🎴 Thẻ biến hình")
	}
	if p.Dead {
		return errors.New(p.Name + " đã chết rồi")
	}
	name, err = cleanName(name)
	if err != nil {
		return err
	}
	valid, err := validateParts(parts)
	if err != nil {
		return err
	}
	st.add("the_skin", -1)
	p.Name, p.Parts = name, valid
	return nil
}

// rename spends a name card: the pet keeps its looks, stats and record, only the name changes.
func rename(st *userState, i int, name string) (string, error) {
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if st.Inv["the_ten"] <= 0 {
		return "", errors.New("bạn không có 🏷️ Thẻ đổi tên")
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	name, err = cleanName(name)
	if err != nil {
		return "", err
	}
	if name == p.Name {
		return "", errors.New("tên mới trùng tên cũ")
	}
	old := p.Name
	st.add("the_ten", -1)
	p.Name = name
	return fmt.Sprintf("🏷️ %s nay tên là %s!", old, name), nil
}

// unequip returns pet i's accessory to the inventory.
func unequip(st *userState, i int) error {
	p, err := st.petAt(i)
	if err != nil {
		return err
	}
	if p.Acc == "" {
		return errors.New("thú không đeo gì")
	}
	st.add(p.Acc, 1)
	p.Acc = ""
	return nil
}

// interact runs a right-click action on living pet i.
func interact(st *userState, i int, action string, now int64) (string, error) {
	def, ok := actions[action]
	if !ok {
		return "", errors.New("thao tác không hợp lệ")
	}
	p, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	if p.Touch == nil {
		p.Touch = map[string]int64{}
	}
	if def.Happy >= 0 {
		switch {
		case p.Effect == "dau" && action != "om":
			return "Hứ! Vừa bị đánh xong, ôm tớ trước đã! 😤", nil
		case p.Sick && (action == "cu" || action == "choi"):
			return "Mệt lắm… không chơi nổi đâu.", nil
		case action == "ngu" && (p.Effect == "cay" || p.Effect == "hyper"):
			return "Tăng động quá, không ngủ được!", nil
		case p.Touch[action] > 0 && now-p.Touch[action] < def.Cooldown*1000:
			return def.Msg + " (từ từ thôi)", nil
		}
	}
	p.Touch[action] = max(now, 1)
	p.Happy = clamp(p.Happy + def.Happy)
	p.Hunger = clamp(p.Hunger + def.Hunger)
	msg := def.Msg
	switch {
	case action == "tam" && p.Effect == "hoi":
		p.Effect, p.EffectUntil = "", 0
		msg = "Hết hôi rồi! Thơm phức!"
	case action == "om" && p.Effect == "dau":
		p.Effect, p.EffectUntil = "", 0
		def.Effect = ""
		msg = "Thôi được, tha cho đó… ♥"
	case action == "dam" && p.Sick:
		def.Health *= 2
	}
	if def.Health > 0 {
		p.Health = clamp(p.Health + def.Health)
	} else if def.Health < 0 {
		p.hurt(-def.Health, "bị đánh đến chết")
	}
	if p.Dead {
		p.DiedAt = now
		return "💀 " + p.Name + " đã chết vì " + p.Cause, nil
	}
	if def.Effect != "" {
		p.setEffect(def.Effect, def.Minutes, now)
	}
	return msg, nil
}

func rollItem(rnd rng) string {
	total := 0
	for _, it := range catalog {
		total += it.Weight
	}
	n := rnd() * float64(total)
	for _, it := range catalog {
		n -= float64(it.Weight)
		if it.Weight > 0 && n < 0 {
			return it.ID
		}
	}
	return "com"
}

func (st *userState) add(id string, n int) {
	st.Inv[id] += n
	if st.Inv[id] <= 0 {
		delete(st.Inv, id)
	}
}

// useItem opens a bag, uses a card or feeds pet i. Nothing changes when it returns an error.
func useItem(st *userState, i int, id string, now int64, rnd rng) (string, error) {
	def := itemByID[id]
	if def == nil || st.Inv[id] <= 0 {
		return "", errors.New("bạn không có món này")
	}
	if def.Kind == "bag" {
		st.add(id, -1)
		return openBag(st, id, rnd), nil
	}
	switch id {
	case "the_skin":
		return "", errors.New("bấm 🎴 Biến hình ở thẻ Thú cưng để dùng thẻ này")
	case "the_ten":
		return "", errors.New("bấm 🏷️ Đổi tên ở thẻ Thú cưng để dùng thẻ này")
	case "the_slot":
		if st.Slots >= maxSlots {
			return "", fmt.Errorf("đã có tối đa %d chuồng", maxSlots)
		}
		st.add(id, -1)
		st.Slots++
		return fmt.Sprintf("Có thêm chuồng! Nuôi được %d con.", st.Slots), nil
	}
	p, err := st.petAt(i)
	if err != nil {
		return "", errors.New("chưa có thú nào, hãy ấp trứng trước")
	}
	if (def.Kind == "acc" || id == "the_nghi") && p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	if def.Kind == "acc" {
		st.add(id, -1)
		if p.Acc != "" {
			st.add(p.Acc, 1)
		}
		p.Acc = id
		return p.Name + " đeo " + def.Name + " trông thật bảnh!", nil
	}
	if id == "sach" {
		if p.Dead {
			return "", errors.New(p.Name + " đã chết rồi")
		}
		if len(p.Skills) < 2 {
			return "", errors.New(p.Name + " chưa có đủ 2 kỹ năng để học lại")
		}
		// The book changes the two starting moves. Level 10/20 unlocks extra slots, which
		// must stay: replacing the whole slice with rollSkills() silently deleted those moves.
		oldPair := strings.Join(p.Skills[:2], ",")
		extra := append([]string(nil), p.Skills[2:]...)
		next := append([]string(nil), p.Skills[:2]...)
		for tries := 0; tries < 20; tries++ {
			pair := rollSkills(rnd)
			clash := false
			for _, id := range pair {
				for _, kept := range extra {
					if id == kept {
						clash = true
					}
				}
			}
			if !clash && strings.Join(pair, ",") != oldPair {
				next = pair
				break
			}
		}
		if strings.Join(next, ",") == oldPair {
			return "", errors.New("chưa tìm được bộ kỹ năng khác, thử lại")
		}

		// Training belongs to the two slots, not to an obsolete skill id. Transfer each
		// rank to its replacement and preserve ranks of every unlocked extra move.
		ranks := map[string]int{}
		for i, newID := range next {
			if lv := p.skillLv(p.Skills[i]); lv > 1 {
				ranks[newID] = lv
			}
		}
		for _, kept := range extra {
			if lv := p.skillLv(kept); lv > 1 {
				ranks[kept] = lv
			}
		}
		p.Skills = append(next, extra...)
		p.SkillLv = ranks
		st.add(id, -1)
		return fmt.Sprintf("%s học được %s %s và %s %s; giữ nguyên %d ô kỹ năng và cấp đã luyện!", p.Name,
			skillByID[next[0]].Emoji, skillByID[next[0]].Name, skillByID[next[1]].Emoji, skillByID[next[1]].Name, len(p.Skills)), nil
	}
	if id == "the_quen" {
		if p.Dead {
			return "", errors.New(p.Name + " đã chết rồi")
		}
		spent := 0
		for _, lv := range p.SkillLv {
			spent += max(lv-1, 0)
		}
		if spent == 0 {
			return "", errors.New(p.Name + " chưa nâng cấp kỹ năng nào")
		}
		st.add(id, -1)
		p.SkillLv = map[string]int{}
		back := spent / 2
		st.add("ngoc", back)
		return fmt.Sprintf("%s quên hết kỹ năng đã luyện, nhận lại %d 💠 ngọc kỹ năng.", p.Name, back), nil
	}
	if id == "phieu_hoan_diem" {
		spent := p.Build.Atk + p.Build.Def + p.Build.HP
		if spent == 0 {
			return "", errors.New(p.Name + " chưa cộng điểm nào")
		}
		st.add(id, -1)
		p.Build = buildDef{}
		return fmt.Sprintf("♻️ Đã hoàn %d điểm tiềm năng của %s về quỹ.", spent, p.Name), nil
	}
	if id == "tinh_the_tiem_nang" {
		if p.Dead {
			return "", errors.New(p.Name + " đã chết rồi")
		}
		st.add(id, -1)
		gain := 1
		if rnd() < 0.2 {
			gain = 2
		}
		p.BuildBonus += gain
		return fmt.Sprintf("🔮 %s nhận thêm %d điểm tiềm năng vĩnh viễn!", p.Name, gain), nil
	}
	if def.Kind == "energy" {
		amount := map[string]int{"nuoc_tang_luc": 5, "tra_sam": 12, "ruou_than": p.maxEnergy()}[id]
		return drinkEnergy(st, i, id, amount, now)
	}
	if id == "the_nang_luong" {
		return growEnergy(st, i, id, 5, now)
	}
	if id == "the_nghi" {
		st.add(id, -1)
		p.FrozenUntil = max(p.FrozenUntil, now) + vacation.Milliseconds()
		return p.Name + " đi nghỉ phép 3 ngày, không lo đói! 🏖️", nil
	}
	if def.Kind == "revive" {
		if !p.Dead {
			return "", errors.New("thú còn sống, để dành lông phượng hoàng nhé")
		}
		st.add(id, -1)
		p.Dead, p.Cause, p.DiedAt, p.Sick, p.LowHours = false, "", 0, false, 0
		p.Health, p.Hunger, p.Happy, p.LastTick = 50, 50, 60, now
		p.setEffect("ngon", 10, now)
		return p.Name + " sống lại rồi!", nil
	}
	if p.Dead {
		return "", errors.New(p.Name + " đã chết rồi")
	}
	if def.Kind == "food" && p.Hunger >= 95 {
		return "", errors.New(p.Name + " no căng bụng rồi, không ăn nữa")
	}
	st.add(id, -1)
	if def.Kind == "food" {
		st.bump("feed", 1)
	}
	msg := feed(p, def, now, rnd)
	if p.Dead {
		p.DiedAt = now
		return "💀 " + p.Name + " đã chết vì " + p.Cause, nil
	}
	return msg, nil
}

// healFor scales a healing item down while the owner wears the cruelty mark.
func healFor(p *pet, health int) float64 {
	if p.Punish && health > 0 {
		return float64(health) * cruelHeal
	}
	return float64(health)
}

func feed(p *pet, def *itemDef, now int64, rnd rng) string {
	if def.Kind == "med" {
		p.Happy = clamp(p.Happy + float64(def.Happy))
		if def.ID == "thuoc" {
			if !p.Sick {
				p.hurt(8, "uống thuốc bừa bãi")
				return "Đắng quá! Có bệnh đâu mà uống thuốc…"
			}
			p.Sick, p.LowHours = false, 0
			p.Health = clamp(p.Health + healFor(p, def.Health))
			return "Khỏi bệnh rồi!"
		}
		was := p.Health
		p.Health = clamp(p.Health + healFor(p, def.Health))
		switch def.ID {
		case "tinhdau":
			if p.Effect == "hoi" {
				p.Effect, p.EffectUntil = "", 0
			}
		case "canhtien":
			p.Sick, p.LowHours = false, 0
			p.Effect, p.EffectUntil = "ngon", now+20*time.Minute.Milliseconds()
			return p.Name + " húp cạn bát canh, khỏe như mới!"
		case "nuocthanh":
			p.Effect, p.EffectUntil = "", 0
			p.Sick, p.LowHours = false, 0
			return "Nước thánh gột sạch mọi thứ xấu!"
		}
		if p.Punish && p.Health-was < float64(def.Health) {
			return fmt.Sprintf("Chỉ hồi được %.0f sức khỏe — thú không còn tin bạn nữa.", p.Health-was)
		}
		return "Khỏe re!"
	}

	p.Fed++
	hunger, happy := float64(def.Hunger), float64(def.Happy)
	effect, minutes := def.Effect, def.Minutes
	var notes []string
	switch def.ID {
	case p.Likes:
		happy += 25
		effect, minutes = "ngon", 20
		p.Known[def.ID] = "like"
		notes = append(notes, "Món khoái khẩu!")
	case p.Hates:
		hunger /= 2
		happy -= 20
		effect = ""
		p.Known[def.ID] = "hate"
		notes = append(notes, "Ghét món này, nhổ ra một nửa!")
	}
	if def.Effect == "cay" && p.Effect == "cay" && now < p.EffectUntil {
		p.hurt(12, "ăn cay quá độ")
		notes = append(notes, "Cay xé lưỡi!!")
	}
	if def.Health < 0 {
		p.hurt(float64(-def.Health), "ăn cay quá độ")
	}
	if def.Sick > 0 && rnd() < def.Sick {
		p.Sick, p.Harm = true, "đau bụng vì "+strings.ToLower(def.Name)
		notes = append(notes, "Ăn xong đau bụng rồi…")
	}
	if def.Poison > 0 {
		if rnd() < def.Poison {
			p.Sick = true
			p.hurt(45, "ngộ độc "+strings.ToLower(def.Name))
			effect, minutes = "ngodoc", 30
			notes = append(notes, "Ngộ độc rồi! Mau cho uống thuốc!")
		} else {
			happy += 15
			effect, minutes = "hyper", 20
			notes = append(notes, "Nấm lạ… phê quá!")
		}
	}
	if total := p.Hunger + hunger; total > 100 {
		over := total - 100
		p.Hunger = 100
		if over >= 15 {
			p.hurt(math.Min(over*0.4, 12), "bội thực")
			notes = append(notes, "No quá, bội thực rồi!")
		}
	} else {
		p.Hunger = total
	}
	p.Happy = clamp(p.Happy + happy)
	if effect != "" {
		p.setEffect(effect, minutes, now)
	}
	if len(notes) > 0 {
		return strings.Join(notes, " ")
	}
	switch effect {
	case "ngon":
		return "Ngon quá!"
	case "cay":
		return "Cay quá! Chạy vòng vòng!"
	case "hoi":
		return "Ọe… hôi quá!"
	case "hyper":
		return "Tỉnh như sáo!"
	}
	return "Măm măm!"
}

// tryDrop sends a food bag after successful activity: at most one every 30 minutes, six a day.
func (st *userState) tryDrop(now int64, rnd rng) bool {
	day := time.UnixMilli(now).In(vnZone).Format("2006-01-02")
	if day != st.DropDay {
		st.DropDay, st.Drops = day, 0
	}
	if st.Drops >= dropsPerDay || now-st.LastDrop < dropCooldown.Milliseconds() || rnd() >= dropChance {
		return false
	}
	st.Drops++
	st.LastDrop = now
	if st.Air < maxAir {
		st.Air++
	} else {
		st.add(rollBag(rnd), 1)
	}
	return true
}

// claimAir moves one airdropped bag into the inventory.
func (st *userState) claimAir(rnd rng) error {
	if st.Air <= 0 {
		return errors.New("không có túi tiếp tế nào")
	}
	st.Air--
	st.add(rollBag(rnd), 1)
	return nil
}

// warnings returns the notifications due after a background tick and updates the warn flags.
func warnings(p *pet, wasDead bool) []string {
	if p == nil || wasDead {
		return nil
	}
	if p.Dead {
		return []string{fmt.Sprintf("💀 **%s** đã chết vì %s. Mở Thú ảo để chôn cất và nuôi con mới.", p.Name, p.Cause)}
	}
	var out []string
	if p.Hunger < 20 && !p.WarnHungry {
		p.WarnHungry = true
		out = append(out, fmt.Sprintf("🍙 **%s** đói lắm rồi (thanh no còn %.0f%%). Cho ăn ngay kẻo chết đói!", p.Name, p.Hunger))
	} else if p.Hunger >= 40 {
		p.WarnHungry = false
	}
	if p.Sick && !p.WarnSick {
		p.WarnSick = true
		out = append(out, fmt.Sprintf("🤒 **%s** bị bệnh, cần 💊 Thuốc cảm. Bệnh lâu sẽ chết đấy.", p.Name))
	} else if !p.Sick {
		p.WarnSick = false
	}
	return out
}

// ---- trading ----

func validateBundle(m map[string]int) error {
	if len(m) > 6 {
		return errors.New("tối đa 6 loại món mỗi lần")
	}
	for id, n := range m {
		if id == "tinhhon" {
			return errors.New("Tinh Hồn không thể giao dịch")
		}
		if itemByID[id] == nil || n < 1 || n > 99 {
			return errors.New("món giao dịch không hợp lệ")
		}
	}
	return nil
}

func (st *userState) has(m map[string]int) bool {
	for id, n := range m {
		if st.Inv[id] < n {
			return false
		}
	}
	return true
}

func (st *userState) move(m map[string]int, sign int) {
	for id, n := range m {
		st.add(id, sign*n)
	}
}

// settle completes an accepted offer: the sender's items were already taken into escrow.
func settle(sender, recipient *userState, o *offer) error {
	if !recipient.has(o.Want) {
		return errors.New("bạn không đủ món người kia muốn")
	}
	recipient.move(o.Want, -1)
	recipient.move(o.Give, 1)
	sender.move(o.Want, 1)
	return nil
}

func bundleText(m map[string]int) string {
	var parts []string
	for _, it := range catalog {
		if n := m[it.ID]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d× %s %s", n, it.Emoji, it.Name))
		}
	}
	return strings.Join(parts, ", ")
}

func removeID(ids []string, id string) []string {
	out := ids[:0]
	for _, x := range ids {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}

// ---- duels ----

type skillDef struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Emoji string  `json:"emoji"`
	Desc  string  `json:"desc"`
	Kind  string  `json:"kind"`
	Base  float64 `json:"base"`
}

var skills = []skillDef{
	{"lua", "Phun lửa", "🔥", "Gây 14 sát thương", "damage", 14},
	{"cuoc", "Liên hoàn cước", "🦶", "Đá 2–3 phát, mỗi phát 6", "multi", 6},
	{"hoi", "Hồi máu", "💚", "Hồi 14 máu", "heal", 14},
	{"khien", "Khiên sắt", "🛡️", "Giảm 50% đòn kế tiếp", "shield", 50},
	{"an", "Tàng hình", "👻", "Né đòn kế tiếp", "hide", 0},
	{"bom", "Bom hôi", "💨", "6 sát thương, đối thủ dễ đánh trượt lượt sau", "stink", 6},
	{"set", "Sấm sét", "⚡", "22 sát thương nhưng trượt một nửa", "risky", 22},
	{"hut", "Hút máu", "🧛", "10 sát thương, hồi lại một nửa", "drain", 10},
}

var skillByID = func() map[string]*skillDef {
	m := make(map[string]*skillDef, len(skills))
	for i := range skills {
		m[skills[i].ID] = &skills[i]
	}
	return m
}()

// skillValue is the authoritative combat effect at a rank. UI labels mirror these formulas;
// fight calls this function directly, so spending a skill stone always changes the real result.
func skillValue(id string, lv int) float64 {
	lv = min(max(lv, 1), maxSkillLv)
	sk := skillByID[id]
	if sk == nil {
		return 0
	}
	switch sk.Kind {
	case "shield":
		return math.Min(.9, .5+float64(lv-1)*.1)
	case "hide":
		return float64(lv-1) * .2
	default:
		return math.Round(sk.Base * (1 + float64(lv-1)*.25))
	}
}

const (
	duelTTL      = 24 * time.Hour
	maxDuelLog   = 10
	duelRewards  = 3  // rewarded wins per day, so alt accounts cannot farm bags
	duelCoin     = 25 // coins for a rewarded win
	maxDuelTurns = 30
	crushChance  = 0.08 // chance the loser takes a crushing 55-point blow
)

func rollSkills(rnd rng) []string {
	a := int(rnd()*float64(len(skills))) % len(skills)
	b := int(rnd()*float64(len(skills)-1)) % (len(skills) - 1)
	if b >= a {
		b++
	}
	return []string{skills[a].ID, skills[b].ID}
}

type duelResult struct {
	ID     string     `json:"id"`
	At     int64      `json:"at"`
	A      duelSide   `json:"a"`
	B      duelSide   `json:"b"`
	Winner int        `json:"winner"` // 0 = A, 1 = B
	Turns  []duelTurn `json:"turns"`
}

type duelSide struct {
	Lv      int               `json:"lv,omitempty"`
	Speed   float64           `json:"speed,omitempty"`
	Atk     float64           `json:"atk,omitempty"`
	Def     float64           `json:"def,omitempty"`
	Gear    map[string]string `json:"gear,omitempty"`
	SkillLv map[string]int    `json:"skill_lv,omitempty"`
	User    string            `json:"user"`
	Born    int64             `json:"born"`
	Name    string            `json:"name"`
	Parts   map[string]int    `json:"parts"`
	Acc     string            `json:"acc,omitempty"`
	Skills  []string          `json:"skills"`
	Points  int               `json:"points"`
	HP      float64           `json:"hp"`
	Sprite  int               `json:"sprite,omitempty"` // wild pets: 1-based index into the monster art, 0 for a player pet
}

type duelTurn struct {
	By     int        `json:"by"`
	Skill  string     `json:"skill,omitempty"` // empty for a basic attack
	Text   string     `json:"text"`
	HP     [2]float64 `json:"hp"`
	Sudden bool       `json:"sudden,omitempty"`
}

func (p *pet) fit() error {
	switch {
	case p.Dead:
		return errors.New(p.Name + " đã chết rồi")
	case p.Sick:
		return errors.New(p.Name + " đang bệnh, chữa khỏi rồi hãy tỷ thí")
	case p.Health < 30:
		return errors.New(p.Name + " yếu quá, cần hồi sức trước")
	}
	return nil
}

func sideOf(user string, p *pet) duelSide {
	g := p.gearOf()
	// The worn title is a separate source from gear: gearOf stays "what the pet is wearing",
	// so the paper doll and gear_totals keep meaning what they say.
	ti := titleOf(p)
	lv := max(p.Lv, 1)
	s := duelSide{User: user, Born: p.Born, Name: p.Name, Parts: p.Parts, Acc: p.Acc, Skills: p.Skills,
		Lv: lv, Speed: p.speedOf(), Atk: g.Atk + ti.Atk + float64(p.Build.Atk)*.8, Def: g.Def + ti.Def + float64(lv-1)*.15 + float64(p.Build.Def)*.6, Gear: p.Gear, SkillLv: map[string]int{}, Points: arenaPoints(p),
		HP: math.Round(40 + p.Health*0.6 + g.HP + ti.HP + float64(lv-1)*2 + float64(p.Build.HP)*4)}
	for _, id := range p.Skills {
		s.SkillLv[id] = p.skillLv(id)
	}
	return s
}

// fight simulates a friendly duel. Battle HP is separate from health, so nobody dies in a duel.
// pickSkill chooses a move like a sensible player: no second shield or hide while the first
// is still up, no heal at high HP, and no big attack into a hidden foe. "" means a plain bite.
func pickSkill(own []string, shielded, hidden, foeHidden bool, hp, maxHP float64, rnd rng) string {
	var ok []string
	for _, id := range own {
		switch {
		case id == "khien" && shielded, id == "an" && hidden, id == "hoi" && hp > maxHP*0.7:
		case foeHidden && id != "khien" && id != "an" && id != "hoi":
		default:
			ok = append(ok, id)
		}
	}
	if len(ok) == 0 || rnd() < 0.3 {
		return ""
	}
	return ok[int(rnd()*float64(len(ok)))%len(ok)]
}

func fight(a, b duelSide, rnd rng) duelResult {
	sides := [2]duelSide{a, b}
	hp := [2]float64{a.HP, b.HP}
	shield, hidden, stink := [2]bool{}, [2]bool{}, [2]bool{}
	shieldCut, hideStay := [2]float64{}, [2]float64{}
	var turns []duelTurn
	by := 0
	if rnd() < 0.5 {
		by = 1
	}
	// A faster pet occasionally acts twice. The chance is capped, so speed shortens fights
	// rather than locking the other side out of the match.
	next := func(cur int) int {
		if rnd() < extraTurnChance(sides[cur].Speed, sides[1-cur].Speed) {
			return cur
		}
		return 1 - cur
	}
	for t := 0; t < maxDuelTurns && hp[0] > 0 && hp[1] > 0; t++ {
		me, foe := sides[by].Name, sides[1-by].Name
		// gear and level shape every hit: the attacker's weapon and level add, the target's armour subtracts
		hit := func(dmg float64) string {
			dmg += sides[by].Atk + float64(max(sides[by].Lv, 1)-1)*0.4
			dmg = math.Max(1, dmg-sides[1-by].Def)
			switch {
			case stink[by] && rnd() < 0.4:
				stink[by] = false
				return "nhưng hôi quá, đánh trượt!"
			case hidden[1-by]:
				hidden[1-by] = rnd() < hideStay[1-by]
				if hidden[1-by] {
					return foe + " tàng hình né mất và vẫn đang ẩn!"
				}
				return foe + " tàng hình né mất!"
			case shield[1-by]:
				shield[1-by] = false
				dmg = math.Ceil(dmg * (1 - shieldCut[1-by]))
			}
			stink[by] = false
			if rnd() < 0.15 {
				dmg = math.Round(dmg * 1.5)
				hp[1-by] = math.Max(0, hp[1-by]-dmg)
				return fmt.Sprintf("CHÍ MẠNG! −%.0f", dmg)
			}
			hp[1-by] = math.Max(0, hp[1-by]-dmg)
			return fmt.Sprintf("−%.0f", dmg)
		}
		turn := duelTurn{By: by}
		if id := pickSkill(sides[by].Skills, shield[by], hidden[by], hidden[1-by], hp[by], sides[by].HP, rnd); id == "" {
			turn.Text = me + " cắn một phát: " + hit(math.Round(6+rnd()*5))
		} else {
			sk := skillByID[id]
			turn.Skill = id
			lv := sides[by].SkillLv[id]
			if lv < 1 {
				lv = 1
			}
			prefix := me + " dùng " + sk.Emoji + " " + sk.Name
			if lv > 1 {
				prefix += fmt.Sprintf(" Lv%d", lv)
			}
			prefix += ": "
			switch id {
			case "lua":
				turn.Text = prefix + hit(skillValue(id, lv))
			case "cuoc":
				n := 2 + int(rnd()*2)
				var parts []string
				for i := 0; i < n && hp[1-by] > 0; i++ {
					parts = append(parts, hit(skillValue(id, lv)))
				}
				turn.Text = prefix + strings.Join(parts, ", ")
			case "hoi":
				heal := skillValue(id, lv)
				hp[by] = math.Min(sides[by].HP, hp[by]+heal)
				stink[by] = false
				turn.Text = prefix + fmt.Sprintf("+%.0f máu", heal)
			case "khien":
				shield[by] = true
				shieldCut[by] = skillValue(id, lv)
				turn.Text = prefix + fmt.Sprintf("giảm %.0f%% đòn kế tiếp", shieldCut[by]*100)
			case "an":
				hidden[by] = true
				hideStay[by] = skillValue(id, lv)
				turn.Text = prefix + "biến mất…"
				if hideStay[by] > 0 {
					turn.Text += fmt.Sprintf(" (%.0f%% giữ ẩn sau khi né)", hideStay[by]*100)
				}
			case "bom":
				turn.Text = prefix + hit(skillValue(id, lv))
				stink[1-by] = true
			case "set":
				if rnd() < 0.5 {
					turn.Text = prefix + "trượt!"
				} else {
					turn.Text = prefix + hit(skillValue(id, lv))
				}
			case "hut":
				before := hp[1-by]
				turn.Text = prefix + hit(skillValue(id, lv))
				gain := math.Round((before - hp[1-by]) / 2)
				hp[by] = math.Min(sides[by].HP, hp[by]+gain)
				if gain > 0 {
					turn.Text += fmt.Sprintf(", hồi %.0f", gain)
				}
			}
		}
		turn.HP = hp
		turns = append(turns, turn)
		by = next(by)
	}
	if hp[0] > 0 && hp[1] > 0 {
		// Eight hits must knock out either pet. This keeps the replay bounded while ensuring
		// a winner is shown only after the loser's battle HP actually reaches zero.
		for hp[0] > 0 && hp[1] > 0 {
			foe := 1 - by
			minimum := math.Ceil(sides[foe].HP / 8)
			dmg := math.Max(minimum, math.Round((8+float64(len(turns)-maxDuelTurns)*2+sides[by].Atk)-sides[foe].Def*.35))
			hp[foe] = math.Max(0, hp[foe]-dmg)
			turns = append(turns, duelTurn{By: by, Text: fmt.Sprintf("Quyết đấu: −%.0f", dmg), HP: hp, Sudden: true})
			by = foe
		}
	}
	winner := 0
	if hp[0] <= 0 {
		winner = 1
	}
	return duelResult{A: a, B: b, Winner: winner, Turns: turns}
}

// petByBorn finds a pet by its birth time, which stays stable when other pets are buried.
func (st *userState) petByBorn(born int64) (int, *pet) {
	for i, p := range st.Pets {
		if p.Born == born {
			return i, p
		}
	}
	return -1, nil
}

// applyDuel records the result for one side. Duels are real fights: both pets get hurt, the loser
// badly, and a rare crushing blow can kill a pet. The winner earns a bag a few times a day.
// Returns a short note for the user.
func applyDuel(st *userState, res duelResult, side int, now int64, rnd rng) string {
	st.Duels = append([]duelResult{res}, st.Duels...)
	if len(st.Duels) > maxDuelLog {
		st.Duels = st.Duels[:maxDuelLog]
	}
	mine := res.A
	if side == 1 {
		mine = res.B
	}
	_, p := st.petByBorn(mine.Born)
	if p == nil || p.Dead {
		return ""
	}
	st.bump("duel", 1)
	won := res.Winner == side
	other := res.B
	if side == 1 {
		other = res.A
	}
	change := arenaPointChange(arenaPoints(p), other.Points, won)
	p.ArenaPoints = max(arenaPointFloor, arenaPoints(p)+change)
	if !won {
		p.Losses++
		p.Happy = clamp(p.Happy - 10)
		p.setEffect("dau", 30, now)
		dmg := 12 + rnd()*13
		if rnd() < crushChance {
			dmg = 55
		}
		p.hurt(dmg, "trọng thương khi tỷ thí")
		if p.Dead {
			p.DiedAt = now
			return fmt.Sprintf("💀 %s trúng đòn chí mạng và đã tử trận… · %d điểm", p.Name, change)
		}
		return fmt.Sprintf("%s thua trận, mất %.0f sức khỏe · %d điểm.", p.Name, dmg, change)
	}
	p.Wins++
	st.bump("duel_win", 1)
	p.Happy = clamp(p.Happy + 10)
	p.hurt(3+rnd()*5, "trọng thương khi tỷ thí")
	day := time.UnixMilli(now).In(vnZone).Format("2006-01-02")
	if st.DuelDay != day {
		st.DuelDay, st.DuelWins = day, 0
	}
	if st.DuelWins >= duelRewards {
		return fmt.Sprintf("%s thắng trận! 🏆 · +%d điểm", p.Name, change)
	}
	st.DuelWins++
	st.add("tui", 1)
	st.Coin += st.payout(duelCoin, now)
	return fmt.Sprintf("%s thắng trận! 🏆 +%d điểm · Thưởng 1 🎁 túi thức ăn và %d 🪙.", p.Name, change, duelCoin)
}
