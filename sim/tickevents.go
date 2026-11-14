package sim

/*

import (
	"log" // <-- Добавь, если еще нет
)

*/

type TickEvent struct {
	PlanetID  int
	StarID    int
	Good      Good
	Deficit   bool
	Processed bool // первый трейдер в этом тике пометил «взял»
}

// checkGoodsCounters — копит дни в крайних состояниях склада и спавнит трейдеров.
// Вызывать в phaseEconomy ДО изменений тика (читает Count прошлого тика).
func (w *World) checkGoodsCounters(p *Planet) {
	for g := 0; g < int(GoodCount); g++ {
		m := &p.Market[g]
		ts := m.TargetStock
		if ts < 1 {
			ts = 1
		}
		maxStock := ts * 3
		if maxStock < 10 {
			maxStock = 10
		}

		// Счётчики
		if m.Count >= maxStock {
			p.SurplusDay[g]++
		} else {
			p.SurplusDay[g] = 0
		}
		
		if m.Count == 0 {
			p.DeficitDay[g]++
		} else {
			p.DeficitDay[g] = 0
		}

		// Новость на переизбыток
		if p.SurplusDay[g] >= 7 {
			// Проверка легальности товара на планете
			if w.IsLegal(p, Good(g)) {
				w.TickEvents = append(w.TickEvents, TickEvent{
					PlanetID: p.ID, StarID: p.StarID, Good: Good(g), Deficit: false,
				})
				
			}
			// Сброс в 0 здесь НЕ НУЖЕН. Он произойдёт автоматически, когда трейдер заберёт товар (Count упадёт).
		}
		
		// Новость на дефицит (создаём событие для поиска)
		if p.DeficitDay[g] >= 7 {
			w.TickEvents = append(w.TickEvents, TickEvent{
				PlanetID: p.ID, StarID: p.StarID, Good: Good(g), Deficit: true,
			})
		}
	}
	// Рост планеты: раз в месяц, не выше ГТУ+3, за деньги.
	if w.Day%36 == 0 {
		if p.TechLevel < w.GalaxyTechLevel+3 && p.TechLevel < 15 {
			mult := 1.0 + float64(p.TechLevel-w.GalaxyTechLevel)
			if mult < 1 {
				mult = 1
			}
			cost := totalCost[p.TechLevel] * (w.techDiscountPlanets(p.TechLevel) + 1) * PlanetCostMult * mult
			if p.Money >= cost + p.MoneyRef {
				p.Money -= cost
				p.TechLevel++
				w.planetUpgradeStock(p, p.TechLevel-1, p.TechLevel)
			}
		} else if p.TechLevel > w.GalaxyTechLevel+3 && p.TechLevel < 15 {
			// Упёрлись в потолок ГТУ (или TL=15) — излишек в сейф для банкира.
			// Копим только то, что сверх MoneyRef (чтобы планета не обеднела).
			surplus := p.Money - p.MoneyRef
			if surplus > 0 {
				p.Money -= surplus
				p.MoneySafe += surplus
			}
		}
	}
}

func (w *World) LogTickEvents() {
	// лог убран — перенесён в EconSummary
}