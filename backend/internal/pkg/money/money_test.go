package money

import "testing"

func TestDecimalRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		input     string
		amount    int64
		formatted string
	}{{"12.34", 1234, "12.34"}, {"0.01", 1, "0.01"}, {"0", 0, "0.00"}, {"0012.3", 1230, "12.30"}, {"92233720368547758.07", 9223372036854775807, "92233720368547758.07"}} {
		amount, err := Parse(tc.input, 2)
		if err != nil || amount != tc.amount {
			t.Fatal(tc, amount, err)
		}
		formatted, err := Format(amount, 2)
		if err != nil || formatted != tc.formatted {
			t.Fatal(formatted, err)
		}
	}
	for _, bad := range []string{"-1", "1e2", "+1", "1.001", "1.", ".1", "NaN", "92233720368547758.08", " 1.00"} {
		if _, err := Parse(bad, 2); err == nil {
			t.Fatal("invalid amount accepted", bad)
		}
	}
}
