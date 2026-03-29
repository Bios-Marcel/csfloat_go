package csfloat_test

import (
	"testing"

	csfloat "github.com/Bios-Marcel/csfloat_go"
)

func Test_ApplyFee(t *testing.T) {
	type testCase struct {
		value                   int
		fee                     float64
		expectedValueWithoutFee int
		expectedAppliedFee      int
	}

	testCases := []testCase{
		{
			value:                   0,
			fee:                     0.02,
			expectedValueWithoutFee: 0,
			expectedAppliedFee:      0,
		},
		{
			value:                   0,
			fee:                     0,
			expectedValueWithoutFee: 0,
			expectedAppliedFee:      0,
		},
		{
			value:                   3,
			fee:                     0.02,
			expectedValueWithoutFee: 2,
			expectedAppliedFee:      1,
		},
		{
			value:                   100,
			fee:                     0.02,
			expectedValueWithoutFee: 98,
			expectedAppliedFee:      2,
		},
		{
			value:                   101,
			fee:                     0.02,
			expectedValueWithoutFee: 98,
			expectedAppliedFee:      3,
		},
		{
			value:                   200,
			fee:                     0.02,
			expectedValueWithoutFee: 196,
			expectedAppliedFee:      4,
		},
		{
			value:                   201,
			fee:                     0.02,
			expectedValueWithoutFee: 196,
			expectedAppliedFee:      5,
		},
	}

	for _, tc := range testCases {
		valueWithoutFee, appliedFee := csfloat.ApplyFee(tc.value, tc.fee)
		if valueWithoutFee != tc.expectedValueWithoutFee || appliedFee != tc.expectedAppliedFee {
			t.Errorf("ApplyFee(%d, %f) = (%d, %d), want (%d, %d)", tc.value, tc.fee, valueWithoutFee, appliedFee, tc.expectedValueWithoutFee, tc.expectedAppliedFee)
		}
	}
}
