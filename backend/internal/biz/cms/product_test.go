package cms

import "testing"

func TestValidateProduct(t *testing.T) {
	valid := Product{Name: "Research material", Slug: "research-material", Status: "draft", Images: []string{"/uploads/example.png", "https://example.com/image.png"}}
	if ValidateProduct(valid) != nil {
		t.Fatal("valid gallery rejected")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*Product)
	}{
		{"blank name", func(p *Product) { p.Name = " " }},
		{"unsafe slug", func(p *Product) { p.Slug = "../secret" }},
		{"unsafe image", func(p *Product) { p.Images = []string{"javascript:alert(1)"} }},
		{"blank image", func(p *Product) { p.Images = []string{""} }},
		{"gallery limit", func(p *Product) { p.Images = make([]string, 21) }},
		{"invalid status", func(p *Product) { p.Status = "deleted" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := valid
			tc.mutate(&p)
			if ValidateProduct(p) == nil {
				t.Fatal("invalid product accepted")
			}
		})
	}
}
