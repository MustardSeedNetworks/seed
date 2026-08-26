package api

// /api/v1/device-credentials endpoints — #1799. The vault is the only
// production SNMP credential store; this is how an operator fills it.
//
//   GET    /api/v1/device-credentials         list (this session's client only)
//   POST   /api/v1/device-credentials         create
//   GET    /api/v1/device-credentials/{id}    fetch one
//   PUT    /api/v1/device-credentials/{id}    full update
//   DELETE /api/v1/device-credentials/{id}    delete
//
// Secrets are write-only. A request carries plaintext; no response ever does.
// polling.Credentials marks every ciphertext field `json:"-"`, so the redaction
// is a property of the type rather than something each handler remembers.
//
// Mutating routes are operator+, matching /polling-targets: a credential is
// polling configuration, and an operator who can add targets but not the
// credentials they reference cannot do the job. User management stays admin.
//
// Every route is scoped to the client on the caller's session claim. There is
// no ?client_id filter and no clientId body field: a credential owned by
// another client is indistinguishable from one that does not exist.

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/credentials"
)

const (
	deviceCredentialsPath       = APIVersionPrefix + "/device-credentials"
	deviceCredentialsPathPrefix = deviceCredentialsPath + "/"
)

// deviceCredentialInput is the request body for POST and PUT.
//
// The secret fields are the only place plaintext appears in this package. They
// are write-only: there is no response type that carries them, and the domain
// type cannot serialise them.
type deviceCredentialInput struct {
	Name         string `json:"name"`
	Community    string `json:"community,omitempty"`
	V3User       string `json:"snmpV3User,omitempty"`
	V3AuthSecret string `json:"snmpV3AuthSecret,omitempty"`
	V3PrivSecret string `json:"snmpV3PrivSecret,omitempty"`
	V3AuthProto  string `json:"snmpV3AuthProto,omitempty"`
	V3PrivProto  string `json:"snmpV3PrivProto,omitempty"`
}

// credentialIDPattern is the shape the repository generates: "cred-" and
// twelve hex characters.
var credentialIDPattern = regexp.MustCompile(`^cred-[0-9a-f]{12}$`)

// validCredentialID rejects anything the server did not generate.
//
// The id arrives from the URL path, so it is caller-controlled. Validating the
// shape is worth doing on its own — a lookup for a malformed id can only miss —
// but it is also what makes the id safe to put in a log line: an unvalidated
// value could carry newlines and forge log entries (CodeQL js/log-injection on
// the first revision of this file).
func validCredentialID(id string) bool {
	return credentialIDPattern.MatchString(id)
}

// handleDeviceCredentials routes the collection endpoint (GET / POST).
func (s *Server) handleDeviceCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listDeviceCredentials(w, r)
	case http.MethodPost:
		s.saveDeviceCredential(w, r, "")
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDeviceCredentialByID routes the resource endpoint (GET / PUT / DELETE).
func (s *Server) handleDeviceCredentialByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, deviceCredentialsPathPrefix)
	if !validCredentialID(id) {
		http.Error(w, "Missing or invalid credential id", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.getDeviceCredential(w, r, id)
	case http.MethodPut:
		s.saveDeviceCredential(w, r, id)
	case http.MethodDelete:
		s.deleteDeviceCredential(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listDeviceCredentials(w http.ResponseWriter, r *http.Request) {
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	if s.deviceCredentials == nil {
		writeCredentialError(w, r, credentials.ErrUnavailable)
		return
	}
	list, err := s.deviceCredentials.List(r.Context(), clientID)
	if err != nil {
		logging.FromContext(r.Context()).ErrorContext(r.Context(),
			"list device_credentials failed", "error", err)
		writeCredentialError(w, r, err)
		return
	}
	writeJSON(w, r, map[string]any{
		jsonKeyCount:  len(list),
		"credentials": list,
	})
}

func (s *Server) getDeviceCredential(w http.ResponseWriter, r *http.Request, id string) {
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	if s.deviceCredentials == nil {
		writeCredentialError(w, r, credentials.ErrUnavailable)
		return
	}
	c, err := s.deviceCredentials.Get(r.Context(), clientID, id)
	if err != nil {
		writeCredentialError(w, r, err)
		return
	}
	writeJSON(w, r, c)
}

// saveDeviceCredential handles both POST (id == "") and PUT.
func (s *Server) saveDeviceCredential(w http.ResponseWriter, r *http.Request, id string) {
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	if s.deviceCredentials == nil {
		writeCredentialError(w, r, credentials.ErrUnavailable)
		return
	}

	var in deviceCredentialInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	// A blank id on POST means create; the repository generates it, so a
	// caller cannot choose its own and collide with another tenant's.
	saved, err := s.deviceCredentials.Save(r.Context(), credentials.Input{
		ID:              id,
		ClientID:        clientID,
		Name:            in.Name,
		Community:       in.Community,
		V3User:          in.V3User,
		V3AuthSecret:    in.V3AuthSecret,
		V3PrivSecret:    in.V3PrivSecret,
		SNMPv3AuthProto: in.V3AuthProto,
		SNMPv3PrivProto: in.V3PrivProto,
	})
	if err != nil {
		// No credential_id field: the logging middleware already records
		// r.URL.Path, which is where the id came from. And no request body —
		// it is the one place in this package that holds plaintext secrets.
		logging.FromContext(r.Context()).WarnContext(r.Context(),
			"save device_credential failed",
			"event", "credential.save.failed")
		writeCredentialError(w, r, err)
		return
	}
	writeJSON(w, r, saved)
}

func (s *Server) deleteDeviceCredential(w http.ResponseWriter, r *http.Request, id string) {
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	if s.deviceCredentials == nil {
		writeCredentialError(w, r, credentials.ErrUnavailable)
		return
	}
	if err := s.deviceCredentials.Delete(r.Context(), clientID, id); err != nil {
		writeCredentialError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeCredentialError maps the use-case's sentinels onto status codes.
//
// In-use is 409 rather than 400: the request is well-formed and the caller is
// authorised; the conflict is with a polling target that still references the
// credential, and deleting it anyway would leave that target unable to
// authenticate.
func writeCredentialError(w http.ResponseWriter, r *http.Request, err error) {
	var ve credentials.ValidationError
	switch {
	case errors.As(err, &ve):
		writeAPITokenError(w, r, http.StatusBadRequest, ErrCodeValidation, ve.Msg)
	case errors.Is(err, credentials.ErrNotFound), errors.Is(err, polling.ErrCredentialsNotFound):
		writeAPITokenError(w, r, http.StatusNotFound, ErrCodeNotFound, "Credential not found")
	case errors.Is(err, credentials.ErrInUse):
		writeAPITokenError(w, r, http.StatusConflict, ErrCodeConflict,
			"Credential is still referenced by a polling target")
	case errors.Is(err, credentials.ErrUnavailable):
		writeAPITokenError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail,
			"Credential store unavailable")
	default:
		writeAPITokenError(w, r, http.StatusInternalServerError, ErrCodeInternal,
			"Failed to process credential")
	}
}
