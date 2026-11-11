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