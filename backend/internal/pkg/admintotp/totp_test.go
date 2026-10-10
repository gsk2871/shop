package admintotp

import (
	"encoding/base32"
	"encoding/base64"
	"testing"
	"time"
)

func TestRFC6238AndEncryption(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))
	code, err := Code(secret, 59/30)
	if err != nil || code != "287082" {
		t.Fatalf("RFC vector: %s %v", code, err)
	}
	if step, ok := Match(secret, code, time.Unix(59, 0)); !ok || step != 1 {
		t.Fatal("valid code rejected")
	}
	if _, ok := Match(secret, "bad", time.Unix(59, 0)); ok {
		t.Fatal("malformed code accepted")
	}
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	codec, err := New("one", map[string]string{"one": key})
	if err != nil {
		t.Fatal(err)
	}
	id, encrypted, err := codec.Encrypt(secret)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := codec.Decrypt(id, encrypted)
	if err != nil || plain != secret {
		t.Fatal("cipher round trip failed")
	}
	if _, err := codec.Decrypt("missing", encrypted); err == nil {
		t.Fatal("missing key accepted")
	}
	b, _ := base64.StdEncoding.DecodeString(encrypted)
	b[len(b)-1] ^= 1
	if _, err := codec.Decrypt(id, base64.StdEncoding.EncodeToString(b)); err == nil {
		t.Fatal("tampered cipher accepted")
	}
}
