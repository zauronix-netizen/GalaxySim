package sim

import (
	"log"
	"time"
)

// ВРЕМЕННО, для замера. Убрать после.
var (
	statFindTrip, statUnload int
	statEcon, statCaravans, statPirates, statLiners, statDiplomats time.Duration
)

func (w *World) SimTick() {
	w.Events = w.Events[:0]

	w.phaseMovement()
	if w.Day >= GalaxyWarmupTurns {
		w.phaseSpawn()
		w.phaseEconomy()
	}
	w.phaseRelations()

	if w.Day > 0 && w.Day%TurnsPerYear == 0 {
		w.ComputeGlobalGoodsPriceBands()
	}

	if len(w.News) > MaxNews {
		w.News = w.News[len(w.News)-MaxNews:]
	}
	w.Day++
}

func (w *World) legDistance(fromPlanet, toPlanet, phase int, routeIdx int, route []int) float64 {
	from := w.Planets[fromPlanet]
	to := w.Planets[toPlanet]
	fromStar := w.Stars[from.StarID]
	toStar := w.Stars[to.StarID]
	switch phase {
	case PhaseLift:
		return fromStar.SystemRadius - from.Orbit
	case PhaseJumpIn:
		return 1.0
	case PhaseJump:
		if routeIdx+1 < len(route) {
			return w.Dist(route[routeIdx], route[routeIdx+1]) / 20.0
		}
		return 1.0
	case PhaseJumpOut:
		return 1.0
	case PhaseDescend:
		return toStar.SystemRadius - to.Orbit
	}
	return 1.0
}

func (w *World) advance(
	phase *int, progress *float64, speed float64,
	fromPlanet, toPlanet int, routeIdx *int, route []int,
	jumpRange float64, fuel *float64, money *float64, hasFuel bool,
) (arrived bool, dead bool) {
	leg := w.legDistance(fromPlanet, toPlanet, *phase, *routeIdx, route)
	if leg < 0.01 {
		*progress = 1.0
	} else {
		*progress += speed / leg
	}
	if *progress < 1.0 {
		return false, false
	}
	*progress = 0
	switch *phase {
	case PhaseLift:
		if len(route) <= 1 {
			*phase = PhaseDescend
		} else {
			*phase = PhaseJumpIn
		}
	case PhaseJumpIn:
		if *routeIdx+1 >= len(route) {
			*phase = PhaseDescend
			return false, false
		}
		if hasFuel {
			d := w.Dist(route[*routeIdx], route[*routeIdx+1])
			owner := oiUninhabited
			cost := w.CalculateFuelCost(d*0.1, owner)
			if *fuel < cost {
				if !w.tryRefuel(route[*routeIdx], fuel, money) {
					return false, true
				}
			}
			*fuel -= cost
		}
		*phase = PhaseJump
	case PhaseJump:
		*phase = PhaseJumpOut
	case PhaseJumpOut:
		*routeIdx++
		if *routeIdx >= len(route)-1 {
			*phase = PhaseDescend
		} else {
			*phase = PhaseJumpIn
		}
	case PhaseDescend:
		return true, false
	}
	return false, false
}

func (w *World) tryRefuel(starID int, fuel, money *float64) bool {
	for _, p := range w.PlanetMap[starID] {
		if !p.Habitable {
			continue
		}
		need := CaravanFuelMax - *fuel
		cost := w.CalculateFuelCost(need, p.OwnerId)
		if *money < cost {
			return false
		}
		*money -= cost
		p.Money += cost
		*fuel = CaravanFuelMax
		return true
	}
	return false
}

func (w *World) phaseMovement() {
	w.rebuildTickCache()
	t0 := time.Now()
	alive := w.Caravans[:0]
	for _, c := range w.Caravans {
		if !c.Alive {
			continue
		}

		if c.Phase >= PhaseUnload {
			switch c.Phase {
			case PhaseUnload:
				statUnload++
				w.phaseUnload(c)
			case PhaseIdle:
				c.Phase = PhaseFindTrip
			case PhaseFindTrip:
				statFindTrip++
				best, good := w.SelectBestTradePlanet(c)
				if best == nil {
					c.Phase = PhaseIdle
					break
				}
				route := w.buildRoute(w.Planets[c.FromPlanet].StarID, best.StarID, c.JumpRange)
				if route == nil {
					c.Phase = PhaseIdle
					break
				}
				c.ToPlanet = best.ID
				c.Good = good
				c.Route = route
				c.RouteIdx = 0
				c.Progress = 0
				c.Phase = PhaseLoad
			case PhaseLoad:
				w.phaseLoad(c)
			}
			alive = append(alive, c)
			continue
		}

		arrived, dead := w.advance(
			&c.Phase, &c.Progress, c.Speed,
			c.FromPlanet, c.ToPlanet, &c.RouteIdx, c.Route,
			c.JumpRange, &c.Fuel, &c.Money, true,
		)
		if dead {
			c.Alive = false
			w.DeathsFuel++
			continue
		}
		if arrived {
			// переезд на планету назначения
			c.FromPlanet = c.ToPlanet
			c.Phase = PhaseUnload
		}
		alive = append(alive, c)
	}
	w.Caravans = alive
	statCaravans += time.Since(t0)
	t1 := time.Now()
	
	// --- ПИРАТЫ ---
	for _, p := range w.Pirates {
		if !p.Alive {
			continue
		}
		if w.Rng.Intn(100) >= 30 {
			continue
		}
		found := false
		for _, c := range w.Caravans {
			if !c.Alive {
				continue
			}
			if c.Phase == PhaseJump {
				continue
			}
			if c.Phase >= PhaseUnload {
				continue
			}
			var cs int
			switch c.Phase {
			case PhaseLift, PhaseJumpIn:
				cs = w.Planets[c.FromPlanet].StarID
			case PhaseJumpOut, PhaseDescend:
				cs = w.Planets[c.ToPlanet].StarID
			default:
				continue
			}
			if cs != w.Planets[p.BasePlanet].StarID {
				continue
			}
			found = true
			if w.Rng.Intn(100) < 5*p.Power {
				lost := c.Cargo / 2
				if lost > 0 {
					c.Cargo -= lost
					c.Money -= float64(lost) * ShopGoodsSellPrice(w.Planets[c.ToPlanet], c.Good)
					if c.Money < 0 {
						c.Money = 0
					}
				}

			} else {
				w.respawnPirate(p)
			}
			break
		}
		if !found {
			w.respawnPirate(p)
		}
	}
	statPirates += time.Since(t1)
	t2 := time.Now()
	
	// --- ЛАЙНЕРЫ ---
	liners := w.Liners[:0]
	for _, l := range w.Liners {
		if !l.Alive {
			continue
		}
		arrived, dead := w.advance(
			&l.Phase, &l.Progress, l.Speed,
			l.FromPlanet, l.ToPlanet, &l.RouteIdx, l.Route,
			l.JumpRange, nil, nil, false,
		)
		if dead {
			l.Alive = false
			continue
		}
		if arrived {
			l.FromPlanet = l.ToPlanet
			from, to, ok := w.randomHabitablePair()
			if !ok {
				l.Alive = false
				continue
			}
			route := w.buildRoute(w.Planets[from].StarID, w.Planets[to].StarID, l.JumpRange)
			if route == nil {
				l.Alive = false
				continue
			}
			l.FromPlanet = from
			l.ToPlanet = to
			l.Route = route
			l.RouteIdx = 0
			l.Phase = PhaseLift
			l.Progress = 0
		}
		liners = append(liners, l)
	}
	w.Liners = liners
	statLiners += time.Since(t2)
	t3 := time.Now()
	
	// --- ДИПЛОМАТЫ ---
	diplomats := w.Diplomats[:0]
	for _, d := range w.Diplomats {
		if !d.Alive {
			continue
		}
		arrived, dead := w.advance(
			&d.Phase, &d.Progress, d.Speed,
			d.FromPlanet, d.ToPlanet, &d.RouteIdx, d.Route,
			d.JumpRange, nil, nil, false,
		)
		if dead {
			d.Alive = false
			continue
		}
		if arrived {
			fromRace := w.Planets[d.FromPlanet].Race
			toRace := w.Planets[d.ToPlanet].Race
			if fromRace != toRace {
				w.pullRelation(fromRace, toRace, 50, 1)
			}
			from, to, ok := w.randomHabitablePair()
			if !ok || w.Planets[from].Race == w.Planets[to].Race {
				d.Alive = false
				continue
			}
			route := w.buildRoute(w.Planets[from].StarID, w.Planets[to].StarID, d.JumpRange)
			if route == nil {
				d.Alive = false
				continue
			}
			d.FromPlanet = from
			d.ToPlanet = to
			d.Route = route
			d.RouteIdx = 0
			d.Phase = PhaseLift
			d.Progress = 0
		}
		diplomats = append(diplomats, d)
	}
	w.Diplomats = diplomats
	statDiplomats += time.Since(t3)
}

func (w *World) phaseLoad(c *Caravan) {
	p := w.Planets[c.FromPlanet]
	buyPrice := ShopGoodsPurchasePrice(p, c.Good)
	if buyPrice <= 0 {
		c.Phase = PhaseFindTrip
		return
	}
	maxQ := int(c.Money / buyPrice)
	if maxQ > CaravanCargo {
		maxQ = CaravanCargo
	}
	if maxQ > p.Market[c.Good].Count {
		maxQ = p.Market[c.Good].Count
	}
	if maxQ <= 0 {
		c.Phase = PhaseFindTrip
		return
	}
	cost, ok := w.Buy(p, c.Good, maxQ, c.Race)
	if !ok {
		c.Phase = PhaseFindTrip
		return
	}
	c.Money -= cost
	c.BaseCost = cost
	c.Cargo = maxQ
	c.Phase = PhaseLift
	c.Progress = 0
}

func (w *World) phaseUnload(c *Caravan) {
	p := w.Planets[c.ToPlanet]
	if c.Cargo > 0 {
		unitCost := c.BaseCost / float64(c.Cargo)
		score, _ := w.FindBestQueuedSellPlanetProfitScore(c, c.Good, unitCost)
		if ShopGoodsSellPrice(p, c.Good) <= unitCost && score >= 80 {
			c.Phase = PhaseFindTrip
			return
		}
		if !w.IsLegal(p, c.Good) {
			w.ApplyIllegalGoodsTradeRelationsPenalty(c.BaseCost)
		}
		rev, ok := w.Sell(p, c.Good, c.Cargo, c.Race)
		if !ok {
			c.Phase = PhaseFindTrip
			return
		}
		c.Money += rev - c.BaseCost
		c.Cargo = 0
	}
	c.Phase = PhaseIdle
}

func (w *World) phaseSpawn() {
	for i := 0; i < 10 && len(w.Caravans) < targetCaravans; i++ {
		if w.SpawnCaravan() == nil {
			break
		}
	}
	for len(w.Pirates) < targetPirates {
		if w.SpawnPirate() == nil {
			break
		}
	}
	for len(w.Liners) < targetLiners {
		if w.SpawnLiner() == nil {
			break
		}
	}
	for len(w.Diplomats) < targetDiplomats {
		if w.SpawnDiplomat() == nil {
			break
		}
	}
}

// phasePopulation — динамика населения (C2).
// Вызывать в phaseEconomy ПЕРВОЙ, до applyProduction и UpdateMarketState.
// Смотрит Count еды vs TargetStock еды ПРОШЛОГО тика (лаг 1 тик, осознанно).
func (w *World) phasePopulation(p *Planet) {
	if !p.Habitable || p.PopRef < 1 {
		return
	}
	food := &p.Market[GoodFood]
	ts := food.TargetStock
	if ts < 1 {
		ts = 1
	}

	// Новость о голоде: раз в 10 дней, если еды меньше пятой части нормы.
	if w.Day%10 == 0 && food.Count < ts/5 {
		/*
		w.News = append(w.News, Event{
			Day:  w.Day,
			Kind: "planet_starving",
			Text: "planet starving",
			Star: p.StarID,
			Good: GoodFood,
		})
		*/
	}

	switch {
	case food.Count >= ts*2:
		// еды хватает → рост, тратим 75% склада
		p.Pop = p.Pop * 1001 / 1000 // +0.1%
		spend := food.Count * 3 / 4
		if spend < 1 {
			spend = 1
		}
		food.Count -= spend
		if food.Count < 0 {
			food.Count = 0
		}
	case food.Count < ts/7:
		// глубокий дефицит → падение (пока оставляем, до решения по production.go)
		p.Pop = p.Pop * 998 / 1000 // -0.2%
	}
	if p.Pop < 1000 {
		p.Pop = 1000
	}
	if p.Pop > 2_000_000 {
		p.Pop = 2_000_000
	}
}

func (w *World) phaseEconomy() {
	tEcon := time.Now()

	// Сводка голодающих — раз в 10 дней, до изменений этого тика.
	if w.Day%10 == 0 {
		starving, total := 0, 0
		for _, p := range w.Planets {
			if !p.Habitable {
				continue
			}
			total++
			food := &p.Market[GoodFood]
			ts := food.TargetStock
			if ts < 1 {
				ts = 1
			}
			if food.Count < ts/2 {
				starving++
			}
		}
		log.Printf("[pop] day=%d starving=%d/%d", w.Day, starving, total)
	}

	for _, p := range w.Planets {
		if !p.Habitable {
			continue
		}
		w.phasePopulation(p) // C2: динамика населения, первой
		w.applyProduction(p) // <-- производство/потребление
		for g := 0; g < int(GoodCount); g++ {
			w.UpdateMarketState(p, Good(g))
		}
	}
	statEcon += time.Since(tEcon)

	// --- Hauler: мгновенный опт (излишек → дефицит, остаток уничтожается) ---
	// Оптимизация: раз за тик собираем дефицитные планеты по каждому товару.
	// O(планет × товаров) вместо O(планет² × товаров).
	// Остаток излишка уничтожается (как потребление игроком).

	// 1) Собрать дефицитные планеты по каждому товару
	type deficitInfo struct {
		p       *Planet
		deficit int
	}
	deficits := [GoodCount][]deficitInfo{}
	for g := 0; g < int(GoodCount); g++ {
		list := make([]deficitInfo, 0, 64)
		for _, q := range w.Planets {
			if !q.Habitable {
				continue
			}
			qm := &q.Market[g]
			qts := qm.TargetStock
			if qts < 1 {
				qts = 1
			}
			if qm.Count < qts/3 {
				list = append(list, deficitInfo{q, qts/3 - qm.Count})
			}
		}
		deficits[g] = list
	}

	// 2) Раздать излишки и уничтожить остаток
	for _, p := range w.Planets {
		if !p.Habitable {
			continue
		}
		for g := 0; g < int(GoodCount); g++ {
			m := &p.Market[g]
			ts := m.TargetStock
			if ts < 1 {
				ts = 1
			}
			surplus := m.Count - ts
			if surplus <= 0 {
				continue
			}
			list := deficits[g]
			for i := range list {
				if surplus <= 0 {
					break
				}
				info := &list[i]
				if info.p.ID == p.ID || info.deficit <= 0 {
					continue
				}
				amount := surplus
				if amount > info.deficit {
					amount = info.deficit
				}
				m.Count -= amount
				info.p.Market[g].Count += amount
				surplus -= amount
				info.deficit -= amount
			}
			// Остаток излишка уничтожаем (потребление)
			if surplus > 0 {
				m.Count -= surplus
				if m.Count < ts {
					m.Count = ts // не ниже TargetStock
				}
			}
		}
	}
}

func (w *World) phaseRelations() {
	if w.Day%10 != 0 {
		return
	}
	for a := 0; a < int(RaceCount); a++ {
		for b := 0; b < int(RaceCount); b++ {
			if a == b {
				continue
			}
			target := w.BaseRel[a][b]
			v := w.Rel[a][b]
			v += (target - v) / 20
			if v < 0 {
				v = 0
			}
			if v > 100 {
				v = 100
			}
			w.Rel[a][b] = v
		}
	}
}

// SimTickTimed — обёртка для замера. SimTick не трогаем.
func (w *World) SimTickTimed() {
	statFindTrip, statUnload = 0, 0
	statEcon, statCaravans, statPirates, statLiners, statDiplomats = 0, 0, 0, 0, 0

	t0 := time.Now()
	w.SimTick()
	el := time.Since(t0)

	// 1) Сводка — всегда.
	log.Printf("day=%d took=%s caravans=%d pirates=%d liners=%d diplomats=%d",
		w.Day, el, len(w.Caravans), len(w.Pirates), len(w.Liners), len(w.Diplomats))

	// 2) Разбивка — всегда (пока замер).
	log.Printf("  day=%d | econ=%s caravans=%s(find=%d unload=%d) pirates=%s liners=%s diplomats=%s",
		w.Day,
		statEcon, statCaravans, statFindTrip, statUnload,
		statPirates, statLiners, statDiplomats)

	// 3) Сводка по экономике — раз в 10 дней, после 300.
	w.EconSummary()
}

// B: пересчёт кэша цен/легальности на текущий тик.
// Вызывать в начале phaseMovement, ДО обхода караванов.
func (w *World) rebuildTickCache() {
	n := len(w.Planets)
	if w.tickBuyPrice == nil || len(w.tickBuyPrice) != n {
		w.tickBuyPrice = make([][]float64, n)
		w.tickSellPrice = make([][]float64, n)
		w.tickLegal = make([][]bool, n)
		w.tickCanTrade = make([][RaceCount]bool, n)
		for i := 0; i < n; i++ {
			w.tickBuyPrice[i] = make([]float64, GoodCount)
			w.tickSellPrice[i] = make([]float64, GoodCount)
			w.tickLegal[i] = make([]bool, GoodCount)
		}
	}
	for i, p := range w.Planets {
		for g := 0; g < int(GoodCount); g++ {
			w.tickBuyPrice[i][g] = ShopGoodsPurchasePrice(p, Good(g))
			w.tickSellPrice[i][g] = ShopGoodsSellPrice(p, Good(g))
			w.tickLegal[i][g] = w.IsLegal(p, Good(g))
		}
		for r := 0; r < int(RaceCount); r++ {
			w.tickCanTrade[i][r] = w.CanTrade(p, Race(r))
		}
	}
}