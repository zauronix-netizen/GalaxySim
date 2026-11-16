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
	// food (6)
	{GoodFood, GoodGrain, GoodAnimalMeat, GoodSyntheticMeat, GoodWater, GoodFertilizer},
	// medicine (6)
	{GoodMedicines, GoodWater, GoodAirProcessors, GoodTextiles, GoodLiquorWines, GoodNarcotics},
	// technics (8)
	{GoodComputers, GoodMachinery, GoodIndustrialParts, GoodFarmMachinery, GoodAirProcessors,
		GoodHeavyPlastics, GoodHydrogenFuel, GoodLiquidOxygen},
	// luxury (9)
	{GoodLuxuries, GoodGemStones, GoodGold, GoodPlatinum, GoodAlienItems,
		GoodTextiles, GoodFurs, GoodAnimalSkins, GoodLiveAnimals},
	// minerals (6) — +Alloys уже есть, +Timber, +Radioactives, +FerrousMetals, +Minerals
	{GoodMinerals, GoodFerrousMetals, GoodAlloys, GoodTimber, GoodRadioactives, GoodHeavyPlastics},
	// alcohol (6) — +Luxuries, +Narcotics, +LiquorWines, +Furs?, +Water?, +LiveAnimals?
	{GoodLiquorWines, GoodLuxuries, GoodNarcotics, GoodWater, GoodLiveAnimals, GoodFurs},
	// arms (6) — +Robots, +Drone, +Machinery?, +Computers?
	{GoodFirearms, GoodBattleWeapons, GoodMilitaryFuel, GoodRobots, GoodDrone, GoodComputers},
	// narcotics (6) — +NerveGas, +Slaves, +Narcotics, +LiquorWines?, +Medicines?, +AlienItems?
	{GoodNarcotics, GoodNerveGas, GoodSlaves, GoodLiquorWines, GoodMedicines, GoodAlienItems},
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

// StationPrice — цена станции сектора от общего стока (сумма TargetStock
// по маппингу PlayerGoodMap) и суммарного count. База — OldGoodsMarket
// (Min/Avg/Max), без рас/правительств/эко/радиуса/pop (агрегат сектора).
// Логика реакции на count/stock — как в UpdateMarketState (targetPrice),
// но без PriceState: targetPrice сразу становится purchase.
func (w *World) StationPrice(pg PlayerGood, count, stock int) (buy, sell float64) {
	gm := &OldGoodsMarket[pg]
	ts := float64(stock)
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
	sell = math.Max(buy/2+1,
		math.Round(target*RemapClamped(c, ts, ts*2.2, 0.99, 0.5)-1))
	return buy, sell
}

// phaseStations — распределение Balance станции по планетам сектора.
//   Balance > 0 (sell): заливаем дефицитные планеты (Count < TargetStock*2.5)
//     до 2.5×, Balance уменьшаем на добавленное.
//   Balance < 0 (buy): забираем с планет избыток (Count > TargetStock*0.25)
//     до 0.25×, Balance уменьшаем (по модулю) на взятое.
// Пока Balance не 0 и есть подходящие планеты. Остаток висит.
// Вызывать каждый тик (делитель — позже).
func (w *World) phaseStations() {
	for _, st := range w.Stations {
		if st == nil {
			continue
		}
		for pg := 0; pg < int(PlayerGoodCount); pg++ {
			bal := st.Balance[pg]
			if bal == 0 {
				continue
			}

			if bal > 0 {
				// sell: заливаем до 2.5× TargetStock.
				for _, g := range PlayerGoodMap[pg] {
					if bal <= 0 {
						break
					}
					for _, starID := range w.sectorStarIDs(st.Sector) {
						if bal <= 0 {
							break
						}
						for _, p := range w.PlanetMap[starID] {
							if bal <= 0 {
								break
							}
							if !p.Habitable || p.IsPirateLair() {
								continue
							}
							m := &p.Market[g]
							ts := m.TargetStock
							if ts < 1 {
								ts = 1
							}
							ceiling := int(float64(ts) * 2.5)
							if m.Count >= ceiling {
								continue
							}
							add := ceiling - m.Count
							if add > bal {
								add = bal
							}
							m.Count += add
							bal -= add
						}
					}
				}
				st.Balance[pg] = bal
			} else {
				// buy: забираем до 0.25× TargetStock.
				need := -bal
				for _, g := range PlayerGoodMap[pg] {
					if need <= 0 {
						break
					}
					for _, starID := range w.sectorStarIDs(st.Sector) {
						if need <= 0 {
							break
						}
						for _, p := range w.PlanetMap[starID] {
							if need <= 0 {
								break
							}
							if !p.Habitable || p.IsPirateLair() {
								continue
							}
							m := &p.Market[g]
							ts := m.TargetStock
							if ts < 1 {
								ts = 1
							}
							floor := int(float64(ts) * 0.25)
							if m.Count <= floor {
								continue
							}
							take := m.Count - floor
							if take > need {
								take = need
							}
							m.Count -= take
							need -= take
						}
					}
				}
				st.Balance[pg] = -need
			}
		}
	}
}

// sectorStarIDs — звёзды сектора (кэш не нужен: 72×~8 звёзд, дёшево).
func (w *World) sectorStarIDs(sector int) []int {
	ids := make([]int, 0, 16)
	for _, s := range w.Stars {
		if s.Sector == sector {
			ids = append(ids, s.ID)
		}
	}
	return ids
}