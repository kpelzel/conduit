// Copyright 2026. Triad National Security, LLC. All rights reserved.

package pki

import (
	"crypto"
	"crypto/x509"
	"fmt"

	"github.com/lanl/conduit/internal/logger"
)

type CertManagerType int

const (
	INTERNAL CertManagerType = iota
	EXTERNAL
)

type CertManager struct {
	log *logger.ConduitLogger
	*InternalCertManager
	*ExternalCertManager
}

// NewCertManager creates the required components for conduit's certificate authentication:
//
// - contains both an internal and external cert manager
func NewCertManager(log *logger.ConduitLogger, internalCertManager *InternalCertManager, externalCertManager *ExternalCertManager) (*CertManager, error) {
	// change prefix for logger
	l := logger.NewConduitLogger(log.GetLevel(), fmt.Sprintf("%scert manager:", log.GetPrefix()))
	if log.GetPrefix() == "" {
		l = logger.NewConduitLogger(log.GetLevel(), "cert manager:")
	}

	// setup a new cert manager
	cm := &CertManager{
		log:                 l,
		InternalCertManager: internalCertManager,
		ExternalCertManager: externalCertManager,
	}

	return cm, nil
}

// GetCertPool returns a cert pool for either the internal or external cert manager
func (cm *CertManager) GetCertPool(certManagerType CertManagerType) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()

	var caCert *x509.Certificate
	var caPrivKey crypto.PrivateKey

	switch certManagerType {
	case INTERNAL:
		caCert = cm.InternalCertManager.caCert
		caPrivKey = cm.InternalCertManager.caPrivKey
	case EXTERNAL:
		caCert = cm.ExternalCertManager.caCert
		caPrivKey = cm.ExternalCertManager.caPrivKey
	default:
		return nil, fmt.Errorf("unrecognized cert manager type: %v", certManagerType)
	}

	pubKey, err := PublicKeyFromPrivateKey(caPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from private key: %v", err)
	}

	signedCert, err := signCert(caCert, caCert, caPrivKey, pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to self sign ca cert: %v", err)
	}

	certPEM, err := certToPEM(signedCert)
	if err != nil {
		return nil, fmt.Errorf("failed to convert server cert to PEM: %v", err)
	}

	ok := certPool.AppendCertsFromPEM(certPEM.Bytes())
	if !ok {
		return nil, fmt.Errorf("failed to append certs to the cert pool")
	}

	return certPool, nil
}
