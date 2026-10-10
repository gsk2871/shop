package whatsapp

import (
	_ "embed"
	"strings"
)

// countries.tab is the public-domain IANA tzdb ISO 3166 table (2025-07-01).
//
//go:embed countries.tab
var countryData string

type CountryOption struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func Countries() []CountryOption {
	items := []CountryOption{}
	for _, line := range strings.Split(countryData, "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.SplitN(line, "\t", 2)
		if len(p) == 2 {
			items = append(items, CountryOption{p[0], p[1]})
		}
	}
	return items
}
func ValidCountry(code string) bool {
	for _, c := range Countries() {
		if c.Code == code {
			return true
		}
	}
	return false
}
