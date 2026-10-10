// Package admintotp implements encrypted RFC 6238 secrets and six digit codes.
package admintotp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Codec struct {
	Active string
	Keys   map[string][]byte
}

func New(active string, keys map[string]string) (*Codec, error) {
	c := &Codec{Active: active, Keys: map[string][]byte{}}
	for id, encoded := range keys {
		key, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(key) != 32 || id == "" {
			return nil, errors.New("TOTP keys must be base64 encoded 32 byte keys")
		}
		c.Keys[id] = key
	}
	if c.Keys[active] == nil {
		return nil, errors.New("TOTP active key is required")
	}
	return c, nil
}
func (c *Codec) Encrypt(secret string) (string, string, error) {
	block, err := aes.NewCipher(c.Keys[c.Active])
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", "", err
	}
	encrypted := gcm.Seal(nonce, nonce, []byte(secret), []byte(c.Active))
	return c.Active, base64.StdEncoding.EncodeToString(encrypted), nil
}
func (c *Codec) Decrypt(id, encoded string) (string, error) {
	key, ok := c.Keys[id]
	if !ok {
		return "", errors.New("referenced TOTP key is missing")
	}
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New("invalid TOTP ciphertext")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(b) < gcm.NonceSize() {
		return "", errors.New("invalid TOTP ciphertext")
	}
	plain, err := gcm.Open(nil, b[:gcm.NonceSize()], b[gcm.NonceSize():], []byte(id))
	if err != nil {
		return "", errors.New("TOTP decryption failed")
	}
	return string(plain), nil
}
func Secret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}
func URI(issuer, user, secret string) string {
	q := url.Values{"secret": {secret}, "issuer": {issuer}, "algorithm": {"SHA1"}, "digits": {"6"}, "period": {"30"}}
	return "otpauth://totp/" + url.PathEscape(issuer+":"+user) + "?" + q.Encode()
}
func Code(secret string, step int64) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(step))
	h := hmac.New(sha1.New, key)
	_, _ = h.Write(b[:])
	digest := h.Sum(nil)
	offset := digest[len(digest)-1] & 15
	value := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", value%1000000), nil
}
func Match(secret, code string, now time.Time) (int64, bool) {
	if len(code) != 6 {
		return 0, false
	}
	if _, err := strconv.Atoi(code); err != nil {
		return 0, false
	}
	step := now.Unix() / 30
	for _, candidate := range []int64{step, step - 1, step + 1} {
		value, err := Code(secret, candidate)
		if err == nil && subtle.ConstantTimeCompare([]byte(value), []byte(code)) == 1 {
			return candidate, true
		}
	}
	return 0, false
}
