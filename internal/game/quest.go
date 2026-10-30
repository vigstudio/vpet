package game

import (
	"errors"
	"fmt"
)

// Quests. Production data showed a reward drought rather than inflation: across 98 active
// pets the median player held 4 gold, the median top pet was level 1, and not one skill
// stone existed. Dailies pay for things players already do, so the early game funds itself
// without handing out a lump sum that would unbalance the market.
const questTargetCap = 9999

type questDef struct {
	ID     string
	Name   string
	Desc   string
	Need   int
	Gold   int
	Item   string
	Amount int
	Daily  bool
	// Achievements come in ladders. Track groups the rungs that read one counter and Tier is
	// the rung, so the panel can show only the rung being worked on instead of a wall of
	// finished rows. Dailies leave both empty.
	Track string
	Tier  int
}

// counters are the raw tallies quests read. They live on userState and only ever go up;
// dailies subtract a snapshot taken at reset, so one counter serves both quest kinds.
var quests = []questDef{
	{ID: "d_feed", Name: "Cho ăn 3 lần", Desc: "Chăm thú là việc mỗi ngày", Need: 3, Gold: 25, Daily: true},
	{ID: "d_hunt", Name: "Đi săn 2 chuyến", Desc: "Mang đồ về nhà", Need: 2, Gold: 35, Item: "gem_shard", Amount: 2, Daily: true},
	{ID: "d_stage", Name: "Vượt 1 ải chiến dịch", Desc: "Tiến thêm một bước", Need: 1, Gold: 40, Item: "gem_shard", Amount: 3, Daily: true},
	{ID: "d_wild", Name: "Hạ 3 thú hoang", Desc: "Luyện tay ngoài hoang dã", Need: 3, Gold: 30, Daily: true},
	{ID: "d_duel", Name: "Đấu 1 trận tỷ thí", Desc: "Thử sức với đồng nghiệp", Need: 1, Gold: 35, Daily: true},
	{ID: "d_boss", Name: "Đánh boss thế giới 1 lần", Desc: "Góp sát thương cùng mọi người", Need: 1, Gold: 45, Item: "gem_shard", Amount: 2, Daily: true},

	// Achievements are ladders. The rungs that already shipped keep their IDs, because
	// QuestDone is keyed by ID and renaming one would hand a finished achievement back out.
	{ID: "a_hunt25", Name: "Thợ săn lành nghề", Desc: "Đi săn 25 chuyến", Need: 25, Gold: 150, Item: "ngoc", Amount: 1, Track: "hunt", Tier: 1},
	{ID: "a_hunt100", Name: "Thợ săn kỳ cựu", Desc: "Đi săn 100 chuyến", Need: 100, Gold: 400, Item: "ngoc", Amount: 2, Track: "hunt", Tier: 2},
	{ID: "a_hunt300", Name: "Bậc thầy đi săn", Desc: "Đi săn 300 chuyến", Need: 300, Gold: 900, Item: "ngoc", Amount: 3, Track: "hunt", Tier: 3},
	{ID: "a_hunt800", Name: "Huyền thoại đường xa", Desc: "Đi săn 800 chuyến", Need: 800, Gold: 2000, Item: "tinh_the_tiem_nang", Amount: 1, Track: "hunt", Tier: 4},

	{ID: "a_wild50", Name: "Khắc tinh thú hoang", Desc: "Hạ 50 thú hoang", Need: 50, Gold: 250, Item: "ngoc", Amount: 1, Track: "wild", Tier: 1},
	{ID: "a_wild200", Name: "Hung thần hoang dã", Desc: "Hạ 200 thú hoang", Need: 200, Gold: 650, Item: "ngoc", Amount: 2, Track: "wild", Tier: 2},
	{ID: "a_wild600", Name: "Chúa tể muông thú", Desc: "Hạ 600 thú hoang", Need: 600, Gold: 1600, Item: "tinh_the_tiem_nang", Amount: 1, Track: "wild", Tier: 3},

	{ID: "a_stage10", Name: "Nhà thám hiểm", Desc: "Vượt 10 ải chiến dịch", Need: 10, Gold: 200, Item: "gem_shard", Amount: 10, Track: "stage", Tier: 1},
	{ID: "a_stage30", Name: "Chinh phục chiến dịch", Desc: "Vượt 30 ải chiến dịch", Need: 30, Gold: 500, Item: "tinh_the_tiem_nang", Amount: 1, Track: "stage", Tier: 2},
	{ID: "a_stage90", Name: "Anh hùng Thiên Đảo", Desc: "Vượt 90 ải chiến dịch", Need: 90, Gold: 1200, Item: "ngoc", Amount: 3, Track: "stage", Tier: 3},
	{ID: "a_stage200", Name: "Vĩ nhân Thiên Đảo", Desc: "Vượt 200 ải chiến dịch", Need: 200, Gold: 2600, Item: "tinh_the_tiem_nang", Amount: 2, Track: "stage", Tier: 4},

	{ID: "a_duel10", Name: "Võ sĩ", Desc: "Thắng 10 trận tỷ thí", Need: 10, Gold: 200, Item: "ngoc", Amount: 1, Track: "duel_win", Tier: 1},
	{ID: "a_duel40", Name: "Kiếm sĩ lão luyện", Desc: "Thắng 40 trận tỷ thí", Need: 40, Gold: 550, Item: "ngoc", Amount: 2, Track: "duel_win", Tier: 2},
	{ID: "a_duel120", Name: "Quán quân đấu trường", Desc: "Thắng 120 trận tỷ thí", Need: 120, Gold: 1400, Item: "tinh_the_tiem_nang", Amount: 1, Track: "duel_win", Tier: 3},

	{ID: "a_forge", Name: "Thợ rèn", Desc: "Ghép đồ 10 lần", Need: 10, Gold: 180, Item: "gem_shard", Amount: 8, Track: "forge", Tier: 1},
	{ID: "a_forge40", Name: "Thợ rèn lão luyện", Desc: "Ghép đồ 40 lần", Need: 40, Gold: 480, Item: "gem_shard", Amount: 20, Track: "forge", Tier: 2},
	{ID: "a_forge120", Name: "Đại sư lò rèn", Desc: "Ghép đồ 120 lần", Need: 120, Gold: 1300, Item: "ngoc", Amount: 3, Track: "forge", Tier: 3},

	{ID: "a_crush", Name: "Thợ đập ngọc", Desc: "Đập 30 món đồ thừa", Need: 30, Gold: 160, Item: "gem_shard", Amount: 6, Track: "crush", Tier: 1},
	{ID: "a_crush150", Name: "Tay đập cừ khôi", Desc: "Đập 150 món đồ thừa", Need: 150, Gold: 450, Item: "gem_shard", Amount: 18, Track: "crush", Tier: 2},
	{ID: "a_crush500", Name: "Nghệ nhân mảnh ngọc", Desc: "Đập 500 món đồ thừa", Need: 500, Gold: 1100, Item: "ngoc", Amount: 2, Track: "crush", Tier: 3},

	{ID: "a_feed50", Name: "Người chăm thú", Desc: "Cho ăn 50 lần", Need: 50, Gold: 160, Item: "gem_shard", Amount: 5, Track: "feed", Tier: 1},
	{ID: "a_feed250", Name: "Bạn đồng hành", Desc: "Cho ăn 250 lần", Need: 250, Gold: 520, Item: "ngoc", Amount: 1, Track: "feed", Tier: 2},
	{ID: "a_feed800", Name: "Tri kỷ của thú", Desc: "Cho ăn 800 lần", Need: 800, Gold: 1500, Item: "tinh_the_tiem_nang", Amount: 1, Track: "feed", Tier: 3},

	{ID: "a_boss10", Name: "Người góp sức", Desc: "Đánh boss thế giới 10 lần", Need: 10, Gold: 220, Item: "gem_shard", Amount: 8, Track: "boss", Tier: 1},
	{ID: "a_boss50", Name: "Tay săn boss", Desc: "Đánh boss thế giới 50 lần", Need: 50, Gold: 700, Item: "ngoc", Amount: 2, Track: "boss", Tier: 2},
	{ID: "a_boss150", Name: "Khắc tinh của boss", Desc: "Đánh boss thế giới 150 lần", Need: 150, Gold: 1800, Item: "tinh_the_tiem_nang", Amount: 1, Track: "boss", Tier: 3},

	{ID: "a_arena30", Name: "Người ham tỷ thí", Desc: "Đấu 30 trận tỷ thí", Need: 30, Gold: 240, Item: "gem_shard", Amount: 8, Track: "duel", Tier: 1},
	{ID: "a_arena120", Name: "Khách quen đấu trường", Desc: "Đấu 120 trận tỷ thí", Need: 120, Gold: 760, Item: "ngoc", Amount: 2, Track: "duel", Tier: 2},
	{ID: "a_arena400", Name: "Linh hồn đấu trường", Desc: "Đấu 400 trận tỷ thí", Need: 400, Gold: 1900, Item: "tinh_the_tiem_nang", Amount: 1, Track: "duel", Tier: 3},
}

var questByID = func() map[string]*questDef {
	m := make(map[string]*questDef, len(quests))
	for i := range quests {
		m[quests[i].ID] = &quests[i]
	}
	return m
}()

// trackTiers counts the rungs on each achievement ladder, so a view can say "rung 2 of 4".
var trackTiers = func() map[string]int {
	m := map[string]int{}
	for i := range quests {
		if t := quests[i].Track; t != "" {
			m[t] = max(m[t], quests[i].Tier)
		}
	}
	return m
}()

// bump records progress. Every gameplay path calls this with the counter it affects, so a
// quest never needs its own hook inside the action it measures.
func (st *userState) bump(counter string, n int) {
	if st.Counts == nil {
		st.Counts = map[string]int{}
	}
	st.Counts[counter] = min(questTargetCap, st.Counts[counter]+n)
}

// questCounter maps a quest to the tally it reads. An achievement names its counter through
// Track; the dailies are listed here because they predate the ladders.
func questCounter(id string) string {
	if q := questByID[id]; q != nil && q.Track != "" {
		return q.Track
	}
	switch id {
	case "d_feed":
		return "feed"
	case "d_hunt":
		return "hunt"
	case "d_stage":
		return "stage"
	case "d_wild":
		return "wild"
	case "d_duel":
		return "duel"
	case "d_boss":
		return "boss"
	}
	return ""
}

// rollDaily resets the daily snapshot once per Vietnamese day. Progress made before the
// reset stays in the lifetime counter, so achievements never lose ground.
func (st *userState) rollDaily(now int64) bool {
	day := dayOf(now)
	if st.QuestDay == day {
		return false
	}
	first := st.QuestDay == ""
	st.QuestDay = day
	st.QuestBase = map[string]int{}
	// On the very first roll there is no previous day to measure against, so today starts
	// from zero and whatever the player already did still counts.
	if !first {
		for k, v := range st.Counts {
			st.QuestBase[k] = v
		}
	}
	// a new day re-opens every daily; achievements keep their claimed flag
	for id := range st.QuestDone {
		if questByID[id] != nil && questByID[id].Daily {
			delete(st.QuestDone, id)
		}
	}
	return true
}

// questProgress is the current tally for one quest: dailies measure from today's snapshot.
func (st *userState) questProgress(q *questDef) int {
	c := questCounter(q.ID)
	have := st.Counts[c]
	if q.Daily {
		have -= st.QuestBase[c]
	}
	return max(0, have)
}

type questView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Have  int    `json:"have"`
	Need  int    `json:"need"`
	Gold  int    `json:"gold"`
	Item  string `json:"item,omitempty"`
	Count int    `json:"count,omitempty"`
	Done  bool   `json:"done"`
	Daily bool   `json:"daily"`
	Track string `json:"track,omitempty"`
	Tier  int    `json:"tier,omitempty"`
	Tiers int    `json:"tiers,omitempty"` // rungs on this ladder, so the panel can say 2/4
}

func (st *userState) questViews(now int64) []questView {
	st.rollDaily(now)
	out := make([]questView, 0, len(quests))
	for i := range quests {
		q := &quests[i]
		out = append(out, questView{ID: q.ID, Name: q.Name, Desc: q.Desc,
			Have: min(st.questProgress(q), q.Need), Need: q.Need, Gold: st.payout(q.Gold, now),
			Item: q.Item, Count: q.Amount, Done: st.QuestDone[q.ID], Daily: q.Daily,
			Track: q.Track, Tier: q.Tier, Tiers: trackTiers[q.Track]})
	}
	return out
}

// claimQuest pays out once. The done flag is the guard: a daily clears it at midnight, an
// achievement keeps it forever.
func claimQuest(st *userState, id string, now int64) (string, error) {
	q := questByID[id]
	if q == nil {
		return "", errors.New("nhiệm vụ không tồn tại")
	}
	st.rollDaily(now)
	if st.QuestDone[id] {
		return "", errors.New("nhiệm vụ này đã nhận thưởng")
	}
	if st.questProgress(q) < q.Need {
		return "", fmt.Errorf("chưa đủ điều kiện: %d/%d", st.questProgress(q), q.Need)
	}
	if st.QuestDone == nil {
		st.QuestDone = map[string]bool{}
	}
	st.QuestDone[id] = true
	gold := st.payout(q.Gold, now)
	st.Coin += gold
	note := fmt.Sprintf("🎯 %s: +%d vàng", q.Name, gold)
	if q.Item != "" && q.Amount > 0 {
		st.add(q.Item, q.Amount)
		note += fmt.Sprintf(", +%d %s", q.Amount, itemByID[q.Item].Name)
	}
	return note, nil
}
