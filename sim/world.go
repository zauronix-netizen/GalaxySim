package sim

import (
	"math"
	"math/rand"
)

type Race int

// Зоны, куда звёзды не ставим (резерв карты).
type NoStarRect struct{ X0, Y0, X1, Y1 float64 }

var NoStarZones = []NoStarRect{
	{5, 235, 55, 275}, // левый низ
	{345, 5, 395, 45}, // правый верх
	{175, 5, 225, 45}, // верх середина
}

func inNoStarZone(x, y float64) bool {
	for _, z := range NoStarZones {
		if x >= z.X0 && x <= z.X1 && y >= z.Y0 && y <= z.Y1 {
			return true
		}
	}
	return false
}

const (
	RaceMaloc Race = iota
	RacePeleng
	RaceHuman
	RaceFeyan
	RaceGaal
	RaceCount
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
	GoodFood           Good = iota // 0
	GoodTextiles                   // 1
	GoodRadioactives               // 2
	GoodSlaves                     // 3
	GoodLiquorWines                // 4
	GoodLuxuries                   // 5
	GoodNarcotics                  // 6
	GoodComputers                  // 7
	GoodMachinery                  // 8
	GoodAlloys                     // 9
	GoodFirearms                   // 10
	GoodFurs                       // 11
	GoodMinerals                   // 12
	GoodGold                       // 13
	GoodPlatinum                   // 14
	GoodGemStones                  // 15
	GoodAlienItems                 // 16
	GoodFerrousMetals              // 17
	GoodTimber                     // 18
	GoodAirProcessors              // 19
	GoodAnimalMeat                 // 20
	GoodAnimalSkins                // 21
	GoodBattleWeapons              // 22
	GoodFarmMachinery              // 23
	GoodFertilizer                 // 24
	GoodGrain                      // 25
	GoodDrone                      // 26
	GoodHeavyPlastics              // 27
	GoodHydrogenFuel               // 28
	GoodIndustrialParts            // 29
	GoodLiquidOxygen               // 30
	GoodLiveAnimals                // 31
	GoodMedicines                  // 32
	GoodMilitaryFuel               // 33
	GoodNerveGas                   // 34
	GoodRobots                     // 35
	GoodSyntheticMeat              // 36
	GoodWater                      // 37
	GoodCount
)

type Economy int

const (
	EcoAgriPoor Economy = iota // 0
	EcoAgriAvg                 // 1
	EcoAgriRich                // 2
	EcoMixedPoor               // 3
	EcoMixedAvg                // 4
	EcoMixedRich               // 5
	EcoIndPoor                 // 6
	EcoIndAvg                  // 7
	EcoIndRich                 // 8
	EcoCount
)

type Government int

const (
	GovAnarchy Government = iota // 0
	GovFeudal                    // 1
	GovMultiGovernment           // 2
	GovDictatorship              // 3
	GovCommunist                 // 4
	GovConfederacy               // 5
	GovDemocracy                 // 6
	GovCorporateState            // 7
	GovMonarchy                  // 8
	GovRepublic                  // 9
	GovCount
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
	ID           	int
	Col, Row     	int
	X, Y         	float64
	SystemRadius 	float64
	Owner        	OwnerId
	IsOrphan 	 	bool // помечена как сирота, не переносится reassignOrphans
	Sector      	int // 0..80 (9×9), считается в NewWorld по SectorOf
	ConstellationID int // номер MST-компоненты (0..K-1)
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
	MoneyRef 		   float64 // стартовый Money (копия при генерации)
	MoneySafe		   float64 // сейф планеты <----- планетарный сейф для дипломата - банкира
	IsMainPiratePlanet bool
	FullFoodDay        int // подряд тиков еда >= 90% TargetStock (сытость)
	HungryDay          int // подряд тиков еда < 10% TargetStock (голод)
	FullMedDay 		   int // подряд тиков Medicine >= 90% TargetStock (здоровье)
	SickDay   		   int // подряд тиков Medicine < 10% TargetStock (болезни)
	SurplusDay         [GoodCount]int // внутреннее: подряд тиков Count max
	DeficitDay         [GoodCount]int // внутреннее: подряд тиков Count zero
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
	ConstellationStars [][]int // [componentID] → []starID
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
	BankersByRace [RaceCount]int // сколько банкиров на расу (для капа 15)
	LastGTL int // последний GTL, при котором спавнили банкиров

	Stations []*Station
	
	Traders []*Trader
	TradersSpawnedThisTick int // счётчик для лимита

	RandomPlanetOrder []int // перемешанные ID обитаемых планет

	Events []Event
	News   []Event
	TickEvents []TickEvent

	Chat *ChatStore
	Shm      *SharedMem // разделяемая память для viewer.exe (nil — выключено)
	ViewStar int        // какая звезда пишется в снимок (для viewer)

	Rel     Relations
	BaseRel Relations

	HealthEffectNewMolizone bool

	GalaxyTechLevel          int

	DeathsFuel  int
	DeathsMoney int
	DeathsPirate int
	LootedCargo  int // число ограблений (пираты)
	DeathsOther int
	
	TickOverrideMs int // 0 = авто (tickInterval), иначе фикс
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
	starCount := 600  // 300 GridCols * GridRows
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
			if inNoStarZone(x, y) {
				ok = false
				continue
			}
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
			habitable := j == 3 || (j >= 2 && j <= 5 && rng.Intn(100) < 60)
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
				p.MoneyRef = p.Money
				p.Government = Government(rng.Intn(int(GovCount)))
				p.Economy = Economy(rng.Intn(int(EcoCount)))
				p.TechLevel = 2 // старт: у всех обитаемых 2 (0x02). Растёт со временем.
			} else {
				p.Race = Race(rng.Intn(int(RaceCount))) // случайная раса (пригодится позже)
				p.OwnerId = oiUninhabited
				p.Pop = 0
				p.Economy = EcoMixedAvg
				// Логово пирата: необитаемая на дальнем краю (j=4..6, orbit 4.2..5.8).
				if p.Orbit >= 4.2 && p.Orbit <= 5.8 {
					p.OwnerId = oiPirate
				}
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
					Count:         0, // пустой склад на старте
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
	// w.reassignOrphans() // <-- добавляем сюда
	w.buildConstellationIDs()
	// Создаем перемешанный порядок обитаемых планет для равномерного спавна/экономики
	w.RandomPlanetOrder = make([]int, 0, len(w.Planets))
	for _, p := range w.Planets {
		if p.Habitable {
			w.RandomPlanetOrder = append(w.RandomPlanetOrder, p.ID)
		}
	}
	w.Rng.Shuffle(len(w.RandomPlanetOrder), func(i, j int) {
		w.RandomPlanetOrder[i], w.RandomPlanetOrder[j] = w.RandomPlanetOrder[j], w.RandomPlanetOrder[i]
	})
	// Спавним по одному трейдеру на 4-й планете каждой звезды
	for _, star := range w.Stars {
		planets := w.PlanetMap[star.ID]
		if len(planets) > 3 && planets[3].Habitable {
			w.SpawnTrader(planets[3].ID, GoodFood)
		}
	}
	w.Chat = NewChatStore(100)
	w.GalaxyTechLevel = 3
	w.Shm = nil     // включается в main.go, если нужен viewer
	w.ViewStar = 0  // потом заменим на случайную/из веба

	w.BankersByRace = [RaceCount]int{}
	w.LastGTL = w.GalaxyTechLevel

	// Станции игрока — 72, по одной на сектор. Пустые (Balance=0).
	w.Stations = make([]*Station, SectorCount)
	for i := 0; i < SectorCount; i++ {
		w.Stations[i] = &Station{Sector: i}
	}

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
	if (g == GoodFood || g == GoodMedicines) && w.HealthEffectNewMolizone {
		return false
	}
	return true
}

// planetUpgradeStock — пересчёт TargetStock при смене TL планеты.
// Пропорционально newTL/oldTL, без накопления. count не трогаем.
func (w *World) planetUpgradeStock(p *Planet, oldTL, newTL int) {
	if oldTL <= 0 || newTL <= oldTL {
		return
	}
	ratio := float64(newTL) / float64(oldTL)
	for g := 0; g < int(GoodCount); g++ {
		p.Market[g].TargetStock = int(float64(p.Market[g].TargetStock) * ratio) * 100
		if p.Market[g].TargetStock < 1 {
			p.Market[g].TargetStock = 1
		}
	}
}

func (p *Planet) IsPirateLair() bool {
	return p.OwnerId == oiPirate
}