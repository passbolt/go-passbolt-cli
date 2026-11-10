package browserauth

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func sealTest(t *testing.T, req Request, payload Payload) Envelope {
	t.Helper()
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	peer, err := ecdh.P256().NewPublicKey(decode(t, req.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	shared, err := key.ECDH(peer)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(shared)
	digest := sha256.Sum256(shared)
	block, _ := aes.NewCipher(digest[:])
	aead, _ := cipher.NewGCM(block)
	nonce := make([]byte, aead.NonceSize())
	rand.Read(nonce)
	plain, _ := json.Marshal(payload)
	defer clear(plain)
	aad, _ := json.Marshal(req)
	return Envelope{ID: req.ID, PublicKey: base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()), Nonce: base64.RawURLEncoding.EncodeToString(nonce), Ciphertext: base64.RawURLEncoding.EncodeToString(aead.Seal(nil, nonce, plain, aad))}
}
func decode(t *testing.T, value string) []byte {
	t.Helper()
	b, e := base64.RawURLEncoding.DecodeString(value)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestHandoffBindsResponseAndConsumesOnlyOnce(t *testing.T) {
	h, err := NewHandoff("https://passbolt.example", "01234567890123456789012345678901234567AB", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	want := Payload{PrivateKey: "private", Grant: "grant", UserID: "user"}
	sealed := sealTest(t, h.Request, want)
	got, err := h.Open(sealed, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %#v", got)
	}
	if _, err = h.Open(sealed, time.Now()); err == nil {
		t.Fatal("replayed handoff accepted")
	}
}
func TestHandoffRejectsTamperingWrongRequestAndExpiry(t *testing.T) {
	for _, scenario := range []string{"tamper", "wrong-id", "expired", "wrong-key", "bad-nonce"} {
		t.Run(scenario, func(t *testing.T) {
			h, _ := NewHandoff("https://passbolt.example", "01234567890123456789012345678901234567AB", time.Now())
			env := sealTest(t, h.Request, Payload{PrivateKey: "private", Grant: "grant", UserID: "user"})
			now := time.Now()
			switch scenario {
			case "tamper":
				data := decode(t, env.Ciphertext)
				data[0] ^= 1
				env.Ciphertext = base64.RawURLEncoding.EncodeToString(data)
			case "wrong-id":
				env.ID = "another"
			case "expired":
				now = now.Add(3 * time.Minute)
			case "wrong-key":
				env.PublicKey = "AA"
			case "bad-nonce":
				env.Nonce = "AA"
			}
			if _, err := h.Open(env, now); err == nil {
				t.Fatal("invalid response accepted")
			}
		})
	}
}
func TestHandoffRejectsInsecureOrAmbiguousOrigin(t *testing.T) {
	for _, origin := range []string{"http://passbolt.example", "https://user@passbolt.example", "https://passbolt.example/path", "https://passbolt.example?x=1", "https://passbolt.example#frag"} {
		if _, err := NewHandoff(origin, "01234567890123456789012345678901234567AB", time.Now()); err == nil {
			t.Fatalf("accepted %s", origin)
		}
	}
}
func TestBrokerCancellationAndUnsolicitedResponses(t *testing.T) {
	b := NewBroker()
	if err := b.Submit(Envelope{ID: "unknown"}); err == nil {
		t.Fatal("accepted unsolicited handoff")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, _, err := b.Authorize(ctx, "https://passbolt.example", "01234567890123456789012345678901234567AB")
		done <- err
	}()
	deadline := time.Now().Add(time.Second)
	for {
		if _, err := b.Pending(); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no pending request")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	if err := <-done; err == nil {
		t.Fatal("cancellation ignored")
	}
	if _, err := b.Pending(); err == nil {
		t.Fatal("canceled request retained")
	}
}
func TestBrokerWaitPendingNotified(t *testing.T) {
	b := NewBroker()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	received := make(chan Request, 1)
	go func() {
		req, err := b.WaitPending(ctx)
		if err == nil {
			received <- req
		}
	}()

	time.Sleep(50 * time.Millisecond)
	authCtx, authCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer authCancel()

	go func() {
		_, _, _ = b.Authorize(authCtx, "https://passbolt.example", "01234567890123456789012345678901234567AB")
	}()

	select {
	case req := <-received:
		if req.Origin != "https://passbolt.example" {
			t.Fatalf("unexpected origin: %s", req.Origin)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for pending request notification")
	}
}
func TestBrokerCancel(t *testing.T) {
	b := NewBroker()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		_, _, err := b.Authorize(ctx, "https://passbolt.example", "01234567890123456789012345678901234567AB")
		errChan <- err
	}()

	time.Sleep(50 * time.Millisecond)
	if err := b.Cancel("cancelled by user"); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	select {
	case err := <-errChan:
		if err == nil || !strings.Contains(err.Error(), "cancelled by user") {
			t.Fatalf("expected cancel error, got: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for cancel to unblock Authorize")
	}
}
func TestNativeFraming(t *testing.T) {
	var wire bytes.Buffer
	if err := WriteNative(&wire, map[string]string{"value": "ok"}); err != nil {
		t.Fatal(err)
	}
	if wire.Bytes()[0] == 0 {
		t.Fatal("native messaging must use little endian framing")
	}
	var got map[string]string
	if err := ReadNative(&wire, &got); err != nil || got["value"] != "ok" {
		t.Fatalf("%v %v", got, err)
	}
	if err := ReadNative(bytes.NewReader([]byte{0xff, 0xff, 0xff, 0xff}), &got); err == nil {
		t.Fatal("oversized frame accepted")
	}
}
