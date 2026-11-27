package sim

/*
	Логика лайнера (идеал)
1. спаунимся на планете с избытком пассажиров
2. читаем местные новости (ищем избыток)
3. берём пассажиров на борт
4. читаем новости приоритетно планета, звезда, созвездие, галактика (ищем дефицит)
5. заправляемся (если надо)
6. взлетаем и летим
7. прилетаем с пассажирами
8. читаем новости (проверяем дефицит)
9. высаживаем и отдыхаем
	(рабочий цикл меняет только поиск избытка)
10. читаем новости приоритетно планета, звезда, созвездие, галактика (ищем избыток)

	СТАТУС: файловая база. Летают пустыми (Passengers=0).
	Маршрутов нет, глобалов нет (World.Liners / Planet.TouristPool — потом).
	Пираты лайнеров пока не видят. Пока НЕ подключены к tick.go.
*/

// ---------- Структуры ----------

/*

type Liner struct {
	ID         int
	Race       Race
	HomePlanet int
	TechLevel  int
	FromPlanet int
	ToPlanet   int
	Phase      int
	Progress   float64
	Speed      float64
	SpeedRng   float64 // 0..1, фикс при спавне (сохраняется при апгрейде)

	// «Груз» лайнера — пассажиры.
	// Имена полей намеренно отличаются от Trader.Cargo/CargoMax,
	// чтобы случайно не перепутать при будущей интеграции.
	Passengers int // текущее число пассажиров (0 — пустой, пока так)
	Capacity   int // вместимость = TechLevel × 10 (как CargoMax у трейдера)

	Money   float64
	Fuel    float64
	FuelMax float64
	Alive   bool

	// Для отладки
	DebugLog  bool
	LastPhase int
}

var nextLinerID int

// ---------- Константы фаз ----------

const (
	LinerPhaseIdle = iota + 10
	LinerPhaseSpawn
	LinerPhaseLoad
	LinerPhaseTakeoff
	LinerPhaseCruise
	LinerPhaseLanding
	LinerPhaseUnload
	LinerPhaseRest
)

// LinerPhaseName возвращает символьное имя фазы лайнера для логов.
func LinerPhaseName(p int) string {
	switch p {
	case LinerPhaseIdle:
		return "LinerIdle"
	case LinerPhaseSpawn:
		return "LinerSpawn"
	case LinerPhaseLoad:
		return "LinerLoad"
	case LinerPhaseTakeoff:
		return "LinerTakeoff"
	case LinerPhaseCruise:
		return "LinerCruise"
	case LinerPhaseLanding:
		return "LinerLanding"
	case LinerPhaseUnload:
		return "LinerUnload"
	case LinerPhaseRest:
		return "LinerRest"
	default:
		return "?"
	}
}

// ---------- Спавн ----------

// SpawnLiner создаёт лайнера. НЕ сохраняет в World — глобалов пока нет.
// Возвращает указатель; кто позовёт — решает, куда положить.
// Когда появится World.Liners, сюда добавится append в конце.
//
// TL спавна — случайно между TL планеты и GTL, как у трейдеров.
// Старт: планеты 2, GTL 3 → линеры на 2 или 3.
func (w *World) SpawnLiner(planetID int) *Liner {
	p := w.Planets[planetID]
	nextLinerID++

	gtl := w.GalaxyTechLevel
	if gtl < 2 {
		gtl = 3 // fallback на warmup / аномалии
	}
	tlLo, tlHi := p.TechLevel, gtl
	if tlLo > tlHi {
		tlLo, tlHi = tlHi, tlLo
	}
	if tlLo < 1 {
		tlLo = 1
	}
	if tlHi > 15 {
		tlHi = 15
	}
	techLevel := tlLo
	if tlHi > tlLo {
		techLevel = tlLo + w.Rng.Intn(tlHi-tlLo+1)
	}

	rngSpeed := w.Rng.Float64()

	l := &Liner{
		ID:         nextLinerID,
		Race:       p.Race,
		HomePlanet: planetID,
		TechLevel:  techLevel,
		FromPlanet: planetID,
		ToPlanet:   planetID,
		Phase:      LinerPhaseIdle,
		Progress:   0,
		Speed:      0.2 + rngSpeed*(float64(techLevel)/10.0),
		SpeedRng:   rngSpeed,
		Passengers: 0, // пустой, пока так
		Capacity:   techLevel * 10,
		Money:      5000.0,
		Fuel:       15.0,
		FuelMax:    15.0,
		Alive:      true,
		DebugLog:   false,
		LastPhase:  -1,
	}
	return l
}

// ---------- Фаза (каркас, пока пустая) ----------

// phaseLiners — каркас. Пока ничего не делает.
//
// TODO (следующие шаги, каждый — отдельно):
//   1. Добавить World.Liners и звать phaseLiners из tick.go.
//   2. Спавн: волны (рядом с трейдерами), SpawnLiner + append в World.Liners.
//   3. FindTrip: избыток пассажиров на планете → Load.
//   4. Load: Passengers = Capacity (бесплатно, как у трейдера).
//   5. FindTripPlus: поиск дефицита пассажиров по новостям.
//   6. Refuel / Takeoff / Cruise / Landing / Unload / Rest — по образцу трейдера.
//   7. Пираты (Liner.CanSeeMe или через общий флаг) — когда решим, что
//      лайнеры им видны.
//
// Пока функция пустая, чтобы файл собирался и был «на месте».
// Подключение к tick.go — отдельный шаг, после появления World.Liners.
func (w *World) phaseLiners() {
	// Пусто. Ждёт World.Liners.
}

*/