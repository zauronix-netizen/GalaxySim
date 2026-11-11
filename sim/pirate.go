package sim

const (
	PiratePhaseIdle = iota + 300 // отдых (тик), потом → Orbit
	PiratePhaseOrbit             // висит на орбите логова, ищет цель
	PiratePhaseChase             // догоняет караван
	PiratePhaseAttack            // атакует (исход)
	PiratePhaseReturn            // возврат к логову
)

// pirateTechLevel — TL от числа побед: 2→1, 4→2, 8→3, 16→4, ... (степени двойки).
func pirateTechLevel(wins int) int {
	if wins < 2 {
		return 1
	}
	tl := 0
	for w := wins; w >= 2; w /= 2 {
		tl++
	}
	if tl > 15 {
		tl = 15
	}
	return tl
}

// phasePirates — обход всех пиратов, switch по Phase.
func (w *World) phasePirates() {
	for _, p := range w.Pirates {
		if !p.Alive {
			continue
		}
		switch p.Phase {
		case PiratePhaseIdle:
			w.pirateIdle(p)
		case PiratePhaseOrbit:
			w.pirateOrbit(p)
		case PiratePhaseChase:
			w.pirateChase(p)
		case PiratePhaseAttack:
			w.pirateAttackPhase(p)
		case PiratePhaseReturn:
			w.pirateReturn(p)
		}
	}
}

func (w *World) pirateIdle(p *Pirate) {
	p.Phase = PiratePhaseOrbit
	p.Progress = 0
}

func (w *World) pirateOrbit(p *Pirate) {
	pirateStar := w.Planets[p.BasePlanet].StarID
	target := w.pirateFindTarget(p, pirateStar)
	if target == nil {
		return
	}
	// 10% «заметил» в тик.
	if w.Rng.Intn(100) >= 10 {
		return
	}
	p.Phase = PiratePhaseChase
	p.Progress = 0
}

func (w *World) pirateChase(p *Pirate) {
	pirateStar := w.Planets[p.BasePlanet].StarID
	target := w.pirateFindTarget(p, pirateStar)
	if target == nil {
		p.Phase = PiratePhaseReturn
		p.Progress = 0
		return
	}
	// Пират медленнее — не догонит, уходим.
	if p.Speed <= target.Speed {
		p.Phase = PiratePhaseReturn
		p.Progress = 0
		return
	}
	p.Progress += p.Speed - target.Speed
	if p.Progress >= 1.0 {
		p.Phase = PiratePhaseAttack
		p.Progress = 0
	}
}

func (w *World) pirateAttackPhase(p *Pirate) {
	pirateStar := w.Planets[p.BasePlanet].StarID
	target := w.pirateFindTarget(p, pirateStar)
	if target == nil {
		p.Phase = PiratePhaseReturn
		p.Progress = 0
		return
	}
	roll := w.Rng.Intn(100)
	switch {
	case roll < 10: // убил
		target.Alive = false
		w.DeathsPirate++
		w.Planets[p.BasePlanet].Money += target.Money // деньги убитого — логову
		w.SpawnCaravan()
		p.Wins++
	case roll < 75: // отобрал половину
		loot := target.Cargo / 2
		target.Cargo -= loot
		w.Planets[p.BasePlanet].Market[target.Good].Count += loot // груз — логову
		w.LootedCargo++                                           // счётчик ограблений
		target.Looted = true
		p.Wins++
	default: // отстал
	}
	// Пересчёт TL и скорости (после switch, перед Return).
	p.TechLevel = pirateTechLevel(p.Wins)
	p.Speed = 0.3 + p.SpeedRng*(float64(p.TechLevel)/10.0)
	p.Phase = PiratePhaseReturn
	p.Progress = 0
}

func (w *World) pirateReturn(p *Pirate) {
	p.Progress++
	if p.Progress >= 1.0 {
		p.Phase = PiratePhaseIdle
		p.Progress = 0
	}
}

// pirateFindTarget — караван в системе пирата, летящий, не ограбленный.
func (w *World) pirateFindTarget(p *Pirate, pirateStar int) *Caravan {
	for _, c := range w.Caravans {
		if !c.Alive || c.Looted {
			continue
		}
		if c.Phase >= PhaseUnload && c.Phase < 100 {
			continue // на земле (торговля) — не цель
		}
		var cs int
		switch c.Phase {
		case CaravanPhaseTakeoff, CaravanPhaseCruise, CaravanPhaseLanding,
			CaravanPhaseRefuelApproach, CaravanPhaseRefuelLanding,
			CaravanPhaseRefuelJump, CaravanPhaseRefuelTakeoff:
			cs = w.Planets[c.FromPlanet].StarID
		case CaravanPhaseToEdge, CaravanPhaseJumpEnter, CaravanPhaseJump,
			CaravanPhaseJumpExit, CaravanPhaseFuelCheck,
			CaravanPhaseEdgeToEdge, CaravanPhasePlanetToEdge, CaravanPhaseEdgeToPlanet:
			cs = w.Planets[c.FromPlanet].StarID
		default:
			continue
		}
		if cs == pirateStar {
			return c
		}
	}
	return nil
}