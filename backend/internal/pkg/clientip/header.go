// Package clientip reads visitor IPs explicitly supplied by the website proxy.
package clientip

import (
	"errors"
	"net/http"
	"net/netip"
	"strings"
)

var ErrInvalid = errors.New("missing or invalid visitor IP header")

// FromHeader never falls back to RemoteAddr: backend callers may be Next.js or
// a reverse proxy rather than the visitor. The ingress must overwrite visitor
// IP headers. X-Client-IP is preferred; standard forwarding headers are accepted
// for integrations that do not provide the dedicated header.
func FromHeader(header http.Header) (netip.Addr, error) {
	for _, name := range []string{"X-Client-IP", "X-Forwarded-For", "X-Real-IP"} {
		values := header.Values(name)
		if len(values) == 0 {
			continue
		}
		if len(values) != 1 {
			return netip.Addr{}, ErrInvalid
		}
		raw := strings.TrimSpace(values[0])
		if name == "X-Forwarded-For" {
			hops := strings.Split(raw, ",")
			if len(hops) > 32 {
				return netip.Addr{}, ErrInvalid
			}
			for _, hop := range hops {
				address, err := netip.ParseAddr(strings.TrimSpace(hop))
				if err != nil || address.Zone() != "" {
					return netip.Addr{}, ErrInvalid
				}
			}
			raw = strings.TrimSpace(hops[0])
		}
		address, err := netip.ParseAddr(raw)
		if err != nil || address.Zone() != "" {
			return netip.Addr{}, ErrInvalid
		}
		return address.Unmap(), nil
	}
	return netip.Addr{}, ErrInvalid
}
