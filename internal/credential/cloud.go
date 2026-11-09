package credential

import (
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauth "k8s.io/client-go/pkg/apis/clientauthentication/v1"
)

var resourceIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func ValidResourceID(value string) bool { return resourceIDPattern.MatchString(value) }

// ResourceData is internal decrypted data, not a wire format. Optional values
// can live in ordinary secret fields or named Passbolt custom fields.
type ResourceData struct {
	Record           Record
	Metadata, Secret map[string]any
}

func (d ResourceData) Field(name string) (*string, error) {
	if name == "" {
		return nil, nil
	}
	for _, fields := range []map[string]any{d.Secret, d.Metadata} {
		if value, ok := fields[name]; ok && value != nil {
			str, ok := value.(string)
			if !ok {
				return nil, ErrResponse
			}
			return &str, nil
		}
	}
	labels := map[string]string{}
	if fields, ok := d.Metadata["custom_fields"].([]any); ok {
		for _, raw := range fields {
			if field, ok := raw.(map[string]any); ok {
				if id, ok := field["id"].(string); ok {
					labels[id], _ = field["metadata_key"].(string)
				}
			}
		}
	}
	var result *string
	for _, source := range []struct {
		fields     map[string]any
		key, value string
	}{{d.Secret, "secret_key", "secret_value"}, {d.Metadata, "metadata_key", "metadata_value"}} {
		fields, _ := source.fields["custom_fields"].([]any)
		for _, raw := range fields {
			field, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			key, _ := field[source.key].(string)
			id, _ := field["id"].(string)
			if key != name && id != name && labels[id] != name {
				continue
			}
			value, ok := field[source.value]
			if !ok || value == nil {
				continue
			}
			text, ok := value.(string)
			if !ok {
				return nil, ErrResponse
			}
			if result != nil {
				return nil, ErrResponse
			}
			result = &text
		}
		if result != nil {
			return result, nil
		}
	}
	return nil, nil
}
func (d ResourceData) Password() (string, error) {
	value, err := d.Field("password")
	if err != nil || value == nil || strings.TrimSpace(*value) == "" {
		return "", ErrResponse
	}
	return *value, nil
}
func (d ResourceData) expiration(field string) (*time.Time, error) {
	value, err := d.Field(field)
	if err != nil {
		return nil, err
	}
	if value == nil || *value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil || !parsed.After(time.Now()) {
		return nil, ErrResponse
	}
	parsed = parsed.UTC().Truncate(time.Second)
	if !parsed.After(time.Now()) {
		return nil, ErrResponse
	}
	return &parsed, nil
}

type AWSProcessCredentials struct {
	Version         int
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string
	SessionToken    *string    `json:",omitempty"`
	Expiration      *time.Time `json:",omitempty"`
}

func WriteAWS(output io.Writer, data ResourceData, tokenField, expirationField string) error {
	password, err := data.Password()
	if err != nil {
		return err
	}
	if strings.TrimSpace(data.Record.Username) == "" || strings.ContainsAny(data.Record.Username+password, "\r\n\x00") {
		return ErrResponse
	}
	token, err := data.Field(tokenField)
	if err != nil {
		return err
	}
	if token != nil {
		if *token == "" {
			token = nil
		} else if strings.ContainsAny(*token, "\r\n\x00") {
			return ErrResponse
		}
	}
	expiration, err := data.expiration(expirationField)
	if err != nil {
		return err
	}
	return writeObject(output, AWSProcessCredentials{Version: 1, AccessKeyID: data.Record.Username, SecretAccessKey: password, SessionToken: token, Expiration: expiration})
}
func WriteKubernetes(output io.Writer, data ResourceData, expirationField string) error {
	token, err := data.Password()
	if err != nil {
		return err
	}
	if strings.ContainsAny(token, "\r\n\x00") {
		return ErrResponse
	}
	expiration, err := data.expiration(expirationField)
	if err != nil {
		return err
	}
	response := clientauth.ExecCredential{Status: &clientauth.ExecCredentialStatus{Token: token}}
	response.APIVersion = "client.authentication.k8s.io/v1"
	response.Kind = "ExecCredential"
	if expiration != nil {
		value := metav1.NewTime(*expiration)
		response.Status.ExpirationTimestamp = &value
	}
	return writeObject(output, response)
}
func writeObject(output io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	payload := make([]byte, len(data)+1)
	copy(payload, data)
	payload[len(data)] = '\n'
	clear(data)
	defer clear(payload)
	_, err = output.Write(payload)
	return err
}

// KubernetesInteractive validates the host's standard exec request without
// copying cluster data or input status into the credential response.
func KubernetesInteractive(value string) (bool, error) {
	if value == "" {
		return true, nil
	}
	if len(value) > maxRequest {
		return false, ErrRequest
	}
	var request clientauth.ExecCredential
	if json.Unmarshal([]byte(value), &request) != nil || request.APIVersion != "client.authentication.k8s.io/v1" || request.Kind != "ExecCredential" {
		return false, ErrRequest
	}
	return request.Spec.Interactive, nil
}
