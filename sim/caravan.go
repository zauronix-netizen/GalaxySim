package sim

import "math"

const (
	caravanDeficitMul    = 1.25 // потолок цены на потребителе
	caravanStaticBuyMul  = 0.90 // типичная скидка производителя (count≈1.3×ts)
	caravanMinRoute      = 8.0  // строгий порог счёта маршрута, %
	caravanMinRouteAny   = 1.0  // фолбэк: лучше слабый рейс, чем вечная стоянка
	caravanMaxDistCost   = 25.0 // максимум % маржи, съедаемых самой дальней дорогой
	caravanRepWeight     = 0.5  // вес репутации в счёте
)

// в World (посчитать один раз при генерации мира):
// DistMax float64 — максимум DistMatrix

// updateSystemMarketSnaps — пересчёт агрегированного спроса/предложения по системам.
// Вызывается раз в день в phaseEconomy после applyProduction/UpdateMarketState.
func (w *World) updateSystemMarketSnaps() {
	nStars := len(w.Stars)
	if len(w.SystemMarketSnaps) != nStars {
		w.SystemMarketSnaps = make([]SystemMarketSnapshot, nStars)
		for i := range w.SystemMarketSnaps {
			w.SystemMarketSnaps[i].StarID = i
			w.SystemMarketSnaps[i].SurplusGoods = make(map[Good]int)
			w.SystemMarketSnaps[i].DeficitGoods = make(map[Good]int)
		}
	}
	for sid := 0; sid < nStars; sid++ {
		snap := &w.SystemMarketSnaps[sid]
		snap.StarID = sid
		// очищаем карты (переиспользуем аллокации)
		for k := range snap.SurplusGoods {
			delete(snap.SurplusGoods, k)
		}
		for k := range snap.DeficitGoods {
			delete(snap.DeficitGoods, k)
		}
		snap.UpdatedDay = w.Day

		planets := w.PlanetMap[sid]
		for _, p := range planets {
			if !p.Habitable {
				continue
			}
			for g := Good(0); g < GoodCount; g++ {
				if !w.IsLegal(p, g) {
					continue
				}
				m := &p.Market[g]
				ts := m.TargetStock
				if ts < 1 {
					ts = 1
				}
				// Избыток: Count > TargetStock
				if m.Count > ts {
					surplus := m.Count - ts
					snap.SurplusGoods[g] += surplus
				}
				// Дефицит: BaseSalePrice > 1.15 * base (как в caravanIsDeficit)
				base := float64(GoodsMarket[g].AveragePrice)
				if m.BaseSalePrice > 1.15*base {
					deficit := int(m.BaseSalePrice - 1.15*base)
					if deficit < 1 {
						deficit = 1
					}
					snap.DeficitGoods[g] += deficit
				}
			}
		}
	}
}

func (w *World) caravanDistCost(fromStar, toStar int) float64 {
	if w.DistMax <= 0 {
		return 0
	}
	return caravanMaxDistCost * float64(w.DistMatrix[fromStar][toStar]) / float64(w.DistMax)
}

// caravanFairPrice — статическая справедливая цена товара на планете
// (общий хелпер для обеих сторон маршрута).
func caravanFairPrice(p *Planet, g Good) float64 {
	gm := &GoodsMarket[g]
	base := float64(gm.AveragePrice) + gm.EcoAdjustPrice*economyRichness(p.Economy)
	if base < 1 {
		base = 1
	}
	fair := base * PlanetRaceMarket[p.Race].Goods.PriceFactor[g] *
		PlanetGovernmentMarket[p.Government].Goods.PriceFactor[g] /
		economyPriceFactor(p.Economy)
	if fair < 1 {
		fair = 1
	}
	return fair
}

// caravanPlanRoute — статический план рейса из справки о мире: товар, который
// производит своя планета, и планета-потребитель с лучшим счётом.
// score% = (sell/buy − 1)×100 − distCost + rep(цель).
func (w *World) caravanPlanRoute(c *Caravan) (*Planet, Good) {
	fromP := w.Planets[c.FromPlanet]
	fromStar := fromP.StarID
	fromConst := w.Stars[fromStar].ConstellationID

	type plan struct {
		p *Planet
		g Good
		s float64
	}
	best := plan{s: -1e9}
	found := false

	// Предварительный фильтр по снапшотам: собираем товары, которые есть в избытке на fromPlanet
	// и есть в дефиците в целевой системе. Это сужает поиск до реально рабочих пар.
	fromSnap := &w.SystemMarketSnaps[fromStar]
	surplusGoods := make([]Good, 0, 8)
	for g := Good(0); g < GoodCount; g++ {
		if fromP.ProducedMask&(1<<uint(g)) != 0 && fromSnap.SurplusGoods[g] > 0 {
			surplusGoods = append(surplusGoods, g)
		}
	}
	if len(surplusGoods) == 0 {
		return nil, 0 // у дома ничего не производится в избытке — рейс невозможен
	}

	scan := func(planets []*Planet, minScore float64) bool {
		for _, p := range planets {
			if !p.Habitable || p.ID == fromP.ID {
				continue
			}
			if !w.CanTrade(p, c.Race) {
				continue
			}
			// память: не летим на только что осушенную цель
			if p.ID == c.LastTarget && w.Day-c.LastTargetAt < CaravanMemoryTicks {
				continue
			}
			// Быстрый отсев по снапшоту системы цели: если в системе нет дефицита по нужным товарам — пропускаем
			targetSnap := &w.SystemMarketSnaps[p.StarID]
			hasDeficit := false
			for _, g := range surplusGoods {
				if targetSnap.DeficitGoods[g] > 0 {
					hasDeficit = true
					break
				}
			}
			if !hasDeficit {
				continue
			}
			for _, g := range surplusGoods {
				if !w.IsLegal(fromP, g) || !w.IsLegal(p, g) {
					continue
				}
				if p.ConsumedMask&(1<<uint(g)) == 0 {
					continue
				}
				// ... дальше тело без изменений: buy, sell, rep, score ...
				buy := caravanFairPrice(fromP, g) * caravanStaticBuyMul
				if buy < 1 {
					buy = 1
				}
				sell := caravanFairPrice(p, g) * caravanDeficitMul
				// репутация цели: накопленный +/- сдвигает цену продажи
				rep := 1 + caravanRepWeight*p.TradeRep[g]/100
				if rep < 0.5 {
					rep = 0.5
				}
				score := (sell*rep/buy - 1) * 100 -
					w.caravanDistCost(fromStar, p.StarID)
				if score >= minScore && score > best.s {
					best = plan{p: p, g: g, s: score}
					found = true
				}
			}
		}
		return found
	}

	// 1. Своя система
	if scan(w.PlanetMap[fromStar], caravanMinRoute) {
		return best.p, best.g
	}
	// 2. Своё созвездие
	for _, sid := range w.ConstellationStars[fromConst] {
		if scan(w.PlanetMap[sid], caravanMinRoute) {
			return best.p, best.g
		}
	}
	// 3. Весь мир: сначала строгий порог...
	if scan(w.Planets, caravanMinRoute) {
		return best.p, best.g
	}
	// ...затем любой положительный счёт — стоять на месте хуже слабого рейса
	if scan(w.Planets, caravanMinRouteAny) {
		return best.p, best.g
	}
	return nil, 0
}


// techDiscountCaravans — скидка/наценка апгрейда каравана (по своим).
func (w *World) techDiscountCaravans(currentTL int) float64 {
	if len(w.Caravans) == 0 {
		return 0
	}
	if currentTL == 0 {
		return 0
	}
	sum := 0
	for _, c := range w.Caravans {
		sum += c.TechLevel
	}
	avg := float64(sum) / float64(len(w.Caravans))
	if currentTL == -1 {
		return avg
	}
	frac := avg - math.Floor(avg)
	inv := math.Floor(avg) - avg + 1
	if float64(currentTL) > avg+1 {
		return inv
	} else if float64(currentTL) > avg {
		return frac
	} else if float64(currentTL) < avg-1 {
		return -inv
	} else {
		return -frac
	}
}

// SpawnCaravanAt — создать караван на конкретной планете.
func (w *World) SpawnCaravanAt(planetID int) *Caravan {
	nextCaravanID++
	rngSpeed := w.Rng.Float64()

	// TL — от планеты, где родился (караван наследует технологии).
	tl := w.Planets[planetID].TechLevel

	// Стартовые деньги — случайно 3000..7000.
	money := 3000.0 + w.Rng.Float64()*4000.0

	c := &Caravan{
		ID:         nextCaravanID,
		Race:       w.Planets[planetID].Race,
		FromPlanet: planetID,
		ToPlanet:   planetID,
		Phase:      PhaseFindTrip,
		Progress:   0,
		SpeedRng:   rngSpeed,
		Speed:      0.2 + rngSpeed*(float64(tl)/10.0),
		JumpRange:  30,
		TechLevel:  tl,
		CargoMax:   tl * 10,
		Money:      money,
		Fuel:       CaravanFuelMax,
		FuelMax:    CaravanFuelMax,
		Alive:      true,
	}
	w.Caravans = append(w.Caravans, c)
	return c
}

// SpawnCaravan — случайная звезда, planets[3]. Используется при респавне.
func (w *World) SpawnCaravan() *Caravan {
	star := w.Stars[w.Rng.Intn(len(w.Stars))]
	planets := w.PlanetMap[star.ID]
	if len(planets) <= 3 || !planets[3].Habitable {
		return nil
	}
	return w.SpawnCaravanAt(planets[3].ID)
}

// SpawnCaravanWave — 1 караван на каждую звезду (planets[3]), ~420 штук.
func (w *World) SpawnCaravanWave() {
	for _, star := range w.Stars {
		planets := w.PlanetMap[star.ID]
		if len(planets) > 3 && planets[3].Habitable {
			w.SpawnCaravanAt(planets[3].ID)
		}
	}
}

// caravanIsSurplus — проверка избытка на планете p для товара g.
func (w *World) caravanIsSurplus(p *Planet, g Good) bool {
	m := &p.Market[g]
	ts := m.TargetStock
	if ts < 1 {
		ts = 1
	}
	maxStock := ts * 3
	if maxStock < 10 {
		maxStock = 10
	}
	return m.Count >= maxStock
}

// caravanIsDeficit — проверка дефицита на планете p для товара g.
// Порог: sale price > 1.15 × базовая.
func (w *World) caravanIsDeficit(p *Planet, g Good) bool {
	base := float64(GoodsMarket[g].AveragePrice)
	return p.Market[g].BaseSalePrice > 1.15*base
}

// === Локальная оценка покупки: насколько % дешевле справедливой цены ===
// Караван стоит на планете-производителе, поэтому знает текущие цену и склады.
// Справедливая цена — из справки (раса/правительство/экономика), фактическая —
// PurchasePrice, в которую UpdateMarketState уже вмазал скидку профицита
// (RemapClamped(count, ts, ts*3, 1, 1.2) — до ×0.83) и дрейф PriceState.
const (
	caravanMinBuyDiscount = 10.0 // минимальная скидка в % к справедливой цене
)

// caravanBuyIndex — насколько % дешевле справедливой цены можно купить good здесь.
func (w *World) caravanBuyIndex(p *Planet, g Good) float64 {
	m := &p.Market[g]
	gm := &GoodsMarket[g]

	purchase := m.PurchasePrice
	if purchase < 1 {
		purchase = 1
	}

	// Справедливая цена на этой планете по статике.
	fair := float64(gm.AveragePrice) *
		PlanetRaceMarket[p.Race].Goods.PriceFactor[g] *
		PlanetGovernmentMarket[p.Government].Goods.PriceFactor[g] /
		economyPriceFactor(p.Economy)
	if fair < 1 {
		fair = 1
	}

	return (1 - purchase/fair) * 100 // проценты ниже справедливой
}


// caravanCheckLocalSurplus — ищем товар, который планета производит по паттерну.
// Возвращает (good, true) если нашли, иначе (0, false).
func (w *World) caravanCheckLocalSurplus(c *Caravan) (Good, bool) {
	p := w.Planets[c.FromPlanet]
	order := w.npcGoodOrder()

	var best Good
	bestScore := caravanMinBuyDiscount
	found := false

	for _, gi := range order {
		g := Good(gi)
		if !w.CanTrade(p, c.Race) {
			continue
		}
		if !w.IsLegal(p, g) {
			continue
		}
		if p.ProducedMask&(1<<uint(g)) == 0 {
			continue
		}
		m := &p.Market[g]
		if m.Count <= 0 || m.TargetStock <= 0 {
			continue
		}
		// Покупать можно только в профиците: count > ts. Если склад провалился
		// ниже цели — рынок в дефицитной зоне, цены завышены (и, возможно,
		// какой-то другой караван этот рынок только что осушил — ждать).
		if float64(m.Count) <= float64(m.TargetStock) {
			continue
		}
		score := w.caravanBuyIndex(p, g)
		if score > bestScore {
			bestScore = score
			best = g
			found = true
		}
	}
	if found {
		return best, true
	}
	return 0, false
}


// caravanFindSurplusNearby — ближайшая планета с избытком любого товара.
// Поиск: система → созвездие → мир. Без телепорта — просто находим цель.
func (w *World) caravanFindSurplusNearby(c *Caravan) (*Planet, Good) {
	fromP := w.Planets[c.FromPlanet]
	fromStar := fromP.StarID
	fromConst := w.Stars[fromStar].ConstellationID

	type cand struct {
		p    *Planet
		g    Good
		dist float64
	}
	var best *cand

	check := func(p *Planet) {
		if !p.Habitable || p.ID == c.FromPlanet {
			return
		}
		if !w.CanTrade(p, c.Race) {
			return
		}
		for g := 0; g < int(GoodCount); g++ {
			gg := Good(g)
			if !goodTraded(gg) {
				continue // NPC не работает с неторгуемыми товарами
			}
			if !w.IsLegal(p, gg) {
				continue
			}
			if !w.caravanIsSurplus(p, gg) {
				continue
			}
			dist := w.DistMatrix[fromStar][p.StarID]
			if best == nil || dist < best.dist {
				best = &cand{p: p, g: gg, dist: dist}
			}
		}
	}

	// 1. Своя система
	for _, p := range w.PlanetMap[fromStar] {
		check(p)
	}
	if best != nil {
		return best.p, best.g
	}

	// 2. Своё созвездие
	for _, sid := range w.ConstellationStars[fromConst] {
		for _, p := range w.PlanetMap[sid] {
			check(p)
		}
	}
	if best != nil {
		return best.p, best.g
	}

	// 3. Вся карта (дорого — O(2710×8). Смотрим took; если станет тяжело, обрежем.)
	for _, p := range w.Planets {
		check(p)
	}
	if best != nil {
		return best.p, best.g
	}
	return nil, 0
}

// === Оценочная функция каравана: маржа продажи товара на планете, только из статики ===
// Караван в полёте не видит чужих складов и текущих цен — он знает только «справку»
// о планете (экономика, раса, правительство). Ожидаемая цена продажи:
//   (AveragePrice + EcoAdjustPrice*richness) * raceFactor * govFactor / ecoFactor,
// умноженная на дефицитный множитель. Планета попала в выбор по ConsumedMask,
// значит её рынок систематически вымывается (прирост в 5% тиков против убыли
// в 20%) и цена держится у потолка дефицита ×1.25 — это устойчивое свойство
// баланса, а не ясновиденье.
const (
	caravanMinMargin   = 10.0  // минимальная маржа в %, ниже — даже не рассматриваем
	caravanDistPenalty = 0.02  // сколько % маржи съедает единица дистанции DistMatrix
)

// caravanFlightIndex — ожидаемая маржа продажи good на планете p в % от AveragePrice.
// Только статические параметры планеты: экономика, раса, правительство.
func caravanFlightIndex(p *Planet, g Good) float64 {
	gm := &GoodsMarket[g]
	ri := &PlanetRaceMarket[p.Race]
	gi := &PlanetGovernmentMarket[p.Government]

	// База с поправкой на богатство: Rich добавляет EcoAdjustPrice, Poor отнимает.
	base := float64(gm.AveragePrice) + gm.EcoAdjustPrice*economyRichness(p.Economy)
	if base < 1 {
		base = 1
	}

	// Статический множитель цены: раса × правительство / фактор экономики.
	price := base * ri.Goods.PriceFactor[g] * gi.Goods.PriceFactor[g] /
		economyPriceFactor(p.Economy)

	// Пиратская планета: если CanTrade их пропускает — учесть надбавку дона.
	if p.OwnerId == oiPirate {
		price *= gm.PirateEconomyFactor
	}

	margin := price*caravanDeficitMul/float64(gm.AveragePrice) - 1
	return margin * 100 // проценты
}


// caravanFindDeficitNearby — выбор цели каравана: не ближайшая потребляющая
// планета, а лучшая по счёту «статическая маржа − штраф за дистанцию» внутри
// зон поиска. Если нигде нет маржи выше caravanMinMargin — вернёт nil (караван
// подождёт вместо бессмысленного перелёта за копейки).
func (w *World) caravanFindDeficitNearby(c *Caravan) *Planet {
	fromP := w.Planets[c.FromPlanet]
	fromStar := fromP.StarID
	fromConst := w.Stars[fromStar].ConstellationID

	var best *Planet
	// score = статическая маржа (%) минус штраф за дорогу.
	// Кандидат обязан набрать хотя бы caravanMinMargin — иначе караван честно
	// вернёт nil и подождёт, вместо того чтобы лететь за копейки.
	bestScore := caravanMinMargin

	// Быстрый отсев: собираем звезды, где есть дефичит по нашему товару.
	// Если в системе нет дефицита — даже не смотрим её планеты.
	deficitStars := make([]int, 0, 32)
	for sid := 0; sid < len(w.SystemMarketSnaps); sid++ {
		if w.SystemMarketSnaps[sid].DeficitGoods[c.Good] > 0 {
			deficitStars = append(deficitStars, sid)
		}
	}
	if len(deficitStars) == 0 {
		return nil
	}

	check := func(p *Planet) {
		if !p.Habitable || p.ID == c.FromPlanet {
			return
		}
		if !w.CanTrade(p, c.Race) {
			return
		}
		if !w.IsLegal(p, c.Good) {
			return
		}
		if p.ConsumedMask&(1<<uint(c.Good)) == 0 {
			return
		}
		score := caravanFlightIndex(p, c.Good) -
			caravanDistPenalty*float64(w.DistMatrix[fromStar][p.StarID])
		if score > bestScore {
			bestScore = score
			best = p
		}
	}

	// 1. Своя система (если она в списке дефицитных)
	if w.SystemMarketSnaps[fromStar].DeficitGoods[c.Good] > 0 {
		for _, p := range w.PlanetMap[fromStar] {
			check(p)
		}
		if best != nil {
			return best
		}
	}

	// 2. Своё созвездие
	for _, sid := range w.ConstellationStars[fromConst] {
		if w.SystemMarketSnaps[sid].DeficitGoods[c.Good] == 0 {
			continue
		}
		for _, p := range w.PlanetMap[sid] {
			check(p)
		}
	}
	if best != nil {
		return best
	}

	// 3. Остальные дефицитные системы (по всей карте)
	for _, sid := range deficitStars {
		if sid == fromStar {
			continue
		}
		// Пропускаем уже проверенные (свою систему и созвездие)
		inConst := false
		for _, cs := range w.ConstellationStars[fromConst] {
			if cs == sid {
				inConst = true
				break
			}
		}
		if inConst {
			continue
		}
		for _, p := range w.PlanetMap[sid] {
			check(p)
		}
	}
	return best
}



// caravanLoad — покупка груза на текущей планете.
// Возвращает true, если караван умер (нет денег на 1 единицу, а товар есть).
func (w *World) caravanLoad(c *Caravan) (dead bool) {
	p := w.Planets[c.FromPlanet]
	buyPrice := ShopGoodsPurchasePrice(p, c.Good)
	if buyPrice <= 0 {
		c.Phase = PhaseFindTrip
		return false
	}

	count := p.Market[c.Good].Count
	if count <= 0 {
		// товара нет — не наша вина, ищем другой
		c.Phase = PhaseFindTrip
		return false
	}

	// Резерв на топливо: караван не покупает "под ноль".
	const fuelReserve = 100.0
	available := c.Money - fuelReserve
	if available <= 0 {
		// денег нет даже на резерв — рестарт: новые деньги, TL = planet.TL - 1
		w.caravanResetBroke(c, p)
		return false
	}

	maxQ := int(available / buyPrice)
	if maxQ > c.CargoMax {
		maxQ = c.CargoMax
	}
	if maxQ > count {
		maxQ = count
	}
	if maxQ <= 0 {
		// товар есть, а денег на 1 единицу (сверх резерва) нет — рестарт
		w.caravanResetBroke(c, p)
		return false
	}

	cost, ok := w.Buy(p, c.Good, maxQ, c.Race)
	if !ok {
		c.Phase = PhaseFindTrip
		return false
	}
	c.Money -= cost * 0.975
	c.BaseCost = cost
	c.buyPrice = cost / float64(maxQ)
	c.Cargo = maxQ
	c.Looted = false

	// История: покупка удалась (buyPrice = цена закупки, sellPrice = 0).
	w.caravanAddHistory(c, c.FromPlanet, c.ToPlanet, c.Good, c.buyPrice, 0, true, "")

	// Апгрейд (как у трейдера)
	if c.TechLevel < 15 {
		upcost := TotalTL(p.TechLevel, 5000) * (w.techDiscountAgents(c.TechLevel) + 1)
		if c.Money >= upcost+5000*float64(c.TechLevel) {
			c.Money -= upcost
			p.Money += upcost * 0.25
			c.TechLevel++
			c.Speed = 0.2 + c.SpeedRng*(float64(c.TechLevel)/10.0)
			c.CargoMax = c.TechLevel * 10
		}
	}

	c.Phase = PhaseFindDeficit
	c.Progress = 0
	return false
}

// caravanResetBroke — караван обанкротился: сброс денег и TL до (planet.TL - 1).
func (w *World) caravanResetBroke(c *Caravan, p *Planet) {
	// Новые стартовые деньги (как при спавне).
	c.Money = 3000.0 + w.Rng.Float64()*4000.0

	// TL = планеты - 1 (минимум 1).
	newTL := p.TechLevel - 1
	if newTL < 1 {
		newTL = 1
	}
	c.TechLevel = newTL
	c.Speed = 0.2 + c.SpeedRng*(float64(newTL)/10.0)
	c.CargoMax = newTL * 10

	// Сбрасываем груз и фазу — ищем новый рейс.
	c.Cargo = 0
	c.BaseCost = 0
	c.buyPrice = 0
	c.Good = 0
	c.Phase = PhaseFindTrip
	c.Progress = 0

	// История: банкротство/рестарт.
	w.caravanAddHistory(c, c.FromPlanet, c.FromPlanet, 0, 0, 0, false, "no_money")
}

const caravanRepStep = 0.001 // шаг репутации за одну сделку


// caravanUnload — продажа груза на текущей планете.
// Репутация планеты по товару: сделка в прибыль (продали дороже, чем купили) —
// планете +, сделка в убыток (продали дешевле покупки) — планете −.
func (w *World) caravanUnload(c *Caravan) {
	p := w.Planets[c.ToPlanet]
	if c.Cargo > 0 {
		// Цена покупки за единицу — снимаем ДО изменения Cargo/BaseCost.
		buyPerUnit := 0.0
		if c.BaseCost > 0 {
			buyPerUnit = c.BaseCost / float64(c.Cargo)
		}

		rev, sold, ok := w.Sell(p, c.Good, c.Cargo, c.Race)
		if ok && sold > 0 {
			actPerUnit := rev / float64(sold)

			c.Money += rev * 1.025
			c.Cargo -= sold
			if c.Cargo <= 0 {
				c.Cargo = 0
				c.BaseCost = 0
				c.buyPrice = 0
			} else {
				// Непроданный остаток несёт свою долю затрат пропорционально.
				c.BaseCost *= float64(c.Cargo) / float64(c.Cargo+sold)
			}

			// --- Репутация планеты: прибыль/убыток сделки ---
			// > — в прибыль: планете +; < — в убыток: планете −; = — ноль-ноль,
			// рынок угадал цену покупки — вердикта нет.
			if buyPerUnit > 0 {
				switch {
				case actPerUnit > c.buyPrice:
					p.TradeRep[c.Good] += caravanRepStep
				case actPerUnit < c.buyPrice:
					p.TradeRep[c.Good] -= caravanRepStep
				}
			}

			// История: продажа удалась (buyPrice = цена закупки, sellPrice = цена продажи).
			w.caravanAddHistory(c, c.FromPlanet, c.ToPlanet, c.Good, c.buyPrice, actPerUnit, true, "")

		} else {
			// История: продажа не удалась.
			reason := "no_goods"
			if !ok {
				reason = "price"
			}
			w.caravanAddHistory(c, c.FromPlanet, c.ToPlanet, c.Good, 0, 0, false, reason)
		}
	} else {
		// История: прилетел, но груза нет (не должно происходить).
		w.caravanAddHistory(c, c.FromPlanet, c.ToPlanet, c.Good, 0, 0, false, "no_cargo")
	}
	if c.Cargo == 0 {
		c.Phase = PhaseIdle
	} else {
		c.Phase = PhaseFindDeficit
	}
	c.Progress = 0
}

// caravanAddHistory — добавить запись в историю каравана (кольцевой буфер).
func (w *World) caravanAddHistory(c *Caravan, sourceID, destID int, g Good, buyPrice, sellPrice float64, fine bool, reason string) {
	idx := c.HistoryHead
	c.History[idx] = CaravanHistoryEntry{
		SourceID:      sourceID,
		DestinationID: destID,
		Good:          g,
		BuyPrice:      buyPrice,
		SellPrice:     sellPrice,
		FineTrade:     fine,
		Reason:        reason,
		Day:           w.Day,
	}
	c.HistoryHead = (c.HistoryHead + 1) % CaravanHistorySize
	if c.HistoryLen < CaravanHistorySize {
		c.HistoryLen++
	}
}
