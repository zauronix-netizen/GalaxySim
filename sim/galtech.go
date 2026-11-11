package sim

import "math"

// computeGalaxyTechLevel пересчитывает ГТУ: максимальное N, при котором
// хотя бы ceil(total × 0.51) обитаемых планет имеют TechLevel >= N.
// Только вверх (без отката). Вызывать раз в TurnsPerYear.
func (w *World) computeGalaxyTechLevel() {
	var counts [16]int
	total := 0
	for _, p := range w.Planets {
		if !p.Habitable {
			continue
		}
		total++
		tl := p.TechLevel
		if tl < 0 {
			tl = 0
		}
		if tl > 15 {
			tl = 15
		}
		counts[tl]++
	}
	if total == 0 {
		return
	}
	threshold := int(math.Ceil(float64(total) * 0.51))

	// Идём сверху вниз: ищем максимальный N, где >= threshold планет с TL>=N
	for n := 15; n >= 0; n-- {
		cnt := 0
		for tl := n; tl <= 15; tl++ {
			cnt += counts[tl]
		}
		if cnt >= threshold {
			if n > w.GalaxyTechLevel {
				w.GalaxyTechLevel = n
			}
			return
		}
	}
}