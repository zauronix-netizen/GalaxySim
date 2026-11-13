package sim

import "math"

func RemapClamped(v, inMin, inMax, outMin, outMax float64) float64 {
	if inMax == inMin {
		return outMin
	}
	t := (v - inMin) / (inMax - inMin)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return outMin + t*(outMax-outMin)
}

func (w *World) NextRandomFloatRange(lo, hi float64) float64 {
	return lo + w.Rng.Float64()*(hi-lo)
}

func (w *World) scaleGoodsPriceByGalaxyAge(base float64) float64 {
	if InflationOff {
		return base
	}
	return math.Round(RemapClamped(float64(w.Day),
		GoodsInflationStartTurn, GoodsInflationEndTurn,
		base*GoodsInflationMin, base*GoodsInflationMax))
}

func (w *World) scaleGoodsStockByGalaxyAge(base float64) float64 {
	if InflationOff {
		return base
	}
	return math.Round(RemapClamped(float64(w.Day),
		GoodsInflationStartTurn, GoodsInflationEndTurn,
		base*GoodsStockMin, base*GoodsStockMax))
}

// UpdateMarketState — порт TPlanet.UpdateMarketState.
func (w *World) UpdateMarketState(p *Planet, g Good) {
	gm := &GoodsMarket[g]
	m := &p.Market[g]
	ri := &PlanetRaceMarket[p.Race]
	gi := &PlanetGovernmentMarket[p.Government]

	// ecoFactor — от экономики планеты (0..8), не от товара.
	// 0 = Agri-Poor ... 8 = Ind-Rich. Rich дешевле, Poor дороже.
	// Нормализуем: база 1.0, шаг ±0.1 на градацию, ±0.05 на тип.
	ecoFactor := economyPriceFactor(p.Economy)
	if p.OwnerId == oiPirate {
		ecoFactor *= gm.PirateEconomyFactor
	}

	radiusRemap := RemapClamped(p.Radius, 60, 100, 0.5, 1.5)

	// C2: множитель населения. PopRef — стартовая Pop планеты.
	popFactor := 1.0
	if p.PopRef >= 1 {
		popFactor = float64(p.Pop) / float64(p.PopRef)
	}

	targetStock := math.Round(float64(gm.BaseStock) *
		ri.Goods.StockFactor[g] *
		gi.Goods.StockFactor[g] *
		ecoFactor *
		radiusRemap *
		popFactor)
	if targetStock < 1 {
		targetStock = 1
	}
	m.TargetStock = int(targetStock)

	// Цена: базовая + поправка от экономики планеты (Elite-стиль, упрощённо).
	// EcoAdjustPrice >0 — дороже в богатых, <0 — дешевле в богатых.
	// Economy 0..8: Poor (0,3,6) — дороже, Rich (2,5,8) — дешевле.
	ecoAdj := gm.EcoAdjustPrice * economyRichness(p.Economy)
	targetPrice := float64(gm.AveragePrice) + ecoAdj
	if targetPrice < 1 {
		targetPrice = 1
	}
	targetPrice *= ri.Goods.PriceFactor[g] * gi.Goods.PriceFactor[g] / ecoFactor

	count := float64(m.Count)
	ts := float64(m.TargetStock)
	if count < ts {
		targetPrice /= RemapClamped(count, ts*0.1, ts, 0.8, 1)
	} else {
		targetPrice /= RemapClamped(count, ts, ts*3, 1, 1.2)
	}

	if float64(gm.MinPrice) < targetPrice {
		if targetPrice > float64(gm.MaxPrice)+1 {
			targetPrice = float64(gm.MaxPrice) + 1
		}
	} else {
		if targetPrice < float64(gm.MinPrice)-1 {
			targetPrice = float64(gm.MinPrice) - 1
		}
	}
	m.TargetPrice = targetPrice

	var step float64
	if m.PriceState-targetPrice >= 0 {
		step = targetPrice * w.NextRandomFloatRange(0.005, 0.008)
	} else {
		step = -targetPrice * w.NextRandomFloatRange(0.005, 0.008)
	}
	roll := w.Rng.Intn(100) + 1
	switch {
	case roll <= 70:
		m.PriceState -= step
	case roll <= 90:
		// без изменений
	default:
		m.PriceState += step
	}

	if m.PriceState < float64(gm.MinPrice)/2 {
		m.PriceState = float64(gm.MinPrice) / 2
	}
	if m.PriceState > float64(gm.MaxPrice)*2 {
		m.PriceState = float64(gm.MaxPrice) * 2
	}

	purchase := math.Max(2, math.Round(m.PriceState))
	baseSale := math.Max(purchase/2+1,
		math.Round(m.PriceState*RemapClamped(count, ts, ts*2.2, 0.99, 0.5)-1))
	m.PurchasePrice = purchase
	m.BaseSalePrice = baseSale

	var stockStep float64
	if count-ts >= 0 {
		stockStep = math.Round(ts*w.NextRandomFloatRange(0.0025, 0.005) + w.Rng.Float64())
	} else {
		stockStep = math.Round(-ts*w.NextRandomFloatRange(0.0025, 0.005) - w.Rng.Float64())
	}
	sroll := w.Rng.Intn(100) + 1
	switch {
	case sroll <= 20:
		m.Count -= int(stockStep)
	case sroll <= 95:
		// без изменений
	default:
		m.Count += int(stockStep)
	}
	if m.Count < 0 {
		m.Count = 0
	}
}

func (w *World) ForceGoodsScarcity(p *Planet, g Good) {
	gm := &GoodsMarket[g]
	m := &p.Market[g]
	m.PriceState = float64(gm.MaxPrice)
	if m.Count > gm.BaseStock/10 {
		m.Count = gm.BaseStock / 10
	}
	m.PurchasePrice = math.Round(m.PriceState)
	m.BaseSalePrice = math.Max(1, math.Round(m.PriceState*0.98-1))
}

func (w *World) ForceGoodsSurplus(p *Planet, g Good) {
	gm := &GoodsMarket[g]
	m := &p.Market[g]
	m.PriceState = float64(gm.MinPrice)
	if m.Count < gm.BaseStock*3 {
		m.Count += gm.BaseStock / 5
		if m.Count > gm.BaseStock*3 {
			m.Count = gm.BaseStock * 3
		}
	}
	m.PurchasePrice = math.Round(m.PriceState)
	m.BaseSalePrice = math.Max(1, math.Round(m.PriceState*0.98-1))
}

func (w *World) ComputeGlobalGoodsPriceBands() {
	for g := 0; g < int(GoodCount); g++ {
		gm := &GoodsMarket[g]
		base := &GoodsMarketBase[g]

		// Считаем от НЕИЗМЕННОЙ базы, чтобы не накапливать сжатие/инфляцию.
		minP := w.scaleGoodsPriceByGalaxyAge(float64(base.MinPrice))
		avgP := w.scaleGoodsPriceByGalaxyAge(float64(base.AveragePrice))
		maxP := w.scaleGoodsPriceByGalaxyAge(float64(base.MaxPrice))
		gm.MinPrice = int(minP)
		gm.AveragePrice = int(avgP)
		gm.MaxPrice = int(maxP)
		gm.BaseStock = int(w.scaleGoodsStockByGalaxyAge(float64(base.BaseStock)))

		spread := float64(gm.AveragePrice - gm.MinPrice)
		minP = float64(gm.MinPrice) + math.Round(spread*Difficulty.MarketPriceBandSqueeze)
		if minP >= float64(gm.AveragePrice)-1 {
			minP = float64(gm.AveragePrice) - 2
		}
		maxP = float64(gm.MaxPrice) - math.Round(spread*Difficulty.MarketPriceBandSqueeze)
		if maxP <= float64(gm.AveragePrice)+1 {
			maxP = float64(gm.AveragePrice) + 2
		}
		gm.MinPrice = int(minP)
		gm.MaxPrice = int(maxP)
	}
}

// economyPriceFactor — множитель цены от экономики планеты.
// Rich дешевле (0.85..0.95), Poor дороже (1.05..1.15), Avg = 1.0.
func economyPriceFactor(e Economy) float64 {
	switch e {
	case EcoAgriPoor, EcoMixedPoor, EcoIndPoor:
		return 1.10
	case EcoAgriAvg, EcoMixedAvg, EcoIndAvg:
		return 1.00
	case EcoAgriRich, EcoMixedRich, EcoIndRich:
		return 0.90
	}
	return 1.0
}

// economyRichness — поправка к EcoAdjustPrice: Rich усиливает (1.0),
// Poor ослабляет (0.5). Avg — 0.75.
func economyRichness(e Economy) float64 {
	switch e {
	case EcoAgriRich, EcoMixedRich, EcoIndRich:
		return 1.0
	case EcoAgriAvg, EcoMixedAvg, EcoIndAvg:
		return 0.75
	case EcoAgriPoor, EcoMixedPoor, EcoIndPoor:
		return 0.5
	}
	return 0.75
}