package sim

// ---------- Товары ----------

type GoodInfo struct {
	InternalName        string
	BaseStock           int
	MinPrice            int
	AveragePrice        int
	MaxPrice            int
	TradeExpFactor      float64
	EconomyFactors      [3]float64 // Agri / Mixed / Ind
	PirateEconomyFactor float64
}

var GoodsMarket = [GoodCount]GoodInfo{
	{"Food", 300, 17, 30, 43, 1.654, [3]float64{1.15, 1.0, 0.85}, 0.9},
	{"Medicine", 160, 27, 40, 53, 2.038, [3]float64{1.10, 1.0, 0.90}, 0.7},
	{"Technics", 100, 62, 80, 98, 2.722, [3]float64{0.85, 1.0, 1.15}, 0.8},
	{"Luxury", 60, 160, 200, 240, 3.0, [3]float64{1.00, 1.1, 1.05}, 0.8},
	{"Minerals", 250, 8, 12, 16, 2.0, [3]float64{1.15, 1.0, 0.85}, 2.5},
	{"Alcohol", 120, 25, 40, 55, 1.833, [3]float64{1.20, 1.0, 0.80}, 1.3},
	{"Arms", 70, 75, 100, 125, 2.5, [3]float64{0.85, 1.0, 1.15}, 1.5},
	{"Narcotics", 30, 250, 400, 550, 1.833, [3]float64{1.10, 1.0, 0.90}, 2.0},
}

// InflationOff — глобальный тумблер инфляции.
// true = инфляция выключена (цены/склады = GoodsMarketBase), false = канон.
const InflationOff = true

// GoodsMarketBase — неизменная копия базовых значений (для ComputeGlobalGoodsPriceBands).
// Считаем от неё, чтобы сжатие/инфляция по возрасту галактики не накапливались.
var GoodsMarketBase [GoodCount]GoodInfo = GoodsMarket

var GoodsTextOrder = [8]int{0, 1, 5, 4, 3, 2, 6, 7}

// ---------- Расы ----------

type RaceGoodsFactor struct {
	PriceFactor [GoodCount]float64
	StockFactor [GoodCount]float64
}

type RaceInfo struct {
	InventionProgressScale      float64
	InitialInventionBoostCount  int
	RevolutionChance            float64
	FriendlyRelationScale       float64
	PirateRelationFactor        float64
	PirateRelationCeiling       int
	Goods                       RaceGoodsFactor
	GovernmentRollThresholds    [5]int
	RaceSkillEvaluationFactors  [6]float64 // Accuracy/Maneuver/Technical/Trading/Charisma/Leadership
}

var PlanetRaceMarket = [RaceCount]RaceInfo{
	{
		InventionProgressScale: 0.85, InitialInventionBoostCount: 5, RevolutionChance: 0.002,
		FriendlyRelationScale: 0.7, PirateRelationFactor: 1.1, PirateRelationCeiling: 60,
		Goods: RaceGoodsFactor{
			PriceFactor: [GoodCount]float64{0.85, 1.10, 1.15, 0.85, 0.87, 1.10, 1.20, 0.80},
			StockFactor: [GoodCount]float64{1.1, 1.1, 0.5, 0.4, 0.7, 0.1, 0.5, 0.2},
		},
		GovernmentRollThresholds:   [5]int{10, 30, 50, 90, 95},
		RaceSkillEvaluationFactors: [6]float64{1.2, 1.1, 0.9, 0.8, 1.0, 1.0},
	},
	{
		InventionProgressScale: 0.95, InitialInventionBoostCount: 6, RevolutionChance: 0.006,
		FriendlyRelationScale: 1.1, PirateRelationFactor: 1.5, PirateRelationCeiling: 80,
		Goods: RaceGoodsFactor{
			PriceFactor: [GoodCount]float64{0.95, 1.05, 1.07, 1.15, 0.95, 1.05, 1.10, 0.95},
			StockFactor: [GoodCount]float64{1.2, 1.0, 0.9, 1.2, 0.8, 1.1, 0.8, 0.6},
		},
		GovernmentRollThresholds:   [5]int{20, 40, 70, 85, 90},
		RaceSkillEvaluationFactors: [6]float64{1.0, 1.2, 0.8, 1.1, 1.0, 0.9},
	},
	{
		InventionProgressScale: 1.00, InitialInventionBoostCount: 7, RevolutionChance: 0.004,
		FriendlyRelationScale: 1.0, PirateRelationFactor: 0.9, PirateRelationCeiling: 45,
		Goods: RaceGoodsFactor{
			PriceFactor: [GoodCount]float64{1, 1, 1, 1, 1, 1, 1, 1},
			StockFactor: [GoodCount]float64{1, 1, 1, 1, 1, 1, 1, 1},
		},
		GovernmentRollThresholds:   [5]int{10, 30, 40, 60, 80},
		RaceSkillEvaluationFactors: [6]float64{0.9, 0.8, 1.0, 1.2, 1.0, 1.1},
	},
	{
		InventionProgressScale: 1.10, InitialInventionBoostCount: 8, RevolutionChance: 0.003,
		FriendlyRelationScale: 1.1, PirateRelationFactor: 0.7, PirateRelationCeiling: 35,
		Goods: RaceGoodsFactor{
			PriceFactor: [GoodCount]float64{1.05, 0.85, 0.87, 1.0, 1.15, 1.15, 0.90, 1.10},
			StockFactor: [GoodCount]float64{0.7, 0.9, 1.4, 0.8, 0.5, 0.4, 0.4, 0.5},
		},
		GovernmentRollThresholds:   [5]int{5, 10, 15, 30, 70},
		RaceSkillEvaluationFactors: [6]float64{1.1, 1.0, 1.2, 0.8, 0.9, 1.0},
	},
	{
		InventionProgressScale: 1.15, InitialInventionBoostCount: 9, RevolutionChance: 0.002,
		FriendlyRelationScale: 1.3, PirateRelationFactor: 0.6, PirateRelationCeiling: 35,
		Goods: RaceGoodsFactor{
			PriceFactor: [GoodCount]float64{1.10, 0.80, 0.80, 0.85, 1.10, 0.90, 0.84, 1.15},
			StockFactor: [GoodCount]float64{0.5, 0.5, 0.8, 0.5, 0.3, 0.3, 0.1, 0.4},
		},
		GovernmentRollThresholds:   [5]int{5, 8, 10, 30, 60},
		RaceSkillEvaluationFactors: [6]float64{0.8, 0.9, 1.1, 1.0, 1.2, 1.0},
	},
}

// ---------- Правительства ----------

type GovGoodsFactor struct {
	PriceFactor [GoodCount]float64
	StockFactor [GoodCount]float64
}

type GovInfo struct {
	RevolutionRelationDelta [3]int // Trader/Pirate/Warrior
	Goods                   GovGoodsFactor
	QuestOfferProbabilities [5]float64
}

var PlanetGovernmentMarket = [GovCount]GovInfo{
	{
		RevolutionRelationDelta: [3]int{-30, 30, 0},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{1.0, 1.0, 1.0, 1.1, 0.8, 0.8, 1.1, 0.9},
			StockFactor: [GoodCount]float64{0.7, 0.7, 0.7, 1.0, 0.8, 1.0, 1.0, 1.0},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.6, 0.8, 0.1, 0.3},
	},
	{
		RevolutionRelationDelta: [3]int{-40, 20, 20},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{1.0, 1.0, 1.0, 1.0, 0.9, 0.9, 1.0, 0.9},
			StockFactor: [GoodCount]float64{0.7, 0.7, 0.7, 1.0, 1.0, 0.9, 0.9, 0.9},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.7, 0.8, 0.2, 0.5},
	},
	{
		RevolutionRelationDelta: [3]int{-10, 0, 10},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{1, 1, 1, 1, 1, 1, 1, 1},
			StockFactor: [GoodCount]float64{0.9, 1.0, 1.0, 1.0, 1.0, 0.8, 0.8, 0.8},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.4, 0.8, 0.6, 0.85},
	},
	{
		RevolutionRelationDelta: [3]int{10, -20, 0},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{1, 1, 1, 1, 1.1, 1, 1, 1.1},
			StockFactor: [GoodCount]float64{1, 1, 1, 1, 1.0, 0.8, 0.7, 0.7},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.5, 0.8, 0.7, 0.9},
	},
	{
		RevolutionRelationDelta: [3]int{15, -30, 0},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{1, 1, 1, 1, 1.1, 1, 1, 1.1},
			StockFactor: [GoodCount]float64{1, 1, 1, 1, 1.0, 0.8, 0.5, 0.6},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.3, 0.8, 0.5, 0.8},
	},
}

// ---------- Легальность ----------

var GoodsLegalOnPlanet [GoodCount][RaceCount][GovCount]bool

func init() {
	for g := 0; g < int(GoodCount); g++ {
		for r := 0; r < int(RaceCount); r++ {
			for v := 0; v < int(GovCount); v++ {
				GoodsLegalOnPlanet[g][r][v] = true
			}
		}
	}

	// --- Текущие (пока оставляем) ---
	// КАНОН: анархия — всё легально. Пока оставляем нелегальным, удалим позже.
	GoodsLegalOnPlanet[GoodLuxury][RaceMaloc][GovAnarchy] = false
	// КАНОН: Feyan/диктат — всё легально. Пока оставляем нелегальным, удалим позже.
	GoodsLegalOnPlanet[GoodAlcohol][RaceFeyan][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodArms][RaceFeyan][GovDemocracy] = false
	for r := 0; r < int(RaceCount); r++ {
		if Race(r) != RacePeleng {
			GoodsLegalOnPlanet[GoodNarcotics][r][GovDemocracy] = false
			GoodsLegalOnPlanet[GoodNarcotics][r][GovRepublic] = false
		}
	}

	// --- КАНОН (добавляем) ---

	// Maloc
	GoodsLegalOnPlanet[GoodAlcohol][RaceMaloc][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodLuxury][RaceMaloc][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceMaloc][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodAlcohol][RaceMaloc][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceMaloc][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodLuxury][RaceMaloc][GovRepublic] = false
	GoodsLegalOnPlanet[GoodLuxury][RaceMaloc][GovDemocracy] = false

	// Human
	GoodsLegalOnPlanet[GoodNarcotics][RaceHuman][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodArms][RaceHuman][GovDemocracy] = false

	// Feyan
	GoodsLegalOnPlanet[GoodArms][RaceFeyan][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceFeyan][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodArms][RaceFeyan][GovRepublic] = false
	GoodsLegalOnPlanet[GoodAlcohol][RaceFeyan][GovRepublic] = false
	GoodsLegalOnPlanet[GoodAlcohol][RaceFeyan][GovDemocracy] = false

	// Gaal
	GoodsLegalOnPlanet[GoodAlcohol][RaceGaal][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceGaal][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodAlcohol][RaceGaal][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceGaal][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodArms][RaceGaal][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodArms][RaceGaal][GovRepublic] = false
	GoodsLegalOnPlanet[GoodArms][RaceGaal][GovDemocracy] = false
}

// ---------- Экономические константы ----------

const (
	TurnsPerYear            = 576
	GalaxyWarmupTurns       = 576
	MaxMonetaryValue        = 100_000_000.0
// PlanetCostMult — множитель стоимости апгрейда планет (замедляет рост ГТУ).
	PlanetCostMult 			= 2.0
	GoodsInflationStartTurn = 1000
	GoodsInflationEndTurn   = 10000
	GoodsInflationMin       = 1.0
	GoodsInflationMax       = 10.0
	GoodsStockMin           = 1.0
	GoodsStockMax           = 0.5
)

type DifficultyTuning struct {
	GoodsEventDurationFactor float64
	MarketPriceBandSqueeze   float64
	StartingPlayerMoney      float64
}

var Difficulty = DifficultyTuning{
	GoodsEventDurationFactor: 1.0,
	MarketPriceBandSqueeze:   0.0,
	StartingPlayerMoney:      1000,
}