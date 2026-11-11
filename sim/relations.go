package sim

import "math"

func (w *World) ApplyIllegalGoodsTradeRelationsPenalty(value float64) {
	delta := int(-math.Round(RemapClamped(value, 100, 1000, 20, 70)))
	if delta < -70 {
		delta = -70
	}
	for a := 0; a < int(RaceCount); a++ {
		for b := 0; b < int(RaceCount); b++ {
			if a == b {
				continue
			}
			w.Rel[a][b] += delta
			if w.Rel[a][b] < 0 {
				w.Rel[a][b] = 0
			}
			if w.Rel[a][b] > 100 {
				w.Rel[a][b] = 100
			}
		}
	}
}

func (w *World) pullRelation(a, b Race, target, step int) {
	if a == b {
		return
	}
	v := w.Rel[a][b]
	if v < target {
		v += step
		if v > target {
			v = target
		}
	} else if v > target {
		v -= step
		if v < target {
			v = target
		}
	}
	w.Rel[a][b] = v
}