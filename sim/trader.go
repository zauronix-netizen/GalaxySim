package sim

/*
	Логика трейдера (идеал)
1. спаунимся на планете с избытком
2. читаем местные новости(ищем избыток)
3. загружаем трюм товаром
4. читаем новости приоритетно планета, звезда, созвездие, галактика(ищем дефицит)
5. заправляемся(если надо)
6. взлетаем и летим
7. прилетаем с грузом
8. читаем новости(проверяем дефицит)
9. разгружаемся и отдыхаем
	(рабочий цикл меняет только поиск излишка)
10. читаем новости приоритетно планета, звезда, созвездие, галактика(ищем избыток)
*/

import (
	"log"
	"math"
	"os"
)

var traderLog *log.Logger

// ---------- Структуры ----------

type FlightPath struct {
	Ax, Ay float64 // Старт
	Cx, Cy float64 // Контрольная точка (обход)
	Bx, By float64 // Финиш (предугаданный)
	Len    float64 // Длина кривой (20 отрезков)
}

type Trader struct {
	ID             int
	Race           Race
	HomePlanet     int
	TechLevel      int
	FromPlanet     int
	ToPlanet       int
	Route          []int
	RouteIdx       int
	Phase          int
	Progress       float64
	Speed          float64
	SpeedRng  	   float64 // 0..1, фикс при спавне (сохраняется при апгрейде)
	JumpRange      float64
	Good           Good
	Cargo          int
	CargoMax       int
	BaseCost       float64
	Money          float64
	Fuel           float64
	FuelMax        float64
	PirateCanSeeMe bool
	Alive          bool
	
	// Механика полёта
	Flight         FlightPath
	NeedsRefuel bool // флаг "надо заправиться" (внешний читатель)
	OnTime      bool // флаг "полная зарплата" (CheckUnload -> Unload)
	Smuggler    bool // флаг "контрабандист" (заготовка)

	// Для отладки
	DebugLog  bool
	LastPhase int
	AfterRefuel int // фаза после RefuelTakeoff (PlanetToEdge / EdgeToPlanet)
}

var currentGalaxyTechLevel int = 3

var nextTraderID int

// ---------- Константы фаз ----------

const (
	TraderPhaseFindTrip = iota + 10
	TraderPhaseFindTripPlus
	TraderPhaseLoad
	TraderPhaseRefuel
	TraderPhaseTakeoff
	TraderPhaseCruise
	TraderPhaseLanding
	TraderPhaseUnload
	TraderPhaseRest
	TraderPhaseCheckLocal
	TraderPhaseCheckUnload // 20
	// Межзвёздный цикл
	TraderPhaseToEdge
	TraderPhaseJumpEnter
	TraderPhaseJump
	TraderPhaseJumpExit
	TraderPhaseFuelCheck
	TraderPhaseEdgeToEdge
	TraderPhaseRefuelApproach
	TraderPhaseRefuelLanding
	TraderPhaseRefuelJump
	TraderPhaseRefuelTakeoff
	TraderPhasePlanetToEdge
	TraderPhaseEdgeToPlanet
	TraderPhasePirateFind
	TraderPhasePirateLoad
	TraderPhasePirateFindMarket
	TraderPhaseUnloadPirate
)

// PhaseName возвращает символьное имя фазы трейдера для логов.
func PhaseName(p int) string {
	switch p {
	case TraderPhaseFindTrip:
		return "FindTrip"
	case TraderPhaseFindTripPlus:
		return "FindTripPlus"
	case TraderPhaseLoad:
		return "Load"
	case TraderPhaseRefuel:
		return "Refuel"
	case TraderPhaseTakeoff:
		return "Takeoff"
	case TraderPhaseCruise:
		return "Cruise"
	case TraderPhaseLanding:
		return "Landing"
	case TraderPhaseUnload:
		return "Unload"
	case TraderPhaseRest:
		return "Rest"
	case TraderPhaseCheckLocal:
		return "CheckLocal"
	case TraderPhaseCheckUnload:
		return "CheckUnload"
	case TraderPhaseToEdge:
		return "ToEdge"
	case TraderPhaseJumpEnter:
		return "JumpEnter"
	case TraderPhaseJump:
		return "Jump"
	case TraderPhaseJumpExit:
		return "JumpExit"
	case TraderPhaseFuelCheck:
		return "FuelCheck"
	case TraderPhaseEdgeToEdge:
		return "EdgeToEdge"
	case TraderPhaseRefuelApproach:
		return "RefuelApproach"
	case TraderPhaseRefuelLanding:
		return "RefuelLanding"
	case TraderPhaseRefuelJump:
		return "RefuelJump"
	case TraderPhaseRefuelTakeoff:
		return "RefuelTakeoff"
	case TraderPhasePlanetToEdge:
		return "PlanetToEdge"
	case TraderPhaseEdgeToPlanet:
		return "EdgeToPlanet"
	case TraderPhasePirateFind:
		return "PirateFind"
	case TraderPhasePirateLoad:
		return "PirateLoad"
	case TraderPhasePirateFindMarket:
		return "PirateFindMarket"
	case TraderPhaseUnloadPirate:
		return "UnloadPirate"
	default:
		return "?"
	}
}

// totalCost — накопительная стоимость апгрейда TechLevel (0→TL).

// totalCost[N] = накопленная стоимость после N шагов.
// totalCost[0] = 0 — начальное состояние.
var totalCost = [16]float64{
	0,        // 0
	0,        // 0→1   (5000·0² = 0)
	5000,     // 0→2   (0 + 5000·1²)
	25000,    // 0→3   (+ 5000·2² = 20000)
	70000,    // 0→4   (+ 45000)
	150000,   // 0→5   (+ 80000)
	275000,   // 0→6   (+ 125000)
	455000,   // 0→7   (+ 180000)
	700000,   // 0→8   (+ 245000)
	1020000,  // 0→9   (+ 320000)
	1425000,  // 0→10  (+ 405000)
	1925000,  // 0→11  (+ 500000)
	2530000,  // 0→12  (+ 605000)
	3250000,  // 0→13  (+ 720000)
	4095000,  // 0→14  (+ 845000)
	5075000,  // 0→15  (+ 980000)
}

func (w *World) techDiscountTraders(currentTL int) float64 {
	if len(w.Traders) == 0 {
		return 0
	}
	
	if currentTL == 0 {
		return 0
	}
	
	sum := 0
	
	for _, t := range w.Traders {
		sum += t.TechLevel
	}
	
	avg := float64(sum) / float64(len(w.Traders))
	
	if currentTL == -1 {
		return avg
	}
	
	frac := avg - math.Floor(avg)
	inv := math.Floor(avg) - avg + 1
	
	if float64(currentTL) > avg+1 {
		return inv          // значительно выше → 1.7 (дорого)
	} else if float64(currentTL) > avg {
		return frac         // чуть выше → 1.3 (чуть дорого)
	} else if float64(currentTL) < avg-1 {
		return -inv         // значительно ниже → 0.3 (дёшево)
	} else {
		return -frac        // чуть ниже → 0.7 (чуть дёшево)
	}
}

func (w *World) techDiscountPlanets(currentTL int) float64 {
	sum, count := 0, 0
	for _, p := range w.Planets {
		if p.Habitable {
			sum += p.TechLevel
			count++
		}
	}
	if count == 0 {
		return 0
	}

	if currentTL == 0 {
		return 0
	}

	avg := float64(sum) / float64(count)

	if currentTL == -1 {
		return avg
	}

	frac := avg - math.Floor(avg)
	inv := math.Floor(avg) - avg + 1

	// Усиленные ветки для сильно отстающих/убегающих — от ГТУ.
	// Работают только если ГТУ известен (>0).
	gtu := w.GalaxyTechLevel
	if gtu > 0 {
		if float64(currentTL) > avg+2 {
			// Убегающие: наценка пропорциональна отрыву от ГТУ.
			v := frac * float64(currentTL-gtu)
			if v > 1.8 {
				v = 1.8
			}
			return v
		}
		if float64(currentTL) < avg-2 {
			// Сильно отстающие: скидка пропорциональна отрыву от ГТУ.
			// TL < GTU → разность отрицательная → v < 0 (скидка).
			v := frac * float64(currentTL-gtu)
			if v < -0.9 {
				v = -0.9
			}
			return v
		}
	}

	// Базовые 4 ветки (без изменений).
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

func init() {
	log.SetFlags(0)
	f, err := os.OpenFile("traders.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
    if err != nil {
        log.Fatalf("error opening traders.log: %v", err)
    }
    traderLog = log.New(f, "", 0)
}

// ---------- Спавн ----------

func (w *World) SpawnTrader(planetID int, g Good) *Trader {
	p := w.Planets[planetID]
	nextTraderID++
	techLevel := 3
	rngSpeed := w.Rng.Float64()
	t := &Trader{
		ID:         nextTraderID,
		Race:       p.Race,
		HomePlanet: planetID,
		TechLevel:  techLevel,
		FromPlanet: planetID,
		ToPlanet:   planetID,
		Phase:      TraderPhaseCheckLocal,
		Good:       g,
		Cargo:      0,
		CargoMax:   techLevel * 100,
		Money:      5000.0,
		Fuel:       15.0,
		FuelMax:    15.0,
		SpeedRng:   rngSpeed,
		Speed:      0.2 + rngSpeed*(float64(techLevel)/10.0),
		JumpRange:  30.0,
		Alive:      true,
		DebugLog:   (nextTraderID == 1),
		LastPhase:  -1,
	}
	w.Traders = append(w.Traders, t)
	return t
}

// ---------- Основная логика (вызывается в phaseTraders) ----------

func (w *World) phaseTraders() {
	// Перемешиваем порядок каждый тик: иначе первые в срезе всегда первыми
	// хватают избытки (помечают TickEvent.Processed), а хвост получает объедки.
	// ID и состояние остаются с трейдером, меняется только порядок обхода.
	w.Rng.Shuffle(len(w.Traders), func(i, j int) {
		w.Traders[i], w.Traders[j] = w.Traders[j], w.Traders[i]
	})

	for _, t := range w.Traders {
		if !t.Alive {
			continue
		}
		
		if t.DebugLog && t.Phase != t.LastPhase {
			// лог убран
//			t.LastPhase = t.Phase
		}
		
		switch t.Phase {
		case TraderPhaseFindTrip:
			w.traderFindTrip(t)
			
		case TraderPhaseFindTripPlus:
			w.traderFindTripPlus(t)
			
		case TraderPhaseLoad:
			w.traderLoad(t)
			
		case TraderPhaseRefuel:
			w.traderRefuel(t)
			
		case TraderPhaseTakeoff:
			t.Progress++
			if t.Progress >= 1.0 {
				t.Progress = 0
				fromStar := w.Planets[t.FromPlanet].StarID
				toStar := w.Planets[t.ToPlanet].StarID
				if fromStar == toStar {
					t.Phase = TraderPhaseCruise
					w.PrepareFlight(t, w.Planets[t.FromPlanet], w.Planets[t.ToPlanet])
				} else {
					t.Route = w.buildRoute(fromStar, toStar, t.JumpRange)
					t.RouteIdx = 0
					exitX, exitY := w.getJumpExitPoint(t.Route[0], t.Route[1])
					w.PrepareJumpFlight(t, w.Planets[t.FromPlanet], exitX, exitY)
					t.Phase = TraderPhaseToEdge
				}
			}
			
		case TraderPhaseCruise:
			if t.AdvanceFlight() {
				t.Phase = TraderPhaseLanding
				t.Progress = 0
				t.FromPlanet = t.ToPlanet
			}
			
		case TraderPhaseLanding:
			t.Progress++
			if t.Progress >= 1.0 {
				t.Progress = 0
				if t.PirateCanSeeMe {
					if t.BaseCost == 0 {
						t.Phase = TraderPhasePirateLoad
					} else {
						t.Phase = TraderPhaseUnloadPirate
					}
				} else {
					t.Phase = TraderPhaseCheckUnload
				}
			}
			
		case TraderPhaseCheckUnload:
			w.traderCheckUnload(t)

		case TraderPhaseUnload:
			w.traderUnload(t)
			
		case TraderPhaseCheckLocal:
			w.traderCheckLocal(t)

		case TraderPhaseRest:
			if t.TechLevel < 15 {
				cost := totalCost[t.TechLevel] * (w.techDiscountTraders(t.TechLevel) + 1)
				if t.Money >= cost + 5000 {
					t.Money -= cost
					w.Planets[t.FromPlanet].Money += cost * 0.01 // 1% апгрейда — планете, где отдыхаем
					t.TechLevel++
					t.Speed = 0.2 + t.SpeedRng*(float64(t.TechLevel)/10.0)
					t.CargoMax = t.TechLevel * 100
				}
			}
			t.Progress++
			if t.Progress >= 1.0 {
				t.Progress = 0
				t.Phase = TraderPhaseCheckLocal
			}
			
		// ---------- Межзвёздный цикл ----------
		case TraderPhaseToEdge:
			if t.AdvanceFlight() {
				t.Phase = TraderPhaseJumpEnter
				t.Progress = 0
			}
			
		case TraderPhaseJumpEnter:
			if t.RouteIdx+1 < len(t.Route) {
				d := w.Dist(t.Route[t.RouteIdx], t.Route[t.RouteIdx+1])
				cost := w.CalculateFuelCost(d*0.1, oiUninhabited)
				t.Fuel -= cost
			}
			t.Phase = TraderPhaseJump
			t.Progress = 0
			
		case TraderPhaseJump:
			if t.RouteIdx+1 < len(t.Route) {
				d := w.Dist(t.Route[t.RouteIdx], t.Route[t.RouteIdx+1])
				leg := d / 20.0
				if leg < 0.01 {
					t.Progress = 1.0
				} else {
					t.Progress += t.Speed / leg
				}
			} else {
				t.Progress = 1.0
			}
			if t.Progress >= 1.0 {
				t.Progress = 0
				t.Phase = TraderPhaseJumpExit
			}
			
		case TraderPhaseJumpExit:
			t.RouteIdx++
			t.Flight.Len = 0 // заправка кончилась, старый Flight не нужен
			t.Phase = TraderPhaseFuelCheck
			t.Progress = 0
			
		case TraderPhaseFuelCheck:
			fuelForJump := w.CalculateFuelCost(t.JumpRange*0.1, oiUninhabited)
			hasNextJump := t.RouteIdx < len(t.Route)-1
			if t.Fuel >= fuelForJump {
				if hasNextJump {
					t.Phase = TraderPhaseEdgeToEdge
				} else {
					t.Phase = TraderPhaseEdgeToPlanet
				}
			} else {
				if hasNextJump {
					t.AfterRefuel = TraderPhasePlanetToEdge
				} else {
					t.AfterRefuel = TraderPhaseEdgeToPlanet
				}
				t.Phase = TraderPhaseRefuelApproach
			}
			t.Progress = 0
			
		case TraderPhaseEdgeToEdge:
			if t.Flight.Len == 0 {
				ax, ay := w.getJumpEntryPoint(t.Route[t.RouteIdx], t.Route[t.RouteIdx-1])
				bx, by := w.getJumpExitPoint(t.Route[t.RouteIdx], t.Route[t.RouteIdx+1])
				w.PrepareJumpSegment(t, ax, ay, bx, by)
			}
			if t.AdvanceFlight() {
				t.Phase = TraderPhaseJumpEnter
				t.Progress = 0
			}
			
		case TraderPhaseRefuelApproach:
			if t.Flight.Len == 0 {
				ax, ay := w.getJumpEntryPoint(t.Route[t.RouteIdx], t.Route[t.RouteIdx-1])
				refuelPlanet := w.PlanetMap[t.Route[t.RouteIdx]][3]
				w.PrepareFlightFromPoint(t, ax, ay, refuelPlanet)
			}
			if t.AdvanceFlight() {
				t.Phase = TraderPhaseRefuelLanding
				t.Progress = 0
			}
			
		case TraderPhaseRefuelLanding:
			t.Progress++
			if t.Progress >= 1.0 {
				t.Progress = 0
				t.Phase = TraderPhaseRefuelJump
			}
			
		case TraderPhaseRefuelJump:
			refuelPlanet := w.PlanetMap[t.Route[t.RouteIdx]][3]
			need := t.FuelMax - t.Fuel
			cost := w.CalculateFuelCost(need, refuelPlanet.OwnerId)
			t.Money -= cost
			refuelPlanet.Money += cost
			t.Fuel = t.FuelMax
			t.Phase = TraderPhaseRefuelTakeoff
			t.Progress = 0
			
		case TraderPhaseRefuelTakeoff:
			t.Progress++
			if t.Progress >= 1.0 {
				t.Progress = 0
				t.Flight.Len = 0 // заправка кончилась, старый Flight не нужен
				t.Phase = t.AfterRefuel
			}
			
		case TraderPhasePlanetToEdge:
			if t.Flight.Len == 0 {
				refuelPlanet := w.PlanetMap[t.Route[t.RouteIdx]][3]
				exitX, exitY := w.getJumpExitPoint(t.Route[t.RouteIdx], t.Route[t.RouteIdx+1])
				w.PrepareJumpFlight(t, refuelPlanet, exitX, exitY)
			}
			if t.AdvanceFlight() {
				t.Phase = TraderPhaseJumpEnter
				t.Progress = 0
			}
			
		case TraderPhaseEdgeToPlanet:
			if t.Flight.Len == 0 {
				ax, ay := w.getJumpEntryPoint(t.Route[t.RouteIdx], t.Route[t.RouteIdx-1])
				w.PrepareFlightFromPoint(t, ax, ay, w.Planets[t.ToPlanet])
			}
			if t.AdvanceFlight() {
				t.Phase = TraderPhaseLanding
				t.Progress = 0
				t.FromPlanet = t.ToPlanet
			}

		case TraderPhasePirateFind:
			w.traderPiratePhaseFind(t)

		case TraderPhasePirateLoad:
			w.traderPirateLoad(t)

		case TraderPhasePirateFindMarket:
			w.traderPirateFindMarket(t)

		case TraderPhaseUnloadPirate:
			w.traderUnloadPirate(t)
			
		default:
			log.Printf("[TRACK] unknown phase=%s(%d) id=%d -> Rest", PhaseName(t.Phase), t.Phase, t.ID)
			t.Phase = TraderPhaseRest
			t.Progress = 0
		}
	}
}

// ---------- Вспомогательные функции ----------

func (w *World) traderCheckLocal(t *Trader) {
	// Если уже есть груз — идем искать дефицит
	if t.Cargo > 0 {
		t.Phase = TraderPhaseFindTrip
		t.Progress = 0
		return
	}

	p := w.Planets[t.FromPlanet]

	// Случайный порядок товаров — чтобы трейдеры не сбивались на первый
	order := w.Rng.Perm(int(GoodCount))

	// 1. Ищем избыток на ТЕКУЩЕЙ планете (по случайному порядку товаров)
	for _, gi := range order {
		g := Good(gi)
		for i := range w.TickEvents {
			e := &w.TickEvents[i]
			if e.Processed || e.Deficit || e.PlanetID != p.ID || e.Good != g { continue }
			if !w.CanTrade(p, t.Race) { continue }
			if !w.IsLegal(p, g) { continue }
			t.Good = g
			e.Processed = true
			t.Phase = TraderPhaseLoad
			t.Progress = 0
			return
		}
	}

	// 2. Телепорт на планету с избытком (по приоритету, по случайному порядку товаров)
	var bestEvent *TickEvent
	bestDist := math.MaxFloat64

	fromStar := p.StarID
	fromConst := w.Stars[fromStar].ConstellationID

	for _, gi := range order {
		g := Good(gi)
		// 2.1. Своя система
		for i := range w.TickEvents {
			e := &w.TickEvents[i]
			if e.Processed || e.Deficit || e.Good != g { continue }
			if !w.CanTrade(w.Planets[e.PlanetID], t.Race) { continue }
			if !w.IsLegal(w.Planets[e.PlanetID], g) { continue }
			if e.StarID == fromStar {
				d := w.DistMatrix[fromStar][e.StarID]
				if d < bestDist { bestDist = d; bestEvent = e }
			}
		}
	}
	// 2.2. Своё созвездие (если в системе не нашли)
	if bestEvent == nil {
		for _, gi := range order {
			g := Good(gi)
			for i := range w.TickEvents {
				e := &w.TickEvents[i]
				if e.Processed || e.Deficit || e.Good != g { continue }
				if !w.CanTrade(w.Planets[e.PlanetID], t.Race) { continue }
				if !w.IsLegal(w.Planets[e.PlanetID], g) { continue }
				if w.Stars[e.StarID].ConstellationID == fromConst {
					d := w.DistMatrix[fromStar][e.StarID]
					if d < bestDist { bestDist = d; bestEvent = e }
				}
			}
		}
	}
	// 2.3. Вся карта (если в созвездии не нашли)
	if bestEvent == nil {
		for _, gi := range order {
			g := Good(gi)
			for i := range w.TickEvents {
				e := &w.TickEvents[i]
				if e.Processed || e.Deficit || e.Good != g { continue }
				if !w.CanTrade(w.Planets[e.PlanetID], t.Race) { continue }
				if !w.IsLegal(w.Planets[e.PlanetID], g) { continue }
				d := w.DistMatrix[fromStar][e.StarID]
				if d < bestDist { bestDist = d; bestEvent = e }
			}
		}
	}

	// 3. Если нашли — телепортируемся и грузимся
	if bestEvent != nil {
		t.FromPlanet = bestEvent.PlanetID
		t.ToPlanet = bestEvent.PlanetID
		t.Good = bestEvent.Good
		t.Phase = TraderPhaseLoad
		t.Progress = 0
		bestEvent.Processed = true
		return
	}

	// 4. Ничего не нашли — идём в пиратский цикл
	t.PirateCanSeeMe = true
	t.Phase = TraderPhasePirateFind
	t.Progress = 0
}

// traderPirateFind — ищем ближайшее логово.
func (w *World) traderPirateFind(t *Trader) {
	fromStar := w.Planets[t.FromPlanet].StarID
	bestDist := math.MaxFloat64
	bestID := -1
	for _, p := range w.Planets {
		if p.OwnerId != oiPirate {
			continue
		}
		d := w.DistMatrix[fromStar][p.StarID]
		if d < bestDist {
			bestDist = d
			bestID = p.ID
		}
	}
	if bestID < 0 {
		t.PirateCanSeeMe = false
		t.Phase = TraderPhaseRest
		t.Progress = 0
		return
	}
	t.ToPlanet = bestID
	t.BaseCost = 0
	t.Phase = TraderPhaseRefuel
	t.Progress = 0
}

// traderPirateFind — ищем ближайшее логово.
func (w *World) traderPiratePhaseFind(t *Trader) {
	fromStar := w.Planets[t.FromPlanet].StarID
	bestDist := math.MaxFloat64
	bestID := -1
	for _, p := range w.Planets {
		if p.OwnerId != oiPirate {
			continue
		}
		d := w.DistMatrix[fromStar][p.StarID]
		if d < bestDist {
			bestDist = d
			bestID = p.ID
		}
	}
	if bestID < 0 {
		t.PirateCanSeeMe = false
		t.Phase = TraderPhaseRest
		t.Progress = 0
		return
	}
	t.ToPlanet = bestID
	t.BaseCost = 0
	t.Phase = TraderPhaseRefuel
	t.Progress = 0
}

// traderPirateLoad — загрузка на логове.
func (w *World) traderPirateLoad(t *Trader) {
	p := w.Planets[t.ToPlanet]
	t.FromPlanet = t.ToPlanet

	var goods []Good
	for g := 0; g < int(GoodCount); g++ {
		if p.Market[g].Count > 0 {
			goods = append(goods, Good(g))
		}
	}
	if len(goods) == 0 {
		t.PirateCanSeeMe = false
		t.Phase = TraderPhaseRest
		t.Progress = 0
		return
	}
	g := goods[w.Rng.Intn(len(goods))]
	buyPrice := ShopGoodsPurchasePrice(p, g)
	if buyPrice <= 0 {
		t.PirateCanSeeMe = false
		t.Phase = TraderPhaseRest
		t.Progress = 0
		return
	}

	const fuelReserve = 100.0
	available := t.Money - fuelReserve
	if available <= 0 {
		w.respawnTrader(t)
		return
	}
	maxQ := int(available / buyPrice)
	if maxQ > t.CargoMax {
		maxQ = t.CargoMax
	}
	if maxQ > p.Market[g].Count {
		maxQ = p.Market[g].Count
	}
	if maxQ <= 0 {
		w.respawnTrader(t)
		return
	}

	cost, ok := w.Buy(p, g, maxQ, t.Race)
	if !ok {
		t.PirateCanSeeMe = false
		t.Phase = TraderPhaseRest
		t.Progress = 0
		return
	}
	t.Money -= cost
	t.BaseCost = cost // как у каравана: сумма за весь груз
	t.Cargo = maxQ
	t.Good = g

	t.Phase = TraderPhasePirateFindMarket
	t.Progress = 0
}

// traderPirateFindMarket — ближайшая обитаемая.
func (w *World) traderPirateFindMarket(t *Trader) {
	fromStar := w.Planets[t.FromPlanet].StarID
	bestDist := math.MaxFloat64
	bestID := -1
	for _, p := range w.Planets {
		if !p.Habitable || p.ID == t.FromPlanet {
			continue
		}
		d := w.DistMatrix[fromStar][p.StarID]
		if d < bestDist {
			bestDist = d
			bestID = p.ID
		}
	}
	if bestID < 0 {
		t.PirateCanSeeMe = false
		t.Phase = TraderPhaseRest
		t.Progress = 0
		return
	}
	t.ToPlanet = bestID
	t.Phase = TraderPhaseRefuel
	t.Progress = 0
}

// traderUnloadPirate — продажа без печати и без зарплаты.
func (w *World) traderUnloadPirate(t *Trader) {
	p := w.Planets[t.ToPlanet]
	t.FromPlanet = t.ToPlanet

	if t.Cargo > 0 {
		rev, sold, ok := w.Sell(p, t.Good, t.Cargo, t.Race)
		if ok && sold > 0 {
			t.Money += rev
			t.Cargo -= sold
		}
	}
	if t.Cargo == 0 {
		t.BaseCost = 0
		t.PirateCanSeeMe = false
		t.Phase = TraderPhaseRest
		t.Progress = 0
		return
	}
	next := w.traderFindDeficitNearby(t)
	if next == nil {
		t.BaseCost = 0
		t.PirateCanSeeMe = false
		t.Phase = TraderPhaseRest
		t.Progress = 0
		return
	}
	t.ToPlanet = next.ID
	t.Phase = TraderPhaseRefuel
	t.Progress = 0
}

// traderFindDeficitNearby — копия caravanFindDeficitNearby для трейдера.
func (w *World) traderFindDeficitNearby(t *Trader) *Planet {
	fromP := w.Planets[t.FromPlanet]
	fromStar := fromP.StarID
	fromConst := w.Stars[fromStar].ConstellationID

	var best *Planet
	bestDist := math.MaxFloat64

	check := func(p *Planet) {
		if !p.Habitable || p.ID == t.FromPlanet {
			return
		}
		if !w.CanTrade(p, t.Race) {
			return
		}
		if !w.IsLegal(p, t.Good) {
			return
		}
		if !w.caravanIsDeficit(p, t.Good) {
			return
		}
		dist := w.DistMatrix[fromStar][p.StarID]
		if dist < bestDist {
			bestDist = dist
			best = p
		}
	}

	for _, p := range w.PlanetMap[fromStar] {
		check(p)
	}
	if best != nil { return best }
	for _, sid := range w.ConstellationStars[fromConst] {
		for _, p := range w.PlanetMap[sid] {
			check(p)
		}
	}
	if best != nil { return best }
	for _, p := range w.Planets {
		check(p)
	}
	return best
}

// respawnTrader — смерть + респавн дома.
func (w *World) respawnTrader(t *Trader) {
	t.Alive = false
	w.SpawnTrader(t.HomePlanet, t.Good)
}

func (w *World) traderCheckUnload(t *Trader) {
	p := w.Planets[t.ToPlanet]
	
	// Проверяем, есть ли новость о дефиците для нашего товара на этой планете
	hasDeficitNews := false
	for i := range w.TickEvents {
		e := &w.TickEvents[i]
		if e.Processed || !e.Deficit || e.Good != t.Good { continue }
		if e.PlanetID == p.ID {
			hasDeficitNews = true
			if !w.IsLegal(p, t.Good) {
				t.Smuggler = true // товар нелегален на планете разгрузки
			}
			break
		}
	}
	
	// OnTime — флаг "полная зарплата" (успел к дефициту)
	t.OnTime = hasDeficitNews
	
	t.Phase = TraderPhaseUnload
	t.Progress = 0
}

func (w *World) traderFindTrip(t *Trader) {
	var bestEvent *TickEvent
	bestDist := math.MaxFloat64
	
	fromStar := w.Planets[t.FromPlanet].StarID
	fromConst := w.Stars[fromStar].ConstellationID
	
	// 1. Своя система
	for i := range w.TickEvents {
		e := &w.TickEvents[i]
		if e.Processed || !e.Deficit || e.Good != t.Good { continue }
		if !w.CanTrade(w.Planets[e.PlanetID], t.Race) { continue }
		if e.PlanetID == t.FromPlanet { continue }
		if e.StarID == fromStar {
			d := w.DistMatrix[fromStar][e.StarID]
			if d < bestDist { bestDist = d; bestEvent = e }
		}
	}
	// 2. Своё созвездие
	if bestEvent == nil {
		for i := range w.TickEvents {
			e := &w.TickEvents[i]
			if e.Processed || !e.Deficit || e.Good != t.Good { continue }
			if !w.CanTrade(w.Planets[e.PlanetID], t.Race) { continue }
			if e.PlanetID == t.FromPlanet { continue }
			if w.Stars[e.StarID].ConstellationID == fromConst {
				d := w.DistMatrix[fromStar][e.StarID]
				if d < bestDist { bestDist = d; bestEvent = e }
			}
		}
	}
	// 3. Вся карта
	if bestEvent == nil {
		for i := range w.TickEvents {
			e := &w.TickEvents[i]
			if e.Processed || !e.Deficit || e.Good != t.Good { continue }
			if !w.CanTrade(w.Planets[e.PlanetID], t.Race) { continue }
			if e.PlanetID == t.FromPlanet { continue }
			d := w.DistMatrix[fromStar][e.StarID]
			if d < bestDist { bestDist = d; bestEvent = e }
		}
	}

	if bestEvent != nil {
		t.ToPlanet = bestEvent.PlanetID
		t.Phase = TraderPhaseRefuel // <-- Заправляемся перед вылетом
		bestEvent.Processed = true
	} else {
		// Не нашли дефицит — продаём на месте (2%, опоздал)
		t.ToPlanet = t.FromPlanet
		t.OnTime = false
		t.Phase = TraderPhaseUnload
		t.Progress = 0
	}
}

func (w *World) traderFindTripPlus(t *Trader) {
	randomGood := Good(w.Rng.Intn(int(GoodCount)))
	if w.traderTryFindSurplus(t, randomGood) {
		return
	}
	
	for g := 0; g < int(GoodCount); g++ {
		if Good(g) == randomGood {
			continue
		}
		if w.traderTryFindSurplus(t, Good(g)) {
			return
		}
	}
	
	t.Phase = TraderPhaseRest
	t.Progress = 0
}

func (w *World) traderTryFindSurplus(t *Trader, g Good) bool {
	var bestEvent *TickEvent
	bestDist := math.MaxFloat64
	
	fromStar := w.Planets[t.FromPlanet].StarID
	fromConst := w.Stars[fromStar].ConstellationID
	
	// 1. Своя система
	for i := range w.TickEvents {
		e := &w.TickEvents[i]
		if e.Processed || e.Deficit || e.Good != g { continue }
		if !w.CanTrade(w.Planets[e.PlanetID], t.Race) { continue }
		if e.StarID == fromStar {
			d := w.DistMatrix[fromStar][e.StarID]
			if d < bestDist { bestDist = d; bestEvent = e }
		}
	}
	// 2. Своё созвездие
	if bestEvent == nil {
		for i := range w.TickEvents {
			e := &w.TickEvents[i]
			if e.Processed || e.Deficit || e.Good != g { continue }
			if !w.CanTrade(w.Planets[e.PlanetID], t.Race) { continue }
			if w.Stars[e.StarID].ConstellationID == fromConst {
				d := w.DistMatrix[fromStar][e.StarID]
				if d < bestDist { bestDist = d; bestEvent = e }
			}
		}
	}
	// 3. Вся карта
	if bestEvent == nil {
		for i := range w.TickEvents {
			e := &w.TickEvents[i]
			if e.Processed || e.Deficit || e.Good != g { continue }
			if !w.CanTrade(w.Planets[e.PlanetID], t.Race) { continue }
			d := w.DistMatrix[fromStar][e.StarID]
			if d < bestDist { bestDist = d; bestEvent = e }
		}
	}
	
	if bestEvent != nil {
		// Проверяем, есть ли куда продать (дефицит с выгодой)
		buyPrice := ShopGoodsPurchasePrice(w.Planets[bestEvent.PlanetID], g)
		profitable := false
		
		for i := range w.TickEvents {
			de := &w.TickEvents[i]
			if de.Processed || !de.Deficit || de.Good != g { continue }
			if !w.CanTrade(w.Planets[de.PlanetID], t.Race) { continue }
			sellPrice := ShopGoodsSellPrice(w.Planets[de.PlanetID], g)
			if sellPrice > buyPrice {
				profitable = true
				break
			}
		}
		
		if profitable {
			t.ToPlanet = bestEvent.PlanetID
			t.Good = g
			t.Phase = TraderPhaseTakeoff
			t.Progress = 0
			bestEvent.Processed = true
			return true
		}
	}
	return false
}

// traderLoad — берет груз с планеты БЕСПЛАТНО (планета отдает его курьеру)
func (w *World) traderLoad(t *Trader) {
	// Если уже есть груз — идем искать дефицит
	if t.Cargo > 0 {
		t.Phase = TraderPhaseFindTrip
		t.Progress = 0
		return
	}

	p := w.Planets[t.FromPlanet]
	m := &p.Market[t.Good]
	
	count := m.Count
	if count > t.CargoMax {
		count = t.CargoMax
	}
	if count <= 0 {
		t.Phase = TraderPhaseCheckLocal
		t.Progress = 0
		return
	}
	
	m.Count -= count
	t.Cargo = count
	t.Smuggler = false // новый рейс — статус чист	

	// планета получает деньги за товар (из воздуха)
	p.Money += float64(count) * ShopGoodsPurchasePrice(p, t.Good) * 0.01

	// TechLevel++ за деньги (как у трейдера)
	if p.TechLevel < w.GalaxyTechLevel+3 && p.TechLevel < 15 {
		mult := 1.0 + float64(p.TechLevel-w.GalaxyTechLevel)
			if mult < 1 {
				mult = 1
			}
		cost := totalCost[p.TechLevel] * (w.techDiscountPlanets(p.TechLevel) + 1) * PlanetCostMult * mult
		if p.Money >= cost + p.MoneyRef {
			p.Money -= cost
			p.TechLevel++
		}
	} else if p.TechLevel > w.GalaxyTechLevel+3 && p.TechLevel < 15  {
		// Упёрлись в потолок ГТУ (или TL=15) — излишек в сейф для банкира.
		surplus := p.Money - p.MoneyRef
		if surplus > 0 {
			p.Money -= surplus
			p.MoneySafe += surplus
		}
	}
	t.BaseCost = 0
	
	t.Phase = TraderPhaseFindTrip
	t.Progress = 0
}

func (w *World) traderRefuel(t *Trader) {
	p := w.Planets[t.FromPlanet]
	need := t.FuelMax - t.Fuel
	if need > 0 {
		cost := w.CalculateFuelCost(need, p.OwnerId)
		if t.Money >= cost {
			t.Money -= cost
			t.Fuel = t.FuelMax
			p.Money += cost
			t.NeedsRefuel = false
		} else {
			t.Phase = TraderPhaseRest
			t.Progress = 0
			return
		}
	}
	t.Phase = TraderPhaseTakeoff
	t.Progress = 0
}

// traderUnload — сдает груз и получает ЗАРПЛАТУ (деньги печатаются)
func (w *World) traderUnload(t *Trader) {
	p := w.Planets[t.ToPlanet]
	if t.Cargo > 0 {
		p.Market[t.Good].Count += t.Cargo
		
		salaryRate := 0.02 // Урезанная (опоздал)
		if t.OnTime  {
			salaryRate = 0.10 // Полная (успел)
		}
		
		salary := float64(t.Cargo) * ShopGoodsSellPrice(p, t.Good) * salaryRate
		t.Money += salary
		t.Cargo = 0
	}
	t.Phase = TraderPhaseRest
	t.Progress = 0
}

func PriceGradeByRatio(purchasePrice, basePrice int) int {
	if basePrice <= 0 || purchasePrice <= 0 {
		return 0
	}
	switch {
	case purchasePrice*100 <= basePrice*70:
		return 3
	case purchasePrice*100 <= basePrice*85:
		return 2
	case purchasePrice*100 <= basePrice*95:
		return 1
	default:
		return 0
	}
}

// GoodName возвращает имя товара для логов (из канона GoodsMarket).
func GoodName(g Good) string {
	if int(g) < 0 || int(g) >= int(GoodCount) {
		return "?"
	}
	return GoodsMarket[g].InternalName
}

// TotalTL возвращает накопленную стоимость за N шагов (N = 1..15)
// при базовом члене base = 5000.
// Формула: S_N = base * (N-1)*N*(2N-1) / 6
func TotalTL(N int, base float64) float64 {
	if N <= 0 {
		return 0
	}
	// защита от выхода за диапазон 1..15, если нужно
	// if N > 15 { N = 15 }
	return base * float64(N-1) * float64(N) * float64(2*N-1) / 6.0
}

// с рекуррентным накоплением (без замкнутой формулы)
func TotalTLIter(N int, base float64) float64 {
	var sum float64
	for k := 1; k <= N; k++ {
		sum += base * float64((k-1)*(k-1))
	}
	return sum
}