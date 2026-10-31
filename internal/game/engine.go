package game

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	userPrefix = "u_"
	offerKey   = "o_"
	duelKey    = "c_"
	listKey    = "m_"
	wantKey    = "w_"
	maxBody    = 8 << 10
)

type User struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Locale      string `json:"locale"`
	Disabled    bool   `json:"disabled"`
}

type Backend interface {
	KVGet(string) ([]byte, error)
	KVSet(string, []byte) error
	KVDelete(string) error
	KVList(page, perPage int) ([]string, error)
	Apply(map[string]any) error
	User(string) (*User, error)
	UserByUsername(string) (*User, error)
	Notify(userID, reason, message, tab, duel string)
	LogError(message string, fields ...any)
}

type Engine struct {
	mu            sync.Mutex
	backend       Backend
	arenaCachedAt int64
	arenaCache    []arenaEntry
	random        rng
}

func NewEngine(backend Backend) *Engine {
	return &Engine{backend: backend, random: rand.Float64}
}

func (p *Engine) Run(ctx context.Context) {
	p.runMaintenance(time.Now().UnixMilli())
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			p.runMaintenance(now.UnixMilli())
		}
	}
}

func (p *Engine) runMaintenance(now int64) {
	p.tickAll(now)
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, _, err := p.ensureWar(now); err != nil {
		p.backend.LogError("vpet: settle territory season", "err", err.Error())
	}
	if err := p.settlePreviousBoss(now); err != nil {
		p.backend.LogError("vpet: settle previous world boss", "err", err.Error())
	}
}

// ---- storage ----

func (p *Engine) load(userID string) (*userState, error) {
	raw, appErr := p.backend.KVGet(userPrefix + userID)
	if appErr != nil {
		return nil, appErr
	}
	if raw == nil {
		return newUserState(), nil
	}
	st := &userState{}
	if err := json.Unmarshal(raw, st); err != nil {
		return nil, err
	}
	st.normalize()
	return st, nil
}

func (p *Engine) save(userID string, st *userState) error {
	raw, err := json.Marshal(st)
	if err != nil {
		return err
	}
	if appErr := p.backend.KVSet(userPrefix+userID, raw); appErr != nil {
		return appErr
	}
	return nil
}

func (p *Engine) loadListing(id string) (*listing, error) {
	raw, appErr := p.backend.KVGet(listKey + id)
	if appErr != nil {
		return nil, appErr
	}
	if raw == nil {
		return nil, nil
	}
	l := &listing{}
	return l, json.Unmarshal(raw, l)
}

func (p *Engine) saveListing(l *listing) error {
	raw, err := json.Marshal(l)
	if err != nil {
		return err
	}
	if appErr := p.backend.KVSet(listKey+l.ID, raw); appErr != nil {
		return appErr
	}
	return nil
}

func (p *Engine) loadBuyOrder(id string) (*buyOrder, error) {
	raw, appErr := p.backend.KVGet(wantKey + id)
	if appErr != nil {
		return nil, appErr
	}
	if raw == nil {
		return nil, nil
	}
	o := &buyOrder{}
	return o, json.Unmarshal(raw, o)
}

func (p *Engine) saveBuyOrder(o *buyOrder) error {
	raw, err := json.Marshal(o)
	if err != nil {
		return err
	}
	if appErr := p.backend.KVSet(wantKey+o.ID, raw); appErr != nil {
		return appErr
	}
	return nil
}

// marketRows reads every live listing. The board is small (a few dozen at most on one
// office server), so a full scan per open is cheaper than keeping a second index in sync.
func (p *Engine) marketRows(userID string, now int64) []listingView {
	out := []listingView{}
	for page := 0; ; page++ {
		keys, appErr := p.backend.KVList(page, 200)
		if appErr != nil {
			p.backend.LogError("vpet: market list", "err", appErr.Error())
			return out
		}
		for _, key := range keys {
			if !strings.HasPrefix(key, listKey) {
				continue
			}
			l, err := p.loadListing(strings.TrimPrefix(key, listKey))
			if err != nil || l == nil || now-l.Created >= marketTTL.Milliseconds() {
				continue
			}
			out = append(out, listingView{listing: *l, SellerName: p.userName(l.Seller), Mine: l.Seller == userID})
		}
		if len(keys) < 200 {
			break
		}
	}
	sortListings(out)
	return out
}

func (p *Engine) buyOrderRows(userID string, now int64) []buyOrderView {
	out := []buyOrderView{}
	for page := 0; ; page++ {
		keys, appErr := p.backend.KVList(page, 200)
		if appErr != nil {
			p.backend.LogError("vpet: buy order list", "err", appErr.Error())
			return out
		}
		for _, key := range keys {
			if !strings.HasPrefix(key, wantKey) {
				continue
			}
			o, err := p.loadBuyOrder(strings.TrimPrefix(key, wantKey))
			if err != nil || o == nil || now-o.Created >= marketTTL.Milliseconds() {
				continue
			}
			out = append(out, buyOrderView{buyOrder: *o, BuyerName: p.userName(o.Buyer), Mine: o.Buyer == userID})
		}
		if len(keys) < 200 {
			break
		}
	}
	sortBuyOrders(out)
	return out
}

func (p *Engine) loadOffer(id string) (*offer, error) {
	raw, appErr := p.backend.KVGet(offerKey + id)
	if appErr != nil {
		return nil, appErr
	}
	if raw == nil {
		return nil, nil
	}
	o := &offer{}
	return o, json.Unmarshal(raw, o)
}

func (p *Engine) saveOffer(o *offer) error {
	raw, _ := json.Marshal(o)
	if appErr := p.backend.KVSet(offerKey+o.ID, raw); appErr != nil {
		return appErr
	}
	return nil
}

// ---- notifications ----

type openLink struct{ Label, Tab, Duel string }

func (p *Engine) notify(userID, reason, message string) {
	p.backend.Notify(userID, reason, localizeText(p.locale(userID), message), "", "")
}

func (p *Engine) dm(userID string, st *userState, message string, link *openLink) {
	if st.Muted {
		return
	}
	tab, duel := "", ""
	if link != nil {
		tab, duel = link.Tab, link.Duel
	}
	p.backend.Notify(userID, "message", localizeText(p.locale(userID), message), tab, duel)
}

// ---- background ----

func (p *Engine) tickAll(now int64) {
	var all []string
	for page := 0; ; page++ {
		keys, appErr := p.backend.KVList(page, 200)
		if appErr != nil {
			p.backend.LogError("vpet: list", "err", appErr.Error())
			return
		}
		all = append(all, keys...)
		if len(keys) < 200 {
			break
		}
	}
	for _, key := range all {
		switch {
		case strings.HasPrefix(key, userPrefix):
			p.tickUser(strings.TrimPrefix(key, userPrefix), now)
		case strings.HasPrefix(key, offerKey):
			p.expireOffer(strings.TrimPrefix(key, offerKey), now)
		case strings.HasPrefix(key, duelKey):
			p.expireChallenge(strings.TrimPrefix(key, duelKey), now)
		case strings.HasPrefix(key, listKey):
			p.expireListing(strings.TrimPrefix(key, listKey), now)
		case strings.HasPrefix(key, wantKey):
			p.expireBuyOrder(strings.TrimPrefix(key, wantKey), now)
		}
	}
}

func (p *Engine) tickUser(userID string, now int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	st, err := p.load(userID)
	if err != nil || len(st.Pets) == 0 {
		return
	}
	var msgs []string
	for _, pt := range st.Pets {
		was := pt.Dead
		pt.tick(now)
		msgs = append(msgs, warnings(pt, was)...)
	}
	if err := p.save(userID, st); err != nil {
		p.backend.LogError("vpet: save", "err", err.Error())
		return
	}
	for _, m := range msgs {
		p.dm(userID, st, m, &openLink{"🐾 Mở Thú ảo", "pet", ""})
	}
	if len(msgs) > 0 {
		p.notify(userID, "tick", "")
	}
}

func (p *Engine) expireOffer(id string, now int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	o, err := p.loadOffer(id)
	if err != nil || o == nil || now-o.Created < offerTTL.Milliseconds() {
		return
	}
	_ = p.closeOffer(o, "hết hạn")
}

// expireListing hands an unsold lot back to its seller after a week.
func (p *Engine) expireListing(id string, now int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	l, err := p.loadListing(id)
	if err != nil || l == nil || now-l.Created < marketTTL.Milliseconds() {
		return
	}
	_ = p.returnListing(l, "hết hạn")
}

func (p *Engine) expireBuyOrder(id string, now int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	o, err := p.loadBuyOrder(id)
	if err != nil || o == nil || now-o.Created < marketTTL.Milliseconds() {
		return
	}
	_ = p.returnBuyOrder(o, "hết hạn")
}

func (p *Engine) returnBuyOrder(o *buyOrder, why string) error {
	buyer, err := p.load(o.Buyer)
	if err != nil {
		return err
	}
	if !hasID(buyer.Buy, o.ID) {
		return errors.New("tin mua không còn trong quầy người mua")
	}
	buyer.Coin += o.Price
	buyer.Buy = removeID(buyer.Buy, o.ID)
	if err := p.commitRows(map[string]any{userPrefix + o.Buyer: buyer, wantKey + o.ID: nil}); err != nil {
		return err
	}
	p.notify(o.Buyer, "market", fmt.Sprintf("Tin mua %d× %s đã %s, hoàn %d vàng.", o.Qty, itemByID[o.Item].Name, why, o.Price))
	return nil
}

// returnListing gives the escrowed items back and deletes the lot. Caller holds p.mu.
func (p *Engine) returnListing(l *listing, why string) error {
	seller, err := p.load(l.Seller)
	if err != nil {
		return err
	}
	seller.add(l.Item, l.Qty)
	seller.Sell = removeID(seller.Sell, l.ID)
	if err := p.commitRows(map[string]any{userPrefix + l.Seller: seller, listKey + l.ID: nil}); err != nil {
		return err
	}
	p.notify(l.Seller, "market", fmt.Sprintf("Lô %d× %s đã %s, hàng về túi.", l.Qty, l.Item, why))
	return nil
}

func (p *Engine) sellItem(userID string, st *userState, id string, qty, price int, now int64) (string, error) {
	if err := checkListing(st, id, qty, price); err != nil {
		return "", err
	}
	l := &listing{ID: newID(), Seller: userID, Item: id, Qty: qty, Price: price, Created: now}
	st.add(id, -qty)
	st.Coin -= marketFee
	st.Sell = append(st.Sell, l.ID)
	if err := p.commitRows(map[string]any{userPrefix + userID: st, listKey + l.ID: l}); err != nil {
		return "", err
	}
	return fmt.Sprintf("Đã đăng bán %d× %s giá %d vàng (phí %d).", qty, itemByID[id].Name, price, marketFee), nil
}

func (p *Engine) buyListing(userID string, buyer *userState, id string, now int64) (string, error) {
	l, err := p.loadListing(id)
	if err != nil {
		return "", err
	}
	if l == nil {
		return "", errors.New("lô hàng không còn")
	}
	if now-l.Created >= marketTTL.Milliseconds() {
		return "", errors.New("lô hàng đã hết hạn")
	}
	if l.Seller == userID {
		return "", errors.New("đây là hàng của bạn, bấm Gỡ để lấy lại")
	}
	if l.Qty < 1 || l.Qty > 99 || l.Price < 1 || itemByID[l.Item] == nil {
		return "", errors.New("lô hàng không hợp lệ")
	}
	if buyer.Coin < l.Price {
		return "", fmt.Errorf("cần %d vàng, bạn còn %d", l.Price, buyer.Coin)
	}
	seller, err := p.load(l.Seller)
	if err != nil {
		return "", err
	}
	if !hasID(seller.Sell, l.ID) {
		return "", errors.New("lô hàng không còn trong quầy người bán")
	}
	tax := taxOn(l.Price)
	buyer.Coin -= l.Price
	buyer.add(l.Item, l.Qty)
	seller.Coin += l.Price - tax
	seller.Sell = removeID(seller.Sell, l.ID)
	if err := p.commitRows(map[string]any{userPrefix + l.Seller: seller, userPrefix + userID: buyer, listKey + l.ID: nil}); err != nil {
		return "", err
	}
	name := itemByID[l.Item].Name
	p.dm(l.Seller, seller, fmt.Sprintf("%s đã mua %d× %s. Bạn nhận %d vàng (thuế chợ %d).", p.userName(userID), l.Qty, name, l.Price-tax, tax), &openLink{"Mở chợ", "market", ""})
	p.notify(l.Seller, "market", "Bán được hàng rồi!")
	return fmt.Sprintf("Mua %d× %s hết %d vàng.", l.Qty, name, l.Price), nil
}

func (p *Engine) cancelListing(userID, id string) (string, error) {
	l, err := p.loadListing(id)
	if err != nil {
		return "", err
	}
	if l == nil {
		return "", errors.New("lô hàng không còn")
	}
	if l.Seller != userID {
		return "", errors.New("đây không phải hàng của bạn")
	}
	if err := p.returnListing(l, "được gỡ"); err != nil {
		return "", err
	}
	return "Đã gỡ lô hàng, vật phẩm về túi.", nil
}

func (p *Engine) postBuyOrder(userID string, st *userState, id string, qty, price int, now int64) (string, error) {
	if err := checkBuyOrder(st, id, qty, price); err != nil {
		return "", err
	}
	o := &buyOrder{ID: newID(), Buyer: userID, Item: id, Qty: qty, Price: price, Created: now}
	st.Coin -= price + marketFee
	st.Buy = append(st.Buy, o.ID)
	if err := p.commitRows(map[string]any{userPrefix + userID: st, wantKey + o.ID: o}); err != nil {
		return "", err
	}
	return fmt.Sprintf("Đã đăng mua %d× %s giá %d vàng; tiền đã được giữ trước (phí %d).", qty, itemByID[id].Name, price, marketFee), nil
}

func (p *Engine) fulfillBuyOrder(userID string, seller *userState, id string, now int64) (string, error) {
	o, err := p.loadBuyOrder(id)
	if err != nil {
		return "", err
	}
	if o == nil {
		return "", errors.New("tin mua không còn")
	}
	if now-o.Created >= marketTTL.Milliseconds() {
		return "", errors.New("tin mua đã hết hạn")
	}
	if o.Buyer == userID {
		return "", errors.New("đây là tin mua của bạn")
	}
	def := itemByID[o.Item]
	if def == nil || o.Qty < 1 || o.Qty > 99 || o.Price < 1 {
		return "", errors.New("tin mua không hợp lệ")
	}
	if seller.Inv[o.Item] < o.Qty {
		return "", fmt.Errorf("cần %d× %s, bạn có %d", o.Qty, def.Name, seller.Inv[o.Item])
	}
	buyer, err := p.load(o.Buyer)
	if err != nil {
		return "", err
	}
	if !hasID(buyer.Buy, o.ID) {
		return "", errors.New("tin mua không còn trong quầy người mua")
	}
	tax := taxOn(o.Price)
	seller.add(o.Item, -o.Qty)
	seller.Coin += o.Price - tax
	buyer.add(o.Item, o.Qty)
	buyer.Buy = removeID(buyer.Buy, o.ID)
	if err := p.commitRows(map[string]any{userPrefix + userID: seller, userPrefix + o.Buyer: buyer, wantKey + o.ID: nil}); err != nil {
		return "", err
	}
	p.dm(o.Buyer, buyer, fmt.Sprintf("%s đã bán cho bạn %d× %s với giá %d vàng.", p.userName(userID), o.Qty, def.Name, o.Price), &openLink{"Mở chợ", "market", ""})
	p.notify(o.Buyer, "market", "Tin mua đã có người bán!")
	return fmt.Sprintf("Bán %d× %s, nhận %d vàng sau thuế %d.", o.Qty, def.Name, o.Price-tax, tax), nil
}

func (p *Engine) cancelBuyOrder(userID, id string) (string, error) {
	o, err := p.loadBuyOrder(id)
	if err != nil {
		return "", err
	}
	if o == nil {
		return "", errors.New("tin mua không còn")
	}
	if o.Buyer != userID {
		return "", errors.New("đây không phải tin mua của bạn")
	}
	if err := p.returnBuyOrder(o, "được hủy"); err != nil {
		return "", err
	}
	return fmt.Sprintf("Đã hủy tin mua, hoàn %d vàng.", o.Price), nil
}

func hasID(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// closeOffer returns escrowed items to the sender and removes the offer. Caller holds p.mu.
func (p *Engine) closeOffer(o *offer, why string) error {
	sender, err := p.load(o.From)
	if err != nil {
		return err
	}
	recipient, err := p.load(o.To)
	if err != nil {
		return err
	}
	sender.move(o.Give, 1)
	sender.Out = removeID(sender.Out, o.ID)
	recipient.In = removeID(recipient.In, o.ID)
	if err := p.commitRows(map[string]any{
		userPrefix + o.From: sender,
		userPrefix + o.To:   recipient,
		offerKey + o.ID:     nil,
	}); err != nil {
		return err
	}
	p.notify(o.From, "offer", "Lời đề nghị trao đổi đã "+why)
	p.notify(o.To, "offer", "")
	return nil
}

func newID() string {
	b := make([]byte, 12)
	_, _ = crand.Read(b)
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (p *Engine) writeError(w http.ResponseWriter, userID string, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": localizeText(p.locale(userID), msg)})
}

func (p *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request, userID string) {
	if userID == "" {
		p.writeError(w, userID, http.StatusUnauthorized, "chưa đăng nhập")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	if r.Method == http.MethodGet {
		switch {
		case path == "/state":
			p.mu.Lock()
			defer p.mu.Unlock()
			p.respondState(w, userID, "")
		case strings.HasPrefix(path, "/pets/"):
			p.handlePeek(w, strings.TrimPrefix(path, "/pets/"))
		default:
			p.writeError(w, userID, http.StatusNotFound, "không có")
		}
		return
	}
	if r.Method != http.MethodPost || r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
		p.writeError(w, userID, http.StatusMethodNotAllowed, "sai phương thức")
		return
	}
	var body struct {
		Name       string         `json:"name"`
		Parts      map[string]int `json:"parts"`
		Item       string         `json:"item"`
		To         string         `json:"to"`
		Give       map[string]int `json:"give"`
		Want       map[string]int `json:"want"`
		Muted      bool           `json:"muted"`
		Pet        int            `json:"pet"`
		Act        string         `json:"action"`
		Map        string         `json:"map"`
		Slot       string         `json:"slot"`
		Skill      string         `json:"skill"`
		Qty        int            `json:"qty"`
		Stage      int            `json:"stage"`
		Difficulty int            `json:"difficulty"`
		Level      int            `json:"level"`
		Gem        string         `json:"gem"`
		SlotAt     int            `json:"slot_at"`
		Price      int            `json:"price"`
		Days       int            `json:"days"`
		ID         string         `json:"id"`
		Tag        string         `json:"tag"`
		Open       bool           `json:"open"`
	}
	// Empty JSON bodies are accepted for actions without parameters.
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody)).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		p.writeError(w, userID, http.StatusBadRequest, "dữ liệu không hợp lệ")
		return
	}
	now := time.Now().UnixMilli()

	p.mu.Lock()
	defer p.mu.Unlock()
	if strings.HasPrefix(path, "/duels/") {
		p.handleDuelAction(w, userID, strings.TrimPrefix(path, "/duels/"), body.Pet, now)
		return
	}
	if strings.HasPrefix(path, "/offers/") {
		p.handleOfferAction(w, userID, strings.TrimPrefix(path, "/offers/"), now)
		return
	}
	st, err := p.load(userID)
	if err != nil {
		p.writeError(w, userID, http.StatusInternalServerError, "không đọc được dữ liệu")
		return
	}
	st.markPets(now)
	for _, pt := range st.Pets {
		pt.tick(now)
	}
	// Establish today's baseline before this action bumps a quest counter.
	st.rollDaily(now)
	var msg string
	switch path {
	case "/pet":
		err = createPet(st, body.Name, body.Parts, now, p.random)
		if err == nil {
			msg = "Trứng nở rồi! Chào " + st.Pets[len(st.Pets)-1].Name + "!"
		}
	case "/use":
		msg, err = useItem(st, body.Pet, body.Item, now, p.random)
	case "/interact":
		msg, err = interact(st, body.Pet, body.Act, now)
	case "/reskin":
		err = reskin(st, body.Pet, body.Name, body.Parts)
		if err == nil {
			msg = "Biến hình xong! ✨"
		}
	case "/unequip":
		err = unequip(st, body.Pet)
	case "/kill":
		msg, err = kill(st, body.Pet, now)
	case "/bury":
		err = bury(st, body.Pet)
		if err == nil {
			msg = "Đã chôn cất. Yên nghỉ nhé 🪦"
		}
	case "/airdrop":
		err = st.claimAir(p.random)
		if err == nil {
			msg = "Nhặt được 🎁 túi thức ăn tiếp tế!"
		}
	case "/hunt":
		var res *huntResult
		res, err = hunt(st, body.Pet, body.Map, now, p.random)
		if err == nil {
			p.rewardActivity(st, path, now)
			if err = p.save(userID, st); err != nil {
				p.writeError(w, userID, http.StatusInternalServerError, "không lưu được")
				return
			}
			p.respondHunt(w, userID, res)
			return
		}
	case "/clinic/buy":
		msg, err = buyCare(st, body.Item, max(body.Qty, 1), now)
	case "/clinic/treat":
		msg, err = treat(st, body.Pet, body.Act, now)
	case "/clinic/care":
		msg, err = boardCare(st, body.Pet, body.Days, now)
	case "/checkin":
		msg, err = claimCheckin(st, now)
	case "/gift/claim":
		msg, err = claimWelcomeGift(st)
	case "/quest/claim":
		msg, err = claimQuest(st, body.ID, now)
	case "/title":
		msg, err = wearTitle(st, body.Pet, body.ID)
	case "/gem/crush":
		msg, err = crush(st, body.Item, max(body.Qty, 1))
	case "/gem/forge":
		msg, err = forgeGem(st, body.Gem, max(body.Stage, 1))
	case "/gem/socket":
		msg, err = socket(st, body.Pet, body.Slot, body.Gem, now)
	case "/gem/unsocket":
		msg, err = unsocket(st, body.Pet, body.Slot, body.SlotAt)
	case "/build":
		msg, err = allocate(st, body.Pet, body.Skill, body.Qty)
		if err == nil {
			if err = p.save(userID, st); err != nil {
				p.writeError(w, userID, http.StatusInternalServerError, "không lưu được")
				return
			}
			p.respondStateFast(w, userID, msg)
			return
		}
	case "/campaign":
		var res *campaignResult
		res, err = campaign(st, body.Pet, body.Map, body.Stage, body.Difficulty, now, p.random, seasonBoostBPS(st, now))
		if err == nil {
			p.rewardActivity(st, path, now)
			if err = p.save(userID, st); err != nil {
				p.writeError(w, userID, 500, "không lưu được tiến độ")
				return
			}
			p.respondAll(w, userID, "", nil, nil, res, nil, nil, true)
			return
		}
	case "/boss/hit", "/boss/claim":
		msg, err = p.bossAction(userID, st, body.Pet, path == "/boss/claim", now)
		if err == nil {
			p.respondState(w, userID, msg)
			return
		}
	case "/party":
		msg, err = p.partyAction(userID, st, body.Pet, body.Act, body.ID, body.Map, now)
		if err == nil {
			p.respondState(w, userID, msg)
			return
		}
	case "/alliance":
		if body.Act == "target" || body.Act == "abandon" || body.Act == "garrison" || body.Act == "attack" {
			var war *duelResult
			msg, war, err = p.allianceWarAction(userID, st, body.Act, body.ID, body.Pet, now)
			if err == nil {
				p.respondAll(w, userID, msg, nil, nil, nil, nil, war, true)
				return
			}
		} else if body.Act == "claim-season" {
			msg, err = p.claimSeasonReward(userID, st, now)
		} else {
			msg, err = p.allianceAction(userID, st, body.Act, body.ID, body.Name, body.Tag, body.To, body.Open, now)
		}
		if err == nil {
			p.respondState(w, userID, msg)
			return
		}
	case "/wild":
		var res *wildResult
		res, err = encounter(st, body.Pet, body.Map, now, p.random, seasonBoostBPS(st, now))
		if err == nil {
			p.rewardActivity(st, path, now)
			if err = p.save(userID, st); err != nil {
				p.writeError(w, userID, http.StatusInternalServerError, "không lưu được")
				return
			}
			p.respondAll(w, userID, res.Note, nil, res, nil, nil, nil, true)
			return
		}
	case "/buy":
		msg, err = buy(st, body.Item, max(body.Qty, 1), now)
	case "/gacha":
		var res *gachaResult
		res, err = gacha(st, p.random)
		if err == nil {
			if err = p.save(userID, st); err != nil {
				p.writeError(w, userID, http.StatusInternalServerError, "không lưu được")
				return
			}
			p.respondAll(w, userID, res.Note, nil, nil, nil, res, nil, true)
			return
		}
	case "/fuse":
		msg, err = fuse(st, body.Item)
	case "/fuse-all":
		msg, err = fuseAll(st)
	case "/rename":
		msg, err = rename(st, body.Pet, body.Name)
	case "/market/sell":
		msg, err = p.sellItem(userID, st, body.Item, body.Qty, body.Price, now)
		if err == nil {
			p.respondState(w, userID, msg)
			return
		}
	case "/market/buy":
		msg, err = p.buyListing(userID, st, body.ID, now)
		if err == nil {
			p.respondState(w, userID, msg)
			return
		}
	case "/market/cancel":
		msg, err = p.cancelListing(userID, body.ID)
		if err == nil {
			p.respondState(w, userID, msg)
			return
		}
	case "/market/want":
		msg, err = p.postBuyOrder(userID, st, body.Item, body.Qty, body.Price, now)
		if err == nil {
			p.respondState(w, userID, msg)
			return
		}
	case "/market/fulfill":
		msg, err = p.fulfillBuyOrder(userID, st, body.ID, now)
		if err == nil {
			p.respondState(w, userID, msg)
			return
		}
	case "/market/cancel-want":
		msg, err = p.cancelBuyOrder(userID, body.ID)
		if err == nil {
			p.respondState(w, userID, msg)
			return
		}
	case "/equip":
		msg, err = equip(st, body.Pet, body.Item, body.Level)
	case "/equip-best":
		msg, err = equipBest(st, body.Pet)
	case "/unequip-slot":
		msg, err = unequipSlot(st, body.Pet, body.Slot)
	case "/enhance":
		msg, err = enhance(st, body.Pet, body.Slot)
	case "/enhance/restore":
		msg, err = restoreEnhanced(st, body.Item, body.Level)
	case "/skill-up":
		msg, err = skillUp(st, body.Pet, body.Skill, now)
	case "/mute":
		st.Muted = body.Muted
	case "/duels":
		msg, err = p.createChallenge(userID, st, body.To, body.Pet, now)
		if err == nil {
			p.respondState(w, userID, msg)
			return
		}
	case "/offers":
		msg, err = p.createOffer(userID, st, body.To, body.Give, body.Want, now)
		if err == nil {
			// createOffer saved both users; reload to answer with the stored state
			p.respondState(w, userID, msg)
			return
		}
	default:
		p.writeError(w, userID, http.StatusNotFound, "không có")
		return
	}
	if err != nil {
		p.writeError(w, userID, http.StatusBadRequest, err.Error())
		return
	}
	p.rewardActivity(st, path, now)
	for _, pt := range st.Pets {
		warnings(pt, false) // resets warn flags after feeding, nothing to send
	}
	if err := p.save(userID, st); err != nil {
		p.writeError(w, userID, http.StatusInternalServerError, "không lưu được")
		return
	}
	if globalStateChanged(path) {
		p.respondState(w, userID, msg)
	} else {
		p.respondStateFast(w, userID, msg)
	}
}

type stateView struct {
	Campaign       [][]stageView       `json:"campaign"`
	BuildPoints    []int               `json:"build_points"`
	Energy         []energyView        `json:"energy"`
	Sockets        [][]socketView      `json:"sockets"`
	Enhance        [][]enhanceView     `json:"enhance"`
	Enhanced       []enhancedStackView `json:"enhanced"`
	Stats          []statSheet         `json:"stats"`
	Clinic         clinicView          `json:"clinic"`
	Quests         []questView         `json:"quests"`
	Checkin        checkinView         `json:"checkin"`
	WelcomeGift    welcomeGiftView     `json:"welcome_gift"`
	Lang           string              `json:"lang"`
	Boss           *bossView           `json:"boss"`
	Parties        []huntParty         `json:"parties"`
	Alliance       allianceView        `json:"alliance"`
	User           string              `json:"user"`
	CampaignResult *campaignResult     `json:"campaign_result,omitempty"`
	WarResult      *duelResult         `json:"war_result,omitempty"`
	GachaResult    *gachaResult        `json:"gacha_result,omitempty"`
	Now            int64               `json:"now"`
	Pets           []*pet              `json:"pets"`
	Slots          int                 `json:"slots"`
	Inv            map[string]int      `json:"inv"`
	Graves         []grave             `json:"graves"`
	Muted          bool                `json:"muted"`
	Air            int                 `json:"air"`
	Catalog        []itemDef           `json:"catalog"`
	In             []offerView         `json:"in"`
	Out            []offerView         `json:"out"`
	DIn            []duelView          `json:"din"`
	DOut           []duelView          `json:"dout"`
	Duels          []duelResult        `json:"duels"`
	Skills         []skillDef          `json:"skills"`
	Maps           []huntMap           `json:"maps"`
	Slots3         any                 `json:"gear_slots"`
	Rarities       []string            `json:"rarities"`
	GearStat       map[string]gearDef  `json:"gear_stats"`
	Coin           int                 `json:"coin"`
	Power          []int               `json:"power"`
	GearTotals     []gearDef           `json:"gear_totals"`
	Sets           []*setBonusDef      `json:"sets"`
	Prices         map[string]int      `json:"prices"`
	CoinLeft       int                 `json:"coin_left"` // still earnable from activity today
	Shop           []shopOffer         `json:"shop"`
	Gacha          int                 `json:"gacha_cost"`
	Forge          []forgeRecipe       `json:"forge"`
	Market         []listingView       `json:"market"`
	BuyOrders      []buyOrderView      `json:"buy_orders"`
	Arena          []arenaEntry        `json:"arena"`
	MarketFx       [3]int              `json:"market_rules"` // posting fee, tax percent, price ceiling
	Cruel          int                 `json:"cruel_days"`   // days left under the cruelty mark, 0 when clean
	Medals         [][]medal           `json:"medals"`       // one list per pet, same order as Pets
	Titles         [][]titleView       `json:"titles"`       // one ladder per pet, same order as Pets
	Hunt           *huntResult         `json:"hunt,omitempty"`
	Wild           *wildResult         `json:"wild,omitempty"`
	Message        string              `json:"message,omitempty"`
	Drops          int                 `json:"drops_left"`
	NextDrop       int64               `json:"next_drop"`
}

type offerView struct {
	offer
	FromName string `json:"from_name"`
	ToName   string `json:"to_name"`
}

// respondHunt answers a hunt with the usual state plus what the pet brought home.
func (p *Engine) respondHunt(w http.ResponseWriter, userID string, res *huntResult) {
	p.respondAll(w, userID, res.Note, res, nil, nil, nil, nil, true)
}

func (p *Engine) respondState(w http.ResponseWriter, userID, msg string) {
	p.respondAll(w, userID, msg, nil, nil, nil, nil, nil, true)
}

// respondStateFast serves local pet changes without scanning every user's market and arena.
// Those global boards are not touched by a build click; omitting the scans removes the
// unrelated KV work from a one-point allocation.
func (p *Engine) respondStateFast(w http.ResponseWriter, userID, msg string) {
	p.respondAll(w, userID, msg, nil, nil, nil, nil, nil, false)
}

func globalStateChanged(path string) bool {
	return strings.HasPrefix(path, "/market/") || strings.HasPrefix(path, "/duels") || strings.HasPrefix(path, "/offers") ||
		strings.HasPrefix(path, "/boss/") || path == "/party" || path == "/alliance"
}

// respondAll answers with the full state, optionally carrying a hunt trip or a wild encounter.
func (p *Engine) respondAll(w http.ResponseWriter, userID, msg string, hunt *huntResult, wild *wildResult, campaign *campaignResult, gacha *gachaResult, war *duelResult, global bool) {
	st, err := p.load(userID)
	if err != nil {
		p.writeError(w, userID, http.StatusInternalServerError, "không đọc được dữ liệu")
		return
	}
	now := time.Now().UnixMilli()
	if st.rollDaily(now) {
		if err := p.save(userID, st); err != nil {
			p.writeError(w, userID, http.StatusInternalServerError, "không lưu được tiến độ nhiệm vụ")
			return
		}
	}
	lang := p.locale(userID)
	v := stateView{Now: now, Pets: []*pet{}, Slots: st.Slots, Inv: st.Inv, Graves: st.Graves, Muted: st.Muted, Air: st.Air, Catalog: translatedCatalog(lang), Message: localizeText(lang, msg),
		In: p.offerViews(st.In), Out: p.offerViews(st.Out),
		DIn: p.duelViews(st.DIn), DOut: p.duelViews(st.DOut), Duels: localizeDuels(lang, st.Duels), Skills: translatedSkills(lang),
		Maps: translatedMaps(lang), Slots3: translatedSlots(lang), Rarities: translatedRarities(lang), GearStat: gearByID, Coin: st.Coin, Shop: st.stockFor(now), Gacha: gachaCost, Forge: st.forgeList(), Enhanced: st.enhancedViews(), MarketFx: [3]int{marketFee, marketTax, marketCeiling}, Cruel: st.cruelLeft(now), Medals: [][]medal{}, Hunt: hunt, Wild: wild, NextDrop: st.LastDrop + dropCooldown.Milliseconds()}
	if global {
		v.Market = p.marketRows(userID, now)
		v.BuyOrders = p.buyOrderRows(userID, now)
		v.Arena = p.arenaRows(userID, now)
	}
	v.User = userID
	v.Boss = p.bossFor(userID, now)
	v.Parties, _ = p.parties(now)
	v.Alliance = p.allianceState(userID, now)
	v.CampaignResult = campaign
	if war != nil {
		localized := localizeDuel(lang, *war)
		v.WarResult = &localized
	}
	v.GachaResult = gacha
	if v.Hunt != nil {
		v.Hunt.Note = localizeText(lang, v.Hunt.Note)
	}
	if v.Wild != nil {
		localizeWildResult(lang, v.Wild)
	}
	if v.CampaignResult != nil {
		localizeCampaignResult(lang, v.CampaignResult)
	}
	if v.GachaResult != nil {
		v.GachaResult.Note = localizeText(lang, v.GachaResult.Note)
	}
	v.Quests = translateQuestViews(lang, st.questViews(now))
	v.Checkin = st.checkinView(now)
	v.WelcomeGift = st.welcomeGiftView()
	v.Clinic = clinicInfo()
	v.Lang = lang
	v.Prices = map[string]int{}
	for _, it := range catalog {
		v.Prices[it.ID] = priceOf(it)
	}
	v.CoinLeft = coinsPerDay
	if st.CoinDay == dayOf(now) {
		v.CoinLeft = max(0, coinsPerDay-st.CoinToday)
	}
	v.Drops = dropsPerDay
	if st.DropDay == time.UnixMilli(now).In(vnZone).Format("2006-01-02") {
		v.Drops = dropsPerDay - st.Drops
	}
	for _, pt := range st.Pets {
		cp := *pt
		cp.tick(now)
		cp.Likes, cp.Hates, cp.Touch = "", "", nil // tastes are discovered by feeding, shown through Known
		v.Pets = append(v.Pets, &cp)
		v.Medals = append(v.Medals, translateMedals(lang, medals(&cp)))
		v.Titles = append(v.Titles, translateTitles(lang, cp.titleViews()))
		v.Power = append(v.Power, boostedPower(&cp, st, now))
		v.BuildPoints = append(v.BuildPoints, cp.buildLeft())
		v.Energy = append(v.Energy, cp.energyView(now))
		v.Sockets = append(v.Sockets, cp.socketViews())
		v.Enhance = append(v.Enhance, cp.enhanceViews(st))
		v.Stats = append(v.Stats, boostedStatsOf(&cp, st, now))
		v.Campaign = append(v.Campaign, cp.campaignViews(st, now))
		v.GearTotals = append(v.GearTotals, cp.gearOf())
		v.Sets = append(v.Sets, translateSet(lang, setOf(cp.Gear)))
	}
	writeJSON(w, http.StatusOK, v)
}

func (p *Engine) offerViews(ids []string) []offerView {
	out := []offerView{}
	for _, id := range ids {
		o, err := p.loadOffer(id)
		if err != nil || o == nil {
			continue
		}
		out = append(out, offerView{offer: *o, FromName: p.userName(o.From), ToName: p.userName(o.To)})
	}
	return out
}

// arenaRows scans every player for the pet they would field. Same reasoning as the market:
// one office server holds a few dozen players, so a scan per open beats a second index.
func (p *Engine) arenaRows(userID string, now int64) []arenaEntry {
	const ttl = 30 * time.Second
	if now-p.arenaCachedAt >= ttl.Milliseconds() || p.arenaCache == nil {
		rows := p.scanArena(now)
		// Plugin reload can briefly return no KV keys. Never cache that transient empty board;
		// retry on the next state request instead of hiding every fighter for 30 seconds.
		if len(rows) > 0 || p.arenaCache == nil {
			p.arenaCache = rows
		}
		if len(rows) > 0 {
			p.arenaCachedAt = now
		}
	}
	n := min(len(p.arenaCache), arenaRows)
	out := append([]arenaEntry(nil), p.arenaCache[:n]...)
	found := false
	for i := range out {
		out[i].Me = out[i].User == userID
		found = found || out[i].Me
	}
	if !found {
		for _, row := range p.arenaCache[n:] {
			if row.User == userID {
				row.Me = true
				out = append(out, row)
				break
			}
		}
	}
	return out
}

func (p *Engine) scanArena(now int64) []arenaEntry {
	out := []arenaEntry{}
	for page := 0; ; page++ {
		keys, appErr := p.backend.KVList(page, 200)
		if appErr != nil {
			p.backend.LogError("vpet: arena list", "err", appErr.Error())
			return out
		}
		for _, key := range keys {
			if !strings.HasPrefix(key, userPrefix) {
				continue
			}
			id := strings.TrimPrefix(key, userPrefix)
			st, err := p.load(id)
			if err != nil || len(st.Pets) == 0 {
				continue
			}
			for _, pt := range st.Pets {
				pt.tick(now)
			}
			pt := best(st)
			if pt == nil {
				continue
			}
			name := p.userName(id)
			if name == "?" {
				continue // deactivated or deleted account
			}
			out = append(out, arenaEntry{User: id, Name: name, Pet: pt.Name, Lv: max(pt.Lv, 1),
				Power: boostedPower(pt, st, now), Wins: pt.Wins, Losses: pt.Losses, Points: arenaPoints(pt), Ready: pt.fit() == nil})
		}
		if len(keys) < 200 {
			break
		}
	}
	sortArena(out)
	for i := range out {
		out[i].Rank = i + 1
	}
	return out
}

// userName is the human-facing account name. Username remains the fallback when no display
// name is set; callers must not prefix it with @. locale is limited to translated languages.
func (p *Engine) locale(userID string) string {
	if u, appErr := p.backend.User(userID); appErr == nil {
		return localeOf(u.Locale)
	}
	return "vi"
}

func (p *Engine) userName(userID string) string {
	if u, appErr := p.backend.User(userID); appErr == nil {
		if name := strings.TrimSpace(u.DisplayName); name != "" {
			return name
		}
		return u.Username
	}
	return "?"
}

// handlePeek shows someone else's pet (profile popover). Inventory and hidden tastes stay private.
func (p *Engine) handlePeek(w http.ResponseWriter, userID string) {
	if !validUserID(userID) {
		p.writeError(w, userID, http.StatusBadRequest, "id không hợp lệ")
		return
	}
	p.mu.Lock()
	st, err := p.load(userID)
	p.mu.Unlock()
	if err != nil || len(st.Pets) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	pets := make([]map[string]any, 0, len(st.Pets))
	for _, pt := range st.Pets {
		cp := *pt
		cp.tick(time.Now().UnixMilli())
		pets = append(pets, map[string]any{
			"name": cp.Name, "parts": cp.Parts, "born": cp.Born, "dead": cp.Dead, "sick": cp.Sick, "acc": cp.Acc,
			"effect": cp.Effect, "hunger": cp.Hunger, "happy": cp.Happy, "health": cp.Health,
			"skills": cp.Skills, "wins": cp.Wins, "losses": cp.Losses,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"pets": pets})
}

func (p *Engine) createOffer(userID string, st *userState, to string, give, want map[string]int, now int64) (string, error) {
	if err := validateBundle(give); err != nil {
		return "", err
	}
	if err := validateBundle(want); err != nil {
		return "", err
	}
	if len(give)+len(want) == 0 {
		return "", fmt.Errorf("chọn ít nhất một món")
	}
	target, appErr := p.backend.UserByUsername(strings.TrimPrefix(strings.TrimSpace(to), "@"))
	if appErr != nil || target.Disabled {
		return "", fmt.Errorf("không tìm thấy người nhận")
	}
	if target.ID == userID {
		return "", fmt.Errorf("không thể tự trao đổi với mình")
	}
	if len(st.Out) >= maxOffers {
		return "", fmt.Errorf("bạn đang có %d lời đề nghị chờ, hủy bớt đi", maxOffers)
	}
	if !st.has(give) {
		return "", fmt.Errorf("bạn không đủ món muốn tặng")
	}
	recipient, err := p.load(target.ID)
	if err != nil {
		return "", err
	}
	if len(recipient.In) >= maxOffers*2 {
		return "", fmt.Errorf("người này đang có quá nhiều lời đề nghị")
	}
	o := &offer{ID: newID(), From: userID, To: target.ID, Give: give, Want: want, Created: now}
	st.move(give, -1) // escrow
	st.Out = append(st.Out, o.ID)
	recipient.In = append(recipient.In, o.ID)
	if err := p.commitRows(map[string]any{
		offerKey + o.ID:        o,
		userPrefix + userID:    st,
		userPrefix + target.ID: recipient,
	}); err != nil {
		return "", err
	}
	from := p.userName(userID)
	text := fmt.Sprintf("🔁 %s muốn trao đổi với bạn:\n- Tặng: %s\n- Muốn nhận: %s",
		from, orNone(bundleText(give)), orNone(bundleText(want)))
	p.dm(target.ID, recipient, text, &openLink{"🔁 Xem lời đề nghị", "trade", ""})
	p.notify(target.ID, "offer", "@"+from+" gửi lời đề nghị trao đổi")
	return "Đã gửi lời đề nghị cho " + p.userName(target.ID), nil
}

func orNone(s string) string {
	if s == "" {
		return "(không có)"
	}
	return s
}

func (p *Engine) handleOfferAction(w http.ResponseWriter, userID, rest string, now int64) {
	id, action, _ := strings.Cut(rest, "/")
	o, err := p.loadOffer(id)
	if err != nil || o == nil {
		p.writeError(w, userID, http.StatusNotFound, "lời đề nghị không còn")
		return
	}
	switch {
	case action == "accept" && o.To == userID:
		sender, err1 := p.load(o.From)
		recipient, err2 := p.load(o.To)
		if err1 != nil || err2 != nil {
			p.writeError(w, userID, http.StatusInternalServerError, "không đọc được dữ liệu")
			return
		}
		if err := settle(sender, recipient, o); err != nil {
			p.writeError(w, userID, http.StatusBadRequest, err.Error())
			return
		}
		sender.Out = removeID(sender.Out, o.ID)
		recipient.In = removeID(recipient.In, o.ID)
		if err := p.commitRows(map[string]any{
			userPrefix + o.From: sender,
			userPrefix + o.To:   recipient,
			offerKey + o.ID:     nil,
		}); err != nil {
			p.writeError(w, userID, http.StatusInternalServerError, "không lưu được")
			return
		}
		p.dm(o.From, sender, fmt.Sprintf("✅ %s đã đồng ý trao đổi. Bạn nhận: %s", p.userName(o.To), orNone(bundleText(o.Want))), nil)
		p.notify(o.From, "offer", "Trao đổi thành công!")
		p.respondState(w, userID, "Trao đổi thành công!")
	case action == "decline" && o.To == userID, action == "cancel" && o.From == userID:
		why := map[string]string{"decline": "bị từ chối", "cancel": "được hủy"}[action]
		if err := p.closeOffer(o, why); err != nil {
			p.writeError(w, userID, http.StatusInternalServerError, "không lưu được")
			return
		}
		p.respondState(w, userID, "Đã "+strings.TrimPrefix(strings.TrimPrefix(why, "bị "), "được "))
	default:
		p.writeError(w, userID, http.StatusForbidden, "không có quyền")
	}
}

// ---- duels ----

type duelView struct {
	challenge
	FromName string    `json:"from_name"`
	ToName   string    `json:"to_name"`
	PetView  *duelSide `json:"pet_view,omitempty"`
}

func (p *Engine) loadChallenge(id string) (*challenge, error) {
	raw, appErr := p.backend.KVGet(duelKey + id)
	if appErr != nil {
		return nil, appErr
	}
	if raw == nil {
		return nil, nil
	}
	c := &challenge{}
	return c, json.Unmarshal(raw, c)
}

func (p *Engine) duelViews(ids []string) []duelView {
	out := []duelView{}
	for _, id := range ids {
		c, err := p.loadChallenge(id)
		if err != nil || c == nil {
			continue
		}
		v := duelView{challenge: *c, FromName: p.userName(c.From), ToName: p.userName(c.To)}
		if st, err := p.load(c.From); err == nil {
			if _, pt := st.petByBorn(c.Pet); pt != nil {
				side := boostedSideOf(c.From, pt, st, time.Now().UnixMilli())
				v.PetView = &side
			}
		}
		out = append(out, v)
	}
	return out
}

// createChallenge invites another user to a duel with pet i. Caller holds p.mu.
func (p *Engine) createChallenge(userID string, st *userState, to string, i int, now int64) (string, error) {
	pt, err := st.petAt(i)
	if err != nil {
		return "", err
	}
	if err := pt.fit(); err != nil {
		return "", err
	}
	// the arena board sends a user id; the old text box still sends a username
	target, appErr := p.backend.UserByUsername(strings.TrimPrefix(strings.TrimSpace(to), "@"))
	if appErr != nil && validUserID(to) {
		target, appErr = p.backend.User(to)
	}
	if appErr != nil || target == nil || target.Disabled {
		return "", fmt.Errorf("không tìm thấy đối thủ")
	}
	if target.ID == userID {
		return "", fmt.Errorf("không thể tự tỷ thí với mình")
	}
	if len(st.DOut) >= maxOffers {
		return "", fmt.Errorf("bạn đang có %d lời thách đấu chờ, hủy bớt đi", maxOffers)
	}
	foe, err := p.load(target.ID)
	if err != nil {
		return "", err
	}
	if len(foe.DIn) >= maxOffers*2 {
		return "", fmt.Errorf("người này đang có quá nhiều lời thách đấu")
	}
	c := &challenge{ID: newID(), From: userID, To: target.ID, Pet: pt.Born, Created: now}
	st.DOut = append(st.DOut, c.ID)
	foe.DIn = append(foe.DIn, c.ID)
	if err := p.commitRows(map[string]any{
		duelKey + c.ID:         c,
		userPrefix + userID:    st,
		userPrefix + target.ID: foe,
	}); err != nil {
		return "", err
	}
	from := p.userName(userID)
	p.dm(target.ID, foe, fmt.Sprintf("⚔️ %s thách đấu bạn bằng **%s**!\n_Thú thua trận sẽ bị thương, xui có thể tử trận._", from, pt.Name), &openLink{"⚔️ Xem lời thách đấu", "duel", ""})
	p.notify(target.ID, "duel", "⚔️ @"+from+" thách đấu bạn!")
	return "Đã gửi lời thách đấu tới " + p.userName(target.ID), nil
}

func (p *Engine) dropChallenge(c *challenge) error {
	from, err := p.load(c.From)
	if err != nil {
		return err
	}
	to, err := p.load(c.To)
	if err != nil {
		return err
	}
	from.DOut = removeID(from.DOut, c.ID)
	to.DIn = removeID(to.DIn, c.ID)
	return p.commitRows(map[string]any{
		userPrefix + c.From: from,
		userPrefix + c.To:   to,
		duelKey + c.ID:      nil,
	})
}

func (p *Engine) expireChallenge(id string, now int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c, err := p.loadChallenge(id)
	if err != nil || c == nil || now-c.Created < duelTTL.Milliseconds() {
		return
	}
	_ = p.dropChallenge(c)
}

// handleDuelAction accepts (fighting with pet i), declines or cancels a challenge. Caller holds p.mu.
func (p *Engine) handleDuelAction(w http.ResponseWriter, userID, rest string, i int, now int64) {
	id, action, _ := strings.Cut(rest, "/")
	c, err := p.loadChallenge(id)
	if err != nil || c == nil {
		p.writeError(w, userID, http.StatusNotFound, "lời thách đấu không còn")
		return
	}
	switch {
	case action == "accept" && c.To == userID:
		attacker, err1 := p.load(c.From)
		defender, err2 := p.load(c.To)
		if err1 != nil || err2 != nil {
			p.writeError(w, userID, http.StatusInternalServerError, "không đọc được dữ liệu")
			return
		}
		attacker.tickAll(now)
		defender.tickAll(now)
		_, ap := attacker.petByBorn(c.Pet)
		if ap == nil || ap.fit() != nil {
			_ = p.dropChallenge(c)
			p.writeError(w, userID, http.StatusBadRequest, "thú của người thách đấu không còn đủ sức, trận bị hủy")
			return
		}
		dp, err := defender.petAt(i)
		if err == nil {
			err = dp.fit()
		}
		if err != nil {
			p.writeError(w, userID, http.StatusBadRequest, err.Error())
			return
		}
		res := fight(boostedSideOf(c.From, ap, attacker, now), boostedSideOf(c.To, dp, defender, now), p.random)
		res.ID, res.At = c.ID, now
		noteA := applyDuel(attacker, res, 0, now, p.random)
		noteD := applyDuel(defender, res, 1, now, p.random)
		p.rewardActivity(attacker, "/duel", now)
		p.rewardActivity(defender, "/duel", now)
		attacker.DOut = removeID(attacker.DOut, c.ID)
		defender.DIn = removeID(defender.DIn, c.ID)
		if err := p.commitRows(map[string]any{
			userPrefix + c.From: attacker,
			userPrefix + c.To:   defender,
			duelKey + c.ID:      nil,
		}); err != nil {
			p.writeError(w, userID, http.StatusInternalServerError, "không lưu được")
			return
		}
		p.arenaCache = nil
		p.arenaCachedAt = 0
		won := map[bool]string{true: "THẮNG 🏆", false: "thua"}[res.Winner == 0]
		p.dm(c.From, attacker, fmt.Sprintf("⚔️ **%s** đấu với **%s** của %s: bạn %s. %s", ap.Name, dp.Name, p.userName(c.To), won, noteA), &openLink{"▶ Xem trận đấu", "duel", res.ID})
		p.notify(c.From, "duel", "⚔️ Trận tỷ thí đã diễn ra!")
		p.respondState(w, userID, noteD)
	case action == "decline" && c.To == userID, action == "cancel" && c.From == userID:
		if err := p.dropChallenge(c); err != nil {
			p.writeError(w, userID, http.StatusInternalServerError, "không lưu được")
			return
		}
		other := c.From
		if action == "cancel" {
			other = c.To
		}
		p.notify(other, "duel", "")
		p.respondState(w, userID, "Đã hủy lời thách đấu")
	default:
		p.writeError(w, userID, http.StatusForbidden, "không có quyền")
	}
}

func validUserID(s string) bool {
	if len(s) < 3 || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func (p *Engine) rewardActivity(st *userState, path string, now int64) {
	group := ""
	switch path {
	case "/interact", "/use", "/clinic/treat":
		group = "care"
	case "/hunt", "/wild", "/campaign", "/boss/hit", "/party", "/duel", "/alliance/attack":
		group = "adventure"
	case "/checkin":
		group = "daily"
	}
	if group == "" {
		return
	}
	if st.ActivityAt == nil {
		st.ActivityAt = map[string]int64{}
	}
	if now-st.ActivityAt[group] < coinCooldown.Milliseconds() {
		return
	}
	st.ActivityAt[group] = now
	st.earn(now, false)
	st.tryDrop(now, p.random)
}

func (p *Engine) commitRows(rows map[string]any) error { return p.backend.Apply(rows) }
