// Package jwz contains implementation of JSON WEB ZERO-Knowledge specification.
package jwz

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/iden3/go-circuits/v2"
	"github.com/iden3/go-rapidsnark/types"
	"github.com/iden3/go-rapidsnark/verifier"
)

const (
	// Groth16 alg
	Groth16 string = "groth16"
)

// AuthGroth16Alg its first auth v1 alg (groth16 vs auth v1 circuit)
var AuthGroth16Alg = ProvingMethodAlg{Groth16, string(circuits.AuthCircuitID)}

// ProvingMethodGroth16Auth defines proofs family and specific circuit
type ProvingMethodGroth16Auth struct {
	ProvingMethodAlg
}

// ProvingMethodGroth16AuthInstance instance for Groth16 proving method with an auth circuit
var (
	ProvingMethodGroth16AuthInstance *ProvingMethodGroth16Auth
)

// nolint : used for init proving method instance

// Alg returns current zk alg
func (m *ProvingMethodGroth16Auth) Alg() string {
	return m.ProvingMethodAlg.Alg
}

// CircuitID returns name of circuit
func (m *ProvingMethodGroth16Auth) CircuitID() string {
	return m.ProvingMethodAlg.CircuitID
}

// Verify performs Groth16 proof verification and checks equality of message hash and proven challenge public signals
func (m *ProvingMethodGroth16Auth) Verify(messageHash []byte, proof *types.ZKProof, verificationKey []byte) error {

	var outputs circuits.AuthPubSignals
	pubBytes, err := json.Marshal(proof.PubSignals)
	if err != nil {
		return err
	}

	err = outputs.PubSignalsUnmarshal(pubBytes)
	if err != nil {
		return err
	}

	if outputs.Challenge.Cmp(new(big.Int).SetBytes(messageHash)) != 0 {
		return errors.New("challenge is not equal to message hash")
	}

	return verifier.VerifyGroth16(*proof, verificationKey)
}
