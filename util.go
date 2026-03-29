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
