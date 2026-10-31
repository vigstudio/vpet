package game

import "fmt"

var campaignNamesByLocale = map[string]map[string][]string{
	"en": {
		"vuon":   {"Field edge", "Flower garden", "Bamboo grove", "Old well", "Meadow", "Garden king", "Mist marsh", "Lost garden", "Thorn maze", "Wood shrine", "Root gate", "Awakened elder tree"},
		"rung":   {"Forest edge", "Blue stream", "Thornwood", "Wolf den", "Great tree", "Forest lord", "Moon forest", "Ancient tracks", "Silver falls", "Wolf shrine", "Deep canopy", "Wolf sovereign"},
		"hang":   {"Cave mouth", "Bat cave", "Stone vein", "Spider nest", "Deep well", "Dark cave king", "Crystal tunnel", "Venom vein", "Chasm", "Stone gate", "Ancient tomb", "Abyss lord"},
		"nui":    {"Foothills", "Wind pass", "Frozen lake", "Snow cliff", "White peak", "Ice tyrant", "Ice bridge", "Blizzard", "Ice citadel", "Wind shrine", "Snow throne", "Ice dragon"},
		"nuilua": {"Ash fields", "Fire river", "Inferno", "Lava citadel", "Dragon nest", "Dragon king", "Ash furnace", "Lava bridge", "Fire shrine", "Hell gate", "Flame throne", "Dragon emperor"},
	},
	"ja": {
		"vuon":   {"畑の端", "花園", "竹林", "古井戸", "草原", "庭園の王", "霧の沼", "忘れられた庭", "茨の迷宮", "木の祠", "根の門", "目覚めた古木"},
		"rung":   {"森の入口", "青い小川", "茨の森", "狼の巣", "大樹", "森の主", "月の森", "古代の足跡", "銀の滝", "狼の祠", "深い樹冠", "狼王"},
		"hang":   {"洞窟の入口", "蝙蝠の洞窟", "岩脈", "蜘蛛の巣", "深井戸", "暗洞の王", "水晶坑道", "毒の鉱脈", "深淵", "石の門", "古墳", "深淵の主"},
		"nui":    {"山麓", "風の峠", "氷の湖", "雪の崖", "白い峰", "氷の暴君", "氷の橋", "吹雪", "氷の城", "風の祠", "雪の玉座", "氷竜"},
		"nuilua": {"灰の地", "火の川", "煉獄", "溶岩城", "竜の巣", "竜王", "灰の炉", "溶岩の橋", "火の祠", "地獄の門", "炎の玉座", "竜帝"},
	},
}

func translatedStageName(locale, mapID string, stage int) string {
	if rows := campaignNamesByLocale[locale][mapID]; stage > 0 && stage <= len(rows) {
		return rows[stage-1]
	}
	return campaignStageName(mapID, stage)
}

func translatedTowerName(locale string, floor int) string {
	if locale == "vi" {
		return towerName(floor)
	}
	if towerIsBoss(floor) {
		names := []string{"Tenth-floor guardian", "Tower sword spirit", "Shadow of thirty", "Sealed war god", "Nameless tower lord"}
		if locale == "ja" {
			names = []string{"十階の守護者", "塔の剣霊", "三十階の影", "封印の戦神", "名もなき塔主"}
		}
		return names[floor/10-1]
	}
	names := []string{"Tower guard", "Stone sentinel", "Wandering shadow", "Tower mage", "Wild knight"}
	if locale == "ja" {
		names = []string{"塔の衛兵", "石の守護者", "彷徨う影", "塔の魔術師", "野生の騎士"}
		return fmt.Sprintf("%d階の%s", floor, names[(floor-1)%5])
	}
	return fmt.Sprintf("%s, floor %d", names[(floor-1)%5], floor)
}
