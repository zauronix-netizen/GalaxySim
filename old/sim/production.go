package sim

// Набросок производства/потребления по расам.
// Основа: вики КР (rangers.fandom.com) + StockFactor из consts.go.
// ProdCoef — множитель производства (сколько товара производится за тик).
// ConsCoef — множитель потребления (сколько товара уходит за тик).
// Итог: Count += (ProdCoef - ConsCoef) * Pop / 10000.
// Пока — набросок, крутим потом.

type RaceProdCons struct {
	ProdCoef [GoodCount]float64
	ConsCoef [GoodCount]float64
}

var PlanetRaceProdCons = [RaceCount]RaceProdCons{
	// Maloc (0): еда/медицина — производят, оружие — мало, алкоголь/наркотики — почти нет.
	{
		ProdCoef: [GoodCount]float64{1.2, 1.1, 0.6, 0.4, 0.8, 0.1, 1.0, 0.1},
		ConsCoef: [GoodCount]float64{1.0, 0.8, 0.9, 1.1, 0.9, 0.2, 0.8, 0.1},
	},
	// Peleng (1): еда/роскошь — производят, наркотики — мало (но легальны).
	{
		ProdCoef: [GoodCount]float64{1.1, 0.9, 0.8, 1.3, 0.7, 1.2, 0.7, 0.4},
		ConsCoef: [GoodCount]float64{1.0, 1.0, 1.0, 0.9, 0.8, 1.3, 0.8, 0.6},
	},
	// Human (2): всё норма, сбалансировано.
	{
		ProdCoef: [GoodCount]float64{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
		ConsCoef: [GoodCount]float64{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
	},
	// Feyan (3): техника — производят, еда/минералы/оружие/наркотики — мало.
	{
		ProdCoef: [GoodCount]float64{0.6, 0.8, 1.5, 0.9, 0.4, 0.3, 0.4, 0.3},
		ConsCoef: [GoodCount]float64{0.7, 0.9, 1.2, 0.8, 0.5, 0.5, 0.5, 0.4},
	},
	// Gaal (4): техника — чуть производят, всё остальное — мало.
	{
		ProdCoef: [GoodCount]float64{0.4, 0.4, 0.9, 0.4, 0.2, 0.2, 0.1, 0.2},
		ConsCoef: [GoodCount]float64{0.5, 0.5, 0.8, 0.4, 0.3, 0.3, 0.2, 0.3},
	},
}

// Множители производства/потребления по экономике (Agri/Mixed/Ind).
// Основа: вики КР (аграрные — еда/алкоголь, индустриальные — техника/оружие).
var EcoProdFactor = [EcoCount][GoodCount]float64{
	// Agri: еда, алкоголь — больше; техника, оружие — меньше.
	{1.5, 1.1, 0.7, 1.0, 1.0, 1.5, 0.7, 1.0},
	// Mixed: всё нейтрально.
	{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
	// Ind: техника, оружие, минералы — больше; еда, алкоголь — меньше.
	{0.7, 1.0, 1.5, 1.1, 1.3, 0.7, 1.5, 1.0},
}

var EcoConsFactor = [EcoCount][GoodCount]float64{
	// Agri: потребление еды/алкоголя — меньше (сами производят).
	{0.8, 1.0, 1.1, 1.0, 1.0, 0.8, 1.1, 1.0},
	// Mixed: нейтрально.
	{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
	// Ind: потребление еды/алкоголя — больше (рабочие).
	{1.2, 1.0, 0.9, 1.0, 1.0, 1.2, 0.9, 1.0},
}

// Множители по политике (Anarchy/Dictatorship/Monarchy/Republic/Democracy).
// Основа: вики КР (налоги, регулирование).
var GovProdFactor = [GovCount][GoodCount]float64{
	// Anarchy: без регулирования.
	{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
	// Dictatorship: оружие +, наркотики − (контроль).
	{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.2, 0.8},
	// Monarchy: роскошь +, еда + (традиции).
	{1.1, 1.0, 1.0, 1.3, 1.0, 1.0, 1.0, 1.0},
	// Republic: техника +, медицина +.
	{1.0, 1.1, 1.2, 1.0, 1.0, 1.0, 1.0, 1.0},
	// Democracy: еда +, медицина +, наркотики − (запреты).
	{1.1, 1.2, 1.0, 1.0, 1.0, 1.0, 1.0, 0.7},
}

var GovConsFactor = [GovCount][GoodCount]float64{
	// Anarchy: нейтрально.
	{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
	// Dictatorship: оружие −, наркотики + (чёрный рынок).
	{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 0.8, 1.2},
	// Monarchy: роскошь + (двор).
	{1.0, 1.0, 1.0, 1.3, 1.0, 1.0, 1.0, 1.0},
	// Republic: техника + (бюрократия).
	{1.0, 1.0, 1.2, 1.0, 1.0, 1.0, 1.0, 1.0},
	// Democracy: еда +, наркотики − (запреты).
	{1.1, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 0.8},
}

// Применить производство/потребление к планете.
// Вызывать в phaseEconomy, ДО UpdateMarketState.
func (w *World) applyProduction(p *Planet) {
	if !p.Habitable {
		return
	}
	rc := &PlanetRaceProdCons[p.Race]
	ecoP := &EcoProdFactor[p.Economy]
	ecoC := &EcoConsFactor[p.Economy]
	govP := &GovProdFactor[p.Government]
	govC := &GovConsFactor[p.Government]
	base := float64(p.Pop) / 10000.0
	if base < 1 {
		base = 1
	}

	// --- Основное производство/потребление (раса × экономика × политика) ---
	for g := 0; g < int(GoodCount); g++ {
		m := &p.Market[g]
		prod := rc.ProdCoef[g] * ecoP[g] * govP[g] * base
		cons := rc.ConsCoef[g] * ecoC[g] * govC[g] * base

		// Бустер еды при голоде: порог тот же, что у новости (count < ts/5).
		if g == int(GoodFood) {
			tsFood := m.TargetStock
			if tsFood < 1 {
				tsFood = 1
			}
/*			if m.Count < tsFood/5 {
				boost := float64(tsFood) * 0.5    // 50% от ts
				need := float64(tsFood - m.Count) // до потолка (ts)
				if boost > need {
					boost = need
				}
				prod += boost
			} */
		}

		// Для веба: сохраняем prod-cons (float) за этот тик — ДО изменения Count.
		m.Delta = prod - cons

		delta := int(prod - cons)
		if delta == 0 {
			if w.Rng.Intn(100) < 50 {
				delta = 1
			} else {
				delta = -1
			}
		}
		m.Count += delta
		if m.Count < 0 {
			m.Count = 0
		}
		maxStock := m.TargetStock * 3
		if maxStock < 10 {
			maxStock = 10
		}
		if m.Count > maxStock {
			m.Count = maxStock
		}
	}

	// --- Спецтовары: дополнительное производство на «подходящих» планетах ---
	// Малые количества. Не везде. Игрок возит (где дорого).
	addExtra := func(g Good, amount int) {
		if amount < 1 {
			amount = 1
		}
		m := &p.Market[g]
		m.Count += amount
		maxStock := m.TargetStock * 3
		if maxStock < 10 {
			maxStock = 10
		}
		if m.Count > maxStock {
			m.Count = maxStock
		}
	}

	// Arms (оружие): Dictatorship / Maloc / Ind
	if p.Government == GovDictatorship || p.Race == RaceMaloc || p.Economy == EcoInd {
		extra := int(base * 0.3)
		addExtra(GoodArms, extra)
	}
	// Alcohol (алкоголь): Peleng / Agri / Monarchy
	if p.Race == RacePeleng || p.Economy == EcoAgri || p.Government == GovMonarchy {
		extra := int(base * 0.3)
		addExtra(GoodAlcohol, extra)
	}
	// Luxury (роскошь): Monarchy / Republic / Peleng / Ind
	if p.Government == GovMonarchy || p.Government == GovRepublic || p.Race == RacePeleng || p.Economy == EcoInd {
		extra := int(base * 0.2)
		addExtra(GoodLuxury, extra)
	}
	// Narcotics (наркотики): Anarchy/Dictatorship + Peleng/Human + не Agri
	if (p.Government == GovAnarchy || p.Government == GovDictatorship) &&
		(p.Race == RacePeleng || p.Race == RaceHuman) && p.Economy != EcoAgri {
		extra := int(base * 0.4)
		addExtra(GoodNarcotics, extra)
	}
}