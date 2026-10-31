package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	allianceIndexKey    = "alliance_index"
	allianceKeyPrefix   = "alliance_"
	allianceMemberKey   = "alliance_member_"
	allianceCreateCost  = 500
	allianceMaxMembers  = 20
	allianceCreateLevel = 10
)

var (
	allianceJoinLock  = 72 * time.Hour
	allianceInviteTTL = 72 * time.Hour
)

type allianceIndex struct {
	Rows []allianceIndexRow `json:"rows"`
}

type allianceIndexRow struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	NameKey string `json:"name_key"`
	Tag     string `json:"tag"`
}

type alliance struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Tag     string           `json:"tag"`
	Open    bool             `json:"open"`
	Created int64            `json:"created"`
	Members []allianceMember `json:"members"`
	Invites map[string]int64 `json:"invites,omitempty"`
}

type allianceMember struct {
	User     string `json:"user"`
	Role     string `json:"role"`
	JoinedAt int64  `json:"joined_at"`
}

type allianceMemberPointer struct {
	Alliance    string `json:"alliance,omitempty"`
	JoinedAt    int64  `json:"joined_at,omitempty"`
	LockedUntil int64  `json:"locked_until,omitempty"`
}

type allianceView struct {
	Mine        *allianceDetailView `json:"mine,omitempty"`
	Rows        []allianceListView  `json:"rows"`
	LockedUntil int64               `json:"locked_until,omitempty"`
}

type allianceDetailView struct {
	ID      string               `json:"id"`
	Name    string               `json:"name"`
	Tag     string               `json:"tag"`
	Open    bool                 `json:"open"`
	Created int64                `json:"created"`
	Role    string               `json:"role"`
	Members []allianceMemberView `json:"members"`
}

type allianceMemberView struct {
	User     string `json:"user"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	JoinedAt int64  `json:"joined_at"`
}

type allianceListView struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Open    bool   `json:"open"`
	Members int    `json:"members"`
	Invited bool   `json:"invited,omitempty"`
}

func cleanAllianceName(name string) (string, string, error) {
	name = strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	if n := utf8.RuneCountInString(name); n < 3 || n > 24 {
		return "", "", errors.New("tên Alliance phải dài 3–24 ký tự")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", "", errors.New("tên Alliance chứa ký tự không hợp lệ")
		}
	}
	return name, strings.ToLower(name), nil
}

func cleanAllianceTag(tag string) (string, error) {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	if len(tag) < 2 || len(tag) > 5 {
		return "", errors.New("tag Alliance phải dài 2–5 ký tự Latin hoặc số")
	}
	for _, r := range tag {
		if !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') {
			return "", errors.New("tag Alliance chỉ được dùng chữ Latin và số")
		}
	}
	return tag, nil
}

func (p *Engine) loadAllianceIndex() (*allianceIndex, error) {
	var idx allianceIndex
	if _, err := p.loadAllianceJSON(allianceIndexKey, &idx); err != nil {
		return nil, err
	}
	if idx.Rows == nil {
		idx.Rows = []allianceIndexRow{}
	}
	return &idx, nil
}

func (p *Engine) loadAlliance(id string) (*alliance, error) {
	if id == "" {
		return nil, nil
	}
	var a alliance
	found, err := p.loadAllianceJSON(allianceKeyPrefix+id, &a)
	if err != nil || !found {
		return nil, err
	}
	if a.Invites == nil {
		a.Invites = map[string]int64{}
	}
	return &a, nil
}

func (p *Engine) loadAllianceMember(userID string) (*allianceMemberPointer, error) {
	var member allianceMemberPointer
	_, err := p.loadAllianceJSON(allianceMemberKey+userID, &member)
	return &member, err
}

func (p *Engine) loadAllianceJSON(key string, dst any) (bool, error) {
	raw, err := p.backend.KVGet(key)
	if err != nil || raw == nil {
		return false, err
	}
	return true, json.Unmarshal(raw, dst)
}

func allianceMemberAt(a *alliance, userID string) int {
	for i := range a.Members {
		if a.Members[i].User == userID {
			return i
		}
	}
	return -1
}

func allianceCanManage(role string) bool { return role == "leader" || role == "officer" }

func (p *Engine) allianceState(userID string, now int64) allianceView {
	out := allianceView{Rows: []allianceListView{}}
	idx, err := p.loadAllianceIndex()
	if err != nil {
		p.backend.LogError("vpet: alliance index", "err", err.Error())
		return out
	}
	member, err := p.loadAllianceMember(userID)
	if err != nil {
		p.backend.LogError("vpet: alliance member", "err", err.Error())
		return out
	}
	out.LockedUntil = member.LockedUntil
	for _, row := range idx.Rows {
		a, loadErr := p.loadAlliance(row.ID)
		if loadErr != nil || a == nil {
			continue
		}
		out.Rows = append(out.Rows, allianceListView{ID: a.ID, Name: a.Name, Tag: a.Tag, Open: a.Open, Members: len(a.Members), Invited: a.Invites[userID] >= now})
		if a.ID != member.Alliance {
			continue
		}
		mine := &allianceDetailView{ID: a.ID, Name: a.Name, Tag: a.Tag, Open: a.Open, Created: a.Created, Members: []allianceMemberView{}}
		for _, m := range a.Members {
			mine.Members = append(mine.Members, allianceMemberView{User: m.User, Name: p.userName(m.User), Role: m.Role, JoinedAt: m.JoinedAt})
			if m.User == userID {
				mine.Role = m.Role
			}
		}
		sort.SliceStable(mine.Members, func(i, j int) bool {
			weight := func(role string) int {
				switch role {
				case "leader":
					return 0
				case "officer":
					return 1
				default:
					return 2
				}
			}
			return weight(mine.Members[i].Role) < weight(mine.Members[j].Role)
		})
		out.Mine = mine
	}
	return out
}

func (p *Engine) allianceAction(userID string, st *userState, action, id, name, tag, to string, open bool, now int64) (string, error) {
	pointer, err := p.loadAllianceMember(userID)
	if err != nil {
		return "", err
	}
	if action == "create" {
		return p.createAlliance(userID, st, pointer, name, tag, open, now)
	}
	if action == "join" {
		return p.joinAlliance(userID, pointer, id, now)
	}
	if pointer.Alliance == "" {
		return "", errors.New("bạn chưa thuộc Alliance nào")
	}
	a, err := p.loadAlliance(pointer.Alliance)
	if err != nil {
		return "", err
	}
	if a == nil {
		return "", errors.New("Alliance không còn tồn tại")
	}
	at := allianceMemberAt(a, userID)
	if at < 0 {
		return "", errors.New("thành viên Alliance không đồng bộ")
	}
	role := a.Members[at].Role
	switch action {
	case "invite":
		return p.inviteAllianceMember(userID, role, a, to, now)
	case "leave":
		if role == "leader" {
			return "", errors.New("leader phải chuyển quyền hoặc giải tán Alliance")
		}
		a.Members = append(a.Members[:at], a.Members[at+1:]...)
		pointer.Alliance, pointer.JoinedAt, pointer.LockedUntil = "", 0, now+allianceJoinLock.Milliseconds()
		if err := p.commitRows(map[string]any{allianceKeyPrefix + a.ID: a, allianceMemberKey + userID: pointer}); err != nil {
			return "", err
		}
		return "Đã rời Alliance; bạn có thể gia nhập lại sau 72 giờ.", nil
	case "kick":
		return p.kickAllianceMember(userID, role, a, id, now)
	case "promote", "demote":
		return p.changeAllianceRole(userID, role, a, id, action)
	case "transfer":
		if role != "leader" {
			return "", errors.New("chỉ leader được chuyển quyền")
		}
		targetAt := allianceMemberAt(a, id)
		if targetAt < 0 || id == userID {
			return "", errors.New("không tìm thấy người nhận quyền")
		}
		a.Members[at].Role = "officer"
		a.Members[targetAt].Role = "leader"
		if err := p.commitRows(map[string]any{allianceKeyPrefix + a.ID: a}); err != nil {
			return "", err
		}
		return "Đã chuyển quyền leader.", nil
	case "open":
		if !allianceCanManage(role) {
			return "", errors.New("chỉ leader hoặc officer được đổi trạng thái")
		}
		if a.Open == open {
			return "", errors.New("trạng thái Alliance không thay đổi")
		}
		a.Open = open
		if err := p.commitRows(map[string]any{allianceKeyPrefix + a.ID: a}); err != nil {
			return "", err
		}
		return "Đã đổi trạng thái Alliance.", nil
	case "disband":
		return p.disbandAlliance(userID, role, pointer, a, now)
	default:
		return "", errors.New("thao tác Alliance không hợp lệ")
	}
}

func (p *Engine) createAlliance(userID string, st *userState, pointer *allianceMemberPointer, name, tag string, open bool, now int64) (string, error) {
	if pointer.Alliance != "" {
		return "", errors.New("bạn đã thuộc một Alliance")
	}
	if pointer.LockedUntil > now {
		return "", errors.New("bạn đang trong thời gian chờ 72 giờ")
	}
	eligible := false
	for _, pt := range st.Pets {
		if !pt.Dead && max(pt.Lv, 1) >= allianceCreateLevel {
			eligible = true
			break
		}
	}
	if !eligible {
		return "", errors.New("cần ít nhất một thú cấp 10 để tạo Alliance")
	}
	if st.Coin < allianceCreateCost {
		return "", fmt.Errorf("cần %d vàng để tạo Alliance", allianceCreateCost)
	}
	cleanName, nameKey, err := cleanAllianceName(name)
	if err != nil {
		return "", err
	}
	cleanTag, err := cleanAllianceTag(tag)
	if err != nil {
		return "", err
	}
	idx, err := p.loadAllianceIndex()
	if err != nil {
		return "", err
	}
	for _, row := range idx.Rows {
		if row.NameKey == nameKey {
			return "", errors.New("tên Alliance đã được dùng")
		}
		if row.Tag == cleanTag {
			return "", errors.New("tag Alliance đã được dùng")
		}
	}
	id := newID()
	a := &alliance{ID: id, Name: cleanName, Tag: cleanTag, Open: open, Created: now, Members: []allianceMember{{User: userID, Role: "leader", JoinedAt: now}}, Invites: map[string]int64{}}
	idx.Rows = append(idx.Rows, allianceIndexRow{ID: id, Name: cleanName, NameKey: nameKey, Tag: cleanTag})
	pointer.Alliance, pointer.JoinedAt, pointer.LockedUntil = id, now, 0
	st.Coin -= allianceCreateCost
	if err := p.commitRows(map[string]any{allianceIndexKey: idx, allianceKeyPrefix + id: a, allianceMemberKey + userID: pointer, userPrefix + userID: st}); err != nil {
		return "", err
	}
	return fmt.Sprintf("Đã thành lập [%s] %s.", cleanTag, cleanName), nil
}

func (p *Engine) joinAlliance(userID string, pointer *allianceMemberPointer, id string, now int64) (string, error) {
	if pointer.Alliance != "" {
		return "", errors.New("bạn đã thuộc một Alliance")
	}
	if pointer.LockedUntil > now {
		return "", errors.New("bạn đang trong thời gian chờ 72 giờ")
	}
	if id == "" {
		return "", errors.New("thiếu Alliance cần gia nhập")
	}
	idx, err := p.loadAllianceIndex()
	if err != nil {
		return "", err
	}
	found := false
	for _, row := range idx.Rows {
		if row.ID == id {
			found = true
			break
		}
	}
	if !found {
		return "", errors.New("Alliance không còn tồn tại")
	}
	a, err := p.loadAlliance(id)
	if err != nil {
		return "", err
	}
	if a == nil {
		return "", errors.New("Alliance không còn tồn tại")
	}
	if len(a.Members) >= allianceMaxMembers {
		return "", errors.New("Alliance đã đủ 20 thành viên")
	}
	if !a.Open && a.Invites[userID] < now {
		return "", errors.New("Alliance này cần lời mời")
	}
	delete(a.Invites, userID)
	a.Members = append(a.Members, allianceMember{User: userID, Role: "member", JoinedAt: now})
	pointer.Alliance, pointer.JoinedAt, pointer.LockedUntil = a.ID, now, 0
	if err := p.commitRows(map[string]any{allianceKeyPrefix + a.ID: a, allianceMemberKey + userID: pointer}); err != nil {
		return "", err
	}
	return fmt.Sprintf("Đã gia nhập [%s] %s.", a.Tag, a.Name), nil
}

func (p *Engine) inviteAllianceMember(actor, role string, a *alliance, username string, now int64) (string, error) {
	if !allianceCanManage(role) {
		return "", errors.New("chỉ leader hoặc officer được mời")
	}
	target, err := p.backend.UserByUsername(strings.TrimSpace(username))
	if err != nil || target == nil || target.Disabled {
		return "", errors.New("không tìm thấy người chơi")
	}
	if target.ID == actor {
		return "", errors.New("bạn đã ở trong Alliance")
	}
	pointer, err := p.loadAllianceMember(target.ID)
	if err != nil {
		return "", err
	}
	if pointer.Alliance != "" {
		return "", errors.New("người chơi đã thuộc Alliance khác")
	}
	if len(a.Members) >= allianceMaxMembers {
		return "", errors.New("Alliance đã đủ 20 thành viên")
	}
	a.Invites[target.ID] = now + allianceInviteTTL.Milliseconds()
	if err := p.commitRows(map[string]any{allianceKeyPrefix + a.ID: a}); err != nil {
		return "", err
	}
	p.backend.Notify(target.ID, "alliance", fmt.Sprintf("Bạn được mời vào Alliance [%s] %s.", a.Tag, a.Name), "world", "")
	return "Đã gửi lời mời.", nil
}

func (p *Engine) kickAllianceMember(actor, role string, a *alliance, target string, now int64) (string, error) {
	if !allianceCanManage(role) {
		return "", errors.New("chỉ leader hoặc officer được đuổi thành viên")
	}
	if target == actor {
		return "", errors.New("hãy dùng thao tác rời Alliance")
	}
	at := allianceMemberAt(a, target)
	if at < 0 {
		return "", errors.New("không tìm thấy thành viên")
	}
	targetRole := a.Members[at].Role
	if targetRole == "leader" || (role == "officer" && targetRole != "member") {
		return "", errors.New("bạn không có quyền đuổi thành viên này")
	}
	pointer, err := p.loadAllianceMember(target)
	if err != nil {
		return "", err
	}
	a.Members = append(a.Members[:at], a.Members[at+1:]...)
	pointer.Alliance, pointer.JoinedAt, pointer.LockedUntil = "", 0, now+allianceJoinLock.Milliseconds()
	if err := p.commitRows(map[string]any{allianceKeyPrefix + a.ID: a, allianceMemberKey + target: pointer}); err != nil {
		return "", err
	}
	p.notify(target, "alliance", fmt.Sprintf("Bạn đã bị rời khỏi Alliance [%s] %s.", a.Tag, a.Name))
	return "Đã loại thành viên; họ có thể gia nhập lại sau 72 giờ.", nil
}

func (p *Engine) changeAllianceRole(actor, role string, a *alliance, target, action string) (string, error) {
	if role != "leader" {
		return "", errors.New("chỉ leader được đổi cấp bậc")
	}
	at := allianceMemberAt(a, target)
	if at < 0 || target == actor {
		return "", errors.New("không tìm thấy thành viên phù hợp")
	}
	want := "officer"
	if action == "demote" {
		want = "member"
	}
	if a.Members[at].Role == want || a.Members[at].Role == "leader" {
		return "", errors.New("cấp bậc không thay đổi")
	}
	a.Members[at].Role = want
	if err := p.commitRows(map[string]any{allianceKeyPrefix + a.ID: a}); err != nil {
		return "", err
	}
	return "Đã cập nhật cấp bậc thành viên.", nil
}

func (p *Engine) disbandAlliance(userID, role string, pointer *allianceMemberPointer, a *alliance, now int64) (string, error) {
	if role != "leader" {
		return "", errors.New("chỉ leader được giải tán Alliance")
	}
	if len(a.Members) != 1 {
		return "", errors.New("phải chuyển hoặc loại hết thành viên trước khi giải tán")
	}
	idx, err := p.loadAllianceIndex()
	if err != nil {
		return "", err
	}
	for i := range idx.Rows {
		if idx.Rows[i].ID == a.ID {
			idx.Rows = append(idx.Rows[:i], idx.Rows[i+1:]...)
			break
		}
	}
	pointer.Alliance, pointer.JoinedAt, pointer.LockedUntil = "", 0, now+allianceJoinLock.Milliseconds()
	if err := p.commitRows(map[string]any{allianceIndexKey: idx, allianceKeyPrefix + a.ID: nil, allianceMemberKey + userID: pointer}); err != nil {
		return "", err
	}
	return "Đã giải tán Alliance.", nil
}
