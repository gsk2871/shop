package faq

import (
	"strings"
	"testing"
)

func TestValidation(t *testing.T) {
	for _, row := range []FAQ{{Question: " ", Answer: "A"}, {Question: "Q", Answer: "\n"}, {Question: strings.Repeat("a", MaxQuestionLength+1), Answer: "A"}, {Question: "Q", Answer: strings.Repeat("a", MaxAnswerLength+1)}, {Question: "Q", Answer: "A", Sort: 2147483648}} {
		if row.Validate() == nil {
			t.Fatal("invalid FAQ accepted")
		}
	}
	row := FAQ{Question: " Q ", Answer: " A\nB ", Sort: -1}
	if err := row.Validate(); err != nil || row.Question != "Q" || row.Answer != "A\nB" {
		t.Fatal(row, err)
	}
}
