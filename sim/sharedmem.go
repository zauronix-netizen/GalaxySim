package sim

import (
	"encoding/binary"
	"encoding/json"
	"sync/atomic"
	"unsafe"
)

// Snapshot — то, что пишем в разделяемую память.
type Snapshot struct {
	Day     int              `json:"day"`
	Star    int              `json:"star"`
	Objects []SnapshotObject `json:"objects"`
}

// SnapshotObject — один объект в системе.
// T: 1 = планета, 2 = корабль.
type SnapshotObject struct {
	T int     `json:"t"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

const (
	shmemName       = "GalaxySimSnapshot"
	shmemHeaderSize = 8                // uint64 index, выровнен
	shmemBufferSize = 64 * 1024        // 64 KB на буфер
	shmemSize       = shmemHeaderSize + shmemBufferSize*2
)

// SharedMem — обёртка над разделяемой памятью. Двойной буфер + атомарный индекс.
type SharedMem struct {
	handle uintptr
	addr   uintptr
	size   int
}

// OpenSharedMemWriter — открыть/создать и получить доступ на запись.
// На не-Windows возвращает (nil, nil) — работаем без shared memory.
func OpenSharedMemWriter() (*SharedMem, error) { return openSharedMemWriter() }

// OpenSharedMemReader — открыть на чтение. На не-Windows — (nil, nil).
func OpenSharedMemReader() (*SharedMem, error) { return openSharedMemReader() }

// Close — освободить.
func (s *SharedMem) Close() {
	if s == nil {
		return
	}
	closeSharedMem(s)
}

// Write — записать data в неактивный буфер и атомарно переключить индекс.
// Формат буфера: [uint32 len][data...].
func (s *SharedMem) Write(data []byte) {
	if s == nil || s.addr == 0 {
		return
	}
	if len(data) > shmemBufferSize-4 {
		data = data[:shmemBufferSize-4]
	}
	idxPtr := (*uint64)(unsafe.Pointer(s.addr))
	cur := atomic.LoadUint64(idxPtr)
	next := cur ^ 1

	base := s.addr + uintptr(shmemHeaderSize) + uintptr(next)*uintptr(shmemBufferSize)
	buf := unsafe.Slice((*byte)(unsafe.Pointer(base)), shmemBufferSize)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(data)))
	copy(buf[4:], data)

	atomic.StoreUint64(idxPtr, next)
}

// Read — прочитать актуальный буфер.
func (s *SharedMem) Read() []byte {
	if s == nil || s.addr == 0 {
		return nil
	}
	idxPtr := (*uint64)(unsafe.Pointer(s.addr))
	idx := atomic.LoadUint64(idxPtr) & 1

	base := s.addr + uintptr(shmemHeaderSize) + uintptr(idx)*uintptr(shmemBufferSize)
	buf := unsafe.Slice((*byte)(unsafe.Pointer(base)), shmemBufferSize)
	n := binary.LittleEndian.Uint32(buf[0:4])
	if n > uint32(shmemBufferSize-4) {
		n = uint32(shmemBufferSize - 4)
	}
	out := make([]byte, n)
	copy(out, buf[4:4+n])
	return out
}

// writeSnapshot — собрать снимок ViewStar и записать в shared memory.
// Вызывать из SimTickTimed. Если Shm == nil — no-op.
func (w *World) writeSnapshot() {
	if w.Shm == nil {
		return
	}
	starID := w.ViewStar
	if starID < 0 || starID >= len(w.Stars) {
		return
	}
	// Раз в день (2048 тиков), а не каждый тик.
	if w.ShmDay == w.Day {
		return
	}
	w.ShmDay = w.Day

	snap := Snapshot{
		Day:     w.Day,
		Star:    starID,
		Objects: make([]SnapshotObject, 0, 32),
	}

	// Планеты системы — все.
	for _, p := range w.PlanetMap[starID] {
		x, y := w.getPlanetPos(p, w.Day)
		snap.Objects = append(snap.Objects, SnapshotObject{T: 1, X: x, Y: y})
	}

	// Корабли — только те, кто сейчас в полёте и в этой системе.
	for _, t := range w.Traders {
		if !w.traderInSystem(t, starID) {
			continue
		}
		x, y := t.GetFlightPoint()
		snap.Objects = append(snap.Objects, SnapshotObject{T: 2, X: x, Y: y})
	}

	data, err := json.Marshal(snap)
	if err != nil {
		return
	}
	w.Shm.Write(data)
}

// traderInSystem — летит ли трейдер сейчас в системе starID.
// Учитываем только те фазы, где Flight.Len > 0 (иначе координат нет).
func (w *World) traderInSystem(t *Trader, starID int) bool {
	if !t.Alive || t.Flight.Len <= 0 {
		return false
	}
	switch t.Phase {
	case TraderPhaseTakeoff, TraderPhaseCruise, TraderPhaseLanding, TraderPhaseToEdge:
		return t.FromPlanet >= 0 && w.Planets[t.FromPlanet].StarID == starID
	case TraderPhaseEdgeToEdge, TraderPhaseRefuelApproach, TraderPhaseRefuelLanding,
		TraderPhaseRefuelJump, TraderPhaseRefuelTakeoff, TraderPhasePlanetToEdge, TraderPhaseEdgeToPlanet:
		return t.RouteIdx >= 0 && t.RouteIdx < len(t.Route) && t.Route[t.RouteIdx] == starID
	}
	return false
}

// SysShip — корабль в системе для карты чата: где он и куда летит.
type SysShip struct {
	Kind  int     `json:"k"`  // 1 трейдер, 2 караван, 3 дипломат, 4 банкир, 5 пират
	X     float64 `json:"x"`  // текущая позиция
	Y     float64 `json:"y"`
	TX    float64 `json:"tx"` // цель (точка встречи Безье)
	TY    float64 `json:"ty"`
	To    int     `json:"to"` // планета-цель (-1 = нет)
	Phase int     `json:"ph"`
}

// caravanInSystem — караван сейчас в системе starID и его координаты валидны.
func (w *World) caravanInSystem(c *Caravan, starID int) bool {
	if !c.Alive || c.Flight.Len <= 0 {
		return false
	}
	switch c.Phase {
	case CaravanPhaseTakeoff, CaravanPhaseCruise, CaravanPhaseLanding,
		CaravanPhaseToEdge, CaravanPhasePlanetToEdge,
		CaravanPhaseRefuelApproach, CaravanPhaseRefuelLanding:
		return c.FromPlanet >= 0 && w.Planets[c.FromPlanet].StarID == starID
	case CaravanPhaseEdgeToPlanet:
		return c.ToPlanet >= 0 && w.Planets[c.ToPlanet].StarID == starID
	case CaravanPhaseEdgeToEdge:
		return c.RouteIdx >= 0 && c.RouteIdx < len(c.Route) && c.Route[c.RouteIdx] == starID
	}
	return false
}

// diplomatInSystem — то же для дипломатов/банкиров (свои константы фаз).
func (w *World) diplomatInSystem(d *Diplomat, starID int) bool {
	if !d.Alive || d.Flight.Len <= 0 {
		return false
	}
	switch d.Phase {
	case DiplomatPhaseTakeoff, DiplomatPhaseCruise, DiplomatPhaseLanding,
		DiplomatPhaseToEdge, DiplomatPhasePlanetToEdge,
		DiplomatPhaseRefuelApproach, DiplomatPhaseRefuelLanding:
		return d.FromPlanet >= 0 && w.Planets[d.FromPlanet].StarID == starID
	case DiplomatPhaseEdgeToPlanet:
		return d.ToPlanet >= 0 && w.Planets[d.ToPlanet].StarID == starID
	case DiplomatPhaseEdgeToEdge:
		return d.RouteIdx >= 0 && d.RouteIdx < len(d.Route) && d.Route[d.RouteIdx] == starID
	}
	return false
}

// MapShips — все корабли в системе для карты: позиция + цель.
func (w *World) MapShips(starID int) []SysShip {
	out := make([]SysShip, 0, 32)
	if starID < 0 || starID >= len(w.Stars) {
		return out
	}
	for _, t := range w.Traders {
		if !w.traderInSystem(t, starID) {
			continue
		}
		x, y := t.GetFlightPoint()
		out = append(out, SysShip{Kind: 1, X: x, Y: y, TX: t.Flight.Bx, TY: t.Flight.By, To: t.ToPlanet, Phase: t.Phase})
	}
	for _, c := range w.Caravans {
		if !w.caravanInSystem(c, starID) {
			continue
		}
		x, y := GetFlightPointCommon(&c.Flight, c.Progress)
		out = append(out, SysShip{Kind: 2, X: x, Y: y, TX: c.Flight.Bx, TY: c.Flight.By, To: c.ToPlanet, Phase: c.Phase})
	}
	for _, d := range w.Diplomats {
		if !w.diplomatInSystem(d, starID) {
			continue
		}
		k := 3
		if d.Role == DiplomatRoleBanker {
			k = 4
		}
		x, y := GetFlightPointCommon(&d.Flight, d.Progress)
		out = append(out, SysShip{Kind: k, X: x, Y: y, TX: d.Flight.Bx, TY: d.Flight.By, To: d.ToPlanet, Phase: d.Phase})
	}
	// Пираты: координат полёта в структуре нет — висят у логова.
	for _, p := range w.Pirates {
		if !p.Alive {
			continue
		}
		if p.BasePlanet < 0 || p.BasePlanet >= len(w.Planets) {
			continue
		}
		if w.Planets[p.BasePlanet].StarID != starID {
			continue
		}
		x, y := w.getPlanetPos(w.Planets[p.BasePlanet], w.Day)
		out = append(out, SysShip{Kind: 5, X: x, Y: y, TX: x, TY: y, To: p.BasePlanet, Phase: p.Phase})
	}
	return out
}