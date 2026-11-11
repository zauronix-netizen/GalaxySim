package sim

import "math"

// ShopGoodsPurchasePrice — цена для торговца с учётом TL планеты (+1% за уровень).
func ShopGoodsPurchasePrice(p *Planet, g Good) float64 {
	base := math.Max(1, p.Market[g].PurchasePrice)
	tlMult := 1.0 + float64(p.TechLevel)/100.0
	return math.Max(1, base*tlMult)
}

// ShopGoodsSellPrice — цена продажи с учётом TL планеты (+1% за уровень).
func ShopGoodsSellPrice(p *Planet, g Good) float64 {
	m := &p.Market[g]
	base := math.Max(1, math.Round(m.BaseSalePrice))
	tlMult := 1.0 + float64(p.TechLevel)/100.0
	return math.Max(1, math.Round(base*tlMult))
}

func (w *World) CanTrade(p *Planet, buyerRace Race) bool {
	return w.Rel[p.Race][buyerRace] >= 20
}

func (w *World) Buy(p *Planet, g Good, q int, buyerRace Race) (cost float64, ok bool) {
	if q <= 0 || !w.CanTrade(p, buyerRace) {
		return 0, false
	}
	m := &p.Market[g]
	if m.Count < q {
		return 0, false
	}
	price := ShopGoodsPurchasePrice(p, g)
	cost = price * float64(q)
	m.Count -= q
	p.Money += cost
	return cost, true
}

func (w *World) Sell(p *Planet, g Good, q int, sellerRace Race) (revenue float64, sold int, ok bool) {
	if q <= 0 || !w.CanTrade(p, sellerRace) {
		return 0, 0, false
	}
	price := ShopGoodsSellPrice(p, g)
	if price <= 0 {
		return 0, 0, false
	}

	// Сколько планета реально может купить?
	maxByMoney := int(p.Money / price)
	if maxByMoney < q {
		q = maxByMoney
	}
	if q <= 0 {
		return 0, 0, false
	}

	revenue = price * float64(q)
	m := &p.Market[g]
	m.Count += q
	p.Money -= revenue // планета платит — деньги реальные
	return revenue, q, true
}

func (w *World) FindBestQueuedSellPlanetProfitScore(c *Caravan, g Good, unitCost float64) (int, *Planet) {
	bestScore := 0
	var best *Planet
	fromStar := w.Planets[c.FromPlanet].StarID
	for _, p := range w.Planets {
		if !p.Habitable {
			continue
		}
		var score int
		if p.StarID == fromStar {
			score = int(RemapClamped(ShopGoodsSellPrice(p, g)/unitCost, 1, 1.4, 0, 100))
		} else {
			score = int(RemapClamped(ShopGoodsSellPrice(p, g)/unitCost, 1, 2, 0, 100))
		}
		if score > bestScore {
			bestScore = score
			best = p
		}
	}
	return bestScore, best
}

// SelectBestTradePlanet — замена reassignCaravan с профит-скором.
func (w *World) SelectBestTradePlanet(c *Caravan) (*Planet, Good) {
	fromPlanet := w.Planets[c.FromPlanet]
	fromID := fromPlanet.ID
	canTradeFrom := w.tickCanTrade[fromID][c.Race]
	legalFrom := w.tickLegal[fromID]
	buyFrom := w.tickBuyPrice[fromID]

	var best *Planet
	var bestGood Good = -1
	bestScore := 0.0

	// C1: кандидаты — своя система + соседние звёзды (Neighbors).
	// Буфер переиспользуемый, без make на каждый караван (GC на T6600).
	cand := w.tickCandidates[:0]
	for _, p := range w.PlanetMap[fromPlanet.StarID] {
		if p.Habitable && p.ID != fromID {
			cand = append(cand, p)
		}
	}
	for _, sid := range w.Neighbors[fromPlanet.StarID] {
		for _, p := range w.PlanetMap[sid] {
			if p.Habitable && p.ID != fromID {
				cand = append(cand, p)
			}
		}
	}
	w.tickCandidates = cand

	for _, p := range cand {
		if !w.tickCanTrade[p.ID][c.Race] || !canTradeFrom {
			continue
		}
		legalTo := w.tickLegal[p.ID]
		sellTo := w.tickSellPrice[p.ID]
		for g := 0; g < int(GoodCount); g++ {
			if !legalFrom[g] || !legalTo[g] {
				continue
			}
			if fromPlanet.Market[g].Count < GoodsMarket[g].BaseStock/3 {
				continue
			}
			buyPrice := buyFrom[g]
			sellPrice := sellTo[g]
			if buyPrice <= 0 {
				continue
			}
			ratio := sellPrice / buyPrice
			if ratio < 1.11 {
				continue
			}
			dist := w.Dist(fromPlanet.StarID, p.StarID)
			distFactor := RemapClamped(dist/c.JumpRange, 0, 3, 1.3, 1)
			score := ratio * distFactor
			if score > bestScore {
				bestScore = score
				best = p
				bestGood = Good(g)
			}
		}
	}
	return best, bestGood
}