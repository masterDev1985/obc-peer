package crypto

import "crypto/x509"

func (node *nodeImpl) initCryptoEngine() error {
	node.log.Info("Initializing node crypto engine...")

	node.rootsCertPool = x509.NewCertPool()

	// Load ECA certs chain
	if err := node.loadECACertsChain(); err != nil {
		return err
	}

	// Load TCA certs chain
	if err := node.loadTCACertsChain(); err != nil {
		return err
	}

	// Load enrollment secret key
	// TODO: finalize encrypted pem support
	if err := node.loadEnrollmentKey(nil); err != nil {
		return err
	}

	// Load enrollment certificate and set validator ID
	if err := node.loadEnrollmentCertificate(); err != nil {
		return err
	}

	// Load enrollment id
	if err := node.loadEnrollmentID(); err != nil {
		return err
	}

	// Load enrollment chain key
	if err := node.loadEnrollmentChainKey(nil); err != nil {
		return err
	}

	node.log.Info("Initializing node crypto engine...done!")

	return nil
}
