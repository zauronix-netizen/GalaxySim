package sim

func (w *World) CalculateFuelCost(amount float64, owner OwnerId) float64 {
	value := amount
	// Инфляция топлива выключена для MMO (чтобы логистика не умирала со временем).
	// value *= RemapClamped(float64(w.Day), 1000, 15000, 1, 10)
	if owner != oiUninhabited {
		value *= ownerFuelPriceFactor(owner)
	}
	value *= Difficulty.GoodsEventDurationFactor
	return value
}

func ownerFuelPriceFactor(o OwnerId) float64 {
	switch o {
	case oiMaloc:
		return 0.85
	case oiPeleng:
		return 0.95
	case oiHuman:
		return 1.0
	case oiFeyan:
		return 1.05
	case oiGaal:
		return 1.15
	case oiPirate:
		return 1.2
	}
	return 1.0
}