package sim

import "math"

// ---------- Общие функции полёта ----------
// Используются караванами (сейчас), трейдерами (потом).
// Принимают указатели на Flight/Progress и speed — без привязки к типу агента.

// PrepareFlightCommon — Безье от fromP к toP с предугадыванием финиша.
func (w *World) PrepareFlightCommon(flight *FlightPath, progress *float64, speed float64, fromP, toP *Planet) {
	ax, ay := w.getPlanetPos(fromP, w.Day)
	bx, by := w.getPlanetPos(toP, w.Day)
	dist := math.Hypot(bx-ax, by-ay)
	for i := 0; i < 3; i++ {
		eta := dist / speed
		bx, by = w.getPlanetPos(toP, w.Day+int(eta))
		dist = math.Hypot(bx-ax, by-ay)
	}
	w.prepareBezierCommon(flight, progress, ax, ay, bx, by)
}

// PrepareFlightFromPointCommon — Безье от точки (ax,ay) к планете toP.
func (w *World) PrepareFlightFromPointCommon(flight *FlightPath, progress *float64, speed float64, ax, ay float64, toP *Planet) {
	bx, by := w.getPlanetPos(toP, w.Day)
	dist := math.Hypot(bx-ax, by-ay)
	for i := 0; i < 3; i++ {
		eta := dist / speed
		bx, by = w.getPlanetPos(toP, w.Day+int(eta))
		dist = math.Hypot(bx-ax, by-ay)
	}
	w.prepareBezierCommon(flight, progress, ax, ay, bx, by)
}

// PrepareJumpSegmentCommon — Безье между двумя статичными точками.
func (w *World) PrepareJumpSegmentCommon(flight *FlightPath, progress *float64, ax, ay, bx, by float64) {
	w.prepareBezierCommon(flight, progress, ax, ay, bx, by)
}

// PrepareJumpFlightCommon — Безье от планеты fromP к точке (exitX,exitY).
func (w *World) PrepareJumpFlightCommon(flight *FlightPath, progress *float64, fromP *Planet, exitX, exitY float64) {
	ax, ay := w.getPlanetPos(fromP, w.Day)
	w.prepareBezierCommon(flight, progress, ax, ay, exitX, exitY)
}

// prepareBezierCommon — общая сборка Безье (контрольная точка, длина).
func (w *World) prepareBezierCommon(flight *FlightPath, progress *float64, ax, ay, bx, by float64) {
	midX := (ax + bx) / 2
	midY := (ay + by) / 2
	dx := bx - ax
	dy := by - ay
	lenAB := math.Hypot(dx, dy)

	if lenAB < 0.01 {
		*flight = FlightPath{Ax: ax, Ay: ay, Cx: ax, Cy: ay, Bx: bx, By: by, Len: 0}
		*progress = 0
		return
	}

	nx := -dy / lenAB
	ny := dx / lenAB
	if midX*nx+midY*ny < 0 {
		nx = -nx
		ny = -ny
	}
	offset := RSafe + lenAB*0.25
	cx := midX + nx*offset
	cy := midY + ny*offset

	*flight = FlightPath{
		Ax: ax, Ay: ay,
		Cx: cx, Cy: cy,
		Bx: bx, By: by,
		Len: bezierLength(ax, ay, cx, cy, bx, by, 20),
	}
	*progress = 0
}

// AdvanceFlightCommon — продвинуть по кривой. true = долетел.
func AdvanceFlightCommon(flight *FlightPath, progress *float64, speed float64) bool {
	if flight.Len <= 0.01 {
		return true
	}
	*progress += speed / flight.Len
	if *progress >= 1.0 {
		*progress = 1.0
		return true
	}
	return false
}

// GetFlightPointCommon — текущие координаты на кривой.
func GetFlightPointCommon(flight *FlightPath, progress float64) (float64, float64) {
	if flight.Len <= 0.01 {
		return flight.Bx, flight.By
	}
	return bezierPoint(flight.Ax, flight.Ay, flight.Cx, flight.Cy, flight.Bx, flight.By, progress)
}