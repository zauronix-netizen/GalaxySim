package sim

// Vec2 — координаты в системе. 2D: вселенная относительно плоская,
// 3D — только в клиенте (Raylib), на сервере не нужно.
type Vec2 struct{ X, Y float64 }

// Ship — корабль игрока.
type Ship struct {
	ID          int
	PlayerID    int
	SystemID    int
	Pos         Vec2
	Vel         Vec2
	Docked      bool
	DockID      int    // planetID или -1
	OrbitPlanet int    // planetID или -1 (на орбите планеты)
	OrbitRadius float64
	Target      *MoveOrder
}

type MoveKind int

const (
	MoveOrbit MoveKind = iota // выйти на орбиту планеты
	MoveGoto                  // лететь в точку
	MoveEdge                  // лететь к краю системы
	MoveDock                  // стыковка
)

// MoveOrder — приказ на перемещение. Сервер исполняет по ходам.
type MoveOrder struct {
	Kind   MoveKind
	Target Vec2
	Planet int
	Radius float64
	ETA    int // сколько ходов осталось (0 = прибыл)
}

// ShipCommand — что клиент шлёт. Один ход = одна команда.
type ShipCommand struct {
	PlayerID int
	ShipID   int
	Kind     MoveKind
	Planet   int
	Radius   float64
	Target   Vec2
}