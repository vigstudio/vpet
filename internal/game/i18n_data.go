package game

import (
	"fmt"
	"strconv"
	"strings"
)

// Catalog translations. Item ids, skill ids and map ids are the stable keys; only the words
// shown to a player change. Vietnamese stays in the Go tables as the source of truth, so a
// missing translation falls back to it rather than to a blank label.

type words struct{ Name, Desc string }

type catalogWords struct {
	Items    map[string]words
	Skills   map[string]words
	Maps     map[string]words
	Slots    map[string]string
	Rarities []string
}

// supportedLocales is what the office actually uses; everything else reads Vietnamese.
var supportedLocales = []string{"vi", "en", "ja"}

func localeOf(raw string) string {
	if len(raw) >= 2 {
		switch raw[:2] {
		case "en":
			return "en"
		case "ja":
			return "ja"
		}
	}
	return "vi"
}

var catalogByLocale = map[string]catalogWords{
	"en": {
		Items: map[string]words{
			"kiemhongmong":       {Name: "Primordial Sword"},
			"giaphongmong":       {Name: "Primordial Armour"},
			"muhongmong":         {Name: "Primordial Helm"},
			"ganghongmong":       {Name: "Primordial Gauntlets"},
			"hiahongmong":        {Name: "Primordial Boots"},
			"nhanhongmong":       {Name: "Primordial Ring"},
			"khienhongmong":      {Name: "Primordial Shield"},
			"aohongmong":         {Name: "Primordial Cloak"},
			"tinhhon":            {"Soul Essence", "Bound material: 3 forge Supreme gear. Defeated world boss: 15% with at least 60% of average contribution; tier-3 territory victory: 5%. Cannot be bought, sold or traded"},
			"tui":                {"Food bag", "Opens into 1-2 random items"},
			"tui_y":              {"Medic bag", "Medicine and herbs"},
			"tui_vukhi":          {"Weapon crate", "One random piece of gear"},
			"tui_vang":           {"Gold chest", "Gold and a rare item"},
			"com":                {"Rice ball", "Filling and gentle"},
			"banhmi":             {"Banh mi", "Solid and filling"},
			"pho":                {"Pho", "Delicious! The pet leaps for joy"},
			"sushi":              {"Sushi", "A treat, very cheering"},
			"banhkem":            {"Cake", "Sweet, maximum joy"},
			"trasua":             {"Bubble tea", "Cheerful but not filling"},
			"ot":                 {"Chilli", "Spicy! Repeated heat hurts health"},
			"micay":              {"Spicy noodles", "Filling but burning hot"},
			"saurieng":           {"Durian", "Fragrant to some, foul to others"},
			"mamtom":             {"Shrimp paste", "Pungent, draws flies"},
			"trungthoi":          {"Rotten egg", "Smelly, upsets the stomach"},
			"nam":                {"Strange mushroom", "A gamble: high or poisoned"},
			"caphe":              {"Iced coffee", "Wide awake, never stops moving"},
			"thuoc":              {"Cold medicine", "Cures illness. Harmful when healthy"},
			"vitamin":            {"Vitamins", "Restores health"},
			"banggac":            {"Bandage", "Restores 18 health, use after a hunt"},
			"thuocbo":            {"Tonic", "Restores 40 health and 15 joy"},
			"canhtien":           {"Fairy soup", "Full health, cures illness"},
			"nuocthanh":          {"Holy water", "Restores 70 health, clears all ailments"},
			"long":               {"Phoenix feather", "Very rare: revives a dead pet"},
			"the_skin":           {"Reshape card", "Redesign one pet, keeping its stats"},
			"the_slot":           {"Stable card", "Raise one more pet (up to 4)"},
			"the_nghi":           {"Holiday card", "No hunger or sadness for 72 hours"},
			"nonla":              {"Conical hat", "Headwear. Swapping returns the old one"},
			"kinh":               {"Sunglasses", "Accessory: effortlessly cool"},
			"no":                 {"Red bow", "Accessory: adorable"},
			"khan":               {"Scarf", "Accessory: cosy"},
			"sach":               {"Martial manual", "Swap one chosen skill for another, keeping its trained rank"},
			"the_ten":            {"Rename card", "Rename a pet, keeping looks and stats"},
			"the_quen":           {"Skill reset card", "Clears skill ranks, refunds half the stones"},
			"phieu_hoan_diem":    {"Respec voucher", "Returns every attack, defence and vitality point"},
			"nuoc_tang_luc":      {"Energy drink", "Restores 5 adventure energy"},
			"tra_sam":            {"Ginseng tea", "Restores 12 adventure energy"},
			"ruou_than":          {"Divine wine", "Fully restores adventure energy"},
			"the_nang_luong":     {"Vigour card", "Permanently +5 max energy (up to +30)"},
			"tinh_the_tiem_nang": {"Potential crystal", "Very rare: 1 or 2 permanent potential points"},
			"banhtet":            {"Tet cake", "Filling, a new-year treat"},
			"chegiai":            {"Cooling dessert", "Soothing, clears the burn"},
			"nuocmia":            {"Sugarcane juice", "Sweet and refreshing"},
			"chaogiai":           {"Rice porridge", "Restores 30 health, settles the stomach"},
			"tinhdau":            {"Cajuput oil", "Restores 22 health and clears the stink"},
			"sammy":              {"Ngoc Linh ginseng", "Restores 55 health and 20 joy"},
			"mu_bao":             {"Helmet", "Accessory: safety first"},
			"vongtay":            {"Grass bracelet", "Accessory: woven from meadow grass"},
			"ngoc":               {"Skill stone", "Raises one duel skill by a rank"},
			"da":                 {"Hide", "Scrap from a hunt"},
			"thaomoc":            {"Herb", "Scrap from a hunt"},
			"mathu":              {"Beast blood", "Scrap from a hunt"},
			"dao":                {Name: "Rusty knife"},
			"kiem":               {Name: "Iron sword"},
			"dinhba":             {Name: "Trident"},
			"gaythan":            {Name: "Divine staff"},
			"aovai":              {Name: "Cloth shirt"},
			"giapda":             {Name: "Leather armour"},
			"giapthep":           {Name: "Steel armour"},
			"longbao":            {Name: "Dragon robe"},
			"nonre":              {Name: "Straw hat"},
			"muda":               {Name: "Leather cap"},
			"muthep":             {Name: "Steel helm"},
			"vuongmien":          {Name: "Crown"},
			"gangvai":            {Name: "Cloth gloves"},
			"gangda":             {Name: "Leather gloves"},
			"gangthep":           {Name: "Steel gauntlets"},
			"gangrong":           {Name: "Dragon gauntlets"},
			"depco":              {Name: "Grass sandals"},
			"giayda":             {Name: "Leather boots"},
			"giaythep":           {Name: "Steel boots"},
			"hiabay":             {Name: "Flying boots"},
			"nhandong":           {Name: "Copper ring"},
			"chuoingoc":          {Name: "Jade beads"},
			"nhanbac":            {Name: "Silver ring"},
			"nhanrong":           {Name: "Dragon ring"},
			"motnap":             {Name: "Pot lid"},
			"khiengo":            {Name: "Wooden shield"},
			"khiensat":           {Name: "Iron shield"},
			"khienbac":           {Name: "Silver shield"},
			"aotoi":              {Name: "Palm cape"},
			"aogam":              {Name: "Brocade coat"},
			"aolong":             {Name: "Feather cloak"},
			"aosao":              {Name: "Starlight cloak"},
			"kiemrong":           {Name: "Dragon sword"},
			"giaprong":           {Name: "Dragon armour"},
			"muthan":             {Name: "Divine helm"},
			"gangthan":           {Name: "Divine gauntlets"},
			"hiathan":            {Name: "Divine boots"},
			"nhanthan":           {Name: "Divine ring"},
			"khienthan":          {Name: "Divine shield"},
			"aothan":             {Name: "Divine cloak"},
		},
		Skills: map[string]words{
			"lua":   {"Flame breath", "Deals 14 damage"},
			"cuoc":  {"Combo kick", "Kicks 2-3 times for 6 each"},
			"hoi":   {"Heal", "Restores 14 HP"},
			"khien": {"Iron shield", "Halves the next blow"},
			"an":    {"Vanish", "Dodges the next blow entirely"},
			"bom":   {"Stink bomb", "6 damage, the foe is likely to miss next turn"},
			"set":   {"Thunder", "22 damage but misses half the time"},
			"hut":   {"Drain", "10 damage, heals half of it back"},
		},
		Maps: map[string]words{
			"vuon":   {"Back garden", "Safe, suits a young pet"},
			"rung":   {"Deep forest", "Herbs and wild beasts"},
			"hang":   {"Dark cave", "Pitch black, easy to get hurt"},
			"nui":    {"Snow mountain", "Biting cold, fine gear at the peak"},
			"nuilua": {"Volcano", "The most dangerous, golden treasure"},
		},
		Slots: map[string]string{
			"vukhi":   "Weapon",
			"giap":    "Armour",
			"mu":      "Helmet",
			"gang":    "Gloves",
			"giay":    "Boots",
			"nhan":    "Ring",
			"khien":   "Shield",
			"aokhoac": "Cloak",
		},
		Rarities: []string{"Common", "Rare", "Epic", "Golden", "Mythic", "Divine", "Supreme"},
	},
	"ja": {
		Items: map[string]words{
			"kiemhongmong":       {Name: "鴻蒙の剣"},
			"giaphongmong":       {Name: "鴻蒙の鎧"},
			"muhongmong":         {Name: "鴻蒙の兜"},
			"ganghongmong":       {Name: "鴻蒙の篭手"},
			"hiahongmong":        {Name: "鴻蒙の靴"},
			"nhanhongmong":       {Name: "鴻蒙の指輪"},
			"khienhongmong":      {Name: "鴻蒙の盾"},
			"aohongmong":         {Name: "鴻蒙のマント"},
			"tinhhon":            {"精魂", "専用素材：至高装備の鍛造に3個。討伐ボスで平均の60%以上貢献すると15%、段階3領土勝利で5%。売買・交換不可"},
			"tui":                {"食料袋", "ランダムで1〜2個"},
			"tui_y":              {"救急袋", "薬と薬草"},
			"tui_vukhi":          {"武器箱", "装備が1つ"},
			"tui_vang":           {"金の宝箱", "金貨とレア品"},
			"com":                {"おにぎり", "腹持ちがよく優しい味"},
			"banhmi":             {"バインミー", "しっかり満腹"},
			"pho":                {"フォー", "美味しくて大喜び"},
			"sushi":              {"寿司", "ごちそう、ご機嫌に"},
			"banhkem":            {"ケーキ", "甘くて最高に嬉しい"},
			"trasua":             {"タピオカミルクティー", "嬉しいが腹持ちは悪い"},
			"ot":                 {"唐辛子", "辛い！続けると体に悪い"},
			"micay":              {"激辛ラーメン", "満腹だが激辛"},
			"saurieng":           {"ドリアン", "好みが分かれる香り"},
			"mamtom":             {"エビ味噌", "強烈な匂い"},
			"trungthoi":          {"腐った卵", "臭くてお腹を壊しやすい"},
			"nam":                {"謎のキノコ", "当たりか毒か"},
			"caphe":              {"ベトナムコーヒー", "目が冴えて動き回る"},
			"thuoc":              {"風邪薬", "病気を治す。健康時は害"},
			"vitamin":            {"ビタミン", "体力を回復"},
			"banggac":            {"包帯", "体力18回復、狩りの後に"},
			"thuocbo":            {"強壮剤", "体力40と喜び15回復"},
			"canhtien":           {"仙人スープ", "全回復し病気も治す"},
			"nuocthanh":          {"聖水", "体力70回復、状態異常解除"},
			"long":               {"不死鳥の羽", "非常に貴重：復活させる"},
			"the_skin":           {"変身カード", "能力はそのままで姿を変更"},
			"the_slot":           {"小屋カード", "ペットをもう1匹（最大4）"},
			"the_nghi":           {"休暇カード", "72時間お腹も気分も減らない"},
			"nonla":              {"ノンラー", "頭の装飾品"},
			"kinh":               {"サングラス", "かっこいい装飾品"},
			"no":                 {"赤いリボン", "かわいい装飾品"},
			"khan":               {"マフラー", "暖かい装飾品"},
			"sach":               {"武術書", "選んだスキル1つを別のスキルに変更、熟練度は維持"},
			"the_ten":            {"改名カード", "姿と能力はそのまま改名"},
			"the_quen":           {"スキル初期化カード", "スキル熟練を戻し半分返還"},
			"phieu_hoan_diem":    {"再振りチケット", "振った能力値を全て返還"},
			"nuoc_tang_luc":      {"エナジードリンク", "冒険の気力を5回復"},
			"tra_sam":            {"高麗人参茶", "冒険の気力を12回復"},
			"ruou_than":          {"神酒", "冒険の気力を全回復"},
			"the_nang_luong":     {"気力カード", "最大気力+5（最大+30）"},
			"tinh_the_tiem_nang": {"潜在結晶", "潜在ポイント1〜2を永久付与"},
			"banhtet":            {"バインテト", "腹持ちのよい正月料理"},
			"chegiai":            {"涼しいチェー", "辛さを和らげる"},
			"nuocmia":            {"サトウキビジュース", "甘くて爽やか"},
			"chaogiai":           {"お粥", "体力30回復、胃に優しい"},
			"tinhdau":            {"カユプテ油", "体力22回復、臭い解除"},
			"sammy":              {"ゴックリン人参", "体力55と喜び20回復"},
			"mu_bao":             {"ヘルメット", "安全第一の装飾品"},
			"vongtay":            {"草の腕輪", "草で編んだ装飾品"},
			"ngoc":               {"スキル玉", "対戦スキルを1段階強化"},
			"da":                 {"皮", "狩りの端材"},
			"thaomoc":            {"薬草", "狩りの端材"},
			"mathu":              {"獣の血", "狩りの端材"},
			"dao":                {Name: "さびた小刀"},
			"kiem":               {Name: "鉄の剣"},
			"dinhba":             {Name: "三叉槍"},
			"gaythan":            {Name: "神の杖"},
			"aovai":              {Name: "布の服"},
			"giapda":             {Name: "革の鎧"},
			"giapthep":           {Name: "鋼の鎧"},
			"longbao":            {Name: "龍袍"},
			"nonre":              {Name: "麦わら帽子"},
			"muda":               {Name: "革の帽子"},
			"muthep":             {Name: "鋼の兜"},
			"vuongmien":          {Name: "王冠"},
			"gangvai":            {Name: "布の手袋"},
			"gangda":             {Name: "革の手袋"},
			"gangthep":           {Name: "鋼の篭手"},
			"gangrong":           {Name: "龍の篭手"},
			"depco":              {Name: "草履"},
			"giayda":             {Name: "革のブーツ"},
			"giaythep":           {Name: "鋼のブーツ"},
			"hiabay":             {Name: "飛翔の靴"},
			"nhandong":           {Name: "銅の指輪"},
			"chuoingoc":          {Name: "翡翠の数珠"},
			"nhanbac":            {Name: "銀の指輪"},
			"nhanrong":           {Name: "龍の指輪"},
			"motnap":             {Name: "鍋のふた"},
			"khiengo":            {Name: "木の盾"},
			"khiensat":           {Name: "鉄の盾"},
			"khienbac":           {Name: "銀の盾"},
			"aotoi":              {Name: "ヤシのマント"},
			"aogam":              {Name: "錦の上着"},
			"aolong":             {Name: "羽毛のマント"},
			"aosao":              {Name: "星光のマント"},
			"kiemrong":           {Name: "龍の剣"},
			"giaprong":           {Name: "龍の鎧"},
			"muthan":             {Name: "神の兜"},
			"gangthan":           {Name: "神の篭手"},
			"hiathan":            {Name: "神の靴"},
			"nhanthan":           {Name: "神の指輪"},
			"khienthan":          {Name: "神の盾"},
			"aothan":             {Name: "神のマント"},
		},
		Skills: map[string]words{
			"lua":   {"火炎", "14ダメージ"},
			"cuoc":  {"連続蹴り", "2〜3回、各6ダメージ"},
			"hoi":   {"回復", "HPを14回復"},
			"khien": {"鉄の盾", "次の一撃を半減"},
			"an":    {"透明化", "次の一撃を完全回避"},
			"bom":   {"臭い爆弾", "6ダメージ、相手は次に外しやすい"},
			"set":   {"雷撃", "22ダメージだが半分外す"},
			"hut":   {"吸血", "10ダメージ、半分回復"},
		},
		Maps: map[string]words{
			"vuon":   {"裏庭", "安全で幼いペット向き"},
			"rung":   {"深い森", "薬草と野生の獣"},
			"hang":   {"暗い洞窟", "真っ暗で怪我しやすい"},
			"nui":    {"雪山", "極寒、頂には良い装備"},
			"nuilua": {"火山", "最も危険、黄金の宝"},
		},
		Slots: map[string]string{
			"vukhi":   "武器",
			"giap":    "鎧",
			"mu":      "兜",
			"gang":    "手袋",
			"giay":    "靴",
			"nhan":    "指輪",
			"khien":   "盾",
			"aokhoac": "マント",
		},
		Rarities: []string{"並", "レア", "エピック", "黄金", "神話", "神級", "至高"},
	},
}

// Derived records are translated after their progress and bonuses are calculated, so the
// gameplay functions remain language-neutral and tests keep using the Vietnamese source data.
var questWords = map[string]map[string]words{
	"en": {
		"d_feed":     {"Feed 3 times", "Daily care matters"},
		"d_hunt":     {"Complete 2 hunts", "Bring something home"},
		"d_stage":    {"Clear 1 campaign stage", "Take one more step"},
		"d_wild":     {"Defeat 3 wild beasts", "Practise in the wild"},
		"d_duel":     {"Fight 1 duel", "Test yourself against a colleague"},
		"d_boss":     {"Attack the world boss once", "Join the office effort"},
		"a_hunt25":   {"Skilled hunter", "Complete 25 hunts"},
		"a_hunt100":  {"Veteran hunter", "Complete 100 hunts"},
		"a_hunt300":  {"Master of the hunt", "Complete 300 hunts"},
		"a_hunt800":  {"Legend of the long road", "Complete 800 hunts"},
		"a_wild50":   {"Wild-beast nemesis", "Defeat 50 wild beasts"},
		"a_wild200":  {"Terror of the wilds", "Defeat 200 wild beasts"},
		"a_wild600":  {"Lord of beasts", "Defeat 600 wild beasts"},
		"a_stage10":  {"Explorer", "Clear 10 campaign stages"},
		"a_stage30":  {"Campaign conqueror", "Clear 30 campaign stages"},
		"a_stage90":  {"Hero of Skyhold", "Clear 90 campaign stages"},
		"a_stage200": {"Great one of Skyhold", "Clear 200 campaign stages"},
		"a_duel10":   {"Fighter", "Win 10 duels"},
		"a_duel40":   {"Veteran blade", "Win 40 duels"},
		"a_duel120":  {"Arena champion", "Win 120 duels"},
		"a_forge":    {"Blacksmith", "Forge gear 10 times"},
		"a_forge40":  {"Seasoned blacksmith", "Forge gear 40 times"},
		"a_forge120": {"Forge grandmaster", "Forge gear 120 times"},
		"a_crush":    {"Gem crusher", "Crush 30 spare items"},
		"a_crush150": {"Expert crusher", "Crush 150 spare items"},
		"a_crush500": {"Shard artisan", "Crush 500 spare items"},
		"a_feed50":   {"Pet carer", "Feed your pet 50 times"},
		"a_feed250":  {"Constant companion", "Feed your pet 250 times"},
		"a_feed800":  {"Soulmate", "Feed your pet 800 times"},
		"a_boss10":   {"Contributor", "Attack the world boss 10 times"},
		"a_boss50":   {"Boss hunter", "Attack the world boss 50 times"},
		"a_boss150":  {"Bane of bosses", "Attack the world boss 150 times"},
		"a_arena30":  {"Keen duellist", "Fight 30 duels"},
		"a_arena120": {"Arena regular", "Fight 120 duels"},
		"a_arena400": {"Soul of the arena", "Fight 400 duels"},
	},
	"ja": {
		"d_feed":     {"3回食事をあげる", "毎日のお世話"},
		"d_hunt":     {"狩りを2回完了", "戦利品を持ち帰る"},
		"d_stage":    {"キャンペーンを1ステージ突破", "一歩前進"},
		"d_wild":     {"野生の獣を3体倒す", "野外で腕を磨く"},
		"d_duel":     {"対戦を1回行う", "同僚に挑戦"},
		"d_boss":     {"ワールドボスを1回攻撃", "みんなでダメージを与える"},
		"a_hunt25":   {"熟練ハンター", "狩りを25回完了"},
		"a_hunt100":  {"ベテランハンター", "狩りを100回完了"},
		"a_hunt300":  {"狩りの達人", "狩りを300回完了"},
		"a_hunt800":  {"長路の伝説", "狩りを800回完了"},
		"a_wild50":   {"野生の天敵", "野生の獣を50体倒す"},
		"a_wild200":  {"荒野の魔神", "野生の獣を200体倒す"},
		"a_wild600":  {"獣の王", "野生の獣を600体倒す"},
		"a_stage10":  {"探検家", "キャンペーンを10ステージ突破"},
		"a_stage30":  {"キャンペーン制覇", "キャンペーンを30ステージ突破"},
		"a_stage90":  {"天島の英雄", "キャンペーンを90ステージ突破"},
		"a_stage200": {"天島の偉人", "キャンペーンを200ステージ突破"},
		"a_duel10":   {"闘士", "対戦で10勝"},
		"a_duel40":   {"熟練の剣士", "対戦で40勝"},
		"a_duel120":  {"闘技場の優勝者", "対戦で120勝"},
		"a_forge":    {"鍛冶職人", "装備を10回鍛造"},
		"a_forge40":  {"熟練の鍛冶師", "装備を40回鍛造"},
		"a_forge120": {"鍛冶の大師", "装備を120回鍛造"},
		"a_crush":    {"宝玉砕き", "余り装備を30個砕く"},
		"a_crush150": {"砕きの名手", "余り装備を150個砕く"},
		"a_crush500": {"欠片の職人", "余り装備を500個砕く"},
		"a_feed50":   {"世話好き", "50回食事をあげる"},
		"a_feed250":  {"いつもの相棒", "250回食事をあげる"},
		"a_feed800":  {"ペットの親友", "800回食事をあげる"},
		"a_boss10":   {"貢献者", "ワールドボスを10回攻撃"},
		"a_boss50":   {"ボスハンター", "ワールドボスを50回攻撃"},
		"a_boss150":  {"ボスの天敵", "ワールドボスを150回攻撃"},
		"a_arena30":  {"対戦好き", "対戦を30回行う"},
		"a_arena120": {"闘技場の常連", "対戦を120回行う"},
		"a_arena400": {"闘技場の魂", "対戦を400回行う"},
	},
}

func translateQuestViews(locale string, in []questView) []questView {
	w := questWords[locale]
	if w == nil {
		return in
	}
	out := make([]questView, len(in))
	copy(out, in)
	for i := range out {
		if t, ok := w[out[i].ID]; ok {
			out[i].Name, out[i].Desc = t.Name, t.Desc
		}
	}
	return out
}

var medalWords = map[string]map[string]words{
	"en": {
		"Tân binh": {"Rookie", "Reach level 5"}, "Lão luyện": {"Seasoned", "Reach level 15"},
		"Cao thủ": {"Master", "Reach level 30"}, "Huyền thoại": {"Legend", "Reach the maximum level"},
		"Lữ khách": {"Traveller", "Complete 20 hunts"}, "Thợ săn": {"Hunter", "Complete 100 hunts"},
		"Thợ diệt thú": {"Beast slayer", "Defeat 25 wild beasts"}, "Chiến binh": {"Warrior", "Win 10 duels"},
		"Vô địch": {"Champion", "Win 50 duels"}, "Tay săn hoàng kim": {"Golden hunter", "Find golden gear"},
	},
	"ja": {
		"Tân binh": {"新人", "レベル5到達"}, "Lão luyện": {"熟練者", "レベル15到達"},
		"Cao thủ": {"達人", "レベル30到達"}, "Huyền thoại": {"伝説", "最大レベル到達"},
		"Lữ khách": {"旅人", "狩りを20回完了"}, "Thợ săn": {"ハンター", "狩りを100回完了"},
		"Thợ diệt thú": {"獣狩り", "野生の獣を25体倒す"}, "Chiến binh": {"戦士", "対戦で10勝"},
		"Vô địch": {"王者", "対戦で50勝"}, "Tay săn hoàng kim": {"黄金ハンター", "黄金装備を発見"},
	},
}

// titleWords is keyed by title ID, not by the Vietnamese name the way medalWords has to be:
// a titleDef carries an ID, so there is no reason to look a translation up by its own display
// text. Descriptions carry the threshold, so they are generated per locale from a format
// string instead of being written out 35 times.
var titleWords = map[string]map[string]string{
	"en": {
		"lv": "Reach level %d", "hunt": "Complete %d hunts", "wild": "Defeat %d wild beasts",
		"duel": "Win %d duels", "star": "Earn %d campaign stars", "gold": "Find %d golden items",
		"arena": "Reach %d arena points",
	},
	"ja": {
		"lv": "レベル%d到達", "hunt": "狩りを%d回完了", "wild": "野生の獣を%d体倒す",
		"duel": "対戦で%d勝", "star": "キャンペーンで星%d獲得", "gold": "黄金装備を%d個発見",
		"arena": "闘技場%dポイント到達",
	},
}

// titleNames holds the five rung names of each track, in track order.
var titleNames = map[string]map[string][titleTiers]string{
	"en": {
		"lv":    {"Sprout", "Wayfarer", "Swordhand", "Grandmaster", "Fate-Chosen"},
		"hunt":  {"Footpath", "Pathfinder", "Explorer", "Cartographer", "Lord of the Long Road"},
		"wild":  {"Fresh Tracker", "Beast's Bane", "Crack Hunter", "Terror of the Wilds", "Lord of Beasts"},
		"duel":  {"Arena Rookie", "Fighter", "Veteran Blade", "Champion", "Peerless"},
		"star":  {"Trailblazer", "Conqueror", "Campaign Hero", "Living Legend", "Great One of Skyhold"},
		"gold":  {"Keen Eye", "Gold Panner", "Golden Hunter", "Fortune's Reaper", "Wealth Incarnate"},
		"arena": {"Bronze", "Silver", "Gold", "Diamond", "Undefeated"},
	},
	"ja": {
		"lv":    {"新芽", "旅人", "剣の担い手", "宗師", "天命の者"},
		"hunt":  {"健脚", "道標", "探検家", "地図師", "長路の覇者"},
		"wild":  {"駆け出しの狩人", "獣の天敵", "凄腕ハンター", "荒野の魔神", "獣の王"},
		"duel":  {"闘技場の新人", "武闘家", "熟練の剣士", "優勝者", "独孤求敗"},
		"star":  {"道を拓く者", "征服者", "キャンペーンの英雄", "生ける伝説", "天島の偉人"},
		"gold":  {"目利き", "砂金採り", "黄金ハンター", "天恵を摘む者", "財神顕現"},
		"arena": {"ブロンズ", "シルバー", "ゴールド", "ダイヤモンド", "無敗"},
	},
}

func translateTitles(locale string, in []titleView) []titleView {
	names, desc := titleNames[locale], titleWords[locale]
	if names == nil || desc == nil {
		return in
	}
	out := make([]titleView, len(in))
	copy(out, in)
	for i := range out {
		if row, ok := names[out[i].Track]; ok && out[i].Tier >= 1 && out[i].Tier <= titleTiers {
			out[i].Name = row[out[i].Tier-1]
		}
		if f, ok := desc[out[i].Track]; ok {
			out[i].Desc = fmt.Sprintf(f, out[i].Need)
		}
	}
	return out
}

func translateMedals(locale string, in []medal) []medal {
	w := medalWords[locale]
	if w == nil {
		return in
	}
	out := make([]medal, len(in))
	copy(out, in)
	for i := range out {
		if t, ok := w[out[i].Name]; ok {
			out[i].Name, out[i].Desc = t.Name, t.Desc
		}
	}
	return out
}

var setNamesByLocale = map[string][]string{
	"en": {"Rookie Set", "Traveller Set", "Warrior Set", "Golden Set", "Mythic Set", "Divine Set", "Supreme Set"},
	"ja": {"新人セット", "旅人セット", "戦士セット", "黄金セット", "神話セット", "神級セット", "至高セット"},
}

func translateSet(locale string, in *setBonusDef) *setBonusDef {
	if in == nil {
		return nil
	}
	names := setNamesByLocale[locale]
	if len(names) != len(setNames) {
		return in
	}
	out := *in
	out.Name = names[min(out.Tier, len(names)-1)]
	return &out
}

// gearDesc writes a piece of gear's blurb from its own stats, so the words follow the table
// instead of a hand-copied sentence that can drift when a number changes.
func gearDesc(locale, id string) string {
	g, ok := gearByID[id]
	if !ok {
		return ""
	}
	slot := gearSlotName(locale, g.Slot)
	var parts []string
	if g.Atk != 0 {
		parts = append(parts, fmt.Sprintf("+%s %s", trimNum(g.Atk), statWord(locale, "atk")))
	}
	if g.Def != 0 {
		parts = append(parts, fmt.Sprintf("+%s %s", trimNum(g.Def), statWord(locale, "def")))
	}
	if g.HP != 0 {
		parts = append(parts, fmt.Sprintf("+%s %s", trimNum(g.HP), statWord(locale, "hp")))
	}
	if len(parts) == 0 {
		return slot
	}
	return slot + ": " + strings.Join(parts, ", ")
}

func gearSlotName(locale, slot string) string {
	if w, ok := catalogByLocale[locale]; ok {
		if name, ok := w.Slots[slot]; ok {
			return name
		}
	}
	for _, s := range gearSlots {
		if s.ID == slot {
			return s.Name
		}
	}
	return slot
}

func statWord(locale, stat string) string {
	words := map[string]map[string]string{
		"en": {"atk": "damage", "def": "armour", "hp": "battle HP"},
		"ja": {"atk": "ダメージ", "def": "防御", "hp": "戦闘HP"},
		"vi": {"atk": "sát thương", "def": "giáp", "hp": "máu đấu"},
	}
	if m, ok := words[locale]; ok {
		return m[stat]
	}
	return words["vi"][stat]
}

func trimNum(v float64) string {
	return strings.TrimSuffix(strings.TrimRight(strconv.FormatFloat(v, 'f', 1, 64), "0"), ".")
}

// translatedCatalog returns the item list in one locale, leaving ids untouched.
func translatedCatalog(locale string) []itemDef {
	w, ok := catalogByLocale[locale]
	if !ok {
		return catalog
	}
	out := make([]itemDef, len(catalog))
	copy(out, catalog)
	for i := range out {
		if t, ok := w.Items[out[i].ID]; ok {
			if t.Name != "" {
				out[i].Name = t.Name
			}
			if t.Desc != "" {
				out[i].Desc = t.Desc
			}
		}
		if out[i].Kind == "gear" {
			if d := gearDesc(locale, out[i].ID); d != "" {
				out[i].Desc = d
			}
		}
	}
	return out
}

func translatedSkills(locale string) []skillDef {
	w, ok := catalogByLocale[locale]
	if !ok {
		return skills
	}
	out := make([]skillDef, len(skills))
	copy(out, skills)
	for i := range out {
		if t, ok := w.Skills[out[i].ID]; ok {
			out[i].Name, out[i].Desc = t.Name, t.Desc
		}
	}
	return out
}

func translatedMaps(locale string) []huntMap {
	w, ok := catalogByLocale[locale]
	if !ok {
		return huntMaps
	}
	out := make([]huntMap, len(huntMaps))
	copy(out, huntMaps)
	for i := range out {
		if t, ok := w.Maps[out[i].ID]; ok {
			out[i].Name, out[i].Desc = t.Name, t.Desc
		}
	}
	return out
}

func translatedSlots(locale string) any {
	w, ok := catalogByLocale[locale]
	if !ok {
		return gearSlots
	}
	type slotView struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Emoji string `json:"emoji"`
	}
	out := make([]slotView, 0, len(gearSlots))
	for _, s := range gearSlots {
		name := s.Name
		if t, ok := w.Slots[s.ID]; ok {
			name = t
		}
		out = append(out, slotView{s.ID, name, s.Emoji})
	}
	return out
}

func translatedRarities(locale string) []string {
	if w, ok := catalogByLocale[locale]; ok && len(w.Rarities) == len(rarityNames) {
		return w.Rarities
	}
	return rarityNames
}
