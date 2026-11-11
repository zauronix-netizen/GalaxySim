package sim

import "math"

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
		CargoMax:   tl * 100,
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

// caravanCheckLocalSurplus — 8 товаров случайно, ищем избыток на текущей планете.
// Возвращает (good, true) если нашли, иначе (0, false).
func (w *World) caravanCheckLocalSurplus(c *Caravan) (Good, bool) {
	p := w.Planets[c.FromPlanet]
	order := w.Rng.Perm(int(GoodCount))
	for _, gi := range order {
		g := Good(gi)
		if !w.CanTrade(p, c.Race) {
			continue
		}
		if !w.IsLegal(p, g) {
			continue
		}
		if w.caravanIsSurplus(p, g) {
			return g, true
		}
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

// caravanFindDeficitNearby — ближайшая планета, где sale price > 1.15×base.
func (w *World) caravanFindDeficitNearby(c *Caravan) *Planet {
	fromP := w.Planets[c.FromPlanet]
	fromStar := fromP.StarID
	fromConst := w.Stars[fromStar].ConstellationID

	var best *Planet
	bestDist := math.MaxFloat64

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
		if !w.caravanIsDeficit(p, c.Good) {
			return
		}
		dist := w.DistMatrix[fromStar][p.StarID]
		if dist < bestDist {
			bestDist = dist
			best = p
		}
	}

	// 1. Своя система
	for _, p := range w.PlanetMap[fromStar] {
		check(p)
	}
	if best != nil {
		return best
	}

	// 2. Своё созвездие
	for _, sid := range w.ConstellationStars[fromConst] {
		for _, p := range w.PlanetMap[sid] {
			check(p)
		}
	}
	if best != nil {
		return best
	}

	// 3. Вся карта
	for _, p := range w.Planets {
		check(p)
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
		// денег нет даже на резерв — смерть
		w.DeathsMoney++
		return true
	}

	maxQ := int(available / buyPrice)
	if maxQ > c.CargoMax {
		maxQ = c.CargoMax
	}
	if maxQ > count {
		maxQ = count
	}
	if maxQ <= 0 {
		// товар есть, а денег на 1 единицу (сверх резерва) нет — смерть
		w.DeathsMoney++
		return true
	}

	cost, ok := w.Buy(p, c.Good, maxQ, c.Race)
	if !ok {
		c.Phase = PhaseFindTrip
		return false
	}
	c.Money -= cost * 0.9
	c.BaseCost = cost
	c.Cargo = maxQ
	c.Looted = false

	// Апгрейд (как у трейдера)
	if c.TechLevel < 15 {
		upcost := totalCost[c.TechLevel] * (w.techDiscountCaravans(c.TechLevel) + 1)
		if c.Money >= upcost+5000*float64(c.TechLevel) {
			c.Money -= upcost
			p.Money += upcost * 0.01
			c.TechLevel++
			c.Speed = 0.2 + c.SpeedRng*(float64(c.TechLevel)/10.0)
			c.CargoMax = c.TechLevel * 100
		}
	}

	c.Phase = PhaseFindDeficit
	c.Progress = 0
	return false
}

// caravanUnload — продажа груза на текущей планете.
func (w *World) caravanUnload(c *Caravan) {
	p := w.Planets[c.ToPlanet]
	if c.Cargo > 0 {
		rev, sold, ok := w.Sell(p, c.Good, c.Cargo, c.Race)
		if ok && sold > 0 {
			c.Money += rev * 1.1
			c.Cargo -= sold
			if c.Cargo <= 0 {
				c.Cargo = 0
				c.BaseCost = 0
			}
		}
	}
	if c.Cargo == 0 {
		c.Phase = PhaseIdle // тик отдыха после рейса (следующий тик → PhaseFindTrip)
	} else {
		c.Phase = PhaseFindDeficit // продали не всё — ищем ещё дефицит
	}
	c.Progress = 0
}