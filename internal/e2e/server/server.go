// Package server runs a throwaway Passbolt in Docker for integration tests.
//
// Keep identical to internal/e2e/server/server.go in go-passbolt.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/passbolt/go-passbolt/api"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Images are pinned by tag and digest. Update both together.
const (
	mariadbImage         = "mariadb:10.11@sha256:8acd4a5561e8897cfc8a99480f25bcecb10b726fa17b1ec7381c6ef36fa00a79"
	defaultPassboltImage = "passbolt/passbolt:5.14.3-1-ce@sha256:9f97ab2a854019737b4fc30a7fdb598f80a547769d4480a76e6f4b6d0bfbd431"

	// PassboltImageEnv overrides the Passbolt image, to test another release.
	PassboltImageEnv = "PASSBOLT_TEST_IMAGE"
)

// PassboltImage returns the Passbolt image Start will use.
func PassboltImage() string {
	if img := strings.TrimSpace(os.Getenv(PassboltImageEnv)); img != "" {
		return img
	}
	return defaultPassboltImage
}

// Credentials of a Passbolt user.
type Credentials struct {
	UserID     string
	Email      string
	Password   string
	PrivateKey string // armored, locked with Password
}

// Passbolt is a running Passbolt server.
type Passbolt struct {
	BaseURL   string
	container testcontainers.Container
	teardown  []func(context.Context) error
}

// Close stops the containers and network. Safe to call more than once.
func (p *Passbolt) Close(ctx context.Context) error {
	var errs []error
	for _, teardown := range slices.Backward(p.teardown) {
		if err := teardown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	p.teardown = nil
	return errors.Join(errs...)
}

// Start runs MariaDB and Passbolt in Docker. The caller must Close it.
func Start(ctx context.Context) (*Passbolt, error) {
	p := &Passbolt{}
	rollback := func(err error) (*Passbolt, error) {
		_ = p.Close(context.Background())
		return nil, err
	}

	net, err := network.New(ctx)
	if err != nil {
		return rollback(fmt.Errorf("create docker network: %w", err))
	}
	p.teardown = append(p.teardown, net.Remove)

	dbReq := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          mariadbImage,
			Networks:       []string{net.Name},
			NetworkAliases: map[string][]string{net.Name: {"db"}},
			Env: map[string]string{
				"MYSQL_ROOT_PASSWORD": "passbolt",
				"MYSQL_DATABASE":      "passbolt",
				"MYSQL_USER":          "passbolt",
				"MYSQL_PASSWORD":      "passbolt",
			},
			// MariaDB restarts once after init, so wait for the second log.
			WaitingFor: wait.ForLog("ready for connections").
				WithOccurrence(2).
				WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	}
	db, err := testcontainers.GenericContainer(ctx, dbReq)
	if err != nil {
		return rollback(fmt.Errorf("start mariadb: %w", err))
	}
	p.teardown = append(p.teardown, func(ctx context.Context) error { return db.Terminate(ctx) })

	pbReq := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        PassboltImage(),
			Networks:     []string{net.Name},
			ExposedPorts: []string{"80/tcp"},
			Env: map[string]string{
				"DATASOURCES_DEFAULT_HOST":     "db",
				"DATASOURCES_DEFAULT_USERNAME": "passbolt",
				"DATASOURCES_DEFAULT_PASSWORD": "passbolt",
				"DATASOURCES_DEFAULT_DATABASE": "passbolt",
				// Tests use the mapped port; Passbolt does not check the Host header.
				"APP_FULL_BASE_URL":  "http://localhost",
				"PASSBOLT_SSL_FORCE": "false",
			},
			WaitingFor: wait.ForHTTP("/healthcheck/status.json").
				WithPort("80/tcp").
				WithStartupTimeout(3 * time.Minute),
		},
		Started: true,
	}
	pb, err := testcontainers.GenericContainer(ctx, pbReq)
	if err != nil {
		return rollback(fmt.Errorf("start passbolt: %w", err))
	}
	p.teardown = append(p.teardown, func(ctx context.Context) error { return pb.Terminate(ctx) })
	p.container = pb

	host, err := pb.Host(ctx)
	if err != nil {
		return rollback(fmt.Errorf("get passbolt host: %w", err))
	}
	port, err := pb.MappedPort(ctx, "80/tcp")
	if err != nil {
		return rollback(fmt.Errorf("get passbolt mapped port: %w", err))
	}
	p.BaseURL = fmt.Sprintf("http://%s:%s", host, port.Port())

	return p, nil
}

// setupURLRegex matches the setup link printed by register_user.
var setupURLRegex = regexp.MustCompile(`https?://\S+/setup/[A-Za-z]+/[A-Za-z0-9-]+/[A-Za-z0-9-]+`)

// RegisterUser invites a user and returns their ID and setup token. role is
// "user" or "admin".
func (p *Passbolt) RegisterUser(ctx context.Context, email, first, last, role string) (userID, token string, err error) {
	// Run as www-data so the files Passbolt writes have the right owner.
	cmd := []string{
		"su", "-m", "-c",
		fmt.Sprintf("bin/cake passbolt register_user -u %s -f %s -l %s -r %s",
			email, first, last, role),
		"-s", "/bin/sh", "www-data",
	}
	rc, reader, err := p.container.Exec(ctx, cmd)
	if err != nil {
		return "", "", fmt.Errorf("exec register_user: %w", err)
	}
	out, _ := io.ReadAll(reader)
	if rc != 0 {
		return "", "", fmt.Errorf("register_user exit=%d output:\n%s", rc, out)
	}
	match := setupURLRegex.Find(out)
	if match == nil {
		return "", "", fmt.Errorf("could not find setup URL in register_user output:\n%s", out)
	}
	uid, tok, err := parseInviteURL(string(match))
	if err != nil {
		return "", "", fmt.Errorf("parse invite URL %q: %w", match, err)
	}
	return uid, tok, nil
}

// parseInviteURL returns the last two path segments of a setup link.
func parseInviteURL(url string) (userID, token string, err error) {
	parts := strings.Split(url, "/")
	if len(parts) < 4 {
		return "", "", fmt.Errorf("invite URL does not have enough slashes: %q", url)
	}
	return parts[len(parts)-2], strings.TrimSuffix(parts[len(parts)-1], ".json"), nil
}

// EnableV5Resources creates a shared metadata key and allows V5 resources,
// which a new server does not. Call it before other clients log in, since
// clients cache these settings at login. admin must have the admin role.
func (p *Passbolt) EnableV5Resources(ctx context.Context, admin Credentials) error {
	c, err := api.NewClient(nil, "go-passbolt-testenv-bootstrap", p.BaseURL, admin.PrivateKey, admin.Password)
	if err != nil {
		return fmt.Errorf("admin bootstrap client: %w", err)
	}
	if err := c.Login(ctx); err != nil {
		return fmt.Errorf("admin login: %w", err)
	}
	defer func() { _ = c.Logout(ctx) }()

	// The type settings below are refused until a metadata key exists.
	if err := p.createSharedMetadataKey(ctx, c); err != nil {
		return fmt.Errorf("create shared metadata key: %w", err)
	}

	types := api.MetadataTypeSettings{
		DefaultResourceType:        api.PassboltAPIVersionTypeV5,
		DefaultFolderType:          api.PassboltAPIVersionTypeV5,
		DefaultTagType:             api.PassboltAPIVersionTypeV5,
		DefaultCommentType:         api.PassboltAPIVersionTypeV5,
		AllowCreationOfV5Resources: true,
		AllowCreationOfV5Folders:   true,
		AllowCreationOfV5Tags:      true,
		AllowCreationOfV5Comments:  true,
		AllowCreationOfV4Resources: true,
		AllowCreationOfV4Folders:   true,
		AllowCreationOfV4Tags:      true,
		AllowCreationOfV4Comments:  true,
		AllowV4V5Upgrade:           true,
		AllowV4V5Downgrade:         true,
	}
	if _, err := c.DoCustomRequestV5(ctx, "POST", "/metadata/types/settings.json", types, nil); err != nil {
		return fmt.Errorf("post metadata type settings: %w", err)
	}

	keys := api.MetadataKeySettings{
		AllowUsageOfPersonalKeys:   true,
		AllowZeroKnowledgeKeyShare: false,
	}
	if _, err := c.DoCustomRequestV5(ctx, "POST", "/metadata/keys/settings.json", keys, nil); err != nil {
		return fmt.Errorf("post metadata key settings: %w", err)
	}
	return nil
}

// CreateUser registers a user and completes their account setup.
func (p *Passbolt) CreateUser(ctx context.Context, email, first, last, role, password string) (Credentials, error) {
	userID, token, err := p.RegisterUser(ctx, email, first, last, role)
	if err != nil {
		return Credentials{}, err
	}

	c, err := api.NewClient(nil, "go-passbolt-testenv", p.BaseURL, "", "")
	if err != nil {
		return Credentials{}, fmt.Errorf("registration client: %w", err)
	}
	privKey, err := completeSetup(ctx, c, userID, token, password)
	if err != nil {
		return Credentials{}, fmt.Errorf("setup account for %s: %w", email, err)
	}
	return Credentials{
		UserID:     userID,
		Email:      email,
		Password:   password,
		PrivateKey: privKey,
	}, nil
}

// createSharedMetadataKey generates a shared metadata key and gives admin a
// copy of the private key.
func (p *Passbolt) createSharedMetadataKey(ctx context.Context, admin *api.Client) error {
	me, err := admin.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}
	if me.GPGKey == nil {
		return fmt.Errorf("admin user has no GPG key in profile")
	}

	pgp := admin.GetPGPHandle()
	key, err := pgp.KeyGeneration().
		AddUserId("Passbolt Shared Metadata Key", "metadata@"+stripScheme(p.BaseURL)).
		New().
		GenerateKey()
	if err != nil {
		return fmt.Errorf("generate metadata keypair: %w", err)
	}
	defer key.ClearPrivateParams()

	publicArmored, err := key.GetArmoredPublicKey()
	if err != nil {
		return fmt.Errorf("armor metadata public key: %w", err)
	}
	privateArmored, err := key.Armor()
	if err != nil {
		return fmt.Errorf("armor metadata private key: %w", err)
	}
	// Passbolt requires an uppercase fingerprint.
	fingerprint := strings.ToUpper(key.GetFingerprint())

	// Server keys have no passphrase. Signed must be set: the browser
	// extension rejects a zero time.
	payload, err := json.Marshal(api.MetadataPrivateKeyData{
		ObjectType:  "PASSBOLT_METADATA_PRIVATE_KEY",
		Domain:      p.BaseURL,
		Fingerprint: fingerprint,
		ArmoredKey:  privateArmored,
		Passphrase:  "",
		Signed:      api.Time{Time: time.Now()},
	})
	if err != nil {
		return fmt.Errorf("marshal metadata private key data: %w", err)
	}

	adminPubKey, err := crypto.NewKeyFromArmored(me.GPGKey.ArmoredKey)
	if err != nil {
		return fmt.Errorf("parse admin public key: %w", err)
	}
	encrypted, err := admin.EncryptMessageWithKey(adminPubKey, string(payload))
	if err != nil {
		return fmt.Errorf("encrypt metadata private key data: %w", err)
	}

	adminID := me.ID
	body := api.MetadataKey{
		Fingerprint: fingerprint,
		ArmoredKey:  publicArmored,
		MetadataPrivateKeys: []api.MetadataPrivateKey{{
			UserID: &adminID,
			Data:   encrypted,
		}},
	}
	if _, err := admin.DoCustomRequestV5(ctx, "POST", "/metadata/keys.json", body, nil); err != nil {
		return fmt.Errorf("post metadata key: %w", err)
	}
	return nil
}

// stripScheme turns "http://localhost:1234" into "localhost:1234".
func stripScheme(url string) string {
	if _, after, ok := strings.Cut(url, "://"); ok {
		return after
	}
	return url
}

// completeSetup generates the user's key, locked with password, and completes
// account setup. It returns the armored private key.
func completeSetup(ctx context.Context, c *api.Client, userID, token, password string) (string, error) {
	install, err := c.SetupInstall(ctx, userID, token)
	if err != nil {
		return "", fmt.Errorf("setup install: %w", err)
	}

	keyName := install.Profile.FirstName + " " + install.Profile.LastName + " " + install.Username
	pgp := c.GetPGPHandle()

	key, err := pgp.KeyGeneration().AddUserId(keyName, install.Username).New().GenerateKey()
	if err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	defer key.ClearPrivateParams()

	publicKey, err := key.GetArmoredPublicKey()
	if err != nil {
		return "", fmt.Errorf("armor public key: %w", err)
	}

	lockedKey, err := pgp.LockKey(key, []byte(password))
	if err != nil {
		return "", fmt.Errorf("lock private key: %w", err)
	}
	defer lockedKey.ClearPrivateParams()

	privateKey, err := lockedKey.Armor()
	if err != nil {
		return "", fmt.Errorf("armor private key: %w", err)
	}

	req := api.SetupCompleteRequest{
		AuthenticationToken: api.AuthenticationToken{Token: token},
		User:                api.User{Locale: api.UserLocaleENUK},
		GPGKey:              api.GPGKey{ArmoredKey: publicKey},
	}
	if err := c.SetupComplete(ctx, userID, req); err != nil {
		return "", fmt.Errorf("setup complete: %w", err)
	}
	return privateKey, nil
}
