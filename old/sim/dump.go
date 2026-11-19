package sim

import "fmt"

func (w *World) Dump() string {
	s := fmt.Sprintf("=== Day %d ===\n", w.Day)
	s += fmt.Sprintf("Stars: %d  Planets: %d\n", len(w.Stars), len(w.Planets))
	s += fmt.Sprintf("Caravans: %d  Pirates: %d  Liners: %d  Diplomats: %d\n",
		len(w.Caravans), len(w.Pirates), len(w.Liners), len(w.Diplomats))
	s += fmt.Sprintf("Events: %d\n", len(w.Events))
	for i, e := range w.Events {
		if i >= 5 {
			s += fmt.Sprintf("  ... (%d more)\n", len(w.Events)-5)
			break
		}
		s += fmt.Sprintf("  [%s] star=%d good=%d\n", e.Kind, e.Star, e.Good)
	}
	for i, c := range w.Caravans {
		if i >= 3 {
			break
		}
		s += fmt.Sprintf("  car %d: phase=%d routeIdx=%d money=%.0f fuel=%.1f\n",
			c.ID, c.Phase, c.RouteIdx, c.Money, c.Fuel)
	}
	return s
}