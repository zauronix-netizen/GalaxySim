package sim

type Caravan struct {
	ID         int
	Race       Race
	FromPlanet int
	ToPlanet   int
	Route      []int
	RouteIdx   int
	Phase      int
	Progress   float64
	Speed      float64
	SpeedRng   float64 // фикс при спавне
	JumpRange  float64
	Good       Good
	Cargo      int
	CargoMax   int
	TechLevel  int
	BaseCost   float64
	Money      float64
	Fuel       float64
	FuelMax    float64
	Looted     bool // уже ограблен пиратом — не атакуем до перегрузки
	Flight     FlightPath
	AfterRefuel int
	Alive      bool
}

type Pirate struct {
	ID         int
	BasePlanet int
	Power      int
	TechLevel  int
	JumpRange  float64
	Speed      float64
	SpeedRng   float64
	Phase      int
	Progress   float64
	Alive      bool
	Wins int
}

type Liner struct {
	ID         int
	FromPlanet int
	ToPlanet   int
	Route      []int
	RouteIdx   int
	Phase      int
	Progress   float64
	Speed      float64
	JumpRange  float64
	Alive      bool
}

const (
	DiplomatRoleDiplomat = 0
	DiplomatRoleBanker   = 1
)

type Diplomat struct {
	ID         int
	Role       int // DiplomatRoleDiplomat / DiplomatRoleBanker
	FromPlanet int
	ToPlanet   int
	Route      []int
	RouteIdx   int
	Phase      int
	Progress   float64
	Speed      float64
	JumpRange  float64
	Fuel       float64
	FuelMax    float64
	Flight     FlightPath
	AfterRefuel int
	Alive      bool
}

const (
	PhaseLift     = 0
	PhaseJumpIn   = 1
	PhaseJump     = 2
	PhaseJumpOut  = 3
	PhaseDescend  = 4
	PhaseUnload   = 5
	PhaseIdle     = 6
	PhaseFindTrip = 7
	PhaseLoad     = 8
	PhaseFindSurplus = 9  // NEW: поиск избытка на других планетах
	PhaseFindDeficit = 10 // NEW: поиск дефицита после загрузки
)

const (
	targetCaravans  = 0
	targetPirates   = 0
	targetLiners    = 0
	targetDiplomats = 0
	CaravanCargo    = 100
	CaravanMoney    = 0.0
	CaravanFuelMax  = 5.0
	CaravanWaves    = 5 // количество волн спавна караванов (по 420 за волну)
)

const (
	CaravanPhaseTakeoff = iota + 100
	CaravanPhaseCruise
	CaravanPhaseLanding
	CaravanPhaseToEdge
	CaravanPhaseJumpEnter
	CaravanPhaseJump
	CaravanPhaseJumpExit
	CaravanPhaseFuelCheck
	CaravanPhaseEdgeToEdge
	CaravanPhaseRefuelApproach
	CaravanPhaseRefuelLanding
	CaravanPhaseRefuelJump
	CaravanPhaseRefuelTakeoff
	CaravanPhasePlanetToEdge
	CaravanPhaseEdgeToPlanet
)

const (
    DiplomatPhaseTakeoff = iota + 200
    DiplomatPhaseCruise
    DiplomatPhaseLanding
    DiplomatPhaseToEdge
    DiplomatPhaseJumpEnter
    DiplomatPhaseJump
    DiplomatPhaseJumpExit
    DiplomatPhaseFuelCheck
    DiplomatPhaseEdgeToEdge
    DiplomatPhaseRefuelApproach
    DiplomatPhaseRefuelLanding
    DiplomatPhaseRefuelJump
    DiplomatPhaseRefuelTakeoff
    DiplomatPhasePlanetToEdge
    DiplomatPhaseEdgeToPlanet
)



var nextCaravanID, nextPirateID, nextLinerID, nextDiplomatID int

func (w *World) randomHabitablePair() (int, int, bool) {
	hab := []int{}
	for _, p := range w.Planets {
		if p.Habitable {
			hab = append(hab, p.ID)
		}
	}
	if len(hab) < 2 {
		return 0, 0, false
	}
	for tries := 0; tries < 100; tries++ {
		a := hab[w.Rng.Intn(len(hab))]
		b := hab[w.Rng.Intn(len(hab))]
		if a == b {
			continue
		}
		if w.Planets[a].StarID == w.Planets[b].StarID {
			continue
		}
		return a, b, true
	}
	return 0, 0, false
}

func (w *World) randomHabitable() (int, bool) {
	hab := []int{}
	for _, p := range w.Planets {
		if p.Habitable {
			hab = append(hab, p.ID)
		}
	}
	if len(hab) == 0 {
		return 0, false
	}
	return hab[w.Rng.Intn(len(hab))], true
}

func (w *World) SpawnPirate() *Pirate {
	base, ok := w.randomHabitable()
	if !ok {
		return nil
	}
	nextPirateID++
	p := &Pirate{
		ID:         nextPirateID,
		BasePlanet: base,
		Power:      1 + w.Rng.Intn(3),
		JumpRange:  40,
		Alive:      true,
	}
	w.Pirates = append(w.Pirates, p)
	return p
}

// findPirateLair — логово пирата (планета уже помечена oiPirate при генерации мира).
func (w *World) findPirateLair(starID int) int {
	for _, p := range w.PlanetMap[starID] {
		if p.OwnerId == oiPirate {
			return p.ID
		}
	}
	return 0
}

// pirateAttack — пират атакует караван в своей системе.
// Возвращает true, если караван убит.
func (w *World) pirateAttack(p *Pirate, c *Caravan) bool {
	if c.Looted {
		return false
	}
	// «Догоню?» 3% в тик.
	if w.Rng.Intn(100) >= 3 {
		return false
	}
	// «Атакую?» 30%.
	if w.Rng.Intn(100) >= 30 {
		return false
	}
	// Исход: 50% убить, 50% отнять 50% груза.
	if w.Rng.Intn(100) < 50 {
		return true // убит
	}
	c.Cargo /= 2
	c.Looted = true
	return false
}

// SpawnPirateAt — пират на пиратской планете (логово).
func (w *World) SpawnPirateAt(planetID int) *Pirate {
	if planetID == 0 {
		return nil
	}
	nextPirateID++
	speedRng := w.Rng.Float64()
	p := &Pirate{
		ID:         nextPirateID,
		BasePlanet: planetID,
		Power:      1 + w.Rng.Intn(3),
		TechLevel:  1,
		JumpRange:  40,
		SpeedRng:   speedRng,
		Speed:      0.3 + speedRng*(1.0/10.0), // TL=1: 0.3..0.4
		Wins:       0,
		Alive:      true,
		Phase: PiratePhaseOrbit,
	}
	w.Pirates = append(w.Pirates, p)
	return p
}

func (w *World) SpawnLiner() *Liner {
	from, to, ok := w.randomHabitablePair()
	if !ok {
		return nil
	}
	route := w.buildRoute(w.Planets[from].StarID, w.Planets[to].StarID, 35)
	if route == nil {
		return nil
	}
	nextLinerID++
	l := &Liner{
		ID:         nextLinerID,
		FromPlanet: from,
		ToPlanet:   to,
		Route:      route,
		RouteIdx:   0,
		Phase:      PhaseLift,
		Progress:   0,
		Speed:      2 + w.Rng.Float64()*3,
		JumpRange:  35,
		Alive:      true,
	}
	w.Liners = append(w.Liners, l)
	return l
}

func (w *World) SpawnDiplomat() *Diplomat {
	from, to, ok := w.randomHabitablePair()
	if !ok {
		return nil
	}
	if w.Planets[from].Race == w.Planets[to].Race {
		return nil
	}
	route := w.buildRoute(w.Planets[from].StarID, w.Planets[to].StarID, 35)
	if route == nil {
		return nil
	}
	nextDiplomatID++
	d := &Diplomat{
		ID:         nextDiplomatID,
		FromPlanet: from,
		ToPlanet:   to,
		Route:      route,
		RouteIdx:   0,
		Phase:      PhaseLift,
		Progress:   0,
		Speed:      2 + w.Rng.Float64()*3,
		JumpRange:  35,
		Alive:      true,
	}
	w.Diplomats = append(w.Diplomats, d)
	return d
}

func (w *World) SpawnInitialAgents() {
	for i := 0; i < targetCaravans; i++ {
		w.SpawnCaravan()
	}
	for i := 0; i < targetPirates; i++ {
		w.SpawnPirate()
	}
	for i := 0; i < targetLiners; i++ {
		w.SpawnLiner()
	}
	for i := 0; i < targetDiplomats; i++ {
		w.SpawnDiplomat()
	}
}

func (w *World) respawnPirate(p *Pirate) {
	// Пират привязан к логову — не мигрирует.
	// Логово уже есть (BasePlanet), просто оживляем.
	p.Alive = true
	p.TechLevel = 1              // сброс на 1 уровень
	p.Wins = 0
	p.Speed = 0.3 + p.SpeedRng*(1.0/10.0)
}

func (w *World) spawnPirate(p *Pirate) {
	candidates := []int{}
	for _, c := range w.Caravans {
		if !c.Alive || c.Phase == PhaseJump {
			continue
		}
		var cs int
		switch c.Phase {
		case PhaseLift, PhaseJumpIn:
			cs = w.Planets[c.FromPlanet].StarID
		case PhaseJumpOut, PhaseDescend:
			cs = w.Planets[c.ToPlanet].StarID
		default:
			continue
		}
		candidates = append(candidates, cs)
	}
	if len(candidates) == 0 {
		return
	}
	newStar := candidates[w.Rng.Intn(len(candidates))]
	planets := w.PlanetMap[newStar]
	if len(planets) == 0 {
		return
	}
	newPlanet := planets[w.Rng.Intn(len(planets))].ID
	nextPirateID++
	p.ID = nextPirateID
	p.BasePlanet = newPlanet
	p.Alive = true
}