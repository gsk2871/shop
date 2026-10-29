package money

import "testing"

func TestQuoteDiscount(t *testing.T) {
	for _, tc := range []struct {
		sale, original int64
		valid          bool
		discount       string
	}{
		{1000, 1000, true, ""}, {800, 1000, true, "USD 2.00"}, {0, 1000, true, "USD 10.00"},
		{1100, 1000, false, ""}, {-1, 1000, false, ""}, {9223372036854775806, 9223372036854775807, true, "USD 0.01"},
	} {
		got, err := Quote(tc.sale, tc.original, 2, "USD")
		if !tc.valid {
			if err == nil {
				t.Fatal("invalid quote accepted", tc)
			}
			continue
		}
		if err != nil || got.DisplayDiscount != tc.discount {
			t.Fatal(tc, got, err)
		}
	}
}
