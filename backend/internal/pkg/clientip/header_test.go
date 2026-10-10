package clientip

import (
	"net/http"
	"testing"
)

func TestVisitorIPHeaders(t *testing.T) {
	for _, tc := range []struct {
		name    string
		headers http.Header
		want    string
	}{
		{"dedicated", http.Header{"X-Client-Ip": {"203.0.113.4"}, "X-Forwarded-For": {"198.51.100.3"}}, "203.0.113.4"},
		{"forwarded IPv6", http.Header{"X-Forwarded-For": {"2001:db8::4, 127.0.0.1"}}, "2001:db8::4"},
		{"real IPv4 mapped", http.Header{"X-Real-Ip": {"::ffff:203.0.113.4"}}, "203.0.113.4"},
		{"missing", http.Header{}, ""},
		{"invalid preferred", http.Header{"X-Client-Ip": {"bad"}, "X-Forwarded-For": {"203.0.113.4"}}, ""},
		{"invalid hop", http.Header{"X-Forwarded-For": {"203.0.113.4, bad"}}, ""},
		{"duplicate", http.Header{"X-Client-Ip": {"203.0.113.4", "198.51.100.3"}}, ""},
		{"address with port", http.Header{"X-Client-Ip": {"203.0.113.4:80"}}, ""},
		{"zone", http.Header{"X-Client-Ip": {"fe80::1%eth0"}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			address, err := FromHeader(tc.headers)
			if tc.want == "" {
				if err == nil {
					t.Fatal("invalid header accepted")
				}
				return
			}
			if err != nil || address.String() != tc.want {
				t.Fatal(address, err)
			}
		})
	}
}
