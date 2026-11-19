package sim

import (
	"math"
	"math/rand"
)

type Race int

const (
	RaceMaloc Race = iota
	RacePeleng
	RaceHuman
	RaceFeyan
	RaceGaal
	RaceCount
)

type Government int

const (
	GovAnarchy Government = iota
	GovDictatorship
	GovMonarchy
	GovRepublic
	GovDemocracy
	GovCount
)

type Economy int

const (
	EcoAgri Economy = iota
	EcoMixed
	EcoInd
	EcoCount
)

type OwnerId int

const (
	oiMaloc OwnerId = iota
	oiPeleng
	oiHuman
	oiFeyan
	oiGaal
	oiPirate
	oiUninhabited
	oiDominator
	OwnerCount
)

type Good int

const (
	GoodFood Good = iota
	GoodMedicine
	GoodTechnics
	GoodLuxury
	GoodMinerals
	GoodAlcohol
	GoodArms
	GoodNarcotics
	GoodCount
)

type MarketEntry struct {
	PriceState    float64
	PurchasePrice float64
	BaseSalePrice float64
	Count         int
	TargetStock   int
	TargetPrice   float64
	Delta         float64 // prod - cons за последний тик (для веба)
}

type Star struct {
	ID           int
	Col, Row     int
	X, Y         float64
	SystemRadius float64
	Owner        OwnerId
	Sector       int // 0..80 (9×9), считается в NewWorld по SectorOf
}

type SectorSeed struct {
	X, Y float64
}

type Planet struct {
	ID                 int
	StarID             int
	Orbit              float64
	Race               Race
	OwnerId            OwnerId
	Pop                int
	PopRef             int // стартовая Pop, точка отсчёта для PopFactor (C2)
	Habitable          bool
	Government         Government
	Economy            Economy
	Radius             float64
	TechLevel          int // 0..15, hex 0x00..0x0F. У необитаемых = 0.
	Market             [GoodCount]MarketEntry
	BaseStock          [GoodCount]int
	Money              float64
	IsMainPiratePlanet bool
}

type World struct {
	Day       int
	Rng       *rand.Rand
	Stars     []*Star
	Planets   []*Planet
	PlanetMap map[int][]*Planet
	Neighbors map[int][]int
	DistMatrix [][]float64
	SectorSeeds     []SectorSeed // 81 семя (9×9), генерация в NewWorld
	SectorNeighbors [][]int      // 81, соседи по расстоянию семян (порог 60)
	ConstellationEdges [][]Edge // [81] сектор → рёбра MST (звёзды)
	// B: кэш на тик (пересчитывается в начале phaseMovement)
	tickBuyPrice   [][]float64 // [planetID][good]
	tickSellPrice  [][]float64
	tickLegal      [][]bool
	tickCanTrade   [][RaceCount]bool // [planetID][race]
	// C1: переиспользуемый буфер кандидатов для SelectBestTradePlanet
	tickCandidates []*Planet
	Caravans  []*Caravan
	Pirates   []*Pirate
	Liners    []*Liner
	Diplomats []*Diplomat

	Events []Event
	News   []Event

	Rel     Relations
	BaseRel Relations

	HealthEffectNewMolizone bool

	DeathsFuel  int
	DeathsMoney int
	DeathsOther int
}

type Event struct {
	Day  int    `json:"day"`
	Kind string `json:"kind"`
	Text string `json:"text"`
	Star int    `json:"star"`
	Good Good   `json:"good"`
}

type Relations [RaceCount][RaceCount]int

const (
	GridCols        = 20
	GridRows        = 15
	MapW            = 400.0
	MapH            = 280.0
	BaseStockPerPop = 1000
	MaxNews         = 1000
)

func NewWorld(seed int64) *World {
	rng := rand.New(rand.NewSource(seed))
	w := &World{
		Rng:       rng,
		PlanetMap: make(map[int][]*Planet),
	}

	starID := 0
	starCount := GridCols * GridRows // 300
	const (
		margin     = 5.0
		minDist    = 8.0
		maxRetries = 200
	)

	type xy struct{ x, y float64 }
	placed := make([]xy, 0, starCount)

	for i := 0; i < starCount; i++ {
		var x, y float64
		ok := false
		for attempt := 0; attempt < maxRetries; attempt++ {
			x = margin + rng.Float64()*(MapW-2*margin)
			y = margin + rng.Float64()*(MapH-2*margin)
			ok = true
			for _, p := range placed {
				dx := x - p.x
				dy := y - p.y
				if dx*dx+dy*dy < minDist*minDist {
					ok = false
					break
				}
			}
			if ok {
				break
			}
		}
		if !ok {
			panic("NewWorld: cannot place star with minDist=8 after maxRetries")
		}
		placed = append(placed, xy{x, y})

		nPlanets := 4 + rng.Intn(6)
		s := &Star{
			ID:           i,
			Col:          int(x / (MapW / float64(GridCols))),
			Row:          int(y / (MapH / float64(GridRows))),
			X:            x,
			Y:            y,
			SystemRadius: 1 + float64(nPlanets-1)*0.8 + 1,
			Owner:        OwnerId(rng.Intn(int(RaceCount))),
		}
		w.Stars = append(w.Stars, s)
		starID++

		for j := 0; j < nPlanets; j++ {
			orbit := 1.0 + float64(j)*0.8
			habitable := j == 3 || rng.Intn(100) < 60
			p := &Planet{
				ID:        len(w.Planets),
				StarID:    s.ID,
				Orbit:     orbit,
				Habitable: habitable,
				Radius:    60 + rng.Float64()*40, // 60..100
			}
			if habitable {
				p.Race = Race(rng.Intn(int(RaceCount)))
				p.OwnerId = OwnerId(p.Race)
				p.Pop = 100000 + rng.Intn(900000)
				p.PopRef = p.Pop // C2: точка отсчёта для PopFactor
				p.Money = float64(p.Pop) * 0.25
				p.Government = Government(rng.Intn(int(GovCount)))
				p.Economy = Economy(rng.Intn(int(EcoCount)))
				p.TechLevel = 3 // старт: у всех обитаемых 3 (0x03). Растёт со временем — потом.
			} else {
				p.Race = RaceMaloc // заполнитель
				p.OwnerId = oiUninhabited
				p.Pop = 0
				p.Economy = EcoMixed
			}
			for g := 0; g < int(GoodCount); g++ {
				base := 0
				if habitable {
					base = p.Pop / BaseStockPerPop
					if base < 10 {
						base = 10
					}
				}
				p.BaseStock[g] = base
				p.Market[g] = MarketEntry{
					PriceState:    float64(GoodsMarket[g].AveragePrice),
					PurchasePrice: float64(GoodsMarket[g].AveragePrice),
					BaseSalePrice: float64(GoodsMarket[g].AveragePrice) / 2,
					Count:         base,
					TargetStock:   base,
					TargetPrice:   float64(GoodsMarket[g].AveragePrice),
				}
			}
			w.Planets = append(w.Planets, p)
			w.PlanetMap[s.ID] = append(w.PlanetMap[s.ID], p)
		}
	}
	
	// Сектора: 81 семя (9×9, джиттер ±20%), соседи по расстоянию, Star.Sector.
	w.genSectorSeeds()
	w.buildSectorNeighbors()
	for _, s := range w.Stars {
		s.Sector = w.SectorOf(s.X, s.Y)
	}
	
	for a := 0; a < int(RaceCount); a++ {
		for b := 0; b < int(RaceCount); b++ {
			var v int
			if a == b {
				v = 100
			} else {
				v = 40 + rng.Intn(31)
			}
			w.Rel[a][b] = v
			w.BaseRel[a][b] = v
		}
	}

	w.Neighbors = make(map[int][]int, len(w.Stars))
	const neighborRange = 35.0
	for i := range w.Stars {
		nb := []int{}
		for j := range w.Stars {
			if i == j {
				continue
			}
			if w.Dist(i, j) <= neighborRange {
				nb = append(nb, j)
			}
		}
		w.Neighbors[i] = nb
	}
	
	// A: кэш расстояний между звёздами
	n := len(w.Stars)
	w.DistMatrix = make([][]float64, n)
	for i := 0; i < n; i++ {
		w.DistMatrix[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			dx := w.Stars[i].X - w.Stars[j].X
			dy := w.Stars[i].Y - w.Stars[j].Y
			w.DistMatrix[i][j] = math.Sqrt(dx*dx + dy*dy)
		}
	}	

	w.buildConstellations()
	w.reassignOrphans() // <-- добавляем сюда

	return w
}

func (w *World) Dist(a, b int) float64 {
	if w.DistMatrix != nil && a < len(w.DistMatrix) && b < len(w.DistMatrix[a]) {
		return w.DistMatrix[a][b]
	}
	dx := w.Stars[a].X - w.Stars[b].X
	dy := w.Stars[a].Y - w.Stars[b].Y
	return math.Sqrt(dx*dx + dy*dy)
}

func (w *World) IsLegal(p *Planet, g Good) bool {
	if p == nil || p.IsMainPiratePlanet {
		return true
	}
	if p.OwnerId == oiPirate {
		return true
	}
	if !GoodsLegalOnPlanet[g][p.Race][p.Government] {
		return false
	}
	if (g == GoodFood || g == GoodMedicine) && w.HealthEffectNewMolizone {
		return false
	}
	return true
}