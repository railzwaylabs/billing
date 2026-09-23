package apikey

import (
	"bytes"
	"strings"
	"testing"

	"github.com/railzwaylabs/billing/internal/iam/domain"
)

func TestGenerateAndParseAPIKey(t *testing.T) {
	codec, err := NewCodec(strings.Repeat("p", 32))
	if err != nil {
		t.Fatal(err)
	}
	generated, err := codec.Generate(domain.APIKey{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(generated.RawKey, "sk_live_") {
		t.Fatalf("unexpected prefix: %s", generated.RawKey)
	}
	keyID, hash, err := codec.ParseAndHash(generated.RawKey)
	if err != nil {
		t.Fatal(err)
	}
	if keyID != generated.APIKey.KeyID || !bytes.Equal(hash, generated.Hash) {
		t.Fatal("parsed key does not match generated key")
	}
}
