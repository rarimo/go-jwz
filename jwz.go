package jwz

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/iden3/go-rapidsnark/types"
	"github.com/rarimo/go-circuits"
)

// HeaderKey represents type for jwz headers keys
type HeaderKey string

const (

	// HeaderType is 'typ' header, so we can set specific typ
	HeaderType HeaderKey = "typ" // we allow to set typ of token

	headerCritical  HeaderKey = "crit"
	headerAlg       HeaderKey = "alg"
	headerCircuitID HeaderKey = "circuitId"
)

// Token represents a JWZ Token.
type Token struct {
	ZkProof *types.ZKProof // The third segment of the token.  Populated when you Parse a token

	Alg       string // fields that are part of headers
	CircuitID string // id of circuit that will be used for proving

	Method ProvingMethod // proving method to create a zkp

	Raw RawJSONWebZeroknowledge // The raw token.  Populated when you Parse a token

	InputsPreparer ProofInputsPreparerHandlerFunc
}

// NewWithPayload creates a new Token with the specified proving method and payload.
func NewWithPayload(prover ProvingMethod, payload []byte, inputsPreparer ProofInputsPreparerHandlerFunc) (*Token, error) {

	token := &Token{
		Alg:            prover.Alg(),
		CircuitID:      prover.CircuitID(),
		Method:         prover,
		InputsPreparer: inputsPreparer,
	}
	token.setDefaultHeaders(prover.Alg(), prover.CircuitID())
	token.setPayload(payload)

	return token, nil
}

// RawJSONWebZeroknowledge is json web token with signature presented by zero knowledge proof
type RawJSONWebZeroknowledge struct {
	Payload   []byte                    `json:"payload,omitempty"`
	Protected []byte                    `json:"protected,omitempty"`
	Header    map[HeaderKey]interface{} `json:"header,omitempty"`
	ZKP       []byte                    `json:"zkp,omitempty"`
}

// setHeader set headers for jwz
func (token *Token) setDefaultHeaders(zkpAlg, circuitID string) {
	headers := map[HeaderKey]interface{}{
		headerAlg:       zkpAlg,
		headerCritical:  []HeaderKey{headerCircuitID},
		headerCircuitID: circuitID,
		HeaderType:      "JWZ",
	}

	token.Raw.Header = headers
}

// WithHeader allows to set or redefine default headers
func (token *Token) WithHeader(key HeaderKey, value interface{}) error {
	token.Raw.Header[key] = value
	return nil
}

// GetHeader returns header
func (token *Token) GetHeader() map[HeaderKey]interface{} {
	return token.Raw.Header
}

// setPayload  set payload for jwz
func (token *Token) setPayload(payload []byte) {
	token.Raw.Payload = payload
}

// GetPayload returns message payload
func (token *Token) GetPayload() []byte {
	return token.Raw.Payload
}

// Parse parses a jwz message in compact or full serialization format.
func Parse(token string) (*Token, error) {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(token, "{") {
		return parseFull(token)
	}
	return parseCompact(token)
}

// parseFull parses a message in full format.
func parseFull(input string) (*Token, error) {
	var parsed RawJSONWebZeroknowledge
	err := json.Unmarshal([]byte(input), &parsed)
	if err != nil {
		return nil, err
	}

	return parsed.sanitized()
}

// parseCompact parses a message in compact format.
func parseCompact(input string) (*Token, error) {
	parts := strings.Split(input, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("iden3/go-jwz: compact JWZ format must have three segments")
	}

	rawProtected, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	proof, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}

	raw := &RawJSONWebZeroknowledge{
		Payload:   rawPayload,
		Protected: rawProtected,
		ZKP:       proof,
	}
	return raw.sanitized()
}

// sanitized produces a cleaned-up JWZ object from the raw JSON.
func (parsed *RawJSONWebZeroknowledge) sanitized() (*Token, error) {
	if parsed.Payload == nil {
		return nil, fmt.Errorf("iden3/go-jwz: missing payload in JWZ message")
	}

	token := &Token{
		Raw: *parsed,
	}

	var headers map[HeaderKey]interface{}

	// all headers are protected
	err := json.Unmarshal(parsed.Protected, &headers)
	if err != nil {
		return nil, err
	}

	// verify that all critical headers are presented
	criticialHeaders := headers[headerCritical].([]interface{})
	for _, key := range criticialHeaders {
		if _, ok := headers[HeaderKey(key.(string))]; !ok {
			return nil, fmt.Errorf("iden3/go-jwz: header is listed in critical %v, but not presented", key)
		}
	}

	token.Raw.Header = headers

	token.Alg = headers[headerAlg].(string)
	token.CircuitID = headers[headerCircuitID].(string)
	token.Method = GetProvingMethod(NewProvingMethodAlg(token.Alg, token.CircuitID))

	// parse proof

	if len(parsed.ZKP) != 0 {
		err := json.Unmarshal(parsed.ZKP, &token.ZkProof)
		if err != nil {
			return nil, err
		}
	}

	return token, nil
}

// ParsePubSignals unmarshalls proof public signals to provided structure.
func (token *Token) ParsePubSignals(out circuits.PubSignalsUnmarshaller) error {
	marshaledPubSignals, err := json.Marshal(token.ZkProof.PubSignals)
	if err != nil {
		return err
	}

	err = out.PubSignalsUnmarshal(marshaledPubSignals)
	if err != nil {
		return err
	}
	return err
}

// Prove creates and returns a complete, proved JWZ.
// The token is proven using the Proving Method specified in the token.
func (token *Token) Prove(provingKey, wasm []byte) (string, error) {

	// all headers must be protected
	headers, err := json.Marshal(token.Raw.Header)
	if err != nil {
		return "", err
	}
	token.Raw.Protected = headers

	msgHash, err := token.GetMessageHash()
	if err != nil {
		return "", err
	}

	inputs, err := token.InputsPreparer.Prepare(msgHash, circuits.CircuitID(token.CircuitID))
	if err != nil {
		return "", err
	}

	proof, err := token.Method.Prove(inputs, provingKey, wasm)
	if err != nil {
		return "", err
	}
	marshaledProof, err := json.Marshal(proof)
	if err != nil {
		return "", err
	}
	token.ZkProof = proof
	token.Raw.ZKP = marshaledProof

	return token.CompactSerialize()
}

// Verify  perform zero knowledge verification.
func (token *Token) Verify(verificationKey []byte) (bool, error) {

	// 1. prepare hash of payload message that had to be proven
	msgHash, err := token.GetMessageHash()
	if err != nil {
		return false, err
	}
	// 2. verify that zkp is valid
	err = token.Method.Verify(msgHash, token.ZkProof, verificationKey)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetMessageHash returns bytes of jwz message hash.
func (token *Token) GetMessageHash() ([]byte, error) {

	headers, err := json.Marshal(token.Raw.Header)
	if err != nil {
		return nil, err
	}
	protectedHeaders := base64.RawURLEncoding.EncodeToString(headers)
	payload := base64.RawURLEncoding.EncodeToString(token.Raw.Payload)

	// JWZ ZkProof input value is ASCII(BASE64URL(UTF8(JWS Protected Header)) || '.' || BASE64URL(JWS Payload)).
	messageToProof := []byte(fmt.Sprintf("%s.%s", protectedHeaders, payload))
	hash, err := Hash(messageToProof)

	if err != nil {
		return nil, err
	}
	return hash.Bytes(), nil
}

// FullSerialize returns marshaled presentation of raw token as json string.
func (token *Token) FullSerialize() (string, error) {

	rawBytes, err := json.Marshal(token.Raw)
	return string(rawBytes), err
}

// CompactSerialize returns token serialized in three parts: base64 encoded headers, payload and proof.
func (token *Token) CompactSerialize() (string, error) {

	if token.Raw.Header == nil || token.Raw.Protected == nil || token.ZkProof == nil {
		return "", errors.New("iden3/jwz:can't serialize without one of components")
	}
	serializedProtected := base64.RawURLEncoding.EncodeToString(token.Raw.Protected)
	proofBytes, err := json.Marshal(token.ZkProof)
	if err != nil {
		return "", err
	}
	serializedProof := base64.RawURLEncoding.EncodeToString(proofBytes)
	serializedPayload := base64.RawURLEncoding.EncodeToString(token.Raw.Payload)

	return fmt.Sprintf("%s.%s.%s", serializedProtected, serializedPayload, serializedProof), nil
}
