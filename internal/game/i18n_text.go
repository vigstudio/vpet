package game

import (
	"regexp"
	"sort"
	"strings"
)

// Text localisation lives at the response boundary. Domain functions keep their tested
// Vietnamese messages; this layer translates the completed text for API replies, replays
// and bot DMs. Rules cover formatted values, then item/skill/map names are translated too.
type textRule struct {
	vi, en, ja string
	re         *regexp.Regexp
}

var exactText = map[string][2]string{
	`🎁 Đã nhận quà 3.000 vàng!`:       {`🎁 Claimed the 3,000-gold gift!`, `🎁 3,000ゴールドのギフトを受け取りました！`},
	`quà 3.000 vàng đã được nhận rồi`: {`the 3,000-gold gift was already claimed`, `3,000ゴールドのギフトは受取済みです`},

	`bấm 🎴 Biến hình ở thẻ Thú cưng để dùng thẻ này`:      {`use the 🎴 Reshape button on the Pet tab`, `ペット画面の🎴変身ボタンから使ってください`},
	`bấm 🏷️ Đổi tên ở thẻ Thú cưng để dùng thẻ này`:       {`use the 🏷️ Rename button on the Pet tab`, `ペット画面の🏷️改名ボタンから使ってください`},
	`hết chuồng rồi. Dùng 🏠 Thẻ thêm chuồng để nuôi thêm`: {`no stable space; use a 🏠 Stable card`, `小屋が満員です。🏠小屋カードを使ってください`},
	`không có thú này`:         {`this pet does not exist`, `このペットはいません`},
	`không có túi tiếp tế nào`: {`there is no supply bag to collect`, `受け取れる補給袋がありません`},

	`chưa đăng nhập`:          {`not signed in`, `未ログインです`},
	`không có`:                {`not found`, `見つかりません`},
	`sai phương thức`:         {`wrong method`, `メソッドが違います`},
	`dữ liệu không hợp lệ`:    {`invalid data`, `データが不正です`},
	`không đọc được dữ liệu`:  {`could not read data`, `データを読み込めません`},
	`không lưu được`:          {`could not save`, `保存できません`},
	`không lưu được tiến độ`:  {`could not save progress`, `進行状況を保存できません`},
	`không có quyền`:          {`not permitted`, `権限がありません`},
	`id không hợp lệ`:         {`invalid id`, `IDが不正です`},
	`lời đề nghị không còn`:   {`the offer is gone`, `交換申請はもうありません`},
	`lời thách đấu không còn`: {`the challenge is gone`, `挑戦はもうありません`},
	`thú của người thách đấu không còn đủ sức, trận bị hủy`: {`the challenger's pet can no longer fight; the match was cancelled`, `挑戦者のペットが戦えないため中止しました`},
	`bạn không có món này`:                                      {`you do not have this item`, `このアイテムを持っていません`},
	`không có món này`:                                          {`this item does not exist`, `このアイテムはありません`},
	`món này không đeo được`:                                    {`this item cannot be equipped`, `このアイテムは装備できません`},
	`món này không ai mua`:                                      {`this item cannot be sold`, `このアイテムは出品できません`},
	`món này không phải ngọc`:                                   {`this is not a gem`, `宝玉ではありません`},
	`món này không rèn được`:                                    {`this item cannot be forged`, `このアイテムは鍛造できません`},
	`món này đã ở bậc cao nhất`:                                 {`this item is already at the highest tier`, `すでに最高段階です`},
	`bạn không có 💠 Ngọc kỹ năng`:                               {`you have no 💠 Skill stone`, `💠 スキル玉がありません`},
	`bạn không có viên ngọc này`:                                {`you do not have this gem`, `この宝玉を持っていません`},
	`bạn không có 🎴 Thẻ biến hình`:                              {`you have no 🎴 Reshape card`, `🎴 変身カードがありません`},
	`bạn không có 🏷️ Thẻ đổi tên`:                               {`you have no 🏷️ Rename card`, `🏷️ 改名カードがありません`},
	`bạn không đủ món muốn tặng`:                                {`you do not have enough of what you offered`, `渡す品が足りません`},
	`bạn không đủ món người kia muốn`:                           {`you do not have enough of what they asked for`, `相手が求める品が足りません`},
	`bệnh viện không bán món này`:                               {`the clinic does not sell this item`, `病院では販売していません`},
	`dịch vụ không hợp lệ`:                                      {`invalid service`, `無効なサービスです`},
	`chọn gói chăm sóc 1, 3, 7 hoặc 14 ngày`:                    {`choose a 1, 3, 7 or 14 day care package`, `1・3・7・14日の預かりプランを選んでください`},
	`hồi sinh thú trước khi cộng điểm`:                          {`revive the pet before spending points`, `ポイントを振る前に蘇生してください`},
	`không đủ điểm tiềm năng`:                                   {`not enough potential points`, `潜在ポイントが足りません`},
	`chọn Công, Thủ hoặc Sinh lực`:                              {`choose Attack, Defence or Vitality`, `攻撃・防御・生命力から選んでください`},
	`chưa có thú nào, hãy ấp trứng trước`:                       {`hatch a pet first`, `先にペットを孵してください`},
	`thú còn sống, để dành lông phượng hoàng nhé`:               {`the pet is alive; save the phoenix feather`, `ペットは生きています。不死鳥の羽は取っておきましょう`},
	`thú không có kỹ năng này`:                                  {`the pet does not have this skill`, `このスキルを覚えていません`},
	`thú không đeo gì`:                                          {`the pet is not wearing anything`, `装備していません`},
	`khung này đang trống`:                                      {`this slot is empty`, `この枠は空です`},
	`thú đang nghỉ phép`:                                        {`the pet is on holiday`, `ペットは休暇中です`},
	`chưa tìm được bộ kỹ năng khác, thử lại`:                    {`could not find a different skill pair; try again`, `別のスキル組を選べませんでした。再試行してください`},
	`ải hoặc độ khó không hợp lệ`:                               {`invalid stage or difficulty`, `ステージまたは難易度が不正です`},
	`chưa mở ải: hoàn thành ải trước và đạt cấp yêu cầu`:        {`stage locked: clear the previous stage and reach the required level`, `未開放：前のステージをクリアし必要レベルに到達してください`},
	`thắng độ khó trước rồi mới thử độ khó này`:                 {`win the previous difficulty first`, `先に前の難易度をクリアしてください`},
	`cần 5 độ no để vào ải`:                                     {`5 fullness is required for the stage`, `ステージには満腹5が必要です`},
	`boss chưa bị hạ, chưa đóng góp hoặc đã nhận thưởng`:        {`the boss is alive, you did not contribute, or you already claimed`, `ボス未討伐・未貢献・受取済みのいずれかです`},
	`boss đã bị hạ; quay lại ngày mai`:                          {`the boss is down; come back tomorrow`, `ボスは討伐済みです。明日また来てください`},
	`boss thế giới yêu cầu cấp 5`:                               {`the world boss requires level 5`, `ワールドボスはレベル5以上です`},
	`cần 10 độ no đánh boss`:                                    {`10 fullness is required to attack the boss`, `ボス攻撃には満腹10が必要です`},
	`đang hồi sức sau đòn đánh boss`:                            {`recovering after the boss attack`, `ボス攻撃後の回復中です`},
	`bạn đã ở trong một đội; rời đội đó trước`:                  {`you are already in a party; leave it first`, `すでにパーティにいます。先に退出してください`},
	`đã đủ 30 đội đang chờ; tham gia một đội có sẵn`:            {`30 parties are already waiting; join one of them`, `待機パーティが30組あります。既存のパーティに参加してください`},
	`đội hết hạn sau 30 phút hoặc không còn`:                    {`the party expired after 30 minutes or is gone`, `パーティは30分で期限切れになったか、存在しません`},
	`đội không còn chỗ hoặc đã xuất phát`:                       {`the party is full or has departed`, `満員または出発済みです`},
	`chỉ chủ đội được xuất phát; cần 2–4 người`:                 {`only the leader can depart; 2-4 people are required`, `出発できるのはリーダーのみ。2〜4人必要です`},
	`không thể rời đội này`:                                     {`you cannot leave this party`, `このパーティから退出できません`},
	`thao tác đội không hợp lệ`:                                 {`invalid party action`, `無効なパーティ操作です`},
	`có thú không còn trong đội`:                                {`a pet is no longer in the party`, `パーティ内に無効なペットがいます`},
	`thú tham gia không còn`:                                    {`a participating pet is gone`, `参加ペットがいません`},
	`chưa có kết quả hoặc đã nhận thưởng`:                       {`no result or reward already claimed`, `結果がないか、報酬受取済みです`},
	`thú chưa sẵn sàng: cần 10 no và hết hồi sức săn`:           {`pet not ready: needs 10 fullness and must finish recovering`, `準備不足：満腹10以上かつ狩りの回復完了が必要です`},
	`không có vùng đất này`:                                     {`this region does not exist`, `この地域はありません`},
	`không có vùng săn`:                                         {`this hunting region does not exist`, `この狩場はありません`},
	`chưa đủ cấp vào vùng này`:                                  {`level is too low for this region`, `この地域にはレベル不足です`},
	`hôm nay cửa hàng không bán món này`:                        {`the shop does not sell this today`, `本日のショップでは販売していません`},
	`số lượng không hợp lệ`:                                     {`invalid quantity`, `数量が不正です`},
	`số lượng phải từ 1 đến 99`:                                 {`quantity must be from 1 to 99`, `数量は1〜99です`},
	`chỉ còn 0 món`:                                             {`sold out`, `売り切れです`},
	`hộp mù đang trống, thử lại sau`:                            {`the prize wheel is empty; try later`, `ルーレットが空です。後でもう一度`},
	`không đập được: đồ Thần thoại và Thần cấp phải rèn mới có`: {`cannot be crushed: mythic and divine gear must be forged`, `砕けません：神話・神級は鍛造でのみ入手できます`},
	`loại ngọc không hợp lệ`:                                    {`invalid gem type`, `宝玉の種類が不正です`},
	`ô này chưa có trang bị để khảm`:                            {`this slot has no gear to socket`, `この枠には装備がありません`},
	`lỗ khảm không tồn tại`:                                     {`this socket does not exist`, `このソケットはありません`},
	`nhiệm vụ không tồn tại`:                                    {`quest does not exist`, `クエストがありません`},
	`nhiệm vụ này đã nhận thưởng`:                               {`quest reward already claimed`, `クエスト報酬は受取済みです`},
	`hôm nay đã điểm danh rồi, mai quay lại nhé`:                {`already checked in today; come back tomorrow`, `本日は受取済みです。明日また来てください`},
	`không thể đăng mua đồ bỏ`:                                  {`scrap cannot be requested`, `素材は購入依頼できません`},
	`giá phải ít nhất 1 vàng`:                                   {`price must be at least 1 gold`, `価格は1金貨以上です`},
	`đây không phải hàng của bạn`:                               {`this is not your listing`, `あなたの出品ではありません`},
	`đây là hàng của bạn, bấm Gỡ để lấy lại`:                    {`this is your listing; take it down to recover the goods`, `あなたの出品です。取り下げて回収してください`},
	`lô hàng không còn`:                                         {`the lot is gone`, `出品はもうありません`},
	`lô hàng đã hết hạn`:                                        {`the lot expired`, `出品は期限切れです`},
	`lô hàng không hợp lệ`:                                      {`invalid lot`, `出品が不正です`},
	`lô hàng không còn trong quầy người bán`:                    {`the lot is no longer in the seller's stall`, `出品者の店頭にありません`},
	`đây không phải tin mua của bạn`:                            {`this is not your buy request`, `あなたの購入依頼ではありません`},
	`đây là tin mua của bạn`:                                    {`this is your own buy request`, `あなた自身の購入依頼です`},
	`tin mua không còn`:                                         {`the buy request is gone`, `購入依頼はもうありません`},
	`tin mua đã hết hạn`:                                        {`the buy request expired`, `購入依頼は期限切れです`},
	`tin mua không hợp lệ`:                                      {`invalid buy request`, `購入依頼が不正です`},
	`tin mua không còn trong quầy người mua`:                    {`the request is no longer in the buyer's stall`, `依頼者の店頭にありません`},
	`không thể tự trao đổi với mình`:                            {`you cannot trade with yourself`, `自分とは交換できません`},
	`không thể tự tỷ thí với mình`:                              {`you cannot challenge yourself`, `自分とは対戦できません`},
	`không tìm thấy đối thủ`:                                    {`opponent not found`, `対戦相手が見つかりません`},
	`không tìm thấy người nhận`:                                 {`recipient not found`, `相手が見つかりません`},
	`người này đang có quá nhiều lời đề nghị`:                   {`this person has too many pending offers`, `この人は保留中の交換が多すぎます`},
	`người này đang có quá nhiều lời thách đấu`:                 {`this person has too many pending challenges`, `この人は保留中の挑戦が多すぎます`},
	`chọn ít nhất một món`:                                      {`choose at least one item`, `最低1個選んでください`},
	`món giao dịch không hợp lệ`:                                {`invalid trade item`, `交換アイテムが不正です`},
	`tối đa 6 loại món mỗi lần`:                                 {`at most 6 item types per trade`, `1回の交換は6種類までです`},
	`giao dịch đang được khôi phục; tải lại chợ để xem kết quả, không đăng lại lô hàng`: {`the transaction is recovering; reload the market and do not post it again`, `取引を復旧中です。マーケットを再読込し、再投稿しないでください`},
	`thao tác không hợp lệ`: {`invalid action`, `無効な操作です`},
	`tên mới trùng tên cũ`:  {`the new name is unchanged`, `同じ名前です`},
}

var formattedText = []*textRule{

	newTextRule(` (%.0f%% giữ ẩn sau khi né)`, ` ($1% chance to stay hidden after dodging)`, `（回避後$1%で隠れ続ける）`),
	newTextRule(` · ✨ %s (%d món)`, ` · ✨ $1 ($2 pieces)`, ` · ✨ $1（$2個）`),
	newTextRule(`%s quên hết kỹ năng đã luyện, nhận lại %d 💠 ngọc kỹ năng.`, `$1 forgot all trained skills and recovered $2 💠 Skill stones.`, `$1はスキル熟練を忘れ、💠スキル玉を$2個取り戻した。`),
	newTextRule(`%s thua %s, mất %.0f sức khỏe. %s`, `$1 lost to $2 and took $3 health damage. $4`, `$1は$2に敗れ体力を$3失った。$4`),
	newTextRule(`%s đã bán cho bạn %d× %s với giá %d vàng.`, `$1 sold you $2× $3 for $4 gold.`, `$1が$3×$2を$4金貨で売りました。`),
	newTextRule(`%s đã mua %d× %s. Bạn nhận %d vàng (thuế chợ %d).`, `$1 bought $2× $3. You received $4 gold (tax $5).`, `$1が$3×$2を購入。受取$4金貨（税$5）。`),
	newTextRule(`%s đã đạt giới hạn +%d năng lượng tối đa`, `$1 has reached the +$2 max-energy limit`, `$1は最大気力+$2の上限です`),
	newTextRule(`%s: %d sao! Thưởng lần đầu +%d vàng. %s`, `$1: $2 stars! First-clear reward +$3 gold. $4`, `$1：星$2！初回報酬+$3金貨。$4`),
	newTextRule(`%s: %d sao! Đã lưu thành tích tốt nhất; thưởng lần đầu đã nhận.`, `$1: $2 stars! Best result saved; first-clear reward already claimed.`, `$1：星$2！最高記録を保存。初回報酬は受取済み。`),
	newTextRule(`+%.0f máu`, `+$1 HP`, `HP+$1`),
	newTextRule(`+%.0f máu đấu`, `+$1 battle HP`, `戦闘HP+$1`),
	newTextRule(`+%.1f giáp`, `+$1 armour`, `防御+$1`),
	newTextRule(`+%.1f sát thương`, `+$1 damage`, `ダメージ+$1`),
	newTextRule(`, hồi %.0f`, `, healed $1`, `、$1回復`),
	newTextRule(`CHÍ MẠNG! −%.0f`, `CRITICAL! −$1`, `クリティカル！−$1`),
	newTextRule(`Quyết đấu: −%.0f`, `Sudden death: −$1`, `決着戦：−$1`),
	newTextRule(`%s cần nghỉ %d giây trước lần thám hiểm tiếp theo`, `$1 must rest $2 seconds before exploring again`, `$1は次の探索まで$2秒休む必要があります`),
	newTextRule(`đã dùng hết %d lượt thám hiểm tự do hôm nay`, `all $1 free explorations have been used today`, `本日の自由探索$1回を使い切りました`),
	newTextRule(`Chỉ hồi được %.0f sức khỏe — thú không còn tin bạn nữa.`, `Only restored $1 health — the pet no longer trusts you.`, `体力は$1しか回復しない——信頼を失っています。`),
	newTextRule(`Khảm vào trang bị: %s`, `Socket into gear: $1`, `装備にはめる：$1`),
	newTextRule(`Lô %d× %s đã %s, hàng về túi.`, `The $1× $2 lot was $3; goods returned to the bag.`, `$2×$1の出品は$3。品物は持ち物へ戻りました。`),
	newTextRule(`Mua %d× %s %s hết %d vàng`, `Bought $1× $2 $3 for $4 gold`, `$2 $3×$1を$4金貨で購入`),
	newTextRule(`Mở hộp mù ra %s %s (%s)!`, `The prize wheel gave $1 $2 ($3)!`, `ルーレットで$1 $2（$3）を獲得！`),
	newTextRule(`Rương vàng: %d 🪙 và %s %s!`, `Gold chest: $1 🪙 and $2 $3!`, `金の宝箱：$1 🪙と$2 $3！`),
	newTextRule(`Săn đội: lực %d / yêu cầu %d. Thắng: %t.`, `Party hunt: power $1 / $2 required. Won: $3.`, `パーティ狩り：戦力$1 / 必要$2。勝利：$3。`),
	newTextRule(`Thua %s. Hồi sức, đổi build rồi thử lại; chưa nhận thưởng.`, `Lost $1. Recover, adjust the build and try again; no reward.`, `$1で敗北。回復・育成調整後に再挑戦。報酬なし。`),
	newTextRule(`Tin mua %d× %s đã %s, hoàn %d vàng.`, `The request for $1× $2 was $3; refunded $4 gold.`, `$2×$1の購入依頼は$3。$4金貨返金。`),
	newTextRule(`bạn đang có %d lời thách đấu chờ, hủy bớt đi`, `you have $1 pending challenges; cancel one first`, `保留中の挑戦が$1件あります。先に取消してください`),
	newTextRule(`bạn đang có %d lời đề nghị chờ, hủy bớt đi`, `you have $1 pending offers; cancel one first`, `保留中の交換が$1件あります。先に取消してください`),
	newTextRule(`bạn đang mang nhãn Kẻ tàn bạo, còn %d ngày nữa mới được nuôi thú mới`, `you bear the Cruel Owner mark; $1 days before you can hatch another pet`, `残虐な飼い主の印：あと$1日、新しいペットを孵せません`),
	newTextRule(`chưa đủ điều kiện: %d/%d`, `not complete: $1/$2`, `未達成：$1/$2`),
	newTextRule(`cần %d vàng tiền công khảm, bạn còn %d`, `socketing costs $1 gold; you have $2`, `装着費$1金貨（所持$2）`),
	newTextRule(`cần %d vàng tiền công, bạn còn %d`, `crafting costs $1 gold; you have $2`, `工賃$1金貨（所持$2）`),
	newTextRule(`cần %d vàng để gỡ ngọc, bạn còn %d`, `removing the gem costs $1 gold; you have $2`, `宝玉を外すには$1金貨（所持$2）`),
	newTextRule(`giảm %.0f%% đòn kế tiếp`, `reduce the next blow by $1%`, `次の一撃を$1%軽減`),
	newTextRule(`ô %s đã đủ %d lỗ khảm`, `$1 already has all $2 sockets filled`, `$1のソケット$2個は満杯です`),
	newTextRule(`Đã gỡ %s khỏi ô %s.`, `Removed $1 from $2.`, `$2から$1を外しました。`),
	newTextRule(`Đập trang bị thừa mà có. Ghép %d mảnh thành một viên ngọc`, `Obtained by crushing spare gear. Forge $1 shards into a gem`, `余り装備を砕いて入手。欠片$1個で宝玉にする`),
	newTextRule(`⚔️ %s thách đấu bạn bằng **%s**!
_Thú thua trận sẽ bị thương, xui có thể tử trận._`, `⚔️ $1 challenged you with **$2**!
_The loser is hurt and may die._`, `⚔️ $1が**$2**で挑戦！
_敗者は負傷し、死亡することもあります。_`),
	newTextRule(`⚔️ **%s** đấu với **%s** của %s: bạn %s. %s`, `⚔️ **$1** fought $3's **$2**: you $4. $5`, `⚔️ **$1** vs $3の**$2**：あなたは$4。$5`),
	newTextRule(`✅ %s đã đồng ý trao đổi. Bạn nhận: %s`, `✅ $1 accepted the trade. You received: $2`, `✅ $1が交換を承認。受取：$2`),
	newTextRule(`🍙 **%s** đói lắm rồi (thanh no còn %.0f%%). Cho ăn ngay kẻo chết đói!`, `🍙 **$1** is starving ($2% fullness). Feed it now!`, `🍙 **$1**は空腹です（満腹$2%）。すぐ食事を！`),
	newTextRule(`🎯 %s: +%d vàng`, `🎯 $1: +$2 gold`, `🎯 $1：+$2金貨`),
	newTextRule(`💀 %s đã bị giết. Bạn mang nhãn **Kẻ tàn bạo** trong %d ngày: không ấp được thú mới, thú còn lại mau đói và buồn, thuốc men kém tác dụng, tiền thưởng giảm nửa.`, `💀 $1 was killed. You bear the **Cruel Owner** mark for $2 days: no new pets, faster decay, weaker medicine and half rewards.`, `💀 $1を殺しました。**残虐な飼い主**の印が$2日間：新しいペット不可、空腹・気分悪化、薬低下、報酬半減。`),
	newTextRule(`💀 **%s** đã chết vì %s. Mở Thú ảo để chôn cất và nuôi con mới.`, `💀 **$1** died from $2. Open Virtual Pet to bury it.`, `💀 **$1**は$2で死亡。バーチャルペットを開いて埋葬してください。`),
	newTextRule(`🔁 %s muốn trao đổi với bạn:
- Tặng: %s
- Muốn nhận: %s`, `🔁 $1 wants to trade:
- Offers: $2
- Wants: $3`, `🔁 $1から交換申請：
- 渡す：$2
- 欲しい：$3`),
	newTextRule(`🤒 **%s** bị bệnh, cần 💊 Thuốc cảm. Bệnh lâu sẽ chết đấy.`, `🤒 **$1** is ill and needs 💊 Cold medicine. Untreated illness can be fatal.`, `🤒 **$1**は病気です。💊風邪薬が必要。放置すると死亡します。`),
	newTextRule(`🪙 +%d vàng`, `🪙 +$1 gold`, `🪙 +$1金貨`),
	newTextRule(`cần %d vàng, bạn còn %d`, `need $1 gold; you have $2`, `$1金貨必要です（所持$2）`),
	newTextRule(`thiếu %d vàng`, `$1 gold short`, `金貨が$1不足`),
	newTextRule(`bạn chỉ có %d× %s`, `you only have $1× $2`, `$2は$1個しかありません`),
	newTextRule(`cần %d× %s, bạn có %d`, `need $1× $2; you have $3`, `$2が$1個必要（所持$3）`),
	newTextRule(`cần %d× %s cấp %d, bạn có %d`, `need $1× tier-$3 $2; you have $4`, `段階$3の$2が$1個必要（所持$4）`),
	newTextRule(`cần %d mảnh ngọc, bạn có %d`, `need $1 gem shards; you have $2`, `宝玉の欠片が$1個必要（所持$2）`),
	newTextRule(`cần %d năng lượng, còn %d. Đủ sau khoảng %d phút`, `need $1 energy; $2 remains. Enough in about $3 minutes`, `気力$1必要、残り$2。約$3分後に回復`),
	newTextRule(`chỉ còn %d món`, `only $1 left`, `残り$1個`),
	newTextRule(`hộp mù giá %d vàng, bạn còn %d`, `the spin costs $1 gold; you have $2`, `ルーレットは$1金貨（所持$2）`),
	newTextRule(`chữa bệnh giá %d vàng, bạn còn %d`, `treatment costs $1 gold; you have $2`, `治療は$1金貨（所持$2）`),
	newTextRule(`nằm viện giá %d vàng, bạn còn %d`, `hospital care costs $1 gold; you have $2`, `入院は$1金貨（所持$2）`),
	newTextRule(`hồi sinh giá %d vàng, bạn còn %d`, `revival costs $1 gold; you have $2`, `蘇生は$1金貨（所持$2）`),
	newTextRule(`gói %d ngày giá %d vàng, bạn còn %d`, `the $1-day package costs $2 gold; you have $3`, `$1日プランは$2金貨（所持$3）`),
	newTextRule(`mỗi lần mua từ 1 đến 20 món`, `buy 1 to 20 items at a time`, `1回1〜20個購入できます`),
	newTextRule(`giá tối đa cho %d× %s là %d vàng`, `the maximum price for $1× $2 is $3 gold`, `$2×$1の上限は$3金貨`),
	newTextRule(`cần %d vàng phí đăng bán, bạn còn %d`, `the $1 gold posting fee is required; you have $2`, `出品料$1金貨が必要（所持$2）`),
	newTextRule(`cần %d vàng gồm giá mua và phí đăng, bạn còn %d`, `$1 gold is required for the price and fee; you have $2`, `購入額と手数料で$1金貨必要（所持$2）`),
	newTextRule(`bạn đang bán %d món, gỡ bớt rồi đăng tiếp`, `you have $1 listings; take one down first`, `出品が$1件あります。先に取り下げてください`),
	newTextRule(`bạn đang có %d tin mua, hủy bớt rồi đăng tiếp`, `you have $1 buy requests; cancel one first`, `購入依頼が$1件あります。先に取消してください`),
	newTextRule(`đã có tối đa %d chuồng`, `all $1 stables are already open`, `小屋は最大$1個です`),
	newTextRule(`Có thêm chuồng! Nuôi được %d con.`, `Stable added! You can raise $1 pets.`, `小屋を追加！$1匹飼えます。`),
	newTextRule(`Đã cộng %d điểm; còn %d điểm tiềm năng.`, `Added $1 points; $2 potential points remain.`, `$1ポイント追加、残り$2。`),
	newTextRule(`tên thú dài 1–%d ký tự`, `pet names must be 1-$1 characters`, `名前は1〜$1文字です`),
	newTextRule(`bộ phận không hợp lệ: %s`, `invalid part: $1`, `不正な部位：$1`),
	newTextRule(`%s cần cấp %d mới vào được %s`, `$1 needs level $2 to enter $3`, `$1はレベル$2で$3に入れます`),
	newTextRule(`%s còn mệt, nghỉ thêm %d phút nữa`, `$1 is tired; rest another $2 minutes`, `$1は疲れています。あと$2分休憩`),
	newTextRule(`%s %s lên cấp %d!`, `$1 $2 reached rank $3!`, `$1の$2がランク$3！`),
	newTextRule(`%s %s đã đạt cấp tối đa`, `$1 $2 is already at max rank`, `$1の$2は最大ランクです`),
	newTextRule(`%s cấp %d`, `$1 tier $2`, `$1 段階$2`),
	newTextRule(`Đã đăng bán %d× %s giá %d vàng (phí %d).`, `Listed $1× $2 for $3 gold (fee $4).`, `$2×$1を$3金貨で出品（手数料$4）。`),
	newTextRule(`Đã đăng mua %d× %s giá %d vàng; tiền đã được giữ trước (phí %d).`, `Requested $1× $2 for $3 gold; funds held (fee $4).`, `$2×$1を$3金貨で依頼、金貨を預かりました（手数料$4）。`),
	newTextRule(`Mua %d× %s hết %d vàng.`, `Bought $1× $2 for $3 gold.`, `$2×$1を$3金貨で購入。`),
	newTextRule(`Bán %d× %s, nhận %d vàng sau thuế %d.`, `Sold $1× $2; received $3 gold after $4 tax.`, `$2×$1を売却、税$4後に$3金貨。`),
	newTextRule(`Đã hủy tin mua, hoàn %d vàng.`, `Cancelled the request; refunded $1 gold.`, `依頼を取消し、$1金貨返金。`),
	newTextRule(`💊 Mua %d× %s hết %d vàng.`, `💊 Bought $1× $2 for $3 gold.`, `💊 $2×$1を$3金貨で購入。`),
	newTextRule(`🏥 Bác sĩ chữa khỏi bệnh cho %s (%d vàng).`, `🏥 The doctor cured $1 ($2 gold).`, `🏥 $1を治療しました（$2金貨）。`),
	newTextRule(`🏥 %s nằm viện, hồi %.0f sức khỏe (%d vàng).`, `🏥 $1 recovered $2 health in hospital ($3 gold).`, `🏥 $1が入院で体力$2回復（$3金貨）。`),
	newTextRule(`🏥 Bệnh viện cứu sống %s (%d vàng).`, `🏥 The clinic revived $1 ($2 gold).`, `🏥 $1を蘇生しました（$2金貨）。`),
	newTextRule(`🏥 Đã gửi %s chăm sóc %d ngày tới %s (%d vàng).`, `🏥 $1 is in care for $2 days until $3 ($4 gold).`, `🏥 $1を$2日間、$3まで預かります（$4金貨）。`),
	newTextRule(`📅 Điểm danh ngày %d: nhận %s %s và %d vàng. Chuỗi %d ngày.`, `📅 Day $1 check-in: $2 $3 and $4 gold. $5-day streak.`, `📅 $1日：$2 $3と$4金貨。連続$5日。`),
	newTextRule(` 🎉 Mốc %d ngày: thêm %s %s và %d vàng!`, ` 🎉 $1-day milestone: extra $2 $3 and $4 gold!`, ` 🎉 $1日達成：$2 $3と$4金貨を追加！`),
	newTextRule(`🔨 Đập %d× %s thành %d mảnh ngọc.`, `🔨 Crushed $1× $2 into $3 gem shards.`, `🔨 $2×$1を宝玉の欠片$3個に。`),
	newTextRule(`💎 Ghép %d mảnh thành %s cấp 1.`, `💎 Forged $1 shards into tier-1 $2.`, `💎 欠片$1個から段階1の$2を作成。`),
	newTextRule(`💎 Nâng %s lên cấp %d.`, `💎 Upgraded $1 to tier $2.`, `💎 $1を段階$2へ強化。`),
	newTextRule(`✨ Khảm %s vào %s.`, `✨ Socketed $1 into $2.`, `✨ $1を$2にはめました。`),
	newTextRule(`🔨 Rèn %d× %s thành %s %s!`, `🔨 Forged $1× $2 into $3 $4!`, `🔨 $2×$1から$3 $4を鍛造！`),
	newTextRule(`🏷️ %s nay tên là %s!`, `🏷️ $1 is now named $2!`, `🏷️ $1の新しい名前は$2！`),
	newTextRule(`🔮 %s nhận thêm %d điểm tiềm năng vĩnh viễn!`, `🔮 $1 gained $2 permanent potential points!`, `🔮 $1は潜在ポイントを永久に$2獲得！`),
	newTextRule(`♻️ Đã hoàn %d điểm tiềm năng của %s về quỹ.`, `♻️ Returned $1 of $2's potential points.`, `♻️ $2の潜在ポイント$1を返還。`),
	newTextRule(`⚡ %s hồi %d năng lượng (%d/%d).`, `⚡ $1 restored $2 energy ($3/$4).`, `⚡ $1の気力を$2回復（$3/$4）。`),
	newTextRule(`🔋 %s tăng %d năng lượng tối đa (%d/%d).`, `🔋 $1 gained $2 max energy ($3/$4).`, `🔋 $1の最大気力+$2（$3/$4）。`),
	newTextRule(`🎉 Lên cấp %d!`, `🎉 Reached level $1!`, `🎉 レベル$1！`),
	newTextRule(`Mở ô kỹ năng mới: %s %s`, `Unlocked a skill slot: $1 $2`, `スキル枠開放：$1 $2`),
	newTextRule(`%s học được %s %s và %s %s; giữ nguyên %d ô kỹ năng và cấp đã luyện!`, `$1 learned $2 $3 and $4 $5, keeping all $6 slots and trained ranks!`, `$1は$2 $3と$4 $5を習得。$6枠と熟練度を維持！`),
	newTextRule(`%s: %d sao! Đánh lại +%d vàng. %s`, `$1: $2 stars! Replay +$3 gold. $4`, `$1：星$2！再挑戦 +$3金貨。$4`),
	newTextRule(`Nhận thưởng boss: %d sát thương (%d%%), %d vàng và %d hòm vũ khí.`, `Boss reward: $1 damage ($2%), $3 gold and $4 weapon crates.`, `ボス報酬：$1ダメージ（$2%）、$3金貨と武器箱$4個。`),
	newTextRule(`Nhận thưởng boss: %d vàng và 1 hòm vũ khí.`, `Boss reward: $1 gold and one weapon crate.`, `ボス報酬：$1金貨と武器箱1個。`),
	newTextRule(`%s gây %d sát thương! Boss còn %d/%d máu.`, `$1 dealt $2 damage! Boss HP $3/$4.`, `$1が$2ダメージ！ボスHP $3/$4。`),
	newTextRule(`%s hạ gục %s, nhặt %d 🪙! %s`, `$1 defeated $2 and found $3 🪙! $4`, `$1は$2を倒し$3 🪙獲得！$4`),
	newTextRule(`%s thắng trận! 🏆 Thưởng 1 🎁 túi thức ăn và %d 🪙.`, `$1 won! 🏆 Reward: one 🎁 food bag and $2 🪙.`, `$1の勝利！🏆 🎁食料袋1個と$2 🪙。`),
	newTextRule(`%s thua trận, mất %.0f sức khỏe.`, `$1 lost and took $2 health damage.`, `$1は敗北し体力を$2失った。`),
	newTextRule(`💀 %s trúng đòn chí mạng và đã tử trận…`, `💀 $1 suffered a crushing blow and died…`, `💀 $1は致命傷で死亡…`),
}

func newTextRule(vi, en, ja string) *textRule {
	var b strings.Builder
	b.WriteString("(?s)^")
	for i := 0; i < len(vi); {
		if vi[i] != '%' {
			j := i + 1
			for j < len(vi) && vi[j] != '%' {
				j++
			}
			b.WriteString(regexp.QuoteMeta(vi[i:j]))
			i = j
			continue
		}
		j := i + 1
		for j < len(vi) && strings.ContainsRune("+#-.0123456789", rune(vi[j])) {
			j++
		}
		if j < len(vi) {
			switch vi[j] {
			case 'd':
				b.WriteString(`(-?[0-9]+)`)
			case 'f':
				b.WriteString(`(-?[0-9]+(?:\.[0-9]+)?)`)
			case 't':
				b.WriteString(`(true|false)`)
			default:
				b.WriteString(`(.+?)`)
			}
			j++
		} else {
			b.WriteString("%")
		}
		i = j
	}
	b.WriteString("$")
	return &textRule{vi: vi, en: braceRefs(en), ja: braceRefs(ja), re: regexp.MustCompile(b.String())}
}

// Go treats "$1金" as a named group "1金". Bracing numeric references keeps adjacent
// Japanese text from swallowing every captured number and item name.
func braceRefs(in string) string {
	var out strings.Builder
	for i := 0; i < len(in); {
		if in[i] != '$' || i+1 >= len(in) || in[i+1] < '0' || in[i+1] > '9' {
			out.WriteByte(in[i])
			i++
			continue
		}
		j := i + 1
		for j < len(in) && in[j] >= '0' && in[j] <= '9' {
			j++
		}
		out.WriteString("${")
		out.WriteString(in[i+1 : j])
		out.WriteByte('}')
		i = j
	}
	return out.String()
}

func localizeText(locale, msg string) string {
	if locale == "vi" || msg == "" {
		return msg
	}
	if v, ok := exactText[msg]; ok {
		if locale == "ja" {
			return localizeNames(locale, v[1])
		}
		return localizeNames(locale, v[0])
	}
	for _, r := range formattedText {
		if r.re.MatchString(msg) {
			repl := r.en
			if locale == "ja" {
				repl = r.ja
			}
			return localizeNames(locale, r.re.ReplaceAllString(msg, repl))
		}
	}
	return localizeNames(locale, msg)
}

func localizeNames(locale, msg string) string {
	if locale == "vi" || msg == "" {
		return msg
	}
	targets := translatedCatalog(locale)
	type pair struct{ from, to string }
	pairs := make([]pair, 0, len(catalog)+len(skills)+len(huntMaps)+len(rarityNames))
	for i, it := range catalog {
		if i < len(targets) && it.Name != targets[i].Name {
			pairs = append(pairs, pair{it.Name, targets[i].Name})
		}
	}
	tSkills := translatedSkills(locale)
	for i, sk := range skills {
		if i < len(tSkills) && sk.Name != tSkills[i].Name {
			pairs = append(pairs, pair{sk.Name, tSkills[i].Name})
		}
	}
	tMaps := translatedMaps(locale)
	for i, m := range huntMaps {
		if i < len(tMaps) && m.Name != tMaps[i].Name {
			pairs = append(pairs, pair{m.Name, tMaps[i].Name})
		}
	}
	for _, land := range territoryDefs {
		if translated := translatedTerritoryName(locale, land.ID); translated != land.Name {
			pairs = append(pairs, pair{land.Name, translated})
		}
	}
	tR := translatedRarities(locale)
	for i, r := range rarityNames {
		if i < len(tR) && r != tR[i] {
			pairs = append(pairs, pair{r, tR[i]})
		}
	}
	gemNames := map[string][2]string{"Hồng Ngọc": {"Ruby", "ルビー"}, "Lam Ngọc": {"Sapphire", "サファイア"}, "Lục Ngọc": {"Emerald", "エメラルド"}, "Mảnh ngọc": {"Gem shard", "宝玉の欠片"}}
	for from, to := range gemNames {
		if locale == "ja" {
			pairs = append(pairs, pair{from, to[1]})
		} else {
			pairs = append(pairs, pair{from, to[0]})
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return len(pairs[i].from) > len(pairs[j].from) })
	for _, p := range pairs {
		msg = strings.ReplaceAll(msg, p.from, p.to)
	}
	return msg
}

func localizeDuel(locale string, in duelResult) duelResult {
	out := in
	out.Turns = append([]duelTurn(nil), in.Turns...)
	for i := range out.Turns {
		out.Turns[i].Text = localizeText(locale, out.Turns[i].Text)
	}
	return out
}

func localizeDuels(locale string, in []duelResult) []duelResult {
	out := make([]duelResult, len(in))
	for i := range in {
		out[i] = localizeDuel(locale, in[i])
	}
	return out
}

func localizeWildResult(locale string, in *wildResult) {
	in.Note = localizeText(locale, in.Note)
	in.Duel = localizeDuel(locale, in.Duel)
}

func localizeCampaignResult(locale string, in *campaignResult) {
	in.Note = localizeText(locale, in.Note)
	in.Duel = localizeDuel(locale, in.Duel)
}
