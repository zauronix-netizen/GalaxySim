package sim

import (
	"math"
)

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

// ---------- Старая модель 8 товаров (копия из old/, потом подгоним) ----------

type OldGoodInfo struct {
	Name         string
	BaseStock    int
	MinPrice     int
	AveragePrice int
	MaxPrice     int
}

// OldGoodsMarket — 8 товаров, старая модель (old/consts.go).
// Порядок = PlayerGood (food, medicine, technics, luxury, minerals, alcohol, arms, narcotics).
var OldGoodsMarket = [PlayerGoodCount]OldGoodInfo{
	{"Food", 300, 17, 30, 43},
	{"Medicine", 160, 27, 40, 53},
	{"Technics", 100, 62, 80, 98},
	{"Luxury", 60, 160, 200, 240},
	{"Minerals", 250, 8, 12, 16},
	{"Alcohol", 120, 25, 40, 55},
	{"Arms", 70, 75, 100, 125},
	{"Narcotics", 30, 250, 400, 550},
}

// ---------- Станция игрока ----------

// Station — станция сектора. Своего Count нет: количество берётся из сима
// (PlayerMarket, 8). Balance — долг: <0 станция должна планетам сектора,
// >0 планеты должны станции. Алгоритм-телепорт пока не подключён.
type Station struct {
	Sector  int
	Balance [PlayerGoodCount]int
}

// StationPrice — упрощённая старая формула цены для станции.
// count — из сима (PlayerMarket.count), ts — старый BaseStock (OldGoodsMarket).
// Без рас/правительств, всё легально.
func StationPrice(pg PlayerGood, count int) (buy, sell float64) {
	gm := &OldGoodsMarket[pg]
	ts := float64(gm.BaseStock)
	if ts < 1 {
		ts = 1
	}
	c := float64(count)

	target := float64(gm.AveragePrice)
	if c < ts {
		target /= RemapClamped(c, ts*0.1, ts, 0.8, 1)
	} else {
		target /= RemapClamped(c, ts, ts*3, 1, 1.2)
	}
	if target > float64(gm.MaxPrice) {
		target = float64(gm.MaxPrice)
	}
	if target < float64(gm.MinPrice) {
		target = float64(gm.MinPrice)
	}

	buy = math.Max(2, math.Round(target))
	sell = math.Max(buy/2+1, math.Round(target*0.5-1))
	return buy, sell
}