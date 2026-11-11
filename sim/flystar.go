package sim

import "math"

const (
	OrbitK = 0.05
	RSafe  = 0.3
)

// FlightPath хранит координаты для кривой Безье.
type FlightPathBak struct {
	Ax, Ay float64 // Старт
	Cx, Cy float64 // Контрольная точка (обход)
	Bx, By float64 // Финиш (предугаданный)
	Len    float64 // Длина кривой (20 отрезков)
}

// getPlanetPos возвращает мировые координаты планеты на текущий день.
func (w *World) getPlanetPos(p *Planet, day int) (float64, float64) {
	r := p.Orbit
	baseAng := float64(p.ID)*137.508*math.Pi/180 + float64(p.StarID)*37.0*math.Pi/180
	speed := OrbitK / p.Orbit
	ang := baseAng + speed*float64(day)
	return r * math.Cos(ang), r * math.Sin(ang)
}

// bezierPoint вычисляет координаты (x, y) на кривой Безье для параметра t (0..1).
func bezierPoint(ax, ay, cx, cy, bx, by, t float64) (float64, float64) {
	mt := 1.0 - t
	x := mt*mt*ax + 2*mt*t*cx + t*t*bx
	y := mt*mt*ay + 2*mt*t*cy + t*t*by
	return x, y
}

// bezierLength считает длину кривой, разбивая её на segments частей.
func bezierLength(ax, ay, cx, cy, bx, by float64, segments int) float64 {
	length := 0.0
	prevX, prevY := ax, ay
	for i := 1; i <= segments; i++ {
		t := float64(i) / float64(segments)
		x, y := bezierPoint(ax, ay, cx, cy, bx, by, t)
		dx := x - prevX
		dy := y - prevY
		length += math.Sqrt(dx*dx + dy*dy)
		prevX, prevY = x, y
	}
	return length
}

// PrepareFlight рассчитывает траекторию Безье с предугадыванием точки встречи.
func (w *World) PrepareFlight(t *Trader, fromP *Planet, toP *Planet) {
	// 1. Старт (текущая позиция)
	ax, ay := w.getPlanetPos(fromP, w.Day)

	// 2. Предугадывание финиша (простая итерация)
	bx, by := w.getPlanetPos(toP, w.Day)
	dist := math.Hypot(bx-ax, by-ay)
	for i := 0; i < 3; i++ {
		eta := dist / t.Speed // сколько тиков лететь
		bx, by = w.getPlanetPos(toP, w.Day+int(eta))
		dist = math.Hypot(bx-ax, by-ay)
	}

	// 3. Поиск управляющей точки (обход звезды)
	midX := (ax + bx) / 2
	midY := (ay + by) / 2
	dx := bx - ax
	dy := by - ay
	lenAB := math.Hypot(dx, dy)
	
	if lenAB < 0.01 {
		// Защита от деления на ноль (если планеты в одной точке)
		t.Flight = FlightPath{Ax: ax, Ay: ay, Cx: ax, Cy: ay, Bx: bx, By: by, Len: 0}
		t.Progress = 0
		return
	}

	// Нормаль к отрезку AB
	nx := -dy / lenAB
	ny := dx / lenAB
	
	// Проверяем, куда смотрит нормаль. Если она смотрит в центр (0,0), инвертируем.
	if midX*nx + midY*ny < 0 {
		nx = -nx
		ny = -ny
	}
	
	// Отодвигаем на RSafe + запас
	offset := RSafe + lenAB*0.25
	cx := midX + nx*offset
	cy := midY + ny*offset

	// 4. Сохраняем
	t.Flight = FlightPath{
		Ax: ax, Ay: ay,
		Cx: cx, Cy: cy,
		Bx: bx, By: by,
		Len: bezierLength(ax, ay, cx, cy, bx, by, 20),
	}
	t.Progress = 0
}

// AdvanceFlight продвигает корабль по кривой. Возвращает true, если долетел.
func (t *Trader) AdvanceFlight() bool {
	if t.Flight.Len <= 0.01 {
		return true
	}
	t.Progress += t.Speed / t.Flight.Len
	if t.Progress >= 1.0 {
		t.Progress = 1.0
		return true
	}
	return false
}

// GetFlightPoint возвращает текущие координаты корабля (для отрисовки в вебе).
func (t *Trader) GetFlightPoint() (float64, float64) {
	if t.Flight.Len <= 0.01 {
		return t.Flight.Bx, t.Flight.By
	}
	return bezierPoint(
		t.Flight.Ax, t.Flight.Ay,
		t.Flight.Cx, t.Flight.Cy,
		t.Flight.Bx, t.Flight.By,
		t.Progress,
	)
}

/*

Exit = выход из системы (в гипер) = вход в прыжок.
Entry = вход в систему (из гипера) = выход из прыжка.

*/

// PrepareJumpSegment строит Безье от точки (ax,ay) к точке (bx,by).
// Без предсказания — обе точки статичны (края систем).
func (w *World) PrepareJumpSegment(t *Trader, ax, ay, bx, by float64) {
	midX := (ax + bx) / 2
	midY := (ay + by) / 2
	dx := bx - ax
	dy := by - ay
	lenAB := math.Hypot(dx, dy)

	if lenAB < 0.01 {
		t.Flight = FlightPath{Ax: ax, Ay: ay, Cx: ax, Cy: ay, Bx: bx, By: by, Len: 0}
		t.Progress = 0
		return
	}

	// Нормаль к отрезку AB
	nx := -dy / lenAB
	ny := dx / lenAB

	// Проверяем, куда смотрит нормаль (от центра или к центру)
	if midX*nx + midY*ny < 0 {
		nx = -nx
		ny = -ny
	}

	// Отодвигаем на RSafe + запас
	offset := RSafe + lenAB*0.25
	cx := midX + nx*offset
	cy := midY + ny*offset

	t.Flight = FlightPath{
		Ax: ax, Ay: ay,
		Cx: cx, Cy: cy,
		Bx: bx, By: by,
		Len: bezierLength(ax, ay, cx, cy, bx, by, 20),
	}
	t.Progress = 0
}

// PrepareFlightFromPoint строит Безье от точки (ax,ay) к планете toP.
// С предсказанием: планета движется (3 итерации eta = dist/Speed).
func (w *World) PrepareFlightFromPoint(t *Trader, ax, ay float64, toP *Planet) {
	// 1. Финиш (предугаданная позиция планеты)
	bx, by := w.getPlanetPos(toP, w.Day)
	dist := math.Hypot(bx-ax, by-ay)
	for i := 0; i < 3; i++ {
		eta := dist / t.Speed
		bx, by = w.getPlanetPos(toP, w.Day+int(eta))
		dist = math.Hypot(bx-ax, by-ay)
	}

	// 2. Контрольная точка (обход звезды)
	midX := (ax + bx) / 2
	midY := (ay + by) / 2
	dx := bx - ax
	dy := by - ay
	lenAB := math.Hypot(dx, dy)

	if lenAB < 0.01 {
		t.Flight = FlightPath{Ax: ax, Ay: ay, Cx: ax, Cy: ay, Bx: bx, By: by, Len: 0}
		t.Progress = 0
		return
	}

	nx := -dy / lenAB
	ny := dx / lenAB
	if midX*nx + midY*ny < 0 {
		nx = -nx
		ny = -ny
	}

	offset := RSafe + lenAB*0.25
	cx := midX + nx*offset
	cy := midY + ny*offset

	t.Flight = FlightPath{
		Ax: ax, Ay: ay,
		Cx: cx, Cy: cy,
		Bx: bx, By: by,
		Len: bezierLength(ax, ay, cx, cy, bx, by, 20),
	}
	t.Progress = 0
}


// getJumpExitPoint возвращает координаты для входа в прыжок.
// Направление — на целевую звезду. Радиус — орбита последней планеты + 2.0.
func (w *World) getJumpExitPoint(fromStarID int, toStarID int) (float64, float64) {
	fromStar := w.Stars[fromStarID]
	toStar := w.Stars[toStarID]
	
	// 1. Находим последнюю (внешнюю) орбиту в системе
	maxOrbit := 1.0
	for _, p := range w.PlanetMap[fromStarID] {
		if p.Orbit > maxOrbit {
			maxOrbit = p.Orbit
		}
	}
	
	// 2. Радиус для прыжка (отступ +2 от последней планеты)
	jumpRadius := maxOrbit + 2.0
	
	// 3. Направление на целевую звезду
	dx := toStar.X - fromStar.X
	dy := toStar.Y - fromStar.Y
	dist := math.Hypot(dx, dy)
	
	if dist < 0.01 {
		return 0, 0
	}
	
	// 4. Итоговые координаты
	exitX := (dx / dist) * jumpRadius
	exitY := (dy / dist) * jumpRadius
	
	return exitX, exitY
}

// getJumpEntryPoint возвращает координаты для выхода из прыжка.
// Симметрично getJumpExitPoint: направление — ОТ предыдущей звезды.
// Радиус — орбита последней планеты + 2.0.
func (w *World) getJumpEntryPoint(starID int, fromStarID int) (float64, float64) {
	star := w.Stars[starID]
	fromStar := w.Stars[fromStarID]

	// 1. Находим последнюю (внешнюю) орбиту в системе
	maxOrbit := 1.0
	for _, p := range w.PlanetMap[starID] {
		if p.Orbit > maxOrbit {
			maxOrbit = p.Orbit
		}
	}

	// 2. Радиус для прыжка (отступ +2 от последней планеты)
	jumpRadius := maxOrbit + 2.0

	// 3. Направление ОТ предыдущей звезды (обратное входу)
	dx := star.X - fromStar.X
	dy := star.Y - fromStar.Y
	dist := math.Hypot(dx, dy)

	if dist < 0.01 {
		return 0, 0
	}

	// 4. Итоговые координаты
	entryX := (dx / dist) * jumpRadius
	entryY := (dy / dist) * jumpRadius

	return entryX, entryY
}

// PrepareJumpFlight строит маршрут к точке входа в прыжок (без предугадывания).
func (w *World) PrepareJumpFlight(t *Trader, fromP *Planet, exitX, exitY float64) {
	// 1. Старт (текущая позиция планеты)
	ax, ay := w.getPlanetPos(fromP, w.Day)

	// 2. Финиш (статичная точка на краю системы)
	bx, by := exitX, exitY

	// 3. Поиск управляющей точки (обход звезды)
	midX := (ax + bx) / 2
	midY := (ay + by) / 2
	dx := bx - ax
	dy := by - ay
	lenAB := math.Hypot(dx, dy)
	
	if lenAB < 0.01 {
		t.Flight = FlightPath{Ax: ax, Ay: ay, Cx: ax, Cy: ay, Bx: bx, By: by, Len: 0}
		t.Progress = 0
		return
	}

	// Нормаль к отрезку AB
	nx := -dy / lenAB
	ny := dx / lenAB
	
	// Проверяем, куда смотрит нормаль (от центра или к центру)
	if midX*nx + midY*ny < 0 {
		nx = -nx
		ny = -ny
	}
	
	// Отодвигаем на RSafe + запас
	offset := RSafe + lenAB*0.25
	cx := midX + nx*offset
	cy := midY + ny*offset

	// 4. Сохраняем (без предугадывания)
	t.Flight = FlightPath{
		Ax: ax, Ay: ay,
		Cx: cx, Cy: cy,
		Bx: bx, By: by,
		Len: bezierLength(ax, ay, cx, cy, bx, by, 20),
	}
	t.Progress = 0
}