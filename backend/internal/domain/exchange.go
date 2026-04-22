package domain

// flavorRate はバニラを 1.0 とした固定の両替レート。
// 整数計算で誤差を出さないため "1000倍" した整数で持つ。
var flavorRate = map[Flavor]int64{
	FlavorVanilla:   1000,
	FlavorChocolate: 1200,
	FlavorMatcha:    1500,
}

// Exchange は from フレーバーで amount マアムを to フレーバーに両替したときの受け取り量を返す。
// 端数は切り捨て。同一フレーバーのときは amount をそのまま返す。
func Exchange(from, to Flavor, amount int64) (int64, error) {
	if amount <= 0 {
		return 0, ErrInvalidAmount
	}
	if from == to {
		return amount, nil
	}
	rf, rt := flavorRate[from], flavorRate[to]
	// floor(amount * rf / rt) を整数演算で。
	converted := amount * rf / rt
	if converted <= 0 {
		return 0, ErrExchangeTooSmall
	}
	return converted, nil
}
