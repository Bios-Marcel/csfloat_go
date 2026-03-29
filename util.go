package csfloat

import "math"

// ApplyFee uses the given value and applies the given fee. It returns the value with the fee deducted and the fee.
// For example, ApplyFee(100, 0.02) returns (98, 2).
// As float always ceils fees, ApplyFee(101, 0.02) returns (98, 3).
func ApplyFee(value int, fee float64) (valueWithoutFee int, appliedFee int) {
	appliedFee = int(math.Ceil(float64(value) * fee))
	valueWithoutFee = value - appliedFee
	return
}

// FloatRange returns the float range for the given quality (fn, mw, ...).
func FloatRange(f float64) (float64, float64) {
	if f < 0.07 {
		return 0.0, 0.07
	} else if f < 0.15 {
		return 0.07, 0.15
	} else if f < 0.38 {
		return 0.15, 0.38
	} else if f < 0.45 {
		return 0.38, 0.45
	}

	return 0.45, 1.0
}
