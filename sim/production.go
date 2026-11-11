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
	// Maloc (0): низшие. Еда/медицина/минералы/оружие — производят, роскошь — нет.
	{
		ProdCoef: [GoodCount]float64{1.3, 1.2, 0.6, 0.3, 1.1, 0.1, 1.1, 0.1},
		ConsCoef: [GoodCount]float64{1.0, 0.9, 0.8, 1.0, 0.9, 0.2, 0.8, 0.1},
	},
	// Peleng (1): торговцы. Еда/роскошь/алкоголь/наркотики — производят.
	{
		ProdCoef: [GoodCount]float64{1.2, 1.0, 0.8, 1.4, 0.8, 1.3, 0.7, 0.5},
		ConsCoef: [GoodCount]float64{1.0, 1.0, 1.0, 0.9, 0.8, 1.2, 0.7, 0.6},
	},
	// Human (2): база, лёгкий плюс, чтобы не стояли.
	{
		ProdCoef: [GoodCount]float64{1.05, 1.05, 1.05, 1.0, 1.05, 1.0, 1.05, 1.0},
		ConsCoef: [GoodCount]float64{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
	},
	// Feyan (3): техно. Technics/Luxury — производят, еда/минералы — импорт.
	{
		ProdCoef: [GoodCount]float64{0.7, 0.9, 1.5, 1.0, 0.4, 0.4, 0.5, 0.3},
		ConsCoef: [GoodCount]float64{0.7, 0.9, 1.1, 0.7, 0.5, 0.5, 0.4, 0.3},
	},
	// Gaal (4): высшие техно. Technics — экспорт, всё остальное — импорт.
	{
		ProdCoef: [GoodCount]float64{0.5, 0.5, 1.3, 0.5, 0.3, 0.3, 0.2, 0.2},
		ConsCoef: [GoodCount]float64{0.5, 0.5, 0.7, 0.4, 0.3, 0.3, 0.2, 0.3},
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

			// ЭКСПЕРИМЕНТ: Delta = (prod-cons)/16 * TechLevel.
			// Временно (TL=3 статичен). При росте TL и Pop-динамике — пересмотреть.
			tl := p.TechLevel
			if tl < 1 {
				tl = 1
			}
			d := (prod - cons) / 16.0 * float64(tl)
			m.Delta = d
			delta := int(d)

			//

		if delta == 0 {
			if d > 0 {
				delta = 1
			} else if d < 0 {
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

	// --- Налог с населения ---
	// Если денег меньше стартового (MoneyRef), планета собирает налог с населения.
	// Pop не страдает (условный "из воздуха", но привязан к населению и бедности).
	// Богатые планеты налог не платят — они уже "живут с торговли".
	if p.Money < p.MoneyRef {
		tax := float64(p.Pop) / 100.0
		p.Money += tax
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

	// Arms (оружие): Dict / Maloc / Ind / Republic / Feyan
	if p.Government == GovDictatorship || p.Government == GovRepublic ||
		p.Race == RaceMaloc || p.Race == RaceFeyan || p.Economy == EcoInd {
		addExtra(GoodArms, int(base*0.3))
	}
	// Alcohol (алкоголь): Peleng / Agri / Monarchy / Anarchy
	if p.Race == RacePeleng || p.Economy == EcoAgri ||
		p.Government == GovMonarchy || p.Government == GovAnarchy {
		addExtra(GoodAlcohol, int(base*0.3))
	}
	// Luxury (роскошь): Monarchy / Republic / Democracy / Peleng / Feyan / Ind
	if p.Government == GovMonarchy || p.Government == GovRepublic ||
		p.Government == GovDemocracy || p.Race == RacePeleng ||
		p.Race == RaceFeyan || p.Economy == EcoInd {
		addExtra(GoodLuxury, int(base*0.2))
	}
	// Narcotics (наркотики): (Anarchy/Dict) + (Peleng/Human/Maloc) + не Agri
	if (p.Government == GovAnarchy || p.Government == GovDictatorship) &&
		(p.Race == RacePeleng || p.Race == RaceHuman || p.Race == RaceMaloc) &&
		p.Economy != EcoAgri {
		addExtra(GoodNarcotics, int(base*0.4))
	}
	// Technics (техника): Feyan / Gaal / Ind / Republic
	if p.Race == RaceFeyan || p.Race == RaceGaal ||
		p.Economy == EcoInd || p.Government == GovRepublic {
		addExtra(GoodTechnics, int(base*0.4))
	}
	// Minerals (минералы): Maloc / Ind / Agri
	if p.Race == RaceMaloc || p.Economy == EcoInd || p.Economy == EcoAgri {
		addExtra(GoodMinerals, int(base*0.3))
	}
}