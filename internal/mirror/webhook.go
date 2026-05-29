package mirror

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

// VerifyGitHubSignature verifies the X-Hub-Signature-256 header against the request body.
func VerifyGitHubSignature(signature string, body []byte, secret string) bool {
	if signature == "" || secret == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

// PushPayload represents a GitHub push event payload.
type PushPayload struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Deleted    bool   `json:"deleted"`
	Repository struct {
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
}

// ParseRef parses a git ref string into ref type and name.
// Examples:
//
//	"refs/heads/main"    -> ("branch", "main")
//	"refs/tags/v1.0.0"  -> ("tag", "v1.0.0")
func ParseRef(ref string) (refType string, name string) {
	parts := strings.Split(ref, "/")
	if len(parts) < 3 {
		return "unknown", ref
	}

	switch parts[1] {
	case "heads":
		return "branch", strings.Join(parts[2:], "/")
	case "tags":
		return "tag", strings.Join(parts[2:], "/")
	default:
		return "unknown", parts[2]
	}
}

// RefAllowed checks if a ref is allowed based on the mirror config.
func RefAllowed(refType, name, branchPattern string, syncTags, syncDeletes bool) bool {
	switch refType {
	case "branch":
		if branchPattern == "" || branchPattern == "*" {
			return true
		}
		matched, _ := path.Match(branchPattern, name)
		return matched
	case "tag":
		return syncTags
	default:
		return false
	}
}

// ParsePushPayload parses a push event payload from the request body.
func ParsePushPayload(body []byte) (*PushPayload, error) {
	var payload PushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse push payload: %w", err)
	}
	return &payload, nil
}

// ReadBody reads the request body and returns it as bytes, replacing the body
// with a new reader for downstream handlers.
func ReadBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	r.Body = io.NopCloser(strings.NewReader(string(body)))
	return body, nil
}
