// Package money converts decimal strings and signed BIGINT units without floats.
package money

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var decimal = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)

func Parse(value string, minorUnit int) (int64, error) {
	if minorUnit < 0 || minorUnit > 6 || len(value) > 100 || !decimal.MatchString(value) {
		return 0, errors.New("invalid amount")
	}
	parts := strings.Split(value, ".")
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > minorUnit {
		return 0, errors.New("amount exceeds currency precision")
	}
	digits := strings.TrimLeft(parts[0], "0") + fraction + strings.Repeat("0", minorUnit-len(fraction))
	if digits == "" {
		digits = "0"
	}
	amount, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || amount < 0 {
		return 0, errors.New("amount overflow")
	}
	return amount, nil
}
func Format(amount int64, minorUnit int) (string, error) {
	if amount < 0 || minorUnit < 0 || minorUnit > 6 {
		return "", errors.New("invalid amount")
	}
	digits := strconv.FormatInt(amount, 10)
	if minorUnit == 0 {
		return digits, nil
	}
	if len(digits) <= minorUnit {
		digits = strings.Repeat("0", minorUnit+1-len(digits)) + digits
	}
	return digits[:len(digits)-minorUnit] + "." + digits[len(digits)-minorUnit:], nil
}

type Price struct {
	Currency     string `json:"currency"`
	SalePrice    string `json:"salePrice"`
	DisplayPrice string `json:"displayPrice"`
}
