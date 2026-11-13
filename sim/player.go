package sim

// PlayerGood — 8 «игровых» товаров (старая база). Отдельный тип,
// чтобы не мешать сим-товарам (GoodCount 38). Стык — через PlayerGoodMap.
type PlayerGood int

const (
	PlayerGoodFood PlayerGood = iota
	PlayerGoodMedicine
	PlayerGoodTechnics
	PlayerGoodLuxury
	PlayerGoodMinerals
	PlayerGoodAlcohol
	PlayerGoodArms
	PlayerGoodNarcotics
	PlayerGoodCount
)

// PlayerGoodNames — имена 8 для веба/логов.
var PlayerGoodNames = [PlayerGoodCount]string{
	"food", "medicine", "technics", "luxury",
	"minerals", "alcohol", "arms", "narcotics",
}

// PlayerGoodMap — какие сим-товары (Good, Elite-индексы) входят в каждую
// из 8 игровых категорий. Все 38 покрыты.
//
// TODO: сюда же добавятся товары вне симуляции (спутники, необитаемые,
// производство игроков), когда появятся — через отдельный диапазон/тип.
var PlayerGoodMap = [PlayerGoodCount][]Good{
	// food
	{GoodFood, GoodGrain, GoodAnimalMeat, GoodSyntheticMeat, GoodLiveAnimals, GoodWater, GoodFertilizer},
	// medicine
	{GoodMedicines},
	// technics
	{GoodComputers, GoodMachinery, GoodRobots, GoodDrone, GoodIndustrialParts,
		GoodFarmMachinery, GoodAirProcessors, GoodHeavyPlastics, GoodAlloys,
		GoodHydrogenFuel, GoodLiquidOxygen},
	// luxury
	{GoodLuxuries, GoodGemStones, GoodGold, GoodPlatinum, GoodAlienItems,
		GoodTextiles, GoodFurs, GoodAnimalSkins},
	// minerals
	{GoodMinerals, GoodFerrousMetals, GoodTimber, GoodRadioactives},
	// alcohol
	{GoodLiquorWines},
	// arms
	{GoodFirearms, GoodBattleWeapons, GoodMilitaryFuel},
	// narcotics
	{GoodNarcotics, GoodNerveGas, GoodSlaves},
}

// PlayerMarketEntry — строка рынка игрока (одна из 8).
type PlayerMarketEntry struct {
	Good  PlayerGood `json:"good"`
	Buy   float64    `json:"buy"`
	Sell  float64    `json:"sell"`
	Count int        `json:"count"`
}

// PlayerMarket — агрегат по сектору для игрока: 8 категорий.
// По обитаемым не-логовам сектора. buy/sell — средневзвешенное по count
// (крупные склады весят больше), count — сумма.
func (w *World) PlayerMarket(sectorID int) [PlayerGoodCount]PlayerMarketEntry {
	// Сначала агрегируем сим-товары по сектору: sumBuy*count, sumSell*count, sumCount, sumCount.
	type acc struct {
		sumBuyWC float64 // Σ buy*count
		sumSellWC float64 // Σ sell*count
		sumCount int
	}
	var mkt [GoodCount]acc

	for _, st := range w.Stars {
		if st.Sector != sectorID {
			continue
		}
		for _, p := range w.PlanetMap[st.ID] {
			if !p.Habitable || p.IsPirateLair() {
				continue
			}
			for g := 0; g < int(GoodCount); g++ {
				m := &p.Market[g]
				c := m.Count
				mkt[g].sumBuyWC += m.PurchasePrice * float64(c)
				mkt[g].sumSellWC += m.BaseSalePrice * float64(c)
				mkt[g].sumCount += c
			}
		}
	}

	var out [PlayerGoodCount]PlayerMarketEntry
	for pg := 0; pg < int(PlayerGoodCount); pg++ {
		var buyWC, sellWC float64
		var cnt int
		for _, g := range PlayerGoodMap[pg] {
			buyWC += mkt[g].sumBuyWC
			sellWC += mkt[g].sumSellWC
			cnt += mkt[g].sumCount
		}
		e := PlayerMarketEntry{Good: PlayerGood(pg), Count: cnt}
		if cnt > 0 {
			e.Buy = buyWC / float64(cnt)
			e.Sell = sellWC / float64(cnt)
		}
		out[pg] = e
	}
	return out
}