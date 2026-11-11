package sim

import (
	"fmt"
	"time"
)

// ВРЕМЕННО, для замера. Убрать после.
var (
	statFindTrip, statUnload int
	statEcon, statCaravans, statPirates, statLiners, statDiplomats time.Duration
)

func (w *World) SimTick() {
	w.Events = w.Events[:0]
	w.TickEvents = w.TickEvents[:0] // чистим в начале: EconSummary читает их после SimTick
	w.TradersSpawnedThisTick = 0 // <-- сбрасываем на каждом тике
	if w.Day >= GalaxyWarmupTurns {
		w.phaseMovement()
		w.phaseSpawn()
		w.phaseEconomy()
		w.phaseTraders()
		w.LogTickEvents()
		w.phaseRelations()
	}
	if !InflationOff && w.Day > 0 && w.Day%TurnsPerYear == 0 {
		w.ComputeGlobalGoodsPriceBands()
	}

	// ГТУ — раз в год, независимо от инфляции.
	if w.Day > 0 && w.Day%TurnsPerYear == 0 {
		w.computeGalaxyTechLevel()
	}

	// Банкиры: по 1 на расу при каждом GTL++ (кап 15 на расу).
	if w.GalaxyTechLevel > w.LastGTL {
		w.LastGTL = w.GalaxyTechLevel
		for r := 0; r < int(RaceCount); r++ {
			if w.BankersByRace[r] >= 15 {
				continue
			}
			race := Race(r)
			// Случайная обитаемая планета этой расы.
			var candidates []*Planet
			for _, p := range w.Planets {
				if p.Habitable && p.Race == race {
					candidates = append(candidates, p)
				}
			}
			if len(candidates) == 0 {
				continue
			}
			p := candidates[w.Rng.Intn(len(candidates))]
			w.SpawnBanker(p.ID, race)
			w.BankersByRace[r]++
		}
	}

	// повторный спавн трейдеров 
	if w.Day == GalaxyWarmupTurns + 1 * TurnsPerYear || w.Day == GalaxyWarmupTurns + 2 * TurnsPerYear || w.Day == GalaxyWarmupTurns + 3 * TurnsPerYear || w.Day == GalaxyWarmupTurns + 4 * TurnsPerYear {
		for _, star := range w.Stars {
			planets := w.PlanetMap[star.ID]
			if len(planets) > 3 && planets[3].Habitable {
				w.SpawnTrader(planets[3].ID, GoodFood)
			}
		}
	}


	// ... (караваны/пираты/лайнеры/дипломаты — как было)

	// Волны караванов: +0.5, +1.5, +2.5, +3.5, +4.5 года от прогрева.
	for i := 0; i < CaravanWaves; i++ {
		waveDay := GalaxyWarmupTurns + i*TurnsPerYear + TurnsPerYear/2
		if w.Day == waveDay {
			w.SpawnCaravanWave()
			break
		}
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
	deathsToRespawn := 0

	// Перемешиваем порядок караванов: иначе первая волна всегда хватает
	// лучшие избытки/дефициты, хвост получает объедки.
	w.Rng.Shuffle(len(w.Caravans), func(i, j int) {
		w.Caravans[i], w.Caravans[j] = w.Caravans[j], w.Caravans[i]
	})
	alive := w.Caravans[:0]
	for _, c := range w.Caravans {
		if !c.Alive {
			continue
		}

		if c.Phase >= PhaseUnload && c.Phase < 100 {
			switch c.Phase {
			case PhaseUnload:
				w.caravanUnload(c)
			case PhaseIdle:
				c.Phase = PhaseFindTrip
			case PhaseFindTrip:
				g, ok := w.caravanCheckLocalSurplus(c)
				if ok {
					c.Good = g
					c.Phase = PhaseLoad
					c.Progress = 0
				} else {
					c.Phase = PhaseFindSurplus
				}
			case PhaseFindSurplus:
				p, g := w.caravanFindSurplusNearby(c)
				if p == nil {
					c.Phase = PhaseIdle
				} else {
					route := w.buildRoute(w.Planets[c.FromPlanet].StarID, p.StarID, c.JumpRange)
					if route == nil {
						c.Phase = PhaseIdle
					} else {
						c.ToPlanet = p.ID
						c.Good = g
						c.Route = route
						c.RouteIdx = 0
						c.Progress = 0
						c.Phase = CaravanPhaseTakeoff
					}
				}
			case PhaseLoad:
				if w.caravanLoad(c) {
					c.Alive = false
					deathsToRespawn++
					continue
				}
			case PhaseFindDeficit:
				p := w.caravanFindDeficitNearby(c)
				if p == nil {
					// дефицита нет — продаём тут же
					c.ToPlanet = c.FromPlanet
					c.Phase = PhaseUnload
				} else {
					route := w.buildRoute(w.Planets[c.FromPlanet].StarID, p.StarID, c.JumpRange)
					if route == nil {
						c.ToPlanet = c.FromPlanet
						c.Phase = PhaseUnload
					} else {
						c.ToPlanet = p.ID
						c.Route = route
						c.RouteIdx = 0
						c.Progress = 0
						c.Phase = CaravanPhaseTakeoff
					}
				}
			}
			alive = append(alive, c)
			continue
		}

		// --- Полёт (фазы как у трейдера, через *Common) ---
		switch c.Phase {
		case CaravanPhaseTakeoff:
			c.Progress++
			if c.Progress >= 1.0 {
				c.Progress = 0
				fromStar := w.Planets[c.FromPlanet].StarID
				toStar := w.Planets[c.ToPlanet].StarID
				if fromStar == toStar {
					c.Phase = CaravanPhaseCruise
					w.PrepareFlightCommon(&c.Flight, &c.Progress, c.Speed, w.Planets[c.FromPlanet], w.Planets[c.ToPlanet])
				} else {
					c.Route = w.buildRoute(fromStar, toStar, c.JumpRange)
					c.RouteIdx = 0
					exitX, exitY := w.getJumpExitPoint(c.Route[0], c.Route[1])
					w.PrepareJumpFlightCommon(&c.Flight, &c.Progress, w.Planets[c.FromPlanet], exitX, exitY)
					c.Phase = CaravanPhaseToEdge
				}
			}

		case CaravanPhaseCruise:
			if AdvanceFlightCommon(&c.Flight, &c.Progress, c.Speed) {
				c.Phase = CaravanPhaseLanding
				c.Progress = 0
				c.FromPlanet = c.ToPlanet
			}

		case CaravanPhaseLanding:
			c.Progress++
			if c.Progress >= 1.0 {
				c.Progress = 0
				if c.Cargo > 0 {
					c.Phase = PhaseUnload
				} else {
					c.Phase = PhaseFindTrip
				}
			}

		case CaravanPhaseToEdge:
			if AdvanceFlightCommon(&c.Flight, &c.Progress, c.Speed) {
				c.Phase = CaravanPhaseJumpEnter
				c.Progress = 0
			}

		case CaravanPhaseJumpEnter:
			if c.RouteIdx+1 < len(c.Route) {
				d := w.Dist(c.Route[c.RouteIdx], c.Route[c.RouteIdx+1])
				cost := w.CalculateFuelCost(d*0.1, oiUninhabited)
				c.Fuel -= cost
			}
			c.Phase = CaravanPhaseJump
			c.Progress = 0

		case CaravanPhaseJump:
			if c.RouteIdx+1 < len(c.Route) {
				d := w.Dist(c.Route[c.RouteIdx], c.Route[c.RouteIdx+1])
				leg := d / 20.0
				if leg < 0.01 {
					c.Progress = 1.0
				} else {
					c.Progress += c.Speed / leg
				}
			} else {
				c.Progress = 1.0
			}
			if c.Progress >= 1.0 {
				c.Progress = 0
				c.Phase = CaravanPhaseJumpExit
			}

		case CaravanPhaseJumpExit:
			c.RouteIdx++
			c.Flight.Len = 0
			c.Phase = CaravanPhaseFuelCheck
			c.Progress = 0

		case CaravanPhaseFuelCheck:
			fuelForJump := w.CalculateFuelCost(c.JumpRange*0.1, oiUninhabited)
			hasNextJump := c.RouteIdx < len(c.Route)-1
			if c.Fuel >= fuelForJump {
				if hasNextJump {
					c.Phase = CaravanPhaseEdgeToEdge
				} else {
					c.Phase = CaravanPhaseEdgeToPlanet
				}
			} else {
				if hasNextJump {
					c.AfterRefuel = CaravanPhasePlanetToEdge
				} else {
					c.AfterRefuel = CaravanPhaseEdgeToPlanet
				}
				c.Phase = CaravanPhaseRefuelApproach
			}
			c.Progress = 0

		case CaravanPhaseEdgeToEdge:
			if c.Flight.Len == 0 {
				ax, ay := w.getJumpEntryPoint(c.Route[c.RouteIdx], c.Route[c.RouteIdx-1])
				bx, by := w.getJumpExitPoint(c.Route[c.RouteIdx], c.Route[c.RouteIdx+1])
				w.PrepareJumpSegmentCommon(&c.Flight, &c.Progress, ax, ay, bx, by)
			}
			if AdvanceFlightCommon(&c.Flight, &c.Progress, c.Speed) {
				c.Phase = CaravanPhaseJumpEnter
				c.Progress = 0
			}

		case CaravanPhaseRefuelApproach:
			if c.Flight.Len == 0 {
				ax, ay := w.getJumpEntryPoint(c.Route[c.RouteIdx], c.Route[c.RouteIdx-1])
				refuelPlanet := w.PlanetMap[c.Route[c.RouteIdx]][3]
				w.PrepareFlightFromPointCommon(&c.Flight, &c.Progress, c.Speed, ax, ay, refuelPlanet)
			}
			if AdvanceFlightCommon(&c.Flight, &c.Progress, c.Speed) {
				c.Phase = CaravanPhaseRefuelLanding
				c.Progress = 0
			}

		case CaravanPhaseRefuelLanding:
			c.Progress++
			if c.Progress >= 1.0 {
				c.Progress = 0
				c.Phase = CaravanPhaseRefuelJump
			}

		case CaravanPhaseRefuelJump:
			refuelPlanet := w.PlanetMap[c.Route[c.RouteIdx]][3]
			need := c.FuelMax - c.Fuel
			cost := w.CalculateFuelCost(need, refuelPlanet.OwnerId)
			if c.Money >= cost {
				c.Money -= cost
				refuelPlanet.Money += cost
				c.Fuel = c.FuelMax
			}
			c.Phase = CaravanPhaseRefuelTakeoff
			c.Progress = 0

		case CaravanPhaseRefuelTakeoff:
			c.Progress++
			if c.Progress >= 1.0 {
				c.Progress = 0
				c.Flight.Len = 0
				c.Phase = c.AfterRefuel
			}

		case CaravanPhasePlanetToEdge:
			if c.Flight.Len == 0 {
				refuelPlanet := w.PlanetMap[c.Route[c.RouteIdx]][3]
				exitX, exitY := w.getJumpExitPoint(c.Route[c.RouteIdx], c.Route[c.RouteIdx+1])
				w.PrepareJumpFlightCommon(&c.Flight, &c.Progress, refuelPlanet, exitX, exitY)
			}
			if AdvanceFlightCommon(&c.Flight, &c.Progress, c.Speed) {
				c.Phase = CaravanPhaseJumpEnter
				c.Progress = 0
			}

		case CaravanPhaseEdgeToPlanet:
			if c.Flight.Len == 0 {
				ax, ay := w.getJumpEntryPoint(c.Route[c.RouteIdx], c.Route[c.RouteIdx-1])
				w.PrepareFlightFromPointCommon(&c.Flight, &c.Progress, c.Speed, ax, ay, w.Planets[c.ToPlanet])
			}
			if AdvanceFlightCommon(&c.Flight, &c.Progress, c.Speed) {
				c.Phase = CaravanPhaseLanding
				c.Progress = 0
				c.FromPlanet = c.ToPlanet
			}
		}

		alive = append(alive, c)
	}
	w.Caravans = alive

	// Респавн мёртвых — сразу, в этом же тике, $5000, случайная planets[3].
	for i := 0; i < deathsToRespawn; i++ {
		w.SpawnCaravan()
	}
	statCaravans += time.Since(t0)
	t1 := time.Now()
	
	// --- ПИРАТЫ ---
	w.phasePirates()
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
	
	// --- ДИПЛОМАТЫ (общий switch, полёт через *Common) ---
	diplomats := w.Diplomats[:0]
	for _, d := range w.Diplomats {
		if !d.Alive {
			continue
		}

		// Idle — ищем маршрут.
		if d.Phase == PhaseIdle {
			if d.Role == DiplomatRoleBanker {
				w.bankerPlanRoute(d)
			} else {
				w.diplomatPlanRoute(d)
			}
			if d.Phase == PhaseIdle {
				diplomats = append(diplomats, d)
				continue
			}
		}

		// Полётные фазы.
		switch d.Phase {
		case DiplomatPhaseTakeoff:
			d.Progress++
			if d.Progress >= 1.0 {
				d.Progress = 0
				fromStar := w.Planets[d.FromPlanet].StarID
				toStar := w.Planets[d.ToPlanet].StarID
				if fromStar == toStar {
					d.Phase = DiplomatPhaseCruise
					w.PrepareFlightCommon(&d.Flight, &d.Progress, d.Speed, w.Planets[d.FromPlanet], w.Planets[d.ToPlanet])
				} else {
					d.Route = w.buildRoute(fromStar, toStar, d.JumpRange)
					d.RouteIdx = 0
					exitX, exitY := w.getJumpExitPoint(d.Route[0], d.Route[1])
					w.PrepareJumpFlightCommon(&d.Flight, &d.Progress, w.Planets[d.FromPlanet], exitX, exitY)
					d.Phase = DiplomatPhaseToEdge
				}
			}

		case DiplomatPhaseCruise:
			if AdvanceFlightCommon(&d.Flight, &d.Progress, d.Speed) {
				d.Phase = DiplomatPhaseLanding
				d.Progress = 0
				d.FromPlanet = d.ToPlanet
			}

		case DiplomatPhaseLanding:
			d.Progress++
			if d.Progress >= 1.0 {
				d.Progress = 0
				d.Phase = PhaseIdle
			}

		case DiplomatPhaseToEdge:
			if AdvanceFlightCommon(&d.Flight, &d.Progress, d.Speed) {
				d.Phase = DiplomatPhaseJumpEnter
				d.Progress = 0
			}

		case DiplomatPhaseJumpEnter:
			if d.RouteIdx+1 < len(d.Route) {
				dist := w.Dist(d.Route[d.RouteIdx], d.Route[d.RouteIdx+1])
				cost := w.CalculateFuelCost(dist*0.1, oiUninhabited)
				d.Fuel -= cost
			}
			d.Phase = DiplomatPhaseJump
			d.Progress = 0

		case DiplomatPhaseJump:
			if d.RouteIdx+1 < len(d.Route) {
				dist := w.Dist(d.Route[d.RouteIdx], d.Route[d.RouteIdx+1])
				leg := dist / 20.0
				if leg < 0.01 {
					d.Progress = 1.0
				} else {
					d.Progress += d.Speed / leg
				}
			} else {
				d.Progress = 1.0
			}
			if d.Progress >= 1.0 {
				d.Progress = 0
				d.Phase = DiplomatPhaseJumpExit
			}

		case DiplomatPhaseJumpExit:
			d.RouteIdx++
			d.Flight.Len = 0
			d.Phase = DiplomatPhaseFuelCheck
			d.Progress = 0

		case DiplomatPhaseFuelCheck:
			fuelForJump := w.CalculateFuelCost(d.JumpRange*0.1, oiUninhabited)
			hasNextJump := d.RouteIdx < len(d.Route)-1
			if d.Fuel >= fuelForJump {
				if hasNextJump {
					d.Phase = DiplomatPhaseEdgeToEdge
				} else {
					d.Phase = DiplomatPhaseEdgeToPlanet
				}
			} else {
				if hasNextJump {
					d.AfterRefuel = DiplomatPhasePlanetToEdge
				} else {
					d.AfterRefuel = DiplomatPhaseEdgeToPlanet
				}
				d.Phase = DiplomatPhaseRefuelApproach
			}
			d.Progress = 0

		case DiplomatPhaseEdgeToEdge:
			if d.Flight.Len == 0 {
				ax, ay := w.getJumpEntryPoint(d.Route[d.RouteIdx], d.Route[d.RouteIdx-1])
				bx, by := w.getJumpExitPoint(d.Route[d.RouteIdx], d.Route[d.RouteIdx+1])
				w.PrepareJumpSegmentCommon(&d.Flight, &d.Progress, ax, ay, bx, by)
			}
			if AdvanceFlightCommon(&d.Flight, &d.Progress, d.Speed) {
				d.Phase = DiplomatPhaseJumpEnter
				d.Progress = 0
			}

		case DiplomatPhaseRefuelApproach:
			if d.Flight.Len == 0 {
				ax, ay := w.getJumpEntryPoint(d.Route[d.RouteIdx], d.Route[d.RouteIdx-1])
				refuelPlanet := w.PlanetMap[d.Route[d.RouteIdx]][3]
				w.PrepareFlightFromPointCommon(&d.Flight, &d.Progress, d.Speed, ax, ay, refuelPlanet)
			}
			if AdvanceFlightCommon(&d.Flight, &d.Progress, d.Speed) {
				d.Phase = DiplomatPhaseRefuelLanding
				d.Progress = 0
			}

		case DiplomatPhaseRefuelLanding:
			d.Progress++
			if d.Progress >= 1.0 {
				d.Progress = 0
				d.Phase = DiplomatPhaseRefuelJump
			}

		case DiplomatPhaseRefuelJump:
			// Дипломат заправляется бесплатно (дипломатический статус).
			d.Fuel = d.FuelMax
			d.Phase = DiplomatPhaseRefuelTakeoff
			d.Progress = 0

		case DiplomatPhaseRefuelTakeoff:
			d.Progress++
			if d.Progress >= 1.0 {
				d.Progress = 0
				d.Flight.Len = 0
				d.Phase = d.AfterRefuel
			}

		case DiplomatPhasePlanetToEdge:
			if d.Flight.Len == 0 {
				refuelPlanet := w.PlanetMap[d.Route[d.RouteIdx]][3]
				exitX, exitY := w.getJumpExitPoint(d.Route[d.RouteIdx], d.Route[d.RouteIdx+1])
				w.PrepareJumpFlightCommon(&d.Flight, &d.Progress, refuelPlanet, exitX, exitY)
			}
			if AdvanceFlightCommon(&d.Flight, &d.Progress, d.Speed) {
				d.Phase = DiplomatPhaseJumpEnter
				d.Progress = 0
			}

		case DiplomatPhaseEdgeToPlanet:
			if d.Flight.Len == 0 {
				ax, ay := w.getJumpEntryPoint(d.Route[d.RouteIdx], d.Route[d.RouteIdx-1])
				w.PrepareFlightFromPointCommon(&d.Flight, &d.Progress, d.Speed, ax, ay, w.Planets[d.ToPlanet])
			}
			if AdvanceFlightCommon(&d.Flight, &d.Progress, d.Speed) {
				d.Phase = DiplomatPhaseLanding
				d.Progress = 0
				d.FromPlanet = d.ToPlanet
			}
		}

		// Прилёт (Landing завершён) — развилка по роли.
		if d.Phase == PhaseIdle && d.FromPlanet == d.ToPlanet && d.Progress == 0 {
			if d.Role == DiplomatRoleBanker {
				w.bankerArrive(d)
			} else {
				fromRace := w.Planets[d.FromPlanet].Race
				toRace := w.Planets[d.ToPlanet].Race
				if fromRace != toRace {
					w.pullRelation(fromRace, toRace, 50, 1)
				}
			}
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
	c.Phase = CaravanPhaseTakeoff
	c.Progress = 0
}

/*

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

*/

func (w *World) phaseSpawn() {
	for i := 0; i < 10 && len(w.Caravans) < targetCaravans; i++ {
		if w.SpawnCaravan() == nil {
			break
		}
	}
	// Спавним по числу логовых планет (метка oiPirate), без константы-потолка.
	// Мёртвые в срезе не блокируют: оживляем на том же логове.
	spawned := true
	for spawned {
		spawned = false
		for _, s := range w.Stars {
			lair := w.findPirateLair(s.ID)
			if lair == 0 {
				continue
			}
			// Уже есть пират (живой или мёртвый) на этой звезде?
			var existing *Pirate
			for _, p := range w.Pirates {
				if w.Planets[p.BasePlanet].StarID == s.ID {
					existing = p
					break
				}
			}
			if existing != nil {
				if existing.Alive {
					continue // живой — пропуск
				}
				w.respawnPirate(existing) // мёртвый — оживляем
				spawned = true
				break
			}
			if w.SpawnPirateAt(lair) != nil {
				spawned = true
				break
			}
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

	// FullFoodDay: считаем дни подряд, когда еда >= 90% TargetStock.
	// Читаем Count прошлого тика (лаг 1 тик, осознанно).
	// food.Count*10 >= ts*9 — целочисленно, без float.
	if food.Count*10 >= ts*9 {
		p.FullFoodDay++
	} else {
		p.FullFoodDay = 0
	}

	// HungryDay: считаем дни подряд, когда еда < 10% TargetStock.
	// Читаем Count прошлого тика. food.Count*10 < ts — целочисленно.
	if food.Count*10 < ts {
		p.HungryDay++
	} else {
		p.HungryDay = 0
	}

}

func (w *World) phaseEconomy() {
	tEcon := time.Now()

	// Счётчики переизбытка/дефицита — каждый тик, все товары, до изменений тика.
	for _, id := range w.RandomPlanetOrder {
		p := w.Planets[id]
		if !p.Habitable {
			continue
		}
		w.checkGoodsCounters(p)
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

// Hauler начало 
/*

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
			// Голодающие планеты (меньше 10% стока)
			if qm.Count * 10 < qts {
				list = append(list, deficitInfo{q, qts/3 - qm.Count}) // Доливка до трети
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
			surplus := m.Count - ts * 2 // больше стока х2
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

// Hauler конец
*/

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

	// Дашборд — каждый тик, фиксированные позиции.
	fmt.Print(w.RenderDashboard(el))

	// EconSummary в консоль выключен — дашборд заменяет.
	// Функция оставлена (пригодится для отладки).
	

	// Снимок системы для viewer.exe — каждый тик (no-op, если Shm == nil).
	w.writeSnapshot()
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