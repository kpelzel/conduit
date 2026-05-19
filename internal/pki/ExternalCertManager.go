// Copyright 2026. Triad National Security, LLC. All rights reserved.

package pki

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/lanl/conduit/internal/logger"
)

const (
	externalCACertExpirationYears     = 10
	externalServerCertExpirationYears = 10
)

type ExternalCertManager struct {
	log           *logger.ConduitLogger
	caCert        *x509.Certificate
	caPrivKey     crypto.PrivateKey
	serverCert    *x509.Certificate
	serverPrivKey crypto.PrivateKey
}

// NewExternalCertManager creates the required components for conduit's certificate authentication used externally with the grpc api:
//
// - Creates a CA if one does not exist at the config specified location
//
// - Creates a server cert for the GRPC server
//
// if ServerIPs or ServerHostnames is nil, a server cert will not be generated
func NewExternalCertManager(log *logger.ConduitLogger, caCertPath string, caKeyPath string, serverIPs []net.IP, serverHostnames []string) (*ExternalCertManager, error) {
	// change prefix for logger
	var l *logger.ConduitLogger
	if log.GetPrefix() == "" {
		l = logger.NewConduitLogger(log.GetLevel(), "external cert manager:")
	} else {
		l = logger.NewConduitLogger(log.GetLevel(), fmt.Sprintf("%s external cert manager:", log.GetPrefix()))
	}

	// setup a new cert manager
	cm := &ExternalCertManager{
		log:       l,
		caCert:    &x509.Certificate{},
		caPrivKey: &ecdsa.PrivateKey{},
	}

	// CA STUFF
	// check if a CA cert and key already exist. If they don't, create new ones
	loadedCACert, loadedCAPrivKey, err := loadCredsFromFile(caCertPath, caKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			cm.log.Infof("no previous CA exists. Making a new one: %v", err)
			newCACert, err := generateCACert(time.Now().AddDate(externalCACertExpirationYears, 0, 0))
			if err != nil {
				return nil, fmt.Errorf("failed to generate CA cert: %v", err)
			}

			newCAPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				return nil, fmt.Errorf("failed to generate CA key: %v", err)
			}

			cm.caCert = newCACert
			cm.caPrivKey = newCAPrivKey

			err = writeCredsToFile(cm.caCert, cm.caPrivKey, caCertPath, caKeyPath, cm.caCert, cm.caPrivKey)
			if err != nil {
				return nil, fmt.Errorf("failed to write CA to file: %v", err)
			}
			cm.log.Debugf("successfully wrote external CA to file[%v]", caCertPath)
		} else {
			return nil, fmt.Errorf("failed to load external CA from file: %v", err)
		}
	} else {
		cm.caCert = loadedCACert
		cm.caPrivKey = loadedCAPrivKey
	}

	// SERVER STUFF
	// we don't need to write these to a file since this cert doesn't need to persist across reboots
	if serverIPs != nil && serverHostnames != nil {
		cm.log.Debug("generating conduit server cert and key")
		sc, err := generateServerCert(serverIPs, serverHostnames, "conduit-server", time.Now().AddDate(externalServerCertExpirationYears, 0, 0))
		if err != nil {
			return nil, fmt.Errorf("failed to generate server cert: %v", err)
		}

		spk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate server key: %v", err)
		}

		cm.serverCert = sc
		cm.serverPrivKey = spk
	}

	return cm, nil
}

func (cm *ExternalCertManager) WriteClientCredsToFile(certPath, keyPath, commonName string, expiration time.Time) error {
	cm.log.Debugf("generating client cert and key for %v", commonName)
	cert, err := GenerateClientCert(commonName, expiration)
	if err != nil {
		return fmt.Errorf("failed to generate client cert: %v", err)
	}

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key for client cert: %v", err)
	}

	err = writeCredsToFile(cm.caCert, cm.caPrivKey, certPath, keyPath, cert, privKey)
	if err != nil {
		return fmt.Errorf("failed to write client cert to file: %v", err)
	}

	return nil
}

func (cm *ExternalCertManager) GetClientCreds(commonName string, expiration time.Time) ([]byte, error) {
	cm.log.Debugf("generating client cert and key for %v", commonName)
	cert, err := GenerateClientCert(commonName, expiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client cert: %v", err)
	}

	// ed25519 is not very compatible with 3rd party clients so for external certs we'll use an ECDSA P-256 key
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key for client cert: %v", err)
	}

	signedCertBytes, err := signCert(cert, cm.caCert, cm.caPrivKey, &privKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign cert: %v", err)
	}

	certPEM, err := certToPEM(signedCertBytes)
	if err != nil {
		return nil, fmt.Errorf("failed convert cert to PEM: %v", err)
	}

	privKeyPEM, err := privKeyToPEM(privKey)
	if err != nil {
		return nil, fmt.Errorf("error getting private key pem: %v", err)
	}

	certAndKey := append(certPEM.Bytes(), privKeyPEM.Bytes()...)

	return certAndKey, nil
}

func (cm *ExternalCertManager) GetServerTLSCert() (*tls.Certificate, error) {
	pubKey, err := PublicKeyFromPrivateKey(cm.serverPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from private key: %v", err)
	}

	signedCert, err := signCert(cm.serverCert, cm.caCert, cm.caPrivKey, pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign server cert: %v", err)
	}

	certPEM, err := certToPEM(signedCert)
	if err != nil {
		return nil, fmt.Errorf("failed to convert server cert to PEM: %v", err)
	}

	certPrivKeyPEM, err := privKeyToPEM(cm.serverPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert server private key to PEM: %v", err)
	}

	keyPair, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to combine cert and key into key pair: %v", err)
	}

	return &keyPair, nil
}
