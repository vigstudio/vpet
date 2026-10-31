package game

import (
	"errors"
	"math"
	"sort"
)

const seasonBoostCapBPS = 500

func (p *Engine) rankAlliances(s *warSeason) []warAllianceRank {
	rows := make([]warAllianceRank, 0, len(s.Scores))
	for id, score := range s.Scores {
		row := warAllianceRank{Alliance: id, Points: score.Points, Lands: len(warOwned(s, id))}
		if a, _ := p.loadAlliance(id); a != nil {
			row.Name, row.Tag = a.Name, a.Tag
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Points != rows[j].Points {
			return rows[i].Points > rows[j].Points
		}
		if rows[i].Lands != rows[j].Lands {
			return rows[i].Lands > rows[j].Lands
		}
		left, right := s.Scores[rows[i].Alliance].ReachedAt, s.Scores[rows[j].Alliance].ReachedAt
		if left != right {
			return left < right
		}
		return rows[i].Alliance < rows[j].Alliance
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows
}

func (p *Engine) rankPlayers(s *warSeason) []warPersonalRank {
	rows := make([]warPersonalRank, 0, len(s.Contribution))
	for id, score := range s.Contribution {
		rows = append(rows, warPersonalRank{User: id, Name: p.userName(id), Alliance: score.Alliance, Points: score.Points, Wins: score.Wins})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Points != rows[j].Points {
			return rows[i].Points > rows[j].Points
		}
		if rows[i].Wins != rows[j].Wins {
			return rows[i].Wins > rows[j].Wins
		}
		left, right := s.Contribution[rows[i].User].ReachedAt, s.Contribution[rows[j].User].ReachedAt
		if left != right {
			return left < right
		}
		return rows[i].User < rows[j].User
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows
}

func (p *Engine) finalizeSeason(s *warSeason) []string {
	if s.Status == "closed" {
		return nil
	}
	users := []string{}
	s.AllianceRanks = p.rankAlliances(s)
	s.PersonalRanks = p.rankPlayers(s)
	allianceBPS := map[string]int{}
	for i, value := range []int{400, 300, 200} {
		if i < len(s.AllianceRanks) {
			allianceBPS[s.AllianceRanks[i].Alliance] = value
		}
	}
	personalBPS := map[string]int{}
	for i, value := range []int{200, 150, 100} {
		if i < len(s.PersonalRanks) {
			personalBPS[s.PersonalRanks[i].User] = value
		}
	}
	until := s.End + int64(warSeasonDays)*24*60*60*1000
	for user, contribution := range s.Contribution {
		alliance := 0
		if pointer, err := p.loadAllianceMember(user); err == nil && s.End-pointer.JoinedAt >= allianceJoinLock.Milliseconds() {
			points := contribution.AlliancePoints[pointer.Alliance]
			if len(contribution.AlliancePoints) == 0 && contribution.Alliance == pointer.Alliance {
				points = contribution.Points
			}
			if points >= warAllianceRewardMin {
				alliance = allianceBPS[pointer.Alliance]
			}
		}
		personal := personalBPS[user]
		if total := min(seasonBoostCapBPS, alliance+personal); total > 0 {
			s.Entitlements[user] = &warEntitlement{User: user, BPS: total, AllianceBPS: alliance, PersonalBPS: personal, Until: until}
			users = append(users, user)
		}
	}
	s.Status = "closed"
	return users
}

func (p *Engine) claimSeasonReward(userID string, st *userState, now int64) (string, error) {
	current, err := p.loadWarCurrent()
	if err != nil || current.Previous == "" {
		return "", errors.New("chưa có thưởng mùa để nhận")
	}
	season, err := p.loadWarSeason(current.Previous)
	if err != nil || season == nil {
		return "", errors.New("không đọc được thưởng mùa")
	}
	reward := season.Entitlements[userID]
	if reward == nil {
		return "", errors.New("bạn không có thưởng mùa này")
	}
	if reward.Claimed {
		return "", errors.New("thưởng mùa đã nhận")
	}
	if now >= reward.Until {
		return "", errors.New("thưởng mùa đã hết hạn")
	}
	reward.Claimed = true
	st.SeasonBoostBPS = min(seasonBoostCapBPS, reward.BPS)
	st.SeasonBoostUntil = reward.Until
	st.SeasonBoostSource = season.ID
	if err := p.commitRows(map[string]any{userPrefix + userID: st, warSeasonKeyPrefix + season.ID: season}); err != nil {
		return "", err
	}
	return "Đã nhận buff sức mạnh mùa giải.", nil
}

func seasonBoostBPS(st *userState, now int64) int {
	if st == nil || now >= st.SeasonBoostUntil {
		return 0
	}
	return min(seasonBoostCapBPS, max(0, st.SeasonBoostBPS))
}

func applySeasonBoost(value float64, st *userState, now int64) float64 {
	return math.Round(value*float64(10000+seasonBoostBPS(st, now))/10000*10) / 10
}

func boostedSideWithBPS(side duelSide, bps int) duelSide {
	bps = min(seasonBoostCapBPS, max(0, bps))
	side.Atk = math.Round(side.Atk*float64(10000+bps)/10000*10) / 10
	side.Def = math.Round(side.Def*float64(10000+bps)/10000*10) / 10
	side.HP = math.Round(side.HP*float64(10000+bps)/10000*10) / 10
	return side
}

func boostedSideOf(user string, pt *pet, st *userState, now int64) duelSide {
	return boostedSideWithBPS(sideOf(user, pt), seasonBoostBPS(st, now))
}

func boostedPower(pt *pet, st *userState, now int64) int {
	return int(math.Round(float64(power(pt)) * float64(10000+seasonBoostBPS(st, now)) / 10000))
}

func boostedStatsOf(pt *pet, st *userState, now int64) statSheet {
	sheet := pt.statsOf()
	sheet.Atk = applySeasonBoost(sheet.Atk, st, now)
	sheet.Def = applySeasonBoost(sheet.Def, st, now)
	sheet.HP = applySeasonBoost(sheet.HP, st, now)
	sheet.Power = boostedPower(pt, st, now)
	return sheet
}
