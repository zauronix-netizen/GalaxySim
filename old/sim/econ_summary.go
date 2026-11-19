package sim

import "log"

// Поля для сводки — храним прошлые значения, чтобы считать прирост.
type econSummaryState struct {
	lastTotalMoney float64
	lastDeathsFuel int
	lastDay        int
	initialized    bool
}

var econState econSummaryState

// EconSummary — вызывать из SimTickTimed (или SimTick) раз в 10 дней.
// Считает дефицит/излишек/норму по каждому товару, средние цены,
// totalMoney (планеты + караваны), deathsFuel, len(Caravans).
func (w *World) EconSummary() {
	if w.Day%10 != 0 {
		return
	}
	if w.Day < GalaxyWarmupTurns {
		return // до 300 экономика спит
	}

	// --- totalMoney ---
	totalMoney := 0.0
	for _, p := range w.Planets {
		totalMoney += p.Money
	}
	for _, c := range w.Caravans {
		totalMoney += c.Money
	}

	// --- прирост с прошлой сводки ---
	deltaMoney := 0.0
	deltaFuelDeaths := 0
	if econState.initialized {
		deltaMoney = totalMoney - econState.lastTotalMoney
		deltaFuelDeaths = w.DeathsFuel - econState.lastDeathsFuel
	}
	econState.lastTotalMoney = totalMoney
	econState.lastDeathsFuel = w.DeathsFuel
	econState.lastDay = w.Day
	econState.initialized = true

	// --- заголовок ---
	log.Printf("[econ-summary] day=%d | totalMoney=%.3e (+%.3e) deathsFuel=%d (+%d) caravans=%d",
		w.Day, totalMoney, deltaMoney, w.DeathsFuel, deltaFuelDeaths, len(w.Caravans))

	// --- по каждому товару ---
	for g := 0; g < int(GoodCount); g++ {
		deficit, surplus, norm := 0, 0, 0
		sumPrice := 0.0
		countHab := 0
		for _, p := range w.Planets {
			if !p.Habitable {
				continue
			}
			countHab++
			m := &p.Market[g]
			ts := m.TargetStock
			if ts < 1 {
				ts = 1
			}
			switch {
			case m.Count < ts/3:
				deficit++
			case m.Count > ts:
				surplus++
			default:
				norm++
			}
			sumPrice += m.PriceState
		}
		avgPrice := 0.0
		if countHab > 0 {
			avgPrice = sumPrice / float64(countHab)
		}
		gm := &GoodsMarket[g]
		log.Printf("  %-10s deficit=%4d surplus=%4d norm=%4d | avgPrice=%7.2f (avg=%7.2f)",
			gm.InternalName, deficit, surplus, norm, avgPrice, float64(gm.AveragePrice))
	}
}