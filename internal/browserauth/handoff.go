// Package browserauth implements the one-shot, encrypted extension-to-agent handoff.
// The native host is an untrusted relay: only the requesting agent can open it.
package browserauth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"
)

const Lifetime = 2 * time.Minute
const MaxFrame = 1 << 20
const HostName = "com.passbolt.cli"

var ErrUnavailable = errors.New("no pending browser authorization; start a CLI command with --agent-auth=browser")

// Request is public. Its exact JSON encoding is authenticated as AES-GCM AAD.
type Request struct {
	Version     int    `json:"version"`
	ID          string `json:"id"`
	Origin      string `json:"origin"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
	Challenge   string `json:"challenge"`
	Expires     int64  `json:"expires"`
}
type Envelope struct {
	ID         string `json:"id"`
	PublicKey  string `json:"public_key"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}
type Payload struct {
	PrivateKey string `json:"private_key"`
	Grant      string `json:"grant"`
	UserID     string `json:"user_id"`
}
type Handoff struct {
	Request  Request
	key      *ecdh.PrivateKey
	verifier string
	consumed bool
}

func NewHandoff(origin, fingerprint string, now time.Time) (*Handoff, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return nil, fmt.Errorf("browser authentication requires an HTTPS server origin without a path, query, or user information")
	}
	if fingerprint != "" {
		if len(fingerprint) != 40 && len(fingerprint) != 64 {
			return nil, fmt.Errorf("invalid account fingerprint")
		}
		for _, c := range fingerprint {
			if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
				return nil, fmt.Errorf("invalid account fingerprint")
			}
		}
		fingerprint = strings.ToUpper(fingerprint)
	}
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	id := make([]byte, 32)
	if _, err = rand.Read(id); err != nil {
		return nil, err
	}
	proof := make([]byte, 32)
	if _, err = rand.Read(proof); err != nil {
		return nil, err
	}
	defer clear(proof)
	verifier := base64.RawURLEncoding.EncodeToString(proof)
	challenge := sha256.Sum256([]byte(verifier))
	return &Handoff{Request: Request{1, base64.RawURLEncoding.EncodeToString(id), strings.TrimSuffix(origin, "/"), fingerprint, base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()), base64.RawURLEncoding.EncodeToString(challenge[:]), now.Add(Lifetime).Unix()}, key: key, verifier: verifier}, nil
}
func (h *Handoff) Open(env Envelope, now time.Time) (Payload, error) {
	var payload Payload
	if h.consumed || h.key == nil || env.ID != h.Request.ID || now.Unix() >= h.Request.Expires {
		return payload, fmt.Errorf("expired, mismatched, or consumed browser handoff")
	}
	public, err := base64.RawURLEncoding.DecodeString(env.PublicKey)
	if err != nil {
		return payload, fmt.Errorf("invalid handoff key")
	}
	peer, err := ecdh.P256().NewPublicKey(public)
	if err != nil {
		return payload, fmt.Errorf("invalid handoff key")
	}
	shared, err := h.key.ECDH(peer)
	if err != nil {
		return payload, err
	}
	defer clear(shared)
	digest := sha256.Sum256(shared)
	defer clear(digest[:])
	block, err := aes.NewCipher(digest[:])
	if err != nil {
		return payload, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return payload, err
	}
	nonce, err := base64.RawURLEncoding.DecodeString(env.Nonce)
	if err != nil || len(nonce) != aead.NonceSize() {
		return payload, fmt.Errorf("invalid handoff nonce")
	}
	sealed, err := base64.RawURLEncoding.DecodeString(env.Ciphertext)
	if err != nil || len(sealed) > MaxFrame {
		return payload, fmt.Errorf("invalid handoff ciphertext")
	}
	aad, _ := json.Marshal(h.Request)
	plain, err := aead.Open(nil, nonce, sealed, aad)
	if err != nil {
		return payload, fmt.Errorf("browser handoff authentication failed")
	}
	defer clear(plain)
	if err = json.Unmarshal(plain, &payload); err != nil {
		return Payload{}, fmt.Errorf("invalid browser handoff")
	}
	if payload.PrivateKey == "" || payload.Grant == "" || payload.UserID == "" {
		return Payload{}, fmt.Errorf("incomplete browser handoff")
	}
	h.consumed = true
	h.key = nil
	return payload, nil
}

// Broker is separate from the serialized command dispatcher: native messages must
// be serviced while an authenticated command is waiting for browser approval.
type Broker struct {
	mu        sync.Mutex
	pending   *Handoff
	responses chan Envelope
	cancels   chan error
	notify    chan struct{}
}

func NewBroker() *Broker {
	return &Broker{
		notify: make(chan struct{}),
	}
}
func (b *Broker) Authorize(ctx context.Context, origin, fingerprint string) (Payload, string, error) {
	h, err := NewHandoff(origin, fingerprint, time.Now())
	if err != nil {
		return Payload{}, "", err
	}
	b.mu.Lock()
	if b.pending != nil {
		b.mu.Unlock()
		return Payload{}, "", fmt.Errorf("browser authorization already pending")
	}
	b.pending = h
	b.responses = make(chan Envelope, 1)
	b.cancels = make(chan error, 1)
	responses := b.responses
	cancels := b.cancels
	notify := b.notify
	b.notify = make(chan struct{})
	b.mu.Unlock()
	if notify != nil {
		close(notify)
	}
	defer func() {
		b.mu.Lock()
		b.pending = nil
		b.responses = nil
		b.cancels = nil
		h.key = nil
		h.verifier = ""
		b.mu.Unlock()
	}()
	timer := time.NewTimer(Lifetime)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return Payload{}, "", ctx.Err()
	case <-timer.C:
		return Payload{}, "", fmt.Errorf("browser authorization timed out")
	case err := <-cancels:
		return Payload{}, "", err
	case envelope := <-responses:
		payload, err := h.Open(envelope, time.Now())
		return payload, h.verifier, err
	}
}
func (b *Broker) Cancel(reason string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pending == nil {
		return ErrUnavailable
	}
	if reason == "" {
		reason = "cancelled by user"
	}
	select {
	case b.cancels <- fmt.Errorf("browser authorization %s", reason):
		return nil
	default:
		return nil
	}
}
func (b *Broker) Pending() (Request, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pending == nil || time.Now().Unix() >= b.pending.Request.Expires {
		return Request{}, ErrUnavailable
	}
	return b.pending.Request, nil
}
func (b *Broker) WaitPending(ctx context.Context) (Request, error) {
	b.mu.Lock()
	if b.pending != nil && time.Now().Unix() < b.pending.Request.Expires {
		req := b.pending.Request
		b.mu.Unlock()
		return req, nil
	}
	notify := b.notify
	if notify == nil {
		b.notify = make(chan struct{})
		notify = b.notify
	}
	b.mu.Unlock()

	select {
	case <-ctx.Done():
		return Request{}, ctx.Err()
	case <-notify:
		b.mu.Lock()
		defer b.mu.Unlock()
		if b.pending != nil && time.Now().Unix() < b.pending.Request.Expires {
			return b.pending.Request, nil
		}
		return Request{}, ErrUnavailable
	}
}
func (b *Broker) Submit(env Envelope) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pending == nil || env.ID != b.pending.Request.ID || time.Now().Unix() >= b.pending.Request.Expires {
		return ErrUnavailable
	}
	select {
	case b.responses <- env:
		return nil
	default:
		return fmt.Errorf("browser response already received")
	}
}

func ReadNative(r io.Reader, v any) error {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return err
	}
	n := binary.LittleEndian.Uint32(header[:])
	if n == 0 || n > MaxFrame {
		return fmt.Errorf("invalid native message size")
	}
	data := make([]byte, n)
	defer clear(data)
	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
func WriteNative(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	defer clear(data)
	if err != nil {
		return err
	}
	if len(data) == 0 || len(data) > MaxFrame {
		return fmt.Errorf("invalid native message size")
	}
	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], uint32(len(data)))
	if _, err = w.Write(header[:]); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
