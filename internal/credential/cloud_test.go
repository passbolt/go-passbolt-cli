package credential

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	clientauth "k8s.io/client-go/pkg/apis/clientauthentication/v1"
)

func TestAWSProcessJSONOptionalFields(t *testing.T) {
	data := ResourceData{Record: Record{Username: "AKIAEXAMPLE"}, Secret: map[string]any{"password": "key-secret"}}
	var output bytes.Buffer
	if err := WriteAWS(&output, data, "session_token", "expiration"); err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(output.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if fields["Version"] != float64(1) || fields["AccessKeyId"] != "AKIAEXAMPLE" || fields["SecretAccessKey"] != "key-secret" || len(fields) != 3 {
		t.Fatalf("AWS output: %s", output.String())
	}
	expires := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	data.Secret["session_token"] = "temporary-token"
	data.Secret["expiration"] = expires.Format(time.RFC3339)
	output.Reset()
	if err := WriteAWS(&output, data, "session_token", "expiration"); err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(output.Bytes(), &fields)
	if fields["SessionToken"] != "temporary-token" || fields["Expiration"] != expires.Format(time.RFC3339) {
		t.Fatalf("AWS session: %s", output.String())
	}
	data.Secret["expiration"] = "bad-expiry"
	output.Reset()
	if err := WriteAWS(&output, data, "session_token", "expiration"); err == nil || output.Len() != 0 {
		t.Fatal("invalid expiration produced partial credentials")
	}
}
func TestKubernetesOfficialExecCredential(t *testing.T) {
	data := ResourceData{Secret: map[string]any{"password": "bearer-token"}}
	var output bytes.Buffer
	if err := WriteKubernetes(&output, data, "expiration"); err != nil {
		t.Fatal(err)
	}
	var parsed clientauth.ExecCredential
	if err := json.Unmarshal(output.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.APIVersion != "client.authentication.k8s.io/v1" || parsed.Kind != "ExecCredential" || parsed.Status == nil || parsed.Status.Token != "bearer-token" || parsed.Status.ExpirationTimestamp != nil {
		t.Fatalf("Kubernetes output: %s", output.String())
	}
	if strings.Contains(output.String(), "SessionToken") {
		t.Fatal("schemas mixed")
	}
}

func TestCloudCustomFieldsAndExpiredCredentials(t *testing.T) {
	expires := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	data := ResourceData{Record: Record{Username: "AKIAEXAMPLE"}, Metadata: map[string]any{"custom_fields": []any{map[string]any{"id": "session", "metadata_key": "aws_session"}, map[string]any{"id": "expiry", "metadata_key": "aws_expiry"}}}, Secret: map[string]any{"password": "key-secret", "custom_fields": []any{map[string]any{"id": "session", "secret_value": "session-value"}, map[string]any{"id": "expiry", "secret_value": expires.Format(time.RFC3339)}}}}
	var output bytes.Buffer
	if err := WriteAWS(&output, data, "aws_session", "aws_expiry"); err != nil {
		t.Fatal(err)
	}
	var result AWSProcessCredentials
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.SessionToken == nil || *result.SessionToken != "session-value" || result.Expiration == nil || !result.Expiration.Equal(expires) {
		t.Fatal("custom fields not mapped")
	}
	data.Secret["expiration"] = time.Now().Add(-time.Hour).Format(time.RFC3339)
	for _, write := range []func(io.Writer) error{func(w io.Writer) error { return WriteAWS(w, data, "", "expiration") }, func(w io.Writer) error { return WriteKubernetes(w, data, "expiration") }} {
		output.Reset()
		if err := write(&output); err == nil || output.Len() != 0 {
			t.Fatal("expired credentials produced output")
		}
	}
}
