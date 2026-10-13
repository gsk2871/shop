package faq

import (
	"math"
	"peptide-site/internal/pkg/errdef"
	"strings"
)

const MaxQuestionLength = 500
const MaxAnswerLength = 10000

type FAQ struct {
	ID              string `json:"id,omitempty" gorm:"-"`
	Question        string `json:"question"`
	Answer          string `json:"answer"`
	Enabled         bool   `json:"enabled"`
	Sort            int    `json:"sort"`
	Version         int64  `json:"version"`
	ExpectedVersion int64  `json:"expectedVersion,omitempty" gorm:"-"`
}

func (c *FAQ) Validate() error {
	c.Question = strings.TrimSpace(c.Question)
	c.Answer = strings.TrimSpace(c.Answer)
	if c.Question == "" || len(c.Question) > MaxQuestionLength || c.Answer == "" || len(c.Answer) > MaxAnswerLength || c.Sort < math.MinInt32 || c.Sort > math.MaxInt32 {
		return errdef.Invalid()
	}
	return nil
}
