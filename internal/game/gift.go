package game

import "errors"

const welcomeGiftCoin = 3000

type welcomeGiftView struct {
	Coin    int  `json:"coin"`
	Claimed bool `json:"claimed"`
}

func (st *userState) welcomeGiftView() welcomeGiftView {
	return welcomeGiftView{Coin: welcomeGiftCoin, Claimed: st.WelcomeGift}
}

func claimWelcomeGift(st *userState) (string, error) {
	if st.WelcomeGift {
		return "", errors.New("quà 3.000 vàng đã được nhận rồi")
	}
	st.WelcomeGift = true
	st.Coin += welcomeGiftCoin
	return "🎁 Đã nhận quà 3.000 vàng!", nil
}
