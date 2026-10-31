package game

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	warCurrentKey        = "war_current"
	warSeasonKeyPrefix   = "war_season_"
	warSeasonDays        = 14
	warMaxTerritories    = 3
	warAttacksPerDay     = 3
	warAttackHunger      = 10
	warMemberWait        = 24 * time.Hour
	warAllianceRewardMin = 60
)

type territoryDef struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Tier     int      `json:"tier"`
	X        int      `json:"x"`
	Y        int      `json:"y"`
	Entrance bool     `json:"entrance,omitempty"`
	Adjacent []string `json:"adjacent"`
}

var territoryDefs = buildTerritoryDefs()

var territoryNames = map[string][]string{
	"en": {"Northwest Gate", "Frost Meadow", "Wind Hill", "Northeast Gate", "Ancient Forest", "Starlight Citadel", "Flame Fortress", "Mountain Pass", "Southwest Gate", "Moon Lake", "Highlands", "Southeast Gate"},
	"ja": {"北西門", "霜の草原", "風の丘", "北東門", "古の森", "星明かりの城", "炎の砦", "山峡", "南西門", "月の湖", "高原", "南東門"},
}

func translatedTerritoryName(locale, id string) string {
	def := territoryDefinition(id)
	if def == nil {
		return id
	}
	names := territoryNames[locale]
	for i := range territoryDefs {
		if territoryDefs[i].ID == id && i < len(names) {
			return names[i]
		}
	}
	return def.Name
}

func buildTerritoryDefs() []territoryDef {
	names := []string{"Cổng Tây Bắc", "Đồng Cỏ Sương", "Đồi Gió", "Cổng Đông Bắc", "Rừng Cổ", "Thành Ánh Sao", "Pháo Đài Lửa", "Hẻm Núi", "Cổng Tây Nam", "Hồ Trăng", "Cao Nguyên", "Cổng Đông Nam"}
	tiers := []int{1, 2, 2, 1, 2, 3, 3, 2, 1, 2, 2, 1}
	out := make([]territoryDef, 12)
	for i := range out {
		x, y := i%4, i/4
		out[i] = territoryDef{ID: fmt.Sprintf("land-%02d", i+1), Name: names[i], Tier: tiers[i], X: x, Y: y, Entrance: i == 0 || i == 3 || i == 8 || i == 11, Adjacent: []string{}}
		for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			nx, ny := x+d[0], y+d[1]
			if nx >= 0 && nx < 4 && ny >= 0 && ny < 3 {
				out[i].Adjacent = append(out[i].Adjacent, fmt.Sprintf("land-%02d", ny*4+nx+1))
			}
		}
	}
	return out
}

type warCurrent struct {
	Season   string `json:"season"`
	Previous string `json:"previous,omitempty"`
}

type warTerritory struct {
	ID       string `json:"id"`
	Owner    string `json:"owner,omitempty"`
	HeldDays int    `json:"held_days,omitempty"`
}

type warContribution struct {
	User           string         `json:"user"`
	Alliance       string         `json:"alliance"` // latest alliance, retained for old season records
	AlliancePoints map[string]int `json:"alliance_points,omitempty"`
	Points         int            `json:"points"`
	Wins           int            `json:"wins"`
	ReachedAt      int64          `json:"reached_at"`
}

type warAllianceScore struct {
	Alliance  string `json:"alliance"`
	Points    int    `json:"points"`
	ReachedAt int64  `json:"reached_at"`
}

type warEntitlement struct {
	User        string `json:"user"`
	BPS         int    `json:"bps"`
	AllianceBPS int    `json:"alliance_bps,omitempty"`
	PersonalBPS int    `json:"personal_bps,omitempty"`
	Until       int64  `json:"until"`
	Claimed     bool   `json:"claimed,omitempty"`
}

type warSeason struct {
	ID            string                                    `json:"id"`
	Start         int64                                     `json:"start"`
	End           int64                                     `json:"end"`
	Status        string                                    `json:"status"`
	Settled       map[string]bool                           `json:"settled"`
	Territories   []warTerritory                            `json:"territories"`
	Targets       map[string]map[string]string              `json:"targets"`
	Nominations   map[string]map[string]map[string]duelSide `json:"nominations"`
	Garrisons     map[string]map[string][]duelSide          `json:"garrisons"`
	Attacks       map[string]map[string]int                 `json:"attacks"`
	Influence     map[string]map[string]map[string]int      `json:"influence"`
	DailyWins     map[string]map[string]int                 `json:"daily_wins"`
	DailyContrib  map[string]map[string]int                 `json:"daily_contrib"`
	Scores        map[string]*warAllianceScore              `json:"scores"`
	Contribution  map[string]*warContribution               `json:"contribution"`
	AllianceRanks []warAllianceRank                         `json:"alliance_ranks,omitempty"`
	PersonalRanks []warPersonalRank                         `json:"personal_ranks,omitempty"`
	Entitlements  map[string]*warEntitlement                `json:"entitlements,omitempty"`
}

type warAllianceRank struct {
	Alliance string `json:"alliance"`
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	Points   int    `json:"points"`
	Lands    int    `json:"lands"`
	Rank     int    `json:"rank"`
}

type warPersonalRank struct {
	User     string `json:"user"`
	Name     string `json:"name"`
	Alliance string `json:"alliance"`
	Points   int    `json:"points"`
	Wins     int    `json:"wins"`
	Rank     int    `json:"rank"`
}

type warTerritoryView struct {
	territoryDef
	Owner      string         `json:"owner,omitempty"`
	OwnerName  string         `json:"owner_name,omitempty"`
	OwnerTag   string         `json:"owner_tag,omitempty"`
	HeldDays   int            `json:"held_days,omitempty"`
	Influence  map[string]int `json:"influence"`
	Mine       bool           `json:"mine,omitempty"`
	Target     bool           `json:"target,omitempty"`
	CanTarget  bool           `json:"can_target,omitempty"`
	CanAbandon bool           `json:"can_abandon,omitempty"`
}

type warView struct {
	Season        string             `json:"season"`
	Start         int64              `json:"start"`
	End           int64              `json:"end"`
	Day           int                `json:"day"`
	DayEnds       int64              `json:"day_ends"`
	Status        string             `json:"status"`
	Territories   []warTerritoryView `json:"territories"`
	AttacksLeft   int                `json:"attacks_left"`
	Target        string             `json:"target,omitempty"`
	GarrisonReady bool               `json:"garrison_ready,omitempty"`
	AllianceRanks []warAllianceRank  `json:"alliance_ranks"`
	PersonalRanks []warPersonalRank  `json:"personal_ranks"`
	Reward        *warEntitlement    `json:"reward,omitempty"`
}

func newWarSeason(start int64) *warSeason {
	id := time.UnixMilli(start).In(vnZone).Format("20060102")
	s := &warSeason{ID: id, Start: start, End: time.UnixMilli(start).In(vnZone).AddDate(0, 0, warSeasonDays).UnixMilli(), Status: "active"}
	s.normalize()
	for _, def := range territoryDefs {
		s.Territories = append(s.Territories, warTerritory{ID: def.ID})
	}
	return s
}

func (s *warSeason) normalize() {
	if s.Settled == nil {
		s.Settled = map[string]bool{}
	}
	if s.Targets == nil {
		s.Targets = map[string]map[string]string{}
	}
	if s.Nominations == nil {
		s.Nominations = map[string]map[string]map[string]duelSide{}
	}
	if s.Garrisons == nil {
		s.Garrisons = map[string]map[string][]duelSide{}
	}
	if s.Attacks == nil {
		s.Attacks = map[string]map[string]int{}
	}
	if s.Influence == nil {
		s.Influence = map[string]map[string]map[string]int{}
	}
	if s.DailyWins == nil {
		s.DailyWins = map[string]map[string]int{}
	}
	if s.DailyContrib == nil {
		s.DailyContrib = map[string]map[string]int{}
	}
	if s.Scores == nil {
		s.Scores = map[string]*warAllianceScore{}
	}
	if s.Contribution == nil {
		s.Contribution = map[string]*warContribution{}
	}
	if s.Entitlements == nil {
		s.Entitlements = map[string]*warEntitlement{}
	}
}

func (p *Engine) loadWarCurrent() (*warCurrent, error) {
	var current warCurrent
	_, err := p.loadAllianceJSON(warCurrentKey, &current)
	return &current, err
}

func (p *Engine) loadWarSeason(id string) (*warSeason, error) {
	if id == "" {
		return nil, nil
	}
	var s warSeason
	found, err := p.loadAllianceJSON(warSeasonKeyPrefix+id, &s)
	if err != nil || !found {
		return nil, err
	}
	s.normalize()
	return &s, nil
}

func warMidnight(now int64) int64 {
	t := time.UnixMilli(now).In(vnZone)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, vnZone).UnixMilli()
}

func warDayKey(s *warSeason, day int) string {
	return time.UnixMilli(s.Start).In(vnZone).AddDate(0, 0, day).Format("2006-01-02")
}

func (p *Engine) ensureWar(now int64) (*warCurrent, *warSeason, error) {
	current, err := p.loadWarCurrent()
	if err != nil {
		return nil, nil, err
	}
	if current.Season == "" {
		s := newWarSeason(warMidnight(now))
		current.Season = s.ID
		if err := p.commitRows(map[string]any{warCurrentKey: current, warSeasonKeyPrefix + s.ID: s}); err != nil {
			return nil, nil, err
		}
		return current, s, nil
	}
	s, err := p.loadWarSeason(current.Season)
	if err != nil || s == nil {
		return nil, nil, errors.New("không đọc được mùa lãnh thổ")
	}
	for now >= s.End {
		for day := 0; day < warSeasonDays; day++ {
			p.settleWarDay(s, day)
		}
		rewardUsers := []string{}
		if s.Status != "closed" {
			rewardUsers = p.finalizeSeason(s)
		}
		next := newWarSeason(s.End)
		current.Previous, current.Season = s.ID, next.ID
		if err := p.commitRows(map[string]any{warCurrentKey: current, warSeasonKeyPrefix + s.ID: s, warSeasonKeyPrefix + next.ID: next}); err != nil {
			return nil, nil, err
		}
		for _, user := range rewardUsers {
			p.notify(user, "alliance", "Mùa lãnh thổ đã kết thúc; bạn có thưởng sức mạnh chờ nhận.")
		}
		s = next
	}
	changed := false
	day := max(0, min(warSeasonDays, int((now-s.Start)/int64(24*time.Hour/time.Millisecond))))
	for i := 0; i < day; i++ {
		changed = p.settleWarDay(s, i) || changed
	}
	if changed {
		if err := p.commitRows(map[string]any{warSeasonKeyPrefix + s.ID: s}); err != nil {
			return nil, nil, err
		}
	}
	return current, s, nil
}

func territoryDefinition(id string) *territoryDef {
	for i := range territoryDefs {
		if territoryDefs[i].ID == id {
			return &territoryDefs[i]
		}
	}
	return nil
}

func warTerritoryAt(s *warSeason, id string) int {
	for i := range s.Territories {
		if s.Territories[i].ID == id {
			return i
		}
	}
	return -1
}

func warOwned(s *warSeason, allianceID string) []string {
	out := []string{}
	for _, land := range s.Territories {
		if land.Owner == allianceID {
			out = append(out, land.ID)
		}
	}
	return out
}

func canTargetTerritory(s *warSeason, allianceID, id string) bool {
	def := territoryDefinition(id)
	if def == nil || len(warOwned(s, allianceID)) >= warMaxTerritories {
		return false
	}
	owned := warOwned(s, allianceID)
	if len(owned) == 0 {
		return def.Entrance
	}
	for _, have := range owned {
		for _, adjacent := range territoryDefinition(have).Adjacent {
			if adjacent == id {
				return s.Territories[warTerritoryAt(s, id)].Owner != allianceID
			}
		}
	}
	return false
}

func (p *Engine) allianceWarAction(userID string, st *userState, action, territory string, petIndex int, now int64) (string, *duelResult, error) {
	_, season, err := p.ensureWar(now)
	if err != nil {
		return "", nil, err
	}
	pointer, err := p.loadAllianceMember(userID)
	if err != nil {
		return "", nil, err
	}
	if pointer.Alliance == "" {
		return "", nil, errors.New("bạn chưa thuộc Alliance nào")
	}
	a, err := p.loadAlliance(pointer.Alliance)
	if err != nil || a == nil {
		return "", nil, errors.New("Alliance không còn tồn tại")
	}
	at := allianceMemberAt(a, userID)
	if at < 0 {
		return "", nil, errors.New("thành viên Alliance không đồng bộ")
	}
	role := a.Members[at].Role
	day := max(0, min(warSeasonDays-1, int((now-season.Start)/int64(24*time.Hour/time.Millisecond))))
	dayKey := warDayKey(season, day)
	if action == "target" || action == "abandon" {
		if !allianceCanManage(role) {
			return "", nil, errors.New("chỉ leader hoặc officer được quản lý mục tiêu")
		}
		if territoryDefinition(territory) == nil {
			return "", nil, errors.New("lãnh thổ không hợp lệ")
		}
		if action == "target" {
			if season.Targets[dayKey] == nil {
				season.Targets[dayKey] = map[string]string{}
			}
			if season.Targets[dayKey][a.ID] != "" {
				return "", nil, errors.New("Alliance đã chọn mục tiêu hôm nay")
			}
			if !canTargetTerritory(season, a.ID, territory) {
				return "", nil, errors.New("mục tiêu phải là cửa vào hoặc vùng kề; tối đa 3 lãnh thổ")
			}
			season.Targets[dayKey][a.ID] = territory
			if err := p.commitRows(map[string]any{warSeasonKeyPrefix + season.ID: season}); err != nil {
				return "", nil, err
			}
			return "Đã chọn mục tiêu lãnh thổ hôm nay.", nil, nil
		}
		land := warTerritoryAt(season, territory)
		if land < 0 || season.Territories[land].Owner != a.ID {
			return "", nil, errors.New("Alliance không sở hữu lãnh thổ này")
		}
		season.Territories[land].Owner, season.Territories[land].HeldDays = "", 0
		if err := p.commitRows(map[string]any{warSeasonKeyPrefix + season.ID: season}); err != nil {
			return "", nil, err
		}
		return "Đã bỏ lãnh thổ.", nil, nil
	}
	pt, err := st.petAt(petIndex)
	if err != nil {
		return "", nil, err
	}
	if err := pt.fit(); err != nil {
		return "", nil, err
	}
	if action == "garrison" {
		nextKey := warDayKey(season, day+1)
		if day+1 >= warSeasonDays {
			return "", nil, errors.New("mùa sắp kết thúc, không thể đề cử phòng thủ")
		}
		if season.Nominations[nextKey] == nil {
			season.Nominations[nextKey] = map[string]map[string]duelSide{}
		}
		if season.Nominations[nextKey][a.ID] == nil {
			season.Nominations[nextKey][a.ID] = map[string]duelSide{}
		}
		season.Nominations[nextKey][a.ID][userID] = boostedSideOf(userID, pt, st, now)
		if err := p.commitRows(map[string]any{warSeasonKeyPrefix + season.ID: season}); err != nil {
			return "", nil, err
		}
		return "Đã đề cử thú phòng thủ cho ngày mai.", nil, nil
	}
	if action != "attack" {
		return "", nil, errors.New("thao tác chiến tranh không hợp lệ")
	}
	if now-pointer.JoinedAt < warMemberWait.Milliseconds() {
		return "", nil, errors.New("thành viên mới phải chờ đủ 24 giờ")
	}
	if season.Targets[dayKey][a.ID] != territory {
		return "", nil, errors.New("lãnh thổ này không phải mục tiêu hôm nay")
	}
	if season.Attacks[dayKey] == nil {
		season.Attacks[dayKey] = map[string]int{}
	}
	if season.Attacks[dayKey][userID] >= warAttacksPerDay {
		return "", nil, errors.New("đã dùng hết 3 lượt đánh hôm nay")
	}
	if pt.Hunger < warAttackHunger {
		return "", nil, errors.New("cần 10 độ no để đánh lãnh thổ")
	}
	landAt := warTerritoryAt(season, territory)
	if landAt < 0 {
		return "", nil, errors.New("lãnh thổ không hợp lệ")
	}
	land, def := &season.Territories[landAt], territoryDefinition(territory)
	attacker := boostedSideOf(userID, pt, st, now)
	defender := neutralTerritorySide(def)
	if g := season.Garrisons[dayKey][land.Owner]; len(g) > 0 {
		defender = g[season.Attacks[dayKey][userID]%len(g)]
	}
	result := fight(attacker, defender, p.random)
	result.ID, result.At = newID(), now
	won := result.Winner == 0
	influence := 35
	if won {
		influence = 100 + underdogBonus(attacker, defender)
		influence = int(math.Round(float64(influence) * float64(100+min(50, land.HeldDays*10)) / 100))
	}
	if season.Influence[dayKey] == nil {
		season.Influence[dayKey] = map[string]map[string]int{}
	}
	if season.Influence[dayKey][territory] == nil {
		season.Influence[dayKey][territory] = map[string]int{}
	}
	season.Influence[dayKey][territory][a.ID] += influence
	season.Attacks[dayKey][userID]++
	if season.DailyWins[dayKey] == nil {
		season.DailyWins[dayKey] = map[string]int{}
	}
	if won {
		season.DailyWins[dayKey][a.ID]++
	}
	if season.DailyContrib[dayKey] == nil {
		season.DailyContrib[dayKey] = map[string]int{}
	}
	gain := 4
	if won {
		gain = 10
	}
	gain = min(gain, max(0, 30-season.DailyContrib[dayKey][userID]))
	season.DailyContrib[dayKey][userID] += gain
	entry := season.Contribution[userID]
	if entry == nil {
		entry = &warContribution{User: userID}
		season.Contribution[userID] = entry
	}
	if entry.AlliancePoints == nil {
		entry.AlliancePoints = map[string]int{}
		if entry.Alliance != "" && entry.Points > 0 {
			entry.AlliancePoints[entry.Alliance] = entry.Points
		}
	}
	entry.Alliance = a.ID
	entry.AlliancePoints[a.ID] += gain
	entry.Points += gain
	if won {
		entry.Wins++
	}
	entry.ReachedAt = now
	pt.Hunger = clamp(pt.Hunger - warAttackHunger)
	if !won {
		pt.Health = max(1, pt.Health-5)
	}
	// Only wins on the hardest lands can yield a bound soul (5%). The existing
	// three daily attempts and atomic user+season commit also bound this drop.
	souls := 0
	if won && def.Tier == 3 && p.random() < 0.05 {
		st.add("tinhhon", 1)
		souls = 1
	}
	p.rewardActivity(st, "/alliance/attack", now)
	if err := p.commitRows(map[string]any{userPrefix + userID: st, warSeasonKeyPrefix + season.ID: season}); err != nil {
		return "", nil, err
	}
	outcome := "thua"
	if won {
		outcome = "thắng"
	}
	return fmt.Sprintf("Trận lãnh thổ %s: +%d ảnh hưởng, +%d Tinh Hồn.", outcome, influence, souls), &result, nil
}

func neutralTerritorySide(def *territoryDef) duelSide {
	lv := 8 + def.Tier*8
	return duelSide{Name: def.Name + " thủ vệ", Sprite: def.Tier, Lv: lv, Speed: speedBase + float64(lv)*speedPerLv, Atk: float64(lv) * .45, Def: float64(lv) * .18, HP: float64(70 + lv*5), Skills: []string{"khien", "lua"}, SkillLv: map[string]int{"khien": def.Tier, "lua": def.Tier}}
}

func duelSidePower(side duelSide) int {
	return int(side.HP + side.Atk*3 + side.Def*3 + float64(side.Lv)*4)
}

func underdogBonus(a, b duelSide) int {
	ap, bp := duelSidePower(a), duelSidePower(b)
	if ap >= bp || bp <= 0 {
		return 0
	}
	return min(25, int(math.Round(float64(bp-ap)/float64(bp)*25)))
}

func (p *Engine) settleWarDay(s *warSeason, day int) bool {
	key := warDayKey(s, day)
	if s.Settled[key] {
		return false
	}
	reachedAt := time.UnixMilli(s.Start).In(vnZone).AddDate(0, 0, day+1).UnixMilli()
	for i := range s.Territories {
		land := &s.Territories[i]
		def := territoryDefinition(land.ID)
		bestID, best, tied := "", 0, false
		for id, score := range s.Influence[key][land.ID] {
			if score > best {
				bestID, best, tied = id, score, false
			} else if score == best && score > 0 {
				tied = true
			}
		}
		old := land.Owner
		if best >= 200+50*def.Tier && (!tied || bestID == old) {
			land.Owner = bestID
		}
		if land.Owner != old {
			p.backend.LogError("vpet: territory settled", "season", s.ID, "day", key, "territory", land.ID, "from", old, "to", land.Owner)
		}
		if land.Owner == old && land.Owner != "" {
			land.HeldDays++
		} else if land.Owner != "" {
			land.HeldDays = 1
		} else {
			land.HeldDays = 0
		}
		if land.Owner != "" {
			score := s.Scores[land.Owner]
			if score == nil {
				score = &warAllianceScore{Alliance: land.Owner}
				s.Scores[land.Owner] = score
			}
			score.Points += []int{0, 100, 150, 200}[def.Tier]
			score.ReachedAt = reachedAt
		}
	}
	for allianceID, wins := range s.DailyWins[key] {
		score := s.Scores[allianceID]
		if score == nil {
			score = &warAllianceScore{Alliance: allianceID}
			s.Scores[allianceID] = score
		}
		score.Points += min(20, wins)
		score.ReachedAt = reachedAt
	}
	nextKey := warDayKey(s, day+1)
	if day+1 < warSeasonDays {
		s.Garrisons[nextKey] = map[string][]duelSide{}
		for allianceID, nominees := range s.Nominations[nextKey] {
			rows := make([]duelSide, 0, len(nominees))
			for _, side := range nominees {
				rows = append(rows, side)
			}
			sort.Slice(rows, func(i, j int) bool { return duelSidePower(rows[i]) > duelSidePower(rows[j]) })
			s.Garrisons[nextKey][allianceID] = append([]duelSide(nil), rows[:min(4, len(rows))]...)
		}
	}
	s.Settled[key] = true
	return true
}

func (p *Engine) warState(userID string, member *allianceMemberPointer, mine *alliance, now int64) warView {
	current, season, err := p.ensureWar(now)
	if err != nil || season == nil {
		return warView{Territories: []warTerritoryView{}, AllianceRanks: []warAllianceRank{}, PersonalRanks: []warPersonalRank{}}
	}
	day := max(0, min(warSeasonDays-1, int((now-season.Start)/int64(24*time.Hour/time.Millisecond))))
	key := warDayKey(season, day)
	out := warView{Season: season.ID, Start: season.Start, End: season.End, Day: day + 1, DayEnds: time.UnixMilli(season.Start).In(vnZone).AddDate(0, 0, day+1).UnixMilli(), Status: season.Status, AttacksLeft: max(0, warAttacksPerDay-season.Attacks[key][userID]), AllianceRanks: p.rankAlliances(season), PersonalRanks: p.rankPlayers(season)}
	if member != nil {
		out.Target = season.Targets[key][member.Alliance]
		out.GarrisonReady = len(season.Nominations[warDayKey(season, day+1)][member.Alliance]) > 0
	}
	locale := p.locale(userID)
	for i, def := range territoryDefs {
		land := season.Territories[i]
		localizedDef := def
		localizedDef.Name = translatedTerritoryName(locale, def.ID)
		view := warTerritoryView{territoryDef: localizedDef, Owner: land.Owner, HeldDays: land.HeldDays, Influence: map[string]int{}, Mine: member != nil && member.Alliance != "" && land.Owner == member.Alliance, Target: out.Target == land.ID}
		for id, value := range season.Influence[key][land.ID] {
			view.Influence[id] = value
		}
		if land.Owner != "" {
			if a, _ := p.loadAlliance(land.Owner); a != nil {
				view.OwnerName, view.OwnerTag = a.Name, a.Tag
			}
		}
		if mine != nil {
			view.CanTarget = canTargetTerritory(season, mine.ID, land.ID) && out.Target == ""
			view.CanAbandon = view.Mine
		}
		out.Territories = append(out.Territories, view)
	}
	if current.Previous != "" {
		if previous, _ := p.loadWarSeason(current.Previous); previous != nil {
			if reward := previous.Entitlements[userID]; reward != nil {
				cp := *reward
				out.Reward = &cp
			}
		}
	}
	return out
}

func (p *Engine) allianceOwnsTerritory(allianceID string) bool {
	current, err := p.loadWarCurrent()
	if err != nil || current.Season == "" {
		return false
	}
	s, _ := p.loadWarSeason(current.Season)
	return s != nil && len(warOwned(s, allianceID)) > 0
}
