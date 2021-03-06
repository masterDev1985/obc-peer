package crypto

import (
	obc "github.com/openblockchain/obc-peer/protos"
)

// Public Interfaces

// Entity represents a crypto object having a name
type Entity interface {

	// GetID returns this entity's name
	GetName() string
}

// Client is an entity able to deploy and invoke chaincode
type Client interface {
	Entity

	// NewChaincodeDeployTransaction is used to deploy chaincode.
	NewChaincodeDeployTransaction(chaincodeDeploymentSpec *obc.ChaincodeDeploymentSpec, uuid string) (*obc.Transaction, error)

	// NewChaincodeExecute is used to execute chaincode's functions.
	NewChaincodeExecute(chaincodeInvocation *obc.ChaincodeInvocationSpec, uuid string) (*obc.Transaction, error)

	// NewChaincodeQuery is used to query chaincode's functions.
	NewChaincodeQuery(chaincodeInvocation *obc.ChaincodeInvocationSpec, uuid string) (*obc.Transaction, error)

	// DecryptQueryResult is used to decrypt the result of a query transaction
	DecryptQueryResult(queryTx *obc.Transaction, result []byte) ([]byte, error)
}

// Peer is an entity able to verify transactions
type Peer interface {
	Entity

	// GetID returns this peer's identifier
	GetID() []byte

	// GetEnrollmentID returns this peer's enrollment id
	GetEnrollmentID() string

	// TransactionPreValidation verifies that the transaction is
	// well formed with the respect to the security layer
	// prescriptions (i.e. signature verification).
	TransactionPreValidation(tx *obc.Transaction) (*obc.Transaction, error)

	// TransactionPreExecution verifies that the transaction is
	// well formed with the respect to the security layer
	// prescriptions (i.e. signature verification). If this is the case,
	// the method prepares the transaction to be executed.
	// TransactionPreExecution returns a clone of tx.
	TransactionPreExecution(tx *obc.Transaction) (*obc.Transaction, error)

	// Sign signs msg with this validator's signing key and outputs
	// the signature if no error occurred.
	Sign(msg []byte) ([]byte, error)

	// Verify checks that signature if a valid signature of message under vkID's verification key.
	// If the verification succeeded, Verify returns nil meaning no error occurred.
	// If vkID is nil, then the signature is verified against this validator's verification key.
	Verify(vkID, signature, message []byte) error

	// GetStateEncryptor returns a StateEncryptor linked to pair defined by
	// the deploy transaction and the execute transaction. Notice that,
	// executeTx can also correspond to a deploy transaction.
	GetStateEncryptor(deployTx, executeTx *obc.Transaction) (StateEncryptor, error)
}

// StateEncryptor is used to encrypt chaincode's state
type StateEncryptor interface {

	// Encrypt encrypts message msg
	Encrypt(msg []byte) ([]byte, error)

	// Decrypt decrypts ciphertext ct obtained
	// from a call of the Encrypt method.
	Decrypt(ct []byte) ([]byte, error)
}