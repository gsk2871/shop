package cms

import (
	"strings"
	"time"
)

// Product keeps a shared image gallery, independent of product attributes.
type Product struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	Category        string    `json:"category"`
	Aliases         string    `json:"aliases"`
	CAS             string    `json:"cas"`
	Formula         string    `json:"formula"`
	MolecularWeight string    `json:"molecularWeight"`
	Sequence        string    `json:"sequence"`
	Purity          string    `json:"purity"`
	Specifications  string    `json:"specifications"`
	Storage         string    `json:"storage"`
	Solubility      string    `json:"solubility"`
	Summary         string    `json:"summary"`
	Description     string    `json:"description"`
	Images          []string  `json:"images" gorm:"serializer:json"`
	Status          string    `json:"status"`
	SEOTitle        string    `json:"seoTitle"`
	SEODescription  string    `json:"seoDescription"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func ValidateProduct(p Product) error {
	if strings.TrimSpace(p.Name) == "" || len(p.Name) > 200 || len(p.Slug) > 120 || !slugPattern.MatchString(p.Slug) || len(p.Images) > 20 {
		return ErrInvalid
	}
	for _, field := range []string{p.Category, p.CAS, p.Formula, p.MolecularWeight, p.Purity, p.SEOTitle} {
		if len(field) > 200 {
			return ErrInvalid
		}
	}
	for _, field := range []string{p.Aliases, p.Sequence, p.Specifications, p.Storage, p.Solubility, p.Summary, p.SEODescription} {
		if len(field) > 5000 {
			return ErrInvalid
		}
	}
	if len(p.Description) > 100000 {
		return ErrInvalid
	}
	for _, img := range p.Images {
		if img == "" || len(img) > 2000 || !SafeURL(img) {
			return ErrInvalid
		}
	}
	switch p.Status {
	case "draft", "published", "offline":
		return nil
	}
	return ErrInvalid
}
