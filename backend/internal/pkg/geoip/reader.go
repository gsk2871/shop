// Package geoip owns an immutable local country database for one resource generation.
package geoip

import (
	"github.com/oschwald/maxminddb-golang/v2"
	"net/netip"
)

type Reader struct{ db *maxminddb.Reader }

func Open(path string) (*Reader, error) {
	r := &Reader{}
	if path == "" {
		return r, nil
	}
	db, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	if err := db.Verify(); err != nil {
		_ = db.Close()
		return nil, err
	}
	r.db = db
	return r, nil
}
func (r *Reader) Available() bool { return r != nil && r.db != nil }
func (r *Reader) Close() {
	if r.Available() {
		_ = r.db.Close()
	}
}
func (r *Reader) Country(ip netip.Addr) (string, error) {
	if !r.Available() || !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return "", nil
	}
	var record struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}
	err := r.db.Lookup(ip).Decode(&record)
	return record.Country.ISOCode, err
}
