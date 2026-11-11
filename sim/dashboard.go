package sim

import (
	"fmt"
	"strings"
	"time"
)

// RenderDashboard собирает кадр дашборда в фиксированных позициях.
// Строки 1-4 — стартовые логи (их пишет main), дашборд с 5-й.
func (w *World) RenderDashboard(took time.Duration) string {
	var b strings.Builder

	put := func(row int, text string) {
		b.WriteString(fmt.Sprintf("\033[%d;1H%s\033[K", row, text))
	}

	// Header (row 5)
	put(1, "=== GalaxySim "+strings.Repeat("=", 62))

	// Date/took (row 6)
	put(2, fmt.Sprintf("date=%s (day=%d) | took=%s",
		dayToDate(w.Day), w.Day, took))

	// Separator (row 7)
	put(3, strings.Repeat("-", 76))
/*
	// Econ-summary header (row 8)
	put(8, "[econ-summary]")

	// Goods — собираем данные
	type goodStats struct {
		name     string
		deficit  int
		surplus  int
		norm     int
		avgPrice float64
		base     int
	}
	rows := make([]goodStats, int(GoodCount))
	for g := 0; g < int(GoodCount); g++ {
		rows[g].name = GoodsMarket[g].InternalName
		rows[g].base = GoodsMarket[g].AveragePrice
	}

	countHab := 0
	for _, p := range w.Planets {
		if !p.Habitable {
			continue
		}
		countHab++
		for g := 0; g < int(GoodCount); g++ {
			m := &p.Market[g]
			ts := m.TargetStock
			if ts < 1 {
				ts = 1
			}
			switch {
			case m.Count < ts/3:
				rows[g].deficit++
			case m.Count > ts:
				rows[g].surplus++
			default:
				rows[g].norm++
			}
			rows[g].avgPrice += m.PriceState
		}
	}
	for g := 0; g < int(GoodCount); g++ {
		if countHab > 0 {
			rows[g].avgPrice /= float64(countHab)
		}
	}

	// Goods — заголовок + строки (9–17)
	put(9, fmt.Sprintf("%-10s %7s %7s %5s | %8s %5s",
		"Good", "deficit", "surplus", "norm", "avgPrice", "base"))
	for g := 0; g < int(GoodCount); g++ {
		r := rows[g]
		put(10+g, fmt.Sprintf("%-10s %7d %7d %5d | %8.2f %5d",
			r.name, r.deficit, r.surplus, r.norm, r.avgPrice, r.base))
	}
*/
	// Separator (row 18)
//	put(18, strings.Repeat("-", 76))

	// News (row 19)
	totalNews := len(w.TickEvents)
	surplusNews, deficitNews := 0, 0
	for _, e := range w.TickEvents {
		if e.Deficit {
			deficitNews++
		} else {
			surplusNews++
		}
	}
	put(4, fmt.Sprintf("PE-news:  total=%-6d surplus=%-6d deficit=%d",
		totalNews, surplusNews, deficitNews))

	// --- Расчёты (без вывода) ---

	// Traders: гистограмма TL
	traderTLHist := [16]int{}
	for _, t := range w.Traders {
		tl := t.TechLevel
		if tl < 0 { tl = 0 }
		if tl > 15 { tl = 15 }
		traderTLHist[tl]++
	}
	thParts := []string{}
	for tl := 0; tl < 16; tl++ {
		if traderTLHist[tl] > 0 {
			thParts = append(thParts, fmt.Sprintf("%d:%d", tl, traderTLHist[tl]))
		}
	}

	// Traders: фазы
	var gIdle, gLoad, gLocal, gStar, gJump, gOther int
	for _, t := range w.Traders {
		if !t.Alive { continue }
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
		case TraderPhasePirateFind, TraderPhasePirateLoad,
			TraderPhasePirateFindMarket, TraderPhaseUnloadPirate:
			gOther++
		default:
			gOther++
		}
	}

	// Planets: гистограмма TL
	tlHist := [16]int{}
	for _, p := range w.Planets {
		if !p.Habitable { continue }
		tl := p.TechLevel
		if tl < 0 { tl = 0 }
		if tl > 15 { tl = 15 }
		tlHist[tl]++
	}
	histParts := []string{}
	for tl := 0; tl < 16; tl++ {
		if tlHist[tl] > 0 {
			histParts = append(histParts, fmt.Sprintf("%d:%d", tl, tlHist[tl]))
		}
	}

	// Caravans: гистограмма TL
	caravanTLHist := [16]int{}
	for _, c := range w.Caravans {
		tl := c.TechLevel
		if tl < 0 { tl = 0 }
		if tl > 15 { tl = 15 }
		caravanTLHist[tl]++
	}
	cvParts := []string{}
	for tl := 0; tl < 16; tl++ {
		if caravanTLHist[tl] > 0 {
			cvParts = append(cvParts, fmt.Sprintf("%d:%d", tl, caravanTLHist[tl]))
		}
	}

	// Caravans: фазы (Land / LocalFly / Refuel / StarFly / Jump / Other)
	var cLand, cLocal, cRefuel, cStar, cJump, cOther int
	for _, c := range w.Caravans {
		if !c.Alive { continue }
		switch c.Phase {
		case PhaseIdle, PhaseFindTrip, PhaseFindSurplus, PhaseFindDeficit,
			PhaseLoad, PhaseUnload, CaravanPhaseTakeoff:
			cLand++
		case CaravanPhaseCruise, CaravanPhaseLanding:
			cLocal++
		case CaravanPhaseRefuelApproach, CaravanPhaseRefuelLanding,
			CaravanPhaseRefuelJump, CaravanPhaseRefuelTakeoff:
			cRefuel++
		case CaravanPhaseToEdge, CaravanPhaseEdgeToEdge,
			CaravanPhasePlanetToEdge, CaravanPhaseEdgeToPlanet:
			cStar++
		case CaravanPhaseJumpEnter, CaravanPhaseJump,
			CaravanPhaseJumpExit, CaravanPhaseFuelCheck:
			cJump++
		default:
			cOther++
		}
	}

	// Деньги по группам
	planetMoney := 0.0
	for _, p := range w.Planets {
		if p.Habitable {
			planetMoney += p.Money
		}
	}
	traderMoney := 0.0
	for _, t := range w.Traders {
		if t.Alive {
			traderMoney += t.Money
		}
	}
	caravanMoney := 0.0
	for _, c := range w.Caravans {
		if c.Alive {
			caravanMoney += c.Money
		}
	}

	piratesAlive := 0
	for _, p := range w.Pirates {
		if p.Alive {
			piratesAlive++
		}
	}

	// Diplomats: счётчики и фазы.
	diplomatsAlive := 0
	bankersAlive := 0
	var dIdle, dLoad, dLocal, dStar, dJump, dOther int
	for _, d := range w.Diplomats {
		if !d.Alive {
			continue
		}
		diplomatsAlive++
		if d.Role == DiplomatRoleBanker {
			bankersAlive++
		}
		switch d.Phase {
		case PhaseIdle:
			dIdle++
		case DiplomatPhaseTakeoff, DiplomatPhaseCruise, DiplomatPhaseLanding:
			dLocal++
		case DiplomatPhaseToEdge, DiplomatPhaseEdgeToEdge, DiplomatPhaseRefuelApproach,
			DiplomatPhaseRefuelLanding, DiplomatPhaseRefuelJump, DiplomatPhaseRefuelTakeoff,
			DiplomatPhasePlanetToEdge, DiplomatPhaseEdgeToPlanet:
			dStar++
		case DiplomatPhaseJumpEnter, DiplomatPhaseJump, DiplomatPhaseJumpExit, DiplomatPhaseFuelCheck:
			dJump++
		default:
			dOther++
		}
	}

	lairMoney := 0.0
	for _, p := range w.Planets {
		if p.OwnerId == oiPirate {
			lairMoney += p.Money
		}
	}
	var pIdle, pOrbit, pChase, pAttack, pReturn int
	for _, p := range w.Pirates {
		if !p.Alive {
			continue
		}
		switch p.Phase {
		case PiratePhaseIdle:
			pIdle++
		case PiratePhaseOrbit:
			pOrbit++
		case PiratePhaseChase:
			pChase++
		case PiratePhaseAttack:
			pAttack++
		case PiratePhaseReturn:
			pReturn++
		}
	}

	pirateTLHist := [16]int{}
	for _, p := range w.Pirates {
		if !p.Alive {
			continue
		}
		tl := p.TechLevel
		if tl < 0 { tl = 0 }
		if tl > 15 { tl = 15 }
		pirateTLHist[tl]++
	}
	pirParts := []string{}
	for tl := 0; tl < 16; tl++ {
		if pirateTLHist[tl] > 0 {
			pirParts = append(pirParts, fmt.Sprintf("%d:%d", tl, pirateTLHist[tl]))
		}
	}

	// --- Вывод (в нужном порядке) ---

	// Planets (rows 5–6)
	put(5, fmt.Sprintf("%-10sGTL=%d   money=%.3e   AvgTL=%.3f",
		"planets:", w.GalaxyTechLevel, planetMoney, w.techDiscountPlanets(-1)))
	put(6, fmt.Sprintf("history: %s", strings.Join(histParts, " ")))

	// Separator (row 7)
	put(7, strings.Repeat("-", 76))

	// Traders (rows 8–9)
	put(8, fmt.Sprintf("%-10s%d   money=%.3e   AvgTech=%.3f",
		"traders:", len(w.Traders), traderMoney, w.techDiscountTraders(-1)))
	put(9, fmt.Sprintf("history: %s", strings.Join(thParts, " ")))

	// Tphases (row 10)
	put(10, fmt.Sprintf("%-10sIdle=%d  Load=%d  LocalFly=%d  StarFly=%d  Jump=%d  Other=%d",
		" phases:", gIdle, gLoad, gLocal, gStar, gJump, gOther))

	// Separator (row 11)
	put(11, strings.Repeat("-", 76))

	// Carvans (rows 12–13)
	put(12, fmt.Sprintf("%-10s%d   money=%.3e   AvgTech=%.3f   deaths: fuel=%d cargo=%d",
		"carvans:", len(w.Caravans), caravanMoney, w.techDiscountCaravans(-1), w.DeathsFuel, w.DeathsMoney))
	put(13, fmt.Sprintf("history: %s", strings.Join(cvParts, " ")))

	// Cphases (row 14)
	put(14, fmt.Sprintf("%-10sLand=%d  LocalFly=%d  Refuel=%d  StarFly=%d  Jump=%d  Other=%d",
		" phases:", cLand, cLocal, cRefuel, cStar, cJump, cOther))

	// Separator (row 15)
	put(15, strings.Repeat("-", 76))

	// Pirates (row 16)
	put(16, fmt.Sprintf("%-10s%d   deaths: pirate=%d   looted=%d   lairMoney=%.3e",
		"pirates:", piratesAlive, w.DeathsPirate, w.LootedCargo, lairMoney))

	// Pirates hist (row 17)
	put(17, fmt.Sprintf("history: %s", strings.Join(pirParts, " ")))

	// Pphases (row 18)
	put(18, fmt.Sprintf("%-10sIdle=%d  Orbit=%d  Chase=%d  Attack=%d  Return=%d",
		" phases:", pIdle, pOrbit, pChase, pAttack, pReturn))

	// Separator (row 19)
	put(19, strings.Repeat("-", 76))

	// Diplomats (row 20)
	put(20, fmt.Sprintf("%-10s%d   (bankers=%d)",
		"dipmats:", diplomatsAlive, bankersAlive))

	// Dphases (row 21)
	put(21, fmt.Sprintf("%-10sIdle=%d  Load=%d  LocalFly=%d  StarFly=%d  Jump=%d  Other=%d",
		" phases:", dIdle, dLoad, dLocal, dStar, dJump, dOther))

	// Separator (row 22)
	put(22, strings.Repeat("-", 76))

	// Резерв (rows 23–25)
	for r := 23; r <= 25; r++ {
		put(r, "")
	}

	b.WriteString("\033[J") // стереть хвост ниже
	return b.String()
}