package branding

import (
	"strings"
	"testing"
)

func TestSocialAccountValidation(t *testing.T) {
	valid := SocialAccount{Name: "Research channel", Platform: "youtube", Link: "https://www.youtube.com/@research", IconMediaID: "1", Enabled: true}
	if err := ValidateSocialAccounts([]SocialAccount{valid, valid}); err != nil {
		t.Fatal("multiple accounts on one platform must be accepted", err)
	}
	for _, link := range []string{"javascript:alert(1)", "data:text/html,test", "//example.com", "http://example.com", "https://user:password@example.com", "https://", "https://example.com\\evil", "https://example.com/\n", "https://example.com/ a"} {
		t.Run(link, func(t *testing.T) {
			account := valid
			account.Link = link
			account.Enabled = false
			if ValidateSocialAccounts([]SocialAccount{account}) == nil {
				t.Fatal("unsafe link accepted, including on a disabled account")
			}
		})
	}
	for _, change := range []func(*SocialAccount){
		func(a *SocialAccount) { a.Name = " " },
		func(a *SocialAccount) { a.Name = strings.Repeat("a", 101) },
		func(a *SocialAccount) { a.Platform = "unknown" },
		func(a *SocialAccount) { a.IconMediaID = "" },
	} {
		account := valid
		change(&account)
		if ValidateSocialAccounts([]SocialAccount{account}) == nil {
			t.Fatal("invalid account accepted", account)
		}
	}
	if ValidateSocialAccounts(make([]SocialAccount, MaxSocialAccounts+1)) == nil {
		t.Fatal("account limit ignored")
	}
	if ValidateSocialAccounts(nil) != nil {
		t.Fatal("legacy settings without accounts rejected")
	}
}

func TestPublicSocialAccounts(t *testing.T) {
	accounts := []SocialAccount{
		{Name: "first", Platform: "x", Link: "https://x.com/first", Icon: "/uploads/first.png", Enabled: true},
		{Name: "hidden", Platform: "x", Enabled: false},
		{Name: "last", Platform: "facebook", Enabled: true},
	}
	public := PublicSocialAccounts(accounts)
	if len(public) != 2 || public[0].Name != "first" || public[1].Name != "last" || public[0].Icon != accounts[0].Icon || public[0].Link != accounts[0].Link {
		t.Fatal("public accounts must filter disabled entries and preserve order", public)
	}
	if PublicSocialAccounts(nil) == nil {
		t.Fatal("empty accounts must serialize as an array")
	}
}
