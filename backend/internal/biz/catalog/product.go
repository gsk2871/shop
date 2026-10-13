// Package catalog defines the new product aggregate with typed media references.
package catalog

import (
	"peptide-site/internal/pkg/errdef"
	"peptide-site/internal/pkg/money"
	"peptide-site/internal/pkg/richtext"

	"regexp"
	"sort"
	"strconv"
	"strings"
)

const MaxSpecs = 6
const MaxValues = 20
const MaxSKUs = 500

type Value struct {
	ID   string `json:"id,omitempty"`
	Key  string `json:"key"`
	Name string `json:"name"`
	Sort int    `json:"sort"`
}
type Spec struct {
	ID     string  `json:"id,omitempty"`
	Key    string  `json:"key"`
	Name   string  `json:"name"`
	Sort   int     `json:"sort"`
	Values []Value `json:"values"`
}
type SKU struct {
	ID            string       `json:"id,omitempty"`
	Key           string       `json:"key"`
	Enabled       bool         `json:"enabled"`
	Sort          int          `json:"sort"`
	SpecValueKeys []string     `json:"specValueKeys"`
	Currency      string       `json:"currency"`
	SalePrice     *string      `json:"salePrice"`
	Price         *money.Price `json:"price,omitempty"`
}
type Product struct {
	ImageMediaID    string            `json:"imageMediaId"`
	VideoMediaID    string            `json:"videoMediaId"`
	ReportMediaID   string            `json:"reportMediaId"`
	Image           string            `json:"image"`
	Video           string            `json:"video"`
	Report          string            `json:"report"`
	ID              string            `json:"id,omitempty"`
	Name            string            `json:"name"`
	Slug            string            `json:"slug"`
	CategoryID      string            `json:"categoryId"`
	Category        string            `json:"category"`
	Aliases         string            `json:"aliases"`
	CAS             string            `json:"cas"`
	Formula         string            `json:"formula"`
	MolecularWeight string            `json:"molecularWeight"`
	Sequence        string            `json:"sequence"`
	Purity          string            `json:"purity"`
	Storage         string            `json:"storage"`
	Solubility      string            `json:"solubility"`
	Summary         string            `json:"summary"`
	Detail          richtext.Document `json:"detail"`
	Status          string            `json:"status"`
	SEOTitle        string            `json:"seoTitle"`
	SEODescription  string            `json:"seoDescription"`
	Version         int64             `json:"version"`
	ExpectedVersion int64             `json:"expectedVersion,omitempty"`
	Specs           []Spec            `json:"specs"`
	SKUs            []SKU             `json:"skus"`
	FromPrice       *money.Price      `json:"fromPrice,omitempty"`
	UpdatedAt       string            `json:"updatedAt,omitempty"`
}

var slug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var key = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,100}$`)

func ParseID(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 || strconv.FormatInt(value, 10) != raw {
		return 0, errdef.Invalid()
	}
	return value, nil
}
func Validate(p *Product, defaultCurrency string) error {
	if id, err := ParseID(p.ImageMediaID); err != nil || id == 0 {
		return errdef.Invalid()
	}
	for _, raw := range []string{p.VideoMediaID, p.ReportMediaID} {
		if _, err := ParseID(raw); err != nil {
			return err
		}
	}
	if id, err := ParseID(p.CategoryID); err != nil || id == 0 {
		return errdef.Invalid()
	}
	if strings.TrimSpace(p.Name) == "" || len(p.Name) > 200 || !slug.MatchString(p.Slug) || len(p.Slug) > 120 || len(p.Specs) > MaxSpecs || len(p.SKUs) > MaxSKUs {
		return errdef.Invalid()
	}
	if p.Status != "draft" && p.Status != "published" && p.Status != "offline" {
		return errdef.Invalid()
	}
	for _, field := range []string{p.Category, p.CAS, p.Formula, p.MolecularWeight, p.Purity, p.SEOTitle} {
		if len(field) > 200 {
			return errdef.Invalid()
		}
	}
	for _, field := range []string{p.Aliases, p.Sequence, p.Storage, p.Solubility, p.Summary, p.SEODescription} {
		if len(field) > 5000 {
			return errdef.Invalid()
		}
	}
	dimensions := map[string]bool{}
	values := map[string]string{}
	names := map[string]bool{}
	for _, spec := range p.Specs {
		if _, err := ParseID(spec.ID); err != nil {
			return err
		}
		if !key.MatchString(spec.Key) || dimensions[spec.Key] || strings.TrimSpace(spec.Name) == "" || len(spec.Name) > 100 || names[strings.ToLower(strings.TrimSpace(spec.Name))] || len(spec.Values) == 0 || len(spec.Values) > MaxValues {
			return errdef.Invalid()
		}
		dimensions[spec.Key] = true
		names[strings.ToLower(strings.TrimSpace(spec.Name))] = true
		valueNames := map[string]bool{}
		for _, value := range spec.Values {
			if _, err := ParseID(value.ID); err != nil {
				return err
			}
			name := strings.ToLower(strings.TrimSpace(value.Name))
			if !key.MatchString(value.Key) || values[value.Key] != "" || name == "" || len(value.Name) > 100 || valueNames[name] {
				return errdef.Invalid()
			}
			values[value.Key] = spec.Key
			valueNames[name] = true
		}
	}
	if len(p.Specs) == 0 && len(p.SKUs) == 0 {
		p.SKUs = []SKU{{Key: "default", Enabled: true, Currency: defaultCurrency, SpecValueKeys: []string{}}}
	}
	if len(p.Specs) == 0 && len(p.SKUs) != 1 {
		return errdef.Invalid()
	}
	if len(p.Specs) > 0 && len(p.SKUs) == 0 && p.Status == "published" {
		return errdef.Invalid()
	}
	skuKeys := map[string]bool{}
	combinations := map[string]bool{}
	for j := range p.SKUs {
		sku := &p.SKUs[j]
		if _, err := ParseID(sku.ID); err != nil {
			return err
		}
		if !key.MatchString(sku.Key) || skuKeys[sku.Key] || len(sku.SpecValueKeys) != len(p.Specs) {
			return errdef.Invalid()
		}
		skuKeys[sku.Key] = true
		seen := map[string]bool{}
		for _, value := range sku.SpecValueKeys {
			dimension := values[value]
			if dimension == "" || seen[dimension] {
				return errdef.Invalid()
			}
			seen[dimension] = true
		}
		ordered := append([]string(nil), sku.SpecValueKeys...)
		sort.Strings(ordered)
		combination := strings.Join(ordered, "|")
		if combinations[combination] {
			return errdef.Invalid()
		}
		combinations[combination] = true
		if sku.Currency == "" {
			if sku.ID != "" {
				return errdef.Invalid()
			}
			sku.Currency = defaultCurrency
		}
		if len(sku.Currency) != 3 {
			return errdef.Invalid()
		}
	}
	return nil
}

type PublicValue struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}
type PublicSpec struct {
	ID     string        `json:"id"`
	Key    string        `json:"key"`
	Name   string        `json:"name"`
	Values []PublicValue `json:"values"`
}
type PublicSKU struct {
	ID            string       `json:"id"`
	Key           string       `json:"key"`
	SpecValueKeys []string     `json:"specValueKeys"`
	Price         *money.Price `json:"price"`
}

func Public(p Product) map[string]any {
	skus := []PublicSKU{}
	for _, sku := range p.SKUs {
		if sku.Enabled && sku.Price != nil {
			skus = append(skus, PublicSKU{sku.ID, sku.Key, sku.SpecValueKeys, sku.Price})
		}
	}
	specs := []PublicSpec{}
	for _, spec := range p.Specs {
		values := []PublicValue{}
		for _, value := range spec.Values {
			values = append(values, PublicValue{value.ID, value.Key, value.Name})
		}
		specs = append(specs, PublicSpec{spec.ID, spec.Key, spec.Name, values})
	}
	return map[string]any{"image": p.Image, "video": p.Video, "report": p.Report, "id": p.ID, "name": p.Name, "slug": p.Slug, "category": p.Category, "categoryId": p.CategoryID, "aliases": p.Aliases, "cas": p.CAS, "formula": p.Formula, "molecularWeight": p.MolecularWeight, "sequence": p.Sequence, "purity": p.Purity, "storage": p.Storage, "solubility": p.Solubility, "summary": p.Summary, "detail": p.Detail, "specs": specs, "skus": skus, "fromPrice": p.FromPrice, "seoTitle": p.SEOTitle, "seoDescription": p.SEODescription}
}
