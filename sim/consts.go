package sim

// ---------- Товары ----------

type GoodInfo struct {
	InternalName        string
	BaseStock           int
	MinPrice            int
	AveragePrice        int
	MaxPrice            int
	TradeExpFactor      float64
	EcoAdjustPrice      float64 // знаковый: >0 дороже в богатых, <0 дешевле
	EcoAdjustQuantity   float64
	Mask                uint8   // какие экономики торгуют (Elite mask)
	PirateEconomyFactor float64
}

var GoodsMarket = [GoodCount]GoodInfo{
	// ---- 17 старых из TableBasePrice ----
	// name, BaseStock, Min, Avg, Max, TradeExp, EcoAdjPrice, EcoAdjQty, Mask, Pirate
	{"Food",         6,  9, 19, 29, 1.654, -2,  0, 0x01, 0.9},
	{"Textiles",    10,  7, 20, 33, 1.500, -1,  0, 0x03, 1.0},
	{"Radioactives", 2, 43, 65, 87, 2.000, -3,  0, 0x07, 1.2},
	{"Slaves",       8, 20, 40, 60, 2.500, -5,  0, 0x1F, 1.5},
	{"LiquorWines", 12, 33, 53, 73, 1.833, -5,  0, 0x0F, 1.3},
	{"Luxuries",     6, 80, 196, 312, 3.000, 8, 0, 0x03, 1.1},
	{"Narcotics",    4, 150, 235, 320, 1.833, 29, 0, 0x78, 2.0},
	{"Computers",    6, 90, 154, 218, 2.722, 14, 0, 0x03, 1.0},
	{"Machinery",    8, 50, 117, 184, 2.500, 6,  0, 0x07, 1.0},
	{"Alloys",       6, 30, 78, 126, 2.000, 1,  0, 0x1F, 1.2},
	{"Firearms",     4, 60, 124, 188, 2.500, 13, 0, 0x07, 1.5},
	{"Furs",         6, 80, 176, 272, 2.000, 9,  0, 0x3F, 1.0},
	{"Minerals",     8, 15, 32, 49, 2.000, 1,  0, 0x03, 2.5},
	{"Gold",         2, 50, 97, 144, 2.500, 1,  0, 0x07, 1.1},
	{"Platinum",     2, 70, 171, 272, 2.500, 2,  0, 0x1F, 1.1},
	{"GemStones",    1, 30, 45, 60, 2.500, 1,  0, 0x0F, 1.2},
	{"AlienItems",   1, 40, 53, 66, 3.000, 15, 0, 0x07, 1.5},

	// ---- 21 новый (черновик, в шкале Elite) ----
	{"FerrousMetals", 6,  30, 60, 90, 2.0,  5,  0, 0x07, 1.5},
	{"Timber",        4,  40, 88, 136, 2.0,  8,  0, 0x03, 1.0},
	{"AirProcessors", 4,  10, 35, 60, 2.0,  7,  0, 0x07, 1.0},
	{"AnimalMeat",   10,  20, 86, 152, 1.8, 10,  0, 0x0F, 0.9},
	{"AnimalSkins",   8,  80, 125, 170, 2.0, 11, 0, 0x07, 1.0},
	{"BattleWeapons", 4,  20, 53, 86, 2.5, 10,  0, 0x07, 1.5},
	{"FarmMachinery", 6,  10, 18, 26, 2.0, 31,  0, 0x07, 1.0},
	{"Fertilizer",    8,  30, 47, 64, 1.8,  4,  0, 0x07, 1.0},
	{"Grain",        12,  20, 82, 144, 1.6,  8,  0, 0x1F, 0.8},
	{"Drone",         4, 120, 210, 300, 2.5, 5,  0, 0x07, 1.1},
	{"HeavyPlastics", 6,  20, 38, 56, 2.0,  5,  0, 0x07, 1.0},
	{"HydrogenFuel",  6, 150, 231, 312, 2.0, 13, 0, 0x0F, 1.0},
	{"IndustrialParts",6, 60, 95, 130, 2.0, 5,  0, 0x07, 1.0},
	{"LiquidOxygen",  4, 100, 171, 242, 2.0, 9,  0, 0x0F, 1.0},
	{"LiveAnimals",   6, 80, 160, 240, 1.8, 17, 0, 0x1F, 1.0},
	{"Medicines",     6,  20, 38, 56, 2.038, 6, 0, 0x0F, 0.7},
	{"MilitaryFuel",  4,  40, 78, 116, 2.0,  9,  0, 0x07, 1.2},
	{"NerveGas",      2,  80, 154, 228, 2.5, 27, 0, 0x74, 2.0},
	{"Robots",        4,  80, 135, 190, 2.5,  9,  0, 0x07, 1.0},
	{"SyntheticMeat", 8, 100, 240, 380, 1.8, 31, 0, 0x06, 1.0},
	{"Water",        10,  30, 78, 126, 1.6, 6,  0, 0x03, 0.8},
}

// InflationOff — глобальный тумблер инфляции.
// true = инфляция выключена (цены/склады = GoodsMarketBase), false = канон.
const InflationOff = true

// GoodsMarketBase — неизменная копия базовых значений (для ComputeGlobalGoodsPriceBands).
// Считаем от неё, чтобы сжатие/инфляция по возрасту галактики не накапливались.
var GoodsMarketBase [GoodCount]GoodInfo = GoodsMarket

// GoodsTextOrder — порядок вывода товаров в вебе, логах, lairs.
// Сгруппировано: еда → сырьё → пром → топливо → роскошь → нелегалка → особые.
var GoodsTextOrder = [GoodCount]int{
	// Еда и сельхоз
	0,  // Food
	25, // Grain
	20, // AnimalMeat
	36, // SyntheticMeat
	31, // LiveAnimals
	37, // Water
	24, // Fertilizer
	// Сырьё
	12, // Minerals
	17, // FerrousMetals
	18, // Timber
	1,  // Textiles
	11, // Furs
	21, // AnimalSkins
	9,  // Alloys
	27, // HeavyPlastics
	// Пром
	8,  // Machinery
	7,  // Computers
	35, // Robots
	26, // Drone
	29, // IndustrialParts
	23, // FarmMachinery
	19, // AirProcessors
	32, // Medicines
	// Топливо
	28, // HydrogenFuel
	33, // MilitaryFuel
	30, // LiquidOxygen
	// Роскошь
	5,  // Luxuries
	4,  // LiquorWines
	// Нелегалка
	6,  // Narcotics
	3,  // Slaves
	34, // NerveGas
	10, // Firearms
	22, // BattleWeapons
	2,  // Radioactives
	// Особые
	13, // Gold
	14, // Platinum
	15, // GemStones
	16, // AlienItems
}

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
	// Maloc (0): сырьё, еда, оружие. Роскошь/техно — дорого/мало.
	{
		InventionProgressScale: 0.85, InitialInventionBoostCount: 5, RevolutionChance: 0.002,
		FriendlyRelationScale: 0.7, PirateRelationFactor: 1.1, PirateRelationCeiling: 60,
		Goods: RaceGoodsFactor{
			PriceFactor: [GoodCount]float64{
				0.85, 1.10, 1.15, 1.20, 1.10, 1.20, 1.10, 1.15, // Food..Computers
				1.10, 1.05, 0.85, 1.10, 0.87, 1.00, 1.00, 1.00, // Machinery..Gold
				1.00, 1.00, 1.10, 1.10, 0.80, 0.90, 1.10, 0.95, // Platinum..Timber
				1.15, 0.90, 1.05, 1.15, 0.90, 0.95, 1.05, 1.10, // AirProc..LiveAnimals
				1.15, 1.10, 1.20, 1.00, 1.10, 0.95,               // Medicines..Water
			},
			StockFactor: [GoodCount]float64{
				1.1, 1.1, 0.5, 0.4, 0.5, 0.2, 0.2, 0.3,
				0.5, 0.7, 1.1, 0.6, 0.7, 0.5, 0.5, 0.5,
				0.4, 0.6, 0.3, 0.3, 1.2, 1.2, 0.4, 1.0,
				0.3, 0.7, 0.6, 0.4, 0.8, 0.8, 0.6, 1.0,
				1.0, 1.1, 0.9, 0.3, 0.5, 1.0,
			},
		},
		GovernmentRollThresholds:   [5]int{10, 30, 50, 90, 95},
		RaceSkillEvaluationFactors: [6]float64{1.2, 1.1, 0.9, 0.8, 1.0, 1.0},
	},
	// Peleng (1): торговцы, нелегалка, роскошь.
	{
		InventionProgressScale: 0.95, InitialInventionBoostCount: 6, RevolutionChance: 0.006,
		FriendlyRelationScale: 1.1, PirateRelationFactor: 1.5, PirateRelationCeiling: 80,
		Goods: RaceGoodsFactor{
			PriceFactor: [GoodCount]float64{
				0.95, 1.05, 1.07, 0.90, 0.95, 1.15, 0.95, 1.10,
				1.10, 1.05, 1.10, 1.15, 0.95, 1.05, 1.05, 1.05,
				1.05, 1.05, 1.05, 1.05, 0.95, 1.05, 1.10, 1.05,
				1.05, 1.05, 1.05, 1.05, 0.95, 0.95, 0.95, 0.95,
				0.95, 1.05, 0.95, 0.80, 1.05, 1.05,
			},
			StockFactor: [GoodCount]float64{
				1.2, 1.0, 0.9, 1.2, 1.2, 0.8, 1.1, 0.8,
				0.8, 1.0, 0.8, 1.0, 0.8, 0.8, 0.8, 0.8,
				0.8, 0.8, 0.8, 0.8, 1.0, 1.0, 0.9, 1.0,
				0.9, 0.9, 0.9, 0.9, 1.0, 1.0, 1.0, 1.0,
				1.0, 0.9, 1.0, 1.1, 0.9, 1.0,
			},
		},
		GovernmentRollThresholds:   [5]int{20, 40, 70, 85, 90},
		RaceSkillEvaluationFactors: [6]float64{1.0, 1.2, 0.8, 1.1, 1.0, 0.9},
	},
	// Human (2): база, нейтрально.
	{
		InventionProgressScale: 1.00, InitialInventionBoostCount: 7, RevolutionChance: 0.004,
		FriendlyRelationScale: 1.0, PirateRelationFactor: 0.9, PirateRelationCeiling: 45,
		Goods: RaceGoodsFactor{
			PriceFactor: [GoodCount]float64{
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1,
			},
			StockFactor: [GoodCount]float64{
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1,
			},
		},
		GovernmentRollThresholds:   [5]int{10, 30, 40, 60, 80},
		RaceSkillEvaluationFactors: [6]float64{0.9, 0.8, 1.0, 1.2, 1.0, 1.1},
	},
	// Feyan (3): техно, роскошь. Еда/сырьё — дорого/мало.
	{
		InventionProgressScale: 1.10, InitialInventionBoostCount: 8, RevolutionChance: 0.003,
		FriendlyRelationScale: 1.1, PirateRelationFactor: 0.7, PirateRelationCeiling: 35,
		Goods: RaceGoodsFactor{
			PriceFactor: [GoodCount]float64{
				1.05, 0.85, 0.87, 1.10, 1.15, 0.90, 1.10, 0.85,
				0.85, 0.90, 0.90, 1.00, 1.15, 1.00, 1.00, 1.00,
				1.00, 1.00, 1.15, 1.10, 1.10, 1.05, 0.90, 0.95,
				1.10, 0.85, 0.90, 0.85, 1.10, 1.10, 1.05, 1.15,
				0.85, 0.90, 1.10, 1.10, 0.90, 1.10,
			},
			StockFactor: [GoodCount]float64{
				0.7, 0.9, 1.4, 0.5, 0.4, 0.8, 0.5, 1.4,
				1.4, 1.2, 0.4, 0.4, 0.5, 0.5, 0.5, 0.5,
				0.5, 0.5, 0.5, 0.5, 0.4, 0.6, 0.8, 1.2,
				0.4, 0.4, 0.4, 0.6, 0.6, 0.6, 0.8, 0.5,
				0.8, 0.4, 0.8, 0.4, 0.6, 0.5,
			},
		},
		GovernmentRollThresholds:   [5]int{5, 10, 15, 30, 70},
		RaceSkillEvaluationFactors: [6]float64{1.1, 1.0, 1.2, 0.8, 0.9, 1.0},
	},
	// Gaal (4): высшие техно. Всё сырьё/еда — дорого/мало.
	{
		InventionProgressScale: 1.15, InitialInventionBoostCount: 9, RevolutionChance: 0.002,
		FriendlyRelationScale: 1.3, PirateRelationFactor: 0.6, PirateRelationCeiling: 35,
		Goods: RaceGoodsFactor{
			PriceFactor: [GoodCount]float64{
				1.10, 0.80, 0.80, 1.15, 1.15, 0.85, 1.15, 0.80,
				0.80, 0.85, 0.85, 0.90, 1.10, 0.90, 0.90, 0.90,
				0.90, 0.90, 1.10, 1.15, 1.15, 1.10, 0.84, 0.90,
				1.15, 0.80, 0.85, 0.80, 1.15, 1.15, 1.10, 1.20,
				0.80, 0.85, 1.15, 1.10, 0.85, 1.15,
			},
			StockFactor: [GoodCount]float64{
				0.5, 0.5, 0.8, 0.3, 0.3, 0.5, 0.4, 1.5,
				1.5, 1.3, 0.3, 0.3, 0.3, 0.4, 0.4, 0.4,
				0.4, 0.4, 0.4, 0.4, 0.3, 0.5, 0.6, 1.4,
				0.3, 0.3, 0.3, 0.5, 0.5, 0.5, 0.7, 0.4,
				0.7, 0.3, 0.7, 0.3, 0.5, 0.4,
			},
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
	// 0 Anarchy — нелегалка дёшево/много, порядок слабый.
	{
		RevolutionRelationDelta: [3]int{-30, 30, 0},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{
				1.0, 1.0, 0.9, 0.8, 0.8, 0.9, 0.7, 1.0,
				1.0, 1.0, 0.8, 1.1, 0.8, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 0.9, 1.0, 1.0, 1.0, 0.9, 1.0, 1.0,
				1.0, 1.0, 0.7, 1.0, 1.0, 1.0,
			},
			StockFactor: [GoodCount]float64{
				0.7, 0.7, 1.0, 1.2, 1.2, 1.0, 1.3, 0.7,
				0.7, 0.7, 1.2, 1.0, 0.8, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.3, 1.0, 1.0, 1.0,
			},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.6, 0.8, 0.1, 0.3},
	},
	// 1 Feudal — роскошь/оружие, слабый контроль.
	{
		RevolutionRelationDelta: [3]int{-20, 10, 10},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{
				1.0, 1.0, 1.0, 1.0, 0.9, 1.1, 0.9, 1.0,
				0.9, 1.0, 0.9, 1.0, 0.9, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 0.9, 1.0, 1.0, 1.0,
			},
			StockFactor: [GoodCount]float64{
				0.8, 0.8, 0.9, 1.0, 1.0, 1.1, 1.0, 0.8,
				0.8, 0.8, 1.0, 1.0, 0.9, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
			},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.5, 0.8, 0.4, 0.6},
	},
	// 2 Multi-Government — хаос, всё понемногу.
	{
		RevolutionRelationDelta: [3]int{-10, 10, 0},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1,
			},
			StockFactor: [GoodCount]float64{
				0.9, 0.9, 1.0, 1.0, 1.0, 1.0, 1.0, 0.9,
				0.9, 0.9, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
			},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.4, 0.8, 0.3, 0.5},
	},
	// 3 Dictatorship — оружие +, наркотики −.
	{
		RevolutionRelationDelta: [3]int{-40, 20, 20},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{
				1.0, 1.0, 1.0, 1.0, 0.9, 1.0, 1.1, 1.0,
				1.0, 1.0, 0.8, 1.0, 0.9, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 0.9, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.1, 1.0, 1.0, 1.0,
			},
			StockFactor: [GoodCount]float64{
				0.7, 0.7, 0.8, 0.8, 0.9, 1.0, 0.9, 0.7,
				0.7, 0.7, 1.2, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.1, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 0.8, 1.0, 1.0, 1.0,
			},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.7, 0.8, 0.2, 0.5},
	},
	// 4 Communist — контроль, еда/медицина +, нелегалка −.
	{
		RevolutionRelationDelta: [3]int{-20, 0, 0},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{
				0.9, 1.0, 1.0, 1.1, 1.1, 1.0, 1.2, 1.0,
				1.0, 1.0, 1.1, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 0.9, 1.0, 1.0, 1.0, 1.0, 1.0, 0.9,
				1.0, 1.0, 1.2, 1.0, 1.0, 1.0,
			},
			StockFactor: [GoodCount]float64{
				1.1, 1.0, 1.0, 0.8, 0.9, 1.0, 0.8, 1.0,
				1.0, 1.0, 0.8, 0.9, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.1, 1.0, 1.0, 1.0, 1.0, 1.0, 1.1,
				1.0, 1.0, 0.7, 1.0, 1.0, 1.0,
			},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.3, 0.8, 0.3, 0.6},
	},
	// 5 Confederacy — свободная торговля, всё нейтрально.
	{
		RevolutionRelationDelta: [3]int{10, 0, 0},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1,
			},
			StockFactor: [GoodCount]float64{
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1,
			},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.5, 0.8, 0.5, 0.7},
	},
	// 6 Democracy — еда/медицина +, наркотики/оружие −.
	{
		RevolutionRelationDelta: [3]int{15, -30, 0},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{
				0.9, 1.0, 1.0, 1.1, 1.1, 1.0, 1.2, 1.0,
				1.0, 1.0, 1.1, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 0.9, 1.0, 1.0, 1.0, 1.0, 1.0, 0.9,
				1.0, 1.0, 1.2, 1.0, 1.0, 1.0,
			},
			StockFactor: [GoodCount]float64{
				1.1, 1.0, 1.0, 0.8, 0.9, 1.0, 0.8, 1.0,
				1.0, 1.0, 0.8, 0.9, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.1, 1.0, 1.0, 1.0, 1.0, 1.0, 1.1,
				1.0, 1.0, 0.7, 1.0, 1.0, 1.0,
			},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.3, 0.8, 0.5, 0.8},
	},
	// 7 Corporate State — стабильно, дорого, роскошь/техно +.
	{
		RevolutionRelationDelta: [3]int{20, -10, 0},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{
				1.1, 1.0, 1.0, 1.0, 1.0, 0.9, 1.0, 0.9,
				0.9, 0.9, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.1, 1.0, 1.0, 1.0, 1.0, 1.0, 1.1,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
			},
			StockFactor: [GoodCount]float64{
				1.0, 1.0, 1.0, 1.0, 1.0, 1.1, 1.0, 1.1,
				1.1, 1.1, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 0.9, 1.0, 1.0, 1.0, 1.0, 1.0, 0.9,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
			},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.4, 0.8, 0.7, 0.9},
	},
	// 8 Monarchy — роскошь/еда +, традиции.
	{
		RevolutionRelationDelta: [3]int{-10, 0, 10},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{
				1.0, 1.0, 1.0, 1.0, 0.9, 0.9, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
			},
			StockFactor: [GoodCount]float64{
				1.0, 1.0, 1.0, 1.0, 1.1, 1.1, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
			},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.4, 0.8, 0.6, 0.85},
	},
	// 9 Republic — техно/медицина +, бюрократия.
	{
		RevolutionRelationDelta: [3]int{10, -20, 0},
		Goods: GovGoodsFactor{
			PriceFactor: [GoodCount]float64{
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 0.9,
				0.9, 0.9, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 0.9,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
			},
			StockFactor: [GoodCount]float64{
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.1,
				1.1, 1.1, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.1,
				1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
			},
		},
		QuestOfferProbabilities: [5]float64{0.2, 0.5, 0.8, 0.7, 0.9},
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

	// Индексы Elite-порядка:
	// 0 Food, 1 Textiles, 2 Radioactives, 3 Slaves, 4 LiquorWines,
	// 5 Luxuries, 6 Narcotics, 7 Computers, 8 Machinery, 9 Alloys,
	// 10 Firearms, 11 Furs, 12 Minerals, 13 Gold, 14 Platinum,
	// 15 GemStones, 16 AlienItems, 17 FerrousMetals, 18 Timber,
	// 19 AirProcessors, 20 AnimalMeat, 21 AnimalSkins, 22 BattleWeapons,
	// 23 FarmMachinery, 24 Fertilizer, 25 Grain, 26 Drone, 27 HeavyPlastics,
	// 28 HydrogenFuel, 29 IndustrialParts, 30 LiquidOxygen, 31 LiveAnimals,
	// 32 Medicines, 33 MilitaryFuel, 34 NerveGas, 35 Robots,
	// 36 SyntheticMeat, 37 Water

	// --- Старые правила (перенесены на новые индексы) ---

	// Maloc: роскошь/алкоголь/наркотики нелегальны в части правительств.
	GoodsLegalOnPlanet[GoodLuxuries][RaceMaloc][GovAnarchy] = false
	GoodsLegalOnPlanet[GoodLiquorWines][RaceMaloc][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodLuxuries][RaceMaloc][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceMaloc][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodLiquorWines][RaceMaloc][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceMaloc][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodLuxuries][RaceMaloc][GovRepublic] = false
	GoodsLegalOnPlanet[GoodLuxuries][RaceMaloc][GovDemocracy] = false

	// Human
	GoodsLegalOnPlanet[GoodNarcotics][RaceHuman][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodFirearms][RaceHuman][GovDemocracy] = false

	// Feyan
	GoodsLegalOnPlanet[GoodLiquorWines][RaceFeyan][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodFirearms][RaceFeyan][GovDemocracy] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceHuman][GovDemocracy] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceHuman][GovRepublic] = false
	GoodsLegalOnPlanet[GoodNarcotics][RacePeleng][GovDemocracy] = false
	GoodsLegalOnPlanet[GoodNarcotics][RacePeleng][GovRepublic] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceFeyan][GovDemocracy] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceFeyan][GovRepublic] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceGaal][GovDemocracy] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceGaal][GovRepublic] = false
	GoodsLegalOnPlanet[GoodFirearms][RaceFeyan][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceFeyan][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodFirearms][RaceFeyan][GovRepublic] = false
	GoodsLegalOnPlanet[GoodLiquorWines][RaceFeyan][GovRepublic] = false
	GoodsLegalOnPlanet[GoodLiquorWines][RaceFeyan][GovDemocracy] = false

	// Gaal
	GoodsLegalOnPlanet[GoodLiquorWines][RaceGaal][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceGaal][GovDictatorship] = false
	GoodsLegalOnPlanet[GoodLiquorWines][RaceGaal][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodNarcotics][RaceGaal][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodFirearms][RaceGaal][GovMonarchy] = false
	GoodsLegalOnPlanet[GoodFirearms][RaceGaal][GovRepublic] = false
	GoodsLegalOnPlanet[GoodFirearms][RaceGaal][GovDemocracy] = false

	// --- Новые правила (38 товаров) ---

	// Slaves: нелегальны почти везде, кроме Anarchy/Feudal/Dictatorship.
	for r := 0; r < int(RaceCount); r++ {
		for v := 0; v < int(GovCount); v++ {
			switch Government(v) {
			case GovAnarchy, GovFeudal, GovDictatorship:
				// легально
			default:
				GoodsLegalOnPlanet[GoodSlaves][Race(r)][Government(v)] = false
			}
		}
	}

	// NerveGas: нелегален почти везде, кроме Dictatorship/Anarchy.
	for r := 0; r < int(RaceCount); r++ {
		for v := 0; v < int(GovCount); v++ {
			switch Government(v) {
			case GovAnarchy, GovDictatorship:
			default:
				GoodsLegalOnPlanet[GoodNerveGas][Race(r)][Government(v)] = false
			}
		}
	}

	// BattleWeapons: нелегальны в Democracy/Communist/Republic.
	for r := 0; r < int(RaceCount); r++ {
		GoodsLegalOnPlanet[GoodBattleWeapons][Race(r)][GovDemocracy] = false
		GoodsLegalOnPlanet[GoodBattleWeapons][Race(r)][GovCommunist] = false
		GoodsLegalOnPlanet[GoodBattleWeapons][Race(r)][GovRepublic] = false
	}

	// Narcotics: нелегальны в Democracy/Republic (кроме Peleng — уже выше).
	for r := 0; r < int(RaceCount); r++ {
		if Race(r) == RacePeleng {
			continue
		}
		GoodsLegalOnPlanet[GoodNarcotics][Race(r)][GovDemocracy] = false
		GoodsLegalOnPlanet[GoodNarcotics][Race(r)][GovRepublic] = false
	}

	// AlienItems: нелегальны в Democracy/Republic/Corporate (ксенофобия).
	for r := 0; r < int(RaceCount); r++ {
		GoodsLegalOnPlanet[GoodAlienItems][Race(r)][GovDemocracy] = false
		GoodsLegalOnPlanet[GoodAlienItems][Race(r)][GovRepublic] = false
		GoodsLegalOnPlanet[GoodAlienItems][Race(r)][GovCorporateState] = false
	}
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