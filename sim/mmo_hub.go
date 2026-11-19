package sim

import (
	"math"
	"sync"
)

// ShipHub — все корабли игроков. Такт — раз в тик симуляции.
type ShipHub struct {
	Ships  map[int]*Ship
	NextID int
	Mu     sync.Mutex
}

func NewShipHub() *ShipHub {
	return &ShipHub{Ships: make(map[int]*Ship), NextID: 1}
}

const (
	MinutesPerHour = 64
	HoursPerDay    = 32
	TicksPerDay    = MinutesPerHour * HoursPerDay
)

// Spawn — создать корабль в доке у планеты (ручной spawn пока).
func (h *ShipHub) Spawn(playerID, systemID, dockID int) *Ship {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	s := &Ship{
		ID: h.NextID, PlayerID: playerID, SystemID: systemID,
		Docked: true, DockID: dockID,
		OrbitPlanet: dockID, OrbitRadius: 0,
	}
	h.NextID++
	h.Ships[s.ID] = s
	return s
}

// ApplyCommand — принять команду, поставить приказ. ETA=1 — заглушка,
// потом здесь будет расчёт "достижимо ли, за сколько ходов" с гравитацией.
// ShipSpeed — ед/ход в системной шкале. Подбирается по темпу.

// ShipSpeed — ед/тик. 0.25 ед/день / TicksPerDay.
const ShipSpeed = 0.5 / TicksPerDay

// ApplyCommand — принять команду, посчитать ETA.
//   MoveOrbit / MoveDock — ETA=1 (манёвр у планеты).
//   MoveGoto / MoveEdge — ETA = ceil(dist(Pos, target) / ShipSpeed).
// Гравитация — только клиент, в ETA не входит.
func (h *ShipHub) ApplyCommand(w *World, cmd ShipCommand) (*Ship, bool) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	s, ok := h.Ships[cmd.ShipID]
	if !ok {
		return nil, false
	}

	// MoveUndock — мгновенно: снять док, встать в точку планеты. Без MoveOrder.
	if cmd.Kind == MoveUndock {
		if cmd.Planet >= 0 && cmd.Planet < len(w.Planets) {
			px, py := w.getPlanetPos(w.Planets[cmd.Planet], w.Day)
			s.Pos = Vec2{X: px, Y: py}
		}
		s.Docked = false
		s.DockID = -1
		s.OrbitPlanet = cmd.Planet
		s.OrbitRadius = 0
		s.Target = nil
		return s, true
	}

	eta := 1
	switch cmd.Kind {
	case MoveGoto, MoveEdge:
		dx := cmd.Target.X - s.Pos.X
		dy := cmd.Target.Y - s.Pos.Y
		dist := math.Hypot(dx, dy)
		eta = int(math.Ceil(dist / ShipSpeed))
		if eta < 1 {
			eta = 1
		}
	}

/*

	case MoveUndock:
		// позиция планеты (dockID = cmd.Planet)
		if cmd.Planet >= 0 && cmd.Planet < len(w.Planets) {
			px, py := w.getPlanetPos(w.Planets[cmd.Planet], w.Day)
			cmd.Target = Vec2{X: px, Y: py}
		}
		eta = 1
	}

*/

	s.Target = &MoveOrder{
		Kind:   cmd.Kind,
		Planet: cmd.Planet,
		Radius: cmd.Radius,
		Target: cmd.Target,
		ETA:    eta,
		Start:  s.Pos,
	}
	return s, true
}

// Tick — один ход (из SimTick). Пока без физики: ETA--, по прибытии
// ставим Docked/Pos заглушкой.
func (h *ShipHub) Tick(w *World) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	for _, s := range h.Ships {
		if s.Target == nil {
			continue
		}
		dx := s.Target.Target.X - s.Pos.X
		dy := s.Target.Target.Y - s.Pos.Y
		dist := math.Hypot(dx, dy)
		step := ShipSpeed

		if dist <= step {
			// прибыл
			s.Pos = s.Target.Target
			switch s.Target.Kind {
			case MoveOrbit:
				s.Docked = false
				s.DockID = -1
				s.OrbitPlanet = s.Target.Planet
				s.OrbitRadius = s.Target.Radius
			case MoveDock:
				s.Docked = true
				s.DockID = s.Target.Planet
				s.OrbitPlanet = s.Target.Planet
				s.OrbitRadius = 0
			case MoveUndock:
				s.Docked = false
				s.DockID = -1
				s.OrbitPlanet = s.Target.Planet
				s.OrbitRadius = 0				
			case MoveGoto, MoveEdge:
				s.Docked = false
				s.DockID = -1
				s.OrbitPlanet = -1
				s.OrbitRadius = 0
			}
			s.Target = nil
			continue
		}

		// шаг к цели
		s.Pos.X += (dx / dist) * step
		s.Pos.Y += (dy / dist) * step
		s.Target.ETA--
		if s.Target.ETA < 0 {
			s.Target.ETA = 0
		}
	}
}