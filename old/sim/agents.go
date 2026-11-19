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
	JumpRange  float64
	Good       Good
	Cargo      int
	BaseCost   float64
	Money      float64
	Fuel       float64
	FuelMax    float64
	Alive      bool
}

type Pirate struct {
	ID         int
	BasePlanet int
	Power      int
	JumpRange  float64
	Alive      bool
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

type Diplomat struct {
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
	PhaseLift     = 0
	PhaseJumpIn   = 1
	PhaseJump     = 2
	PhaseJumpOut  = 3
	PhaseDescend  = 4
	PhaseUnload   = 5
	PhaseIdle     = 6
	PhaseFindTrip = 7
	PhaseLoad     = 8
)

const (
	targetCaravans  = 2500
	targetPirates   = 200
	targetLiners    = 50
	targetDiplomats = 50
	CaravanCargo    = 100
	CaravanMoney    = 20000.0
	CaravanFuelMax  = 25.0
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

func (w *World) SpawnCaravan() *Caravan {
	from, ok := w.randomHabitable()
	if !ok {
		return nil
	}
	return w.newCaravan(from, nil)
}

func (w *World) newCaravan(from int, route []int) *Caravan {
	nextCaravanID++
	c := &Caravan{
		ID:         nextCaravanID,
		Race:       w.Planets[from].Race,
		FromPlanet: from,
		ToPlanet:   from,
		Route:      route,
		RouteIdx:   0,
		Phase:      PhaseFindTrip,
		Progress:   0,
		Speed:      2 + w.Rng.Float64()*3,
		JumpRange:  30,
		Good:       0,
		Cargo:      0,
		BaseCost:   0,
		Money:      CaravanMoney,
		Fuel:       CaravanFuelMax,
		FuelMax:    CaravanFuelMax,
		Alive:      true,
	}
	w.Caravans = append(w.Caravans, c)
	return c
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