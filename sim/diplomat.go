package sim

// SpawnBanker — создать банкира на планете spawnPlanetID, раса race.
func (w *World) SpawnBanker(spawnPlanetID int, race Race) *Diplomat {
	nextDiplomatID++
	d := &Diplomat{
		ID:         nextDiplomatID,
		Role:       DiplomatRoleBanker,
		FromPlanet: spawnPlanetID,
		ToPlanet:   spawnPlanetID,
		Phase:      PhaseIdle,
		Progress:   0,
		Speed:      2 + w.Rng.Float64()*3,
		JumpRange:  35,
		Fuel:       CaravanFuelMax,
		FuelMax:    CaravanFuelMax,
		Alive:      true,
	}
	w.Diplomats = append(w.Diplomats, d)
	return d
}

// bankerFindSource — планета своей расы с максимальным TL, Money > effRef.
func (w *World) bankerFindSource(race Race) *Planet {
	var best *Planet
	for _, p := range w.Planets {
		if !p.Habitable || p.Race != race {
			continue
		}
		effRef := p.MoneyRef * (1.0 + float64(p.TechLevel)/10.0)
		if p.Money+p.MoneySafe <= effRef {
			continue
		}
		if best == nil || p.TechLevel > best.TechLevel {
			best = p
		}
	}
	return best
}

// bankerFindTarget — планета своей расы с минимальным TL, Money < effRef.
// Исключает fromPlanet (источник).
func (w *World) bankerFindTarget(race Race, fromPlanet int) *Planet {
	var best *Planet
	for _, p := range w.Planets {
		if !p.Habitable || p.Race != race || p.ID == fromPlanet {
			continue
		}
		effRef := p.MoneyRef * (1.0 + float64(p.TechLevel)/10.0)
		if p.Money >= effRef {
			continue
		}
		if best == nil || p.TechLevel < best.TechLevel {
			best = p
		}
	}
	return best
}

// bankerArrive — банкир прилетел к приёмнику. Переводит деньги.
func (w *World) bankerArrive(d *Diplomat) {
	race := w.Planets[d.FromPlanet].Race

	src := w.bankerFindSource(race)
	if src == nil {
		d.Phase = PhaseIdle
		return
	}
	srcEffRef := src.MoneyRef * (1.0 + float64(src.TechLevel)/10.0)
	surplus := src.Money + src.MoneySafe - srcEffRef
	if surplus <= 0 {
		d.Phase = PhaseIdle
		return
	}

	trg := w.Planets[d.ToPlanet]
	trgEffRef := trg.MoneyRef * (1.0 + float64(trg.TechLevel)/10.0)
	need := trgEffRef - trg.Money
	if need <= 0 {
		d.Phase = PhaseIdle
		return
	}

	transfer := surplus
	if transfer > need {
		transfer = need
	}
	// Сначала списываем с кубышки, потом с Money.
	if src.MoneySafe >= transfer {
		src.MoneySafe -= transfer
	} else {
		rest := transfer - src.MoneySafe
		src.MoneySafe = 0
		src.Money -= rest
	}
	trg.Money += transfer

	d.Phase = PhaseIdle
}

// bankerPlanRoute — банкир ищет маршрут: источник → приёмник.
func (w *World) bankerPlanRoute(d *Diplomat) {
	race := w.Planets[d.FromPlanet].Race
	src := w.bankerFindSource(race)
	if src == nil {
		return
	}
	trg := w.bankerFindTarget(race, src.ID)
	if trg == nil {
		return
	}

	route := w.buildRoute(src.StarID, trg.StarID, d.JumpRange)
	if route == nil {
		return
	}

	d.FromPlanet = src.ID
	d.ToPlanet = trg.ID
	d.Route = route
	d.RouteIdx = 0
	d.Phase = DiplomatPhaseTakeoff
	d.Progress = 0
}

// diplomatPlanRoute — обычный дипломат ищет маршрут: разные расы, случайная пара.
func (w *World) diplomatPlanRoute(d *Diplomat) {
	from, to, ok := w.randomHabitablePair()
	if !ok || w.Planets[from].Race == w.Planets[to].Race {
		return
	}
	route := w.buildRoute(w.Planets[from].StarID, w.Planets[to].StarID, d.JumpRange)
	if route == nil {
		return
	}
	d.FromPlanet = from
	d.ToPlanet = to
	d.Route = route
	d.RouteIdx = 0
	d.Phase = DiplomatPhaseTakeoff
	d.Progress = 0
}