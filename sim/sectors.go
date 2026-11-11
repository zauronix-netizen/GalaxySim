package sim

import "math"

// SectorGridN — было 9×9 = 81. Теперь 10×6 = 60.
const (
	SectorGridCols = 9
	SectorGridRows = 8
	SectorCount    = SectorGridCols * SectorGridRows // 60
	SectorJitter   = 0.20
	SectorNeighR   = 60.0
)

const MstMaxEdge = 35.0 // тест: порог ребра MST (прыжок ≤35)

// genSectorSeeds — 81 семя: сетка 9×9 по полю MapW×MapH, с джиттером ±20% шага.
func (w *World) genSectorSeeds() {
	stepX := MapW / float64(SectorGridCols)
	stepY := MapH / float64(SectorGridRows)
	jitX := stepX * SectorJitter
	jitY := stepY * SectorJitter

	w.SectorSeeds = make([]SectorSeed, 0, SectorCount)
	for j := 0; j < SectorGridRows; j++ {
		for i := 0; i < SectorGridCols; i++ {
			x := (float64(i)+0.5)*stepX + (w.Rng.Float64()*2-1)*jitX
			y := (float64(j)+0.5)*stepY + (w.Rng.Float64()*2-1)*jitY
			w.SectorSeeds = append(w.SectorSeeds, SectorSeed{X: x, Y: y})
		}
	}
}

// SectorOf — индекс ближайшего семени (0..80). O(81).
func (w *World) SectorOf(x, y float64) int {
	best := 0
	bestD := math.MaxFloat64
	for i, s := range w.SectorSeeds {
		dx := x - s.X
		dy := y - s.Y
		d := dx*dx + dy*dy
		if d < bestD {
			bestD = d
			best = i
		}
	}
	return best
}

// buildSectorNeighbors — соседи по расстоянию семян ≤ SectorNeighR. Симметрично.
func (w *World) buildSectorNeighbors() {
	n := len(w.SectorSeeds)
	w.SectorNeighbors = make([][]int, n)
	for i := 0; i < n; i++ {
		nb := []int{}
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			dx := w.SectorSeeds[i].X - w.SectorSeeds[j].X
			dy := w.SectorSeeds[i].Y - w.SectorSeeds[j].Y
			if dx*dx+dy*dy <= SectorNeighR*SectorNeighR {
				nb = append(nb, j)
			}
		}
		w.SectorNeighbors[i] = nb
	}
}

// Edge — ребро созвездия (star IDs).
type Edge struct {
	A, B int
}

// buildConstellations — для каждого сектора строит MST (Прим) по DistMatrix.
// Звёзды группируются по Star.Sector. O(n²) на шаг, n ≤ ~10.
// ВАЖНО: вызывать ПОСЛЕ заполнения DistMatrix в NewWorld.
func (w *World) buildConstellations() {
	sectorStars := make([][]int, SectorCount)
	for _, s := range w.Stars {
		if s.Sector < 0 || s.Sector >= SectorCount {
			continue
		}
		sectorStars[s.Sector] = append(sectorStars[s.Sector], s.ID)
	}

	w.ConstellationEdges = make([][]Edge, SectorCount)
	for sec := 0; sec < SectorCount; sec++ {
		w.ConstellationEdges[sec] = mstSector(sectorStars[sec], w.DistMatrix)
	}
}

func (w *World) buildConstellationIDs() {
	n := len(w.Stars)
	visited := make([]bool, n)
	w.ConstellationStars = w.ConstellationStars[:0]

	// adjacency по рёбрам MST (из ConstellationEdges)
	adj := make([][]int, n)
	for _, edges := range w.ConstellationEdges {
		for _, e := range edges {
			adj[e.A] = append(adj[e.A], e.B)
			adj[e.B] = append(adj[e.B], e.A)
		}
	}

	for id := 0; id < n; id++ {
		if visited[id] {
			continue
		}
		comp := []int{}
		queue := []int{id}
		visited[id] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			comp = append(comp, cur)
			w.Stars[cur].ConstellationID = len(w.ConstellationStars)
			for _, nb := range adj[cur] {
				if !visited[nb] {
					visited[nb] = true
					queue = append(queue, nb)
				}
			}
		}
		w.ConstellationStars = append(w.ConstellationStars, comp)
	}
}

// mstSector — MST по Приму. Пусто, если ids < 2.
// Tie-break по ID при равных расстояниях — детерминизм.
func mstSector(ids []int, dist [][]float64) []Edge {
	n := len(ids)
	if n < 2 {
		return nil
	}
	inTree := make([]bool, n)
	inTree[0] = true
	edges := make([]Edge, 0, n-1)
	for len(edges) < n-1 {
		bestI, bestJ := -1, -1
		bestD := math.MaxFloat64
		for i := 0; i < n; i++ {
			if !inTree[i] {
				continue
			}
			for j := 0; j < n; j++ {
				if inTree[j] {
					continue
				}
				d := dist[ids[i]][ids[j]]
				if d > MstMaxEdge {
					continue
				}
				if d < bestD || (d == bestD && (bestI == -1 || ids[i] < ids[bestI] || (ids[i] == ids[bestI] && ids[j] < ids[bestJ]))) {
					bestD = d
					bestI, bestJ = i, j
				}
			}
		}
		if bestI == -1 {
			break
		}
		inTree[bestJ] = true
		edges = append(edges, Edge{A: ids[bestI], B: ids[bestJ]})
	}
	return edges
}

// reassignOrphans — переносит звёзд-сирот (где ConstellationEdges пусто)
// в сектор ближайшей звезды, у которой есть созвездие.
// Вызывать ПОСЛЕ buildConstellations. В конце сам вызывает buildConstellations.
func (w *World) reassignOrphans() {
	// Ищем ближайшую «не-сироту» для каждой сироты
	changed := false
	marked := false
	for _, s := range w.Stars {
		if len(w.ConstellationEdges[s.Sector]) > 0 {
			continue
		}
		if !marked {
			s.IsOrphan = true
			marked = true
			continue
		}
    // ... остальные сироты — как раньше (поиск ближайшей, перенос)

		// Ищем ближайшую звезду, у которой есть созвездие
		bestID := -1
		bestD := math.MaxFloat64
		for _, t := range w.Stars {
			if t.ID == s.ID {
				continue
			}
			// Нам нужна звезда, которая уже в созвездии
			if len(w.ConstellationEdges[t.Sector]) == 0 {
				continue
			}
			d := w.DistMatrix[s.ID][t.ID]
			if d < bestD {
				bestD = d
				bestID = t.ID
			}
		}

		// Если нашли — переносим
		if bestID >= 0 {
			s.Sector = w.Stars[bestID].Sector
			changed = true
		}
	}

	// Если кого-то перенесли — пересобираем MST
	if changed {
		w.buildConstellations()
	}
}

