package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

// The boss used to be a flat 18,000 HP. With 200+ owners on the server a handful of geared
// pets emptied that bar within minutes, so everyone who logged in later found it already dead
// and never got to join. Its health now grows as people turn up: each new fighter adds their
// own share, so the bar lasts roughly as long as the server keeps hitting it regardless of
// whether three people show up or eighty.
const (
	bossBaseHP    = 500000          // a day-long target: a geared pet lands ~800 a blow
	bossHPPerJoin = 15000           // added the first time each fighter lands a blow
	bossMaxHPCap  = 3000000         // a ceiling, so a surge of players cannot make it unkillable
	bossHitCap    = 0.04            // one blow may take at most this share of the current pool
	bossCooldown  = 3 * time.Minute // shorter than before: the limit is the bar, not the clock
	partyTTL      = 30 * time.Minute
)

// bossMaxHP is the starting pool. Kept as a name because tests and the client read it.
const bossMaxHP = bossBaseHP

type worldBoss struct {
	Day     string           `json:"day"`
	Name    string           `json:"name"`
	HP      int              `json:"hp"`
	Max     int              `json:"max"` // grows with each new fighter; 0 on pre-1.20 records
	Damage  map[string]int   `json:"damage"`
	Last    map[string]int64 `json:"last"`
	Claimed map[string]bool  `json:"claimed"`
}

// maxHP answers the pool size, filling in the starting value for bosses stored before the
// pool became elastic.
func (b *worldBoss) maxHP() int {
	if b.Max <= 0 {
		return bossMaxHP
	}
	return b.Max
}

type bossMember struct {
	User   string `json:"user"`
	Name   string `json:"name"`
	Damage int    `json:"damage"`
}
type bossView struct {
	Name     string       `json:"name"`
	Day      string       `json:"day"`
	HP       int          `json:"hp"`
	MaxHP    int          `json:"max_hp"`
	Mine     int          `json:"mine"`
	Next     int64        `json:"next"`
	Claimed  bool         `json:"claimed"`
	Fighters int          `json:"fighters"`
	Share    int          `json:"share"` // this user's percent of the damage done so far
	Tiers    [4]int       `json:"tiers"` // damage needed for 1..4 crates, at today's turnout
	Gold     int          `json:"gold"`  // what the player would be paid right now
	Top      []bossMember `json:"top"`
}

func (p *Engine) loadBoss(now int64) (*worldBoss, error) {
	day := dayOf(now)
	raw, e := p.backend.KVGet("boss_" + day)
	if e != nil {
		return nil, e
	}
	b := &worldBoss{Day: day, Name: "Cổ Long Thiên Đảo", HP: bossMaxHP, Max: bossMaxHP, Damage: map[string]int{}, Last: map[string]int64{}, Claimed: map[string]bool{}}
	if raw != nil {
		if err := json.Unmarshal(raw, b); err != nil {
			return nil, err
		}
	}
	return b, nil
}
func (p *Engine) bossFor(user string, now int64) *bossView {
	b, err := p.loadBoss(now)
	if err != nil {
		return nil
	}
	v := &bossView{Name: b.Name, Day: b.Day, HP: b.HP, MaxHP: b.maxHP(), Mine: b.Damage[user], Next: b.Last[user] + bossCooldown.Milliseconds(),
		Claimed: b.Claimed[user], Fighters: len(b.Damage), Top: []bossMember{}}
	// Crate thresholds move with the turnout, so the screen has to be told them rather than
	// printing fixed numbers that stopped being true the moment the pool grew.
	fighters := max(1, len(b.Damage))
	pool := max(1, b.maxHP()-b.HP)
	average := float64(pool) / float64(fighters)
	for i, weight := range [4]float64{0, 0.6, 1.5, 2.5} {
		v.Tiers[i] = int(average * weight)
	}
	v.Gold = bossGold(b.Damage[user], pool, fighters)
	if total := b.maxHP() - b.HP; total > 0 {
		v.Share = b.Damage[user] * 100 / total
	}
	for id, n := range b.Damage {
		v.Top = append(v.Top, bossMember{User: id, Name: p.userName(id), Damage: n})
	}
	sort.Slice(v.Top, func(i, j int) bool { return v.Top[i].Damage > v.Top[j].Damage })
	return v
}

// Boss pay is deliberately not proportional to damage. A geared pet hits for ~4,000 and a new
// one for ~275, so paying per point handed the same fifty blows fifteen times the gold and left
// newcomers permanently behind — players called the fight unfair, and they were right.
//
// Instead the reward follows the share of the pool a player took, which is what "turning up and
// helping" actually means, and it is capped. Everyone who lands a real contribution clears the
// floor; the hardest hitter earns roughly three times the newcomer, not fifteen.
const (
	bossGoldFloor = 25  // for anyone whose blows mattered at all
	bossGoldCap   = 150 // one blind box: worth the day, never a second economy
)

// bossGold pays for pulling your weight, measured against the average fighter rather than
// against the whole pool, so the same effort is worth the same whether five people show up or
// a hundred and sixty.
func bossGold(mine, pool, fighters int) int {
	if mine <= 0 || pool <= 0 || fighters <= 0 {
		return 0
	}
	// Square root flattens the curve: ten times the damage pays about three times the gold,
	// which keeps a geared pet ahead without leaving a new one feeling pointless.
	weight := float64(mine) / (float64(pool) / float64(fighters))
	gold := bossGoldFloor + int(float64(bossGoldCap-bossGoldFloor)*math.Sqrt(min(weight/2.5, 1)))
	return min(gold, bossGoldCap)
}

// bossCrates measures a player against the rest of the field rather than against the pool. A
// fixed share cannot work: twenty fighters each take 5% of the bar and a hundred and sixty
// each take 0.6%, so a share threshold pays the same effort three crates on a quiet day and
// one on a busy one — exactly the "it gets worse when more people play" complaint.
//
// Pulling your weight means roughly a 1/fighters share, so that is the yardstick: the ratio of
// your damage to the average. Turn up and do your part and the crowd size stops mattering.
func bossCrates(mine, pool, fighters int) int {
	if mine <= 0 || pool <= 0 || fighters <= 0 {
		return 0
	}
	average := float64(pool) / float64(fighters)
	switch weight := float64(mine) / average; {
	case weight >= 2.5:
		return 4
	case weight >= 1.5:
		return 3
	case weight >= 0.6:
		return 2
	default:
		return 1
	}
}

func bossReward(st *userState, b *worldBoss, user string, now int64, rnd rng) string {
	mine := b.Damage[user]
	pool := max(1, b.maxHP()-b.HP)
	share := mine * 100 / pool
	// payout() only halves the cruelty penalty; the daily cap is applied by earnBoss so boss
	// gold cannot route around the economy the way it did before.
	fighters := max(1, len(b.Damage))
	gold := st.earnBoss(st.payout(bossGold(mine, pool, fighters), now), now)
	crates := bossCrates(mine, pool, fighters)
	st.add("tui_vukhi", crates)
	// One roll per settled boss, only for a meaningful contribution. The existing
	// claimed marker and inventory commit together; retries cannot mint extra souls.
	souls := 0
	if b.HP <= 0 && crates >= 2 && rnd() < 0.15 {
		st.add("tinhhon", 1)
		souls = 1
	}
	return fmt.Sprintf("Nhận thưởng boss: %d sát thương (%d%%), %d vàng, %d hòm vũ khí và %d Tinh Hồn.", mine, share, gold, crates, souls)
}

// settleBoss pays everybody when the boss dies. Manual claiming remains as a fallback, but a
// player no longer loses the reward merely because midnight passed before they opened the panel.
func (p *Engine) settleBoss(b *worldBoss, now int64) error {
	if b == nil || b.HP > 0 {
		return nil
	}
	if b.Claimed == nil {
		b.Claimed = map[string]bool{}
	}
	users := make([]string, 0, len(b.Damage))
	for user, damage := range b.Damage {
		if damage > 0 && !b.Claimed[user] {
			users = append(users, user)
		}
	}
	sort.Strings(users)
	var first error
	for _, user := range users {
		st, err := p.load(user)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		msg := bossReward(st, b, user, now, p.random)
		b.Claimed[user] = true
		if err = p.commitRows(map[string]any{userPrefix + user: st, "boss_" + b.Day: b}); err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		p.dm(user, st, "🎁 "+msg, &openLink{"🐉 Mở Boss thế giới", "campaign", ""})
		p.notify(user, "boss_reward", "")
	}
	return first
}

func (p *Engine) settlePreviousBoss(now int64) error {
	previous := time.UnixMilli(now).In(vnZone).AddDate(0, 0, -1).Format("2006-01-02")
	raw, appErr := p.backend.KVGet("boss_" + previous)
	if appErr != nil || raw == nil {
		if appErr != nil {
			return appErr
		}
		return nil
	}
	var b worldBoss
	if err := json.Unmarshal(raw, &b); err != nil {
		return err
	}
	return p.settleBoss(&b, now)
}

func (p *Engine) bossAction(user string, st *userState, petIndex int, claim bool, now int64) (string, error) {
	b, err := p.loadBoss(now)
	if err != nil {
		return "", err
	}
	if claim {
		if b.HP > 0 || b.Damage[user] <= 0 || b.Claimed[user] {
			return "", errors.New("boss chưa bị hạ, chưa đóng góp hoặc đã nhận thưởng")
		}
		msg := bossReward(st, b, user, now, p.random)
		b.Claimed[user] = true
		if err := p.commitRows(map[string]any{userPrefix + user: st, "boss_" + b.Day: b}); err != nil {
			return "", err
		}
		return msg, nil
	}
	pt, err := st.petAt(petIndex)
	if err != nil {
		return "", err
	}
	if err := pt.fit(); err != nil {
		return "", err
	}
	if pt.FrozenUntil > now {
		return "", errors.New("thú đang nghỉ phép")
	}
	if max(pt.Lv, 1) < 5 {
		return "", errors.New("boss thế giới yêu cầu cấp 5")
	}
	if b.HP <= 0 {
		return "", errors.New("boss đã bị hạ; quay lại ngày mai")
	}
	if b.Last[user] > 0 && now-b.Last[user] < bossCooldown.Milliseconds() {
		return "", errors.New("đang hồi sức sau đòn đánh boss")
	}
	if pt.Hunger < 10 {
		return "", errors.New("cần 10 độ no đánh boss")
	}
	// A newcomer brings their own share of health with them, so joining late still leaves
	// something to fight rather than a corpse.
	if _, seen := b.Damage[user]; !seen && b.maxHP() < bossMaxHPCap {
		grown := min(bossMaxHPCap, b.maxHP()+bossHPPerJoin)
		b.HP += grown - b.maxHP()
		b.Max = grown
	}
	// No single pet may delete the bar: the strongest hit is still a slice, which keeps the
	// fight open for everybody else and makes the damage board a contest rather than a race.
	dmg := min(b.HP, 60+boostedPower(pt, st, now)*2+int(p.random()*30))
	dmg = min(dmg, max(1, int(float64(b.maxHP())*bossHitCap)))
	b.HP -= dmg
	b.Damage[user] += dmg
	b.Last[user] = now
	pt.Hunger = clamp(pt.Hunger - 10)
	pt.Health = max(1, pt.Health-5)
	st.bump("boss", 1)
	p.rewardActivity(st, "/boss/hit", now)
	if err := p.commitRows(map[string]any{userPrefix + user: st, "boss_" + b.Day: b}); err != nil {
		return "", err
	}
	msg := fmt.Sprintf("%s gây %d sát thương! Boss còn %d/%d máu.", pt.Name, dmg, b.HP, b.maxHP())
	if b.HP == 0 {
		if err := p.settleBoss(b, now); err != nil {
			p.backend.LogError("vpet: settle world boss", "day", b.Day, "err", err.Error())
		}
	}
	return msg, nil
}

type partyMember struct {
	User    string `json:"user"`
	Name    string `json:"name"`
	Born    int64  `json:"born"`
	Pet     string `json:"pet"`
	Power   int    `json:"power"`
	Claimed bool   `json:"claimed"`
}
type huntParty struct {
	ID      string        `json:"id"`
	Host    string        `json:"host"`
	Map     string        `json:"map"`
	Created int64         `json:"created"`
	Members []partyMember `json:"members"`
	Done    bool          `json:"done"`
	Won     bool          `json:"won"`
	Power   int           `json:"power"`
	Target  int           `json:"target"`
}

func (p *Engine) parties(now int64) ([]huntParty, error) {
	out := []huntParty{}
	for page := 0; ; page++ {
		keys, e := p.backend.KVList(page, 200)
		if e != nil {
			return nil, e
		}
		for _, key := range keys {
			if len(key) < 6 || key[:6] != "party_" {
				continue
			}
			raw, e := p.backend.KVGet(key)
			if e != nil {
				return nil, e
			}
			if raw == nil {
				continue
			}
			var v huntParty
			if err := json.Unmarshal(raw, &v); err != nil {
				return nil, err
			}
			if now-v.Created < partyTTL.Milliseconds() {
				out = append(out, v)
			}
		}
		if len(keys) < 200 {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created > out[j].Created })
	return out, nil
}
func (p *Engine) partyAction(user string, st *userState, petIndex int, action, id, mapID string, now int64) (string, error) {
	rooms, err := p.parties(now)
	if err != nil {
		return "", err
	}
	findMember := func(room *huntParty) int {
		for i, m := range room.Members {
			if m.User == user {
				return i
			}
		}
		return -1
	}
	var room *huntParty
	for i := range rooms {
		if rooms[i].ID == id {
			room = &rooms[i]
			break
		}
	}
	if action == "create" || action == "join" {
		for i := range rooms {
			if findMember(&rooms[i]) >= 0 && !rooms[i].Done {
				return "", errors.New("bạn đã ở trong một đội; rời đội đó trước")
			}
		}
		if action == "create" {
			active := 0
			for _, r := range rooms {
				if !r.Done {
					active++
				}
			}
			if active >= 30 {
				return "", errors.New("đã đủ 30 đội đang chờ; tham gia một đội có sẵn")
			}
			m := mapByID[mapID]
			if m == nil {
				return "", errors.New("không có vùng săn")
			}
			room = &huntParty{ID: newID(), Host: user, Map: mapID, Created: now, Target: 110 + m.MinLv*12, Members: []partyMember{}}
		} else if room == nil || room.Done || len(room.Members) >= 4 {
			return "", errors.New("đội không còn chỗ hoặc đã xuất phát")
		}
		pt, err := st.petAt(petIndex)
		if err != nil {
			return "", err
		}
		if err := pt.fit(); err != nil {
			return "", err
		}
		if max(pt.Lv, 1) < mapByID[room.Map].MinLv {
			return "", errors.New("chưa đủ cấp vào vùng này")
		}
		if pt.FrozenUntil > now || pt.Hunger < 10 || (pt.LastHunt > 0 && now-pt.LastHunt < huntCooldown.Milliseconds()) {
			return "", errors.New("thú chưa sẵn sàng: cần 10 no và hết hồi sức săn")
		}
		room.Members = append(room.Members, partyMember{User: user, Name: p.userName(user), Born: pt.Born, Pet: pt.Name, Power: boostedPower(pt, st, now)})
		if err := p.commitRows(map[string]any{"party_" + room.ID: room}); err != nil {
			return "", err
		}
		return "Đã vào đội; chủ đội có thể xuất phát khi đủ ít nhất 2 người.", nil
	}
	if room == nil {
		return "", errors.New("đội hết hạn sau 30 phút hoặc không còn")
	}
	at := findMember(room)
	switch action {
	case "leave":
		if at < 0 || room.Done {
			return "", errors.New("không thể rời đội này")
		}
		if room.Host == user {
			if err := p.commitRows(map[string]any{"party_" + room.ID: nil}); err != nil {
				return "", err
			}
			return "Đã giải tán đội.", nil
		}
		room.Members = append(room.Members[:at], room.Members[at+1:]...)
	case "start":
		if room.Host != user || room.Done || len(room.Members) < 2 {
			return "", errors.New("chỉ chủ đội được xuất phát; cần 2–4 người")
		}
		writes := map[string]any{}
		players := map[string]*userState{}
		for _, member := range room.Members {
			player, err := p.load(member.User)
			if err != nil {
				return "", err
			}
			player.markPets(now)
			_, pt := player.petByBorn(member.Born)
			if pt == nil {
				return "", errors.New("có thú không còn trong đội")
			}
			pt.tick(now)
			if err := pt.fit(); err != nil {
				return "", fmt.Errorf("%s: %w", member.Name, err)
			}
			if max(pt.Lv, 1) < mapByID[room.Map].MinLv {
				return "", errors.New(member.Name + " chưa đủ cấp")
			}
			if pt.FrozenUntil > now || pt.Hunger < 10 || (pt.LastHunt > 0 && now-pt.LastHunt < huntCooldown.Milliseconds()) {
				return "", errors.New(member.Name + " chưa sẵn sàng: cần 10 no, không nghỉ phép, hết hồi sức đi săn")
			}
			room.Power += boostedPower(pt, player, now)
			players[member.User] = player
		}
		room.Done = true
		room.Won = room.Power >= room.Target
		for _, member := range room.Members {
			player := players[member.User]
			_, pt := player.petByBorn(member.Born)
			pt.LastHunt = now
			pt.Hunger = clamp(pt.Hunger - 10)
			if !room.Won {
				pt.Health = max(1, pt.Health-12)
			}
			p.rewardActivity(player, "/party", now)
			writes[userPrefix+member.User] = player
		}
		writes["party_"+room.ID] = room
		if err := p.commitRows(writes); err != nil {
			return "", err
		}
		for _, member := range room.Members {
			p.notify(member.User, "party", "Đội đã hoàn thành chuyến săn; mở tab Săn đội nhận kết quả.")
		}
		return fmt.Sprintf("Săn đội: lực %d / yêu cầu %d. Thắng: %t.", room.Power, room.Target, room.Won), nil
	case "claim":
		if at < 0 || !room.Done || room.Members[at].Claimed {
			return "", errors.New("chưa có kết quả hoặc đã nhận thưởng")
		}
		_, pt := st.petByBorn(room.Members[at].Born)
		if pt == nil {
			return "", errors.New("thú tham gia không còn")
		}
		room.Members[at].Claimed = true
		if room.Won {
			st.Coin += st.payout(20+mapByID[room.Map].MinLv*2, now)
			st.add("tui", 1)
			gainXP(pt, mapByID[room.Map].XP+12, p.random)
		}
		if err := p.commitRows(map[string]any{userPrefix + user: st, "party_" + room.ID: room}); err != nil {
			return "", err
		}
		if room.Won {
			return "Nhận thưởng săn đội: vàng, kinh nghiệm và 1 túi thức ăn.", nil
		}
		return "Đội chưa đủ lực. Chưa có thưởng; hồi phục và tìm thêm đồng đội.", nil
	default:
		return "", errors.New("thao tác đội không hợp lệ")
	}
	if err := p.commitRows(map[string]any{"party_" + room.ID: room}); err != nil {
		return "", err
	}
	return "Đã rời đội.", nil
}
