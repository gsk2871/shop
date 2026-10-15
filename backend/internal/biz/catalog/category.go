package catalog

import (
	"math"
	"peptide-site/internal/pkg/errdef"
	"strings"
)

const MaxCategoryName = 200

type Category struct {
	CoverMediaID    string `json:"coverMediaId" gorm:"-"`
	Cover           string `json:"cover" gorm:"-"`
	ID              string `json:"id,omitempty" gorm:"-"`
	Name            string `json:"name"`
	Sort            int    `json:"sort"`
	Version         int64  `json:"version"`
	ExpectedVersion int64  `json:"expectedVersion,omitempty" gorm:"-"`
}

func (c *Category) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	if _, err := ParseID(c.CoverMediaID); err != nil {
		return err
	}
	if c.Name == "" || len(c.Name) > MaxCategoryName || c.Sort < math.MinInt32 || c.Sort > math.MaxInt32 {
		return errdef.Invalid()
	}
	return nil
}
