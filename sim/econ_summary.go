package sim

import (
	"log"
	"fmt"
)

// Поля для сводки — храним прошлые значения, чтобы считать прирост.
type econSummaryState struct {
	lastTotalMoney float64
	lastDeathsFuel int
	lastDay        int
	initialized    bool
}

var econState econSummaryState

// dayToDate — галактический календарь: 36 дней/мес, 16 мес/год, год с 3300.
// Тот же формат, что в вебе (dayToDate в index.html).
func dayToDate(day int) string {
	const (
		daysInMonth = 36
		monthsInYear = 16
		daysInYear  = daysInMonth * monthsInYear // 576
		yearBase    = 3300
	)
	year := yearBase + day/daysInYear
	dayOfYear := day % daysInYear
	month := dayOfYear / daysInMonth
	dayOfMonth := (dayOfYear % daysInMonth) + 1
	return fmt.Sprintf("%02d/0x%02X/%d", dayOfMonth, month, year)
}

// EconSummary — вызывать из SimTickTimed (или SimTick) раз в 10 дней.
// Считает дефицит/излишек/норму по каждому товару, средние цены,
// totalMoney (планеты + караваны), deathsFuel, len(Caravans).
func (w *World) EconSummary() {
	if w.Day%12 != 0 {
		return
	}

	if w.Day < GalaxyWarmupTurns {
		return
	}

	// --- totalMoney ---
	totalMoney := 0.0
	for _, p := range w.Planets {
		totalMoney += p.Money
	}
	for _, c := range w.Caravans {
		totalMoney += c.Money
	}

	traderMoney := 0.0
	for _, t := range w.Traders {
		traderMoney += t.Money
	}
	phases := make(map[int]int)
	for _, t := range w.Traders {
		phases[t.Phase]++
	}

	// --- прирост с прошлой сводки ---
	deltaMoney := 0.0
//	deltaFuelDeaths := 0
	if econState.initialized {
		deltaMoney = totalMoney - econState.lastTotalMoney
	//	deltaFuelDeaths = w.DeathsFuel - econState.lastDeathsFuel
	}
	econState.lastTotalMoney = totalMoney
	econState.lastDeathsFuel = w.DeathsFuel
	econState.lastDay = w.Day
	econState.initialized = true

	// --- заголовок ---
	log.Printf("[econ-summary]\ndate=%s (day=%d) | totalMoney=%.3e (+%.3e) traders=%d | traderMoney=%.3e",
		dayToDate(w.Day), w.Day, totalMoney, deltaMoney, len(w.Traders), traderMoney)

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
	
	surplus, deficit := 0, 0
	for _, e := range w.TickEvents {
		if e.Deficit {
			deficit++
		} else {
			surplus++
		}
	}
	log.Printf(" news: total=%d surplus=%d deficit=%d", len(w.TickEvents), surplus, deficit)
	
	// trader-phases (сгруппировано)
	var gIdle, gLoad, gLocal, gStar, gJump, gOther int
	for _, t := range w.Traders {
		if !t.Alive {
			continue
		}
		switch t.Phase {
		case TraderPhaseRest, TraderPhaseCheckLocal:
			gIdle++
		case TraderPhaseLoad, TraderPhaseFindTrip, TraderPhaseCheckUnload, TraderPhaseUnload:
			gLoad++
		case TraderPhaseTakeoff, TraderPhaseCruise, TraderPhaseLanding, TraderPhaseRefuel:
			gLocal++
		case TraderPhaseJumpEnter, TraderPhaseJump, TraderPhaseJumpExit, TraderPhaseFuelCheck:
			gJump++
		case TraderPhaseToEdge, TraderPhaseEdgeToEdge, TraderPhaseRefuelApproach,
			TraderPhaseRefuelLanding, TraderPhaseRefuelJump, TraderPhaseRefuelTakeoff,
			TraderPhasePlanetToEdge, TraderPhaseEdgeToPlanet:
			gStar++
		default:
			gOther++
		}
	}
	log.Printf(" phases: Idle=%d Load=%d LocalFly=%d StarFly=%d Jump=%d Other=%d",
		gIdle, gLoad, gLocal, gStar, gJump, gOther)
}