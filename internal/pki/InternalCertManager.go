// Copyright 2026. Triad National Security, LLC. All rights reserved.

package pki

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/lanl/conduit/internal/logger"
)

const (
	internalCACertExpirationYears     = 10
	internalServerCertExpirationYears = 10
	internalClientCertExpirationYears = 1
	SchedulerClientCertCName          = "conduit-scheduler"
)

type InternalCertManager struct {
	log       *logger.ConduitLogger
	caCert    *x509.Certificate
	caPrivKey ed25519.PrivateKey

	serverCert       *x509.Certificate  // This cert is used for any internal server, such as the runner
	serverPrivKey    ed25519.PrivateKey // This key is used for any internal server, such as the runner
	etcdClientCert   *x509.Certificate  // This cert is used by conduit to authenticate with etcd
	etcdClientKey    ed25519.PrivateKey // This key is used by conduit to authenticate with etcd
	rqliteClientCert *x509.Certificate  // This cert is used by conduit to authenticate with rqlite
	rqliteClientKey  ed25519.PrivateKey // This key is used by conduit to authenticate with rqlite
}

// NewInternalCertManager creates the required components for conduit's certificate authentication internally:
//
// - Creates a CA if one does not exist at the config specified location
//
// - Creates root etcd client cert
//
// - Creates root rqlite client cert
//
// if ServerIPs or ServerHostnames is nil, a server cert will not be generated
func NewInternalCertManager(log *logger.ConduitLogger, caCertPath string, caKeyPath string, serverIPs []net.IP, serverHostnames []string) (*InternalCertManager, error) {
	// change prefix for logger
	var l *logger.ConduitLogger
	if log.GetPrefix() == "" {
		l = logger.NewConduitLogger(log.GetLevel(), "internal cert manager:")
	} else {
		l = logger.NewConduitLogger(log.GetLevel(), fmt.Sprintf("%s internal cert manager:", log.GetPrefix()))
	}

	// setup a new cert manager
	cm := &InternalCertManager{
		log:       l,
		caCert:    &x509.Certificate{},
		caPrivKey: ed25519.PrivateKey{},
	}

	// CA STUFF
	// check if a CA cert and key already exist. If they don't, create new ones
	loadedCACert, loadedCAPrivKey, err := loadCredsFromFile(caCertPath, caKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			e, ok := err.(*os.PathError)
			if ok {
				if e.Path == caCertPath {
					cm.log.Infof("no previous CA cert exists at [%v]. Making a new CA cert and key: %v", caCertPath, err)
				}
				if e.Path == caKeyPath {
					cm.log.Infof("no previous CA key exists at [%v]. Making a new CA cert and key: %v", caKeyPath, err)
				}
			} else {
				cm.log.Infof("no previous CA cert and/or key exists at [%v][%v]. Making a new ca and key: %v", caCertPath, caKeyPath, err)
			}

			newCACert, err := generateCACert(time.Now().AddDate(internalCACertExpirationYears, 0, 0))
			if err != nil {
				return nil, fmt.Errorf("failed to generate CA cert: %v", err)
			}
			_, newCAPrivKey, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				return nil, fmt.Errorf("failed to generate CA key: %v", err)
			}

			cm.caCert = newCACert
			cm.caPrivKey = newCAPrivKey

			err = writeCredsToFile(cm.caCert, cm.caPrivKey, caCertPath, caKeyPath, cm.caCert, cm.caPrivKey)
			if err != nil {
				return nil, fmt.Errorf("failed to write CA to file: %v", err)
			}
			cm.log.Debugf("successfully wrote internal CA cert and key to file. cert:[%v] key:[%v]", caCertPath, caKeyPath)
		} else {
			return nil, fmt.Errorf("failed to load internal CA from file[%v]: %v", caCertPath, err)
		}
	} else {
		// verify that the loaded private key is an ed25519 key
		if pk, ok := loadedCAPrivKey.(ed25519.PrivateKey); ok {
			cm.caPrivKey = pk
		} else {
			return nil, fmt.Errorf("provided internal private key is not ed25519. This is required for the internal certs")
		}
		cm.caCert = loadedCACert
	}

	// SERVER STUFF
	// we don't need to write these to a file since this cert doesn't need to persist across reboots
	if serverIPs != nil && serverHostnames != nil {
		cm.log.Debug("generating conduit server cert and key")
		sc, err := generateServerCert(serverIPs, serverHostnames, "conduit-server", time.Now().AddDate(externalServerCertExpirationYears, 0, 0))
		if err != nil {
			return nil, fmt.Errorf("failed to generate server cert: %v", err)
		}

		_, spk, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate server key: %v", err)
		}

		cm.serverCert = sc
		cm.serverPrivKey = spk
	}

	// generate a client cert and key for use with etcd
	// client cert expires in 10 years
	cm.etcdClientCert, err = GenerateClientCert("root", time.Now().AddDate(10, 0, 0))
	if err != nil {
		return nil, fmt.Errorf("failed to generate etcd client cert: %v", err)
	}
	_, cm.etcdClientKey, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key for etcd client cert: %v", err)
	}

	// generate a client cert and key for use with rqlite
	// client cert expires in 10 years
	cm.rqliteClientCert, err = GenerateClientCert("conduit", time.Now().AddDate(10, 0, 0))
	if err != nil {
		return nil, fmt.Errorf("failed to generate rqlite client cert: %v", err)
	}
	_, cm.rqliteClientKey, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key for rqlite client cert: %v", err)
	}

	return cm, nil
}

func (cm *InternalCertManager) createSignedClientCert(user string, expiration time.Time) (*tls.Certificate, error) {
	cert, err := GenerateClientCert(user, expiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client cert: %v", err)
	}

	_, certKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA key: %v", err)
	}

	certBytes, err := signCert(cert, cm.caCert, cm.caPrivKey, certKey.Public().(ed25519.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign client cert: %v", err)
	}

	certPEM, err := certToPEM(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert cert to PEM: %v", err)
	}

	certPrivKeyPEM, err := privKeyToPEM(certKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert cert key to PEM")
	}

	keypair, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to create tls keypair: %v", err)
	}

	return &keypair, nil
}

func (cm *InternalCertManager) CreateSignedClientCert(user string, expiration time.Time) ([]byte, error) {
	keyPair, err := cm.createSignedClientCert(user, expiration)
	if err != nil {
		return nil, err
	}

	certPEM, err := certToPEM(keyPair.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert cert to PEM: %v", err)
	}

	certPrivKeyPEM, err := privKeyToPEM(keyPair.PrivateKey.(ed25519.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to convert cert key to PEM")
	}

	finalPEM := append(certPEM.Bytes(), certPrivKeyPEM.Bytes()...)

	return finalPEM, nil
}

// creates a client cert for the conduit scheduler to use when connecting to the runner
func (cm *InternalCertManager) CreateSchedulerClientCert(expiration time.Time, schedulerID uuid.UUID) (*tls.Certificate, error) {
	keyPair, err := cm.createSignedClientCert(fmt.Sprintf("%s-%s", SchedulerClientCertCName, schedulerID), expiration)
	if err != nil {
		return nil, err
	}

	return keyPair, nil
}

func (cm *InternalCertManager) WriteServerCredsToFile(certPath, keyPath string, hostnames []string, ips []net.IP, commonName string) error {
	cm.log.Debugf("generating server cert and key for %v %v", hostnames, ips)
	cert, err := generateServerCert(ips, hostnames, commonName, time.Now().AddDate(internalServerCertExpirationYears, 0, 0))
	if err != nil {
		return fmt.Errorf("failed to generate server cert: %v", err)
	}

	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key for server cert: %v", err)
	}

	err = writeCredsToFile(cm.caCert, cm.caPrivKey, certPath, keyPath, cert, privKey)
	if err != nil {
		return fmt.Errorf("failed to write server cert to file: %v", err)
	}
	return nil
}

func (cm *InternalCertManager) WriteClientCredsToFile(certPath, keyPath string, commonName string, expiration time.Time) error {
	cm.log.Debugf("generating client cert and key for %v", commonName)
	cert, err := GenerateClientCert(commonName, expiration)
	if err != nil {
		return fmt.Errorf("failed to generate client cert: %v", err)
	}

	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key for client cert: %v", err)
	}

	err = writeCredsToFile(cm.caCert, cm.caPrivKey, certPath, keyPath, cert, privKey)
	if err != nil {
		return fmt.Errorf("failed to write client cert to file: %v", err)
	}

	return nil
}

func (cm *InternalCertManager) GetETCDClientTLSCert() (*tls.Certificate, error) {
	signedCert, err := signCert(cm.etcdClientCert, cm.caCert, cm.caPrivKey, cm.etcdClientKey.Public().(ed25519.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign etcd cert: %v", err)
	}

	certPEM, err := certToPEM(signedCert)
	if err != nil {
		return nil, fmt.Errorf("failed to convert etcd cert to PEM: %v", err)
	}

	certPrivKeyPEM, err := privKeyToPEM(cm.etcdClientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert etcd private key to PEM: %v", err)
	}

	keyPair, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to combine etcd cert and key into key pair: %v", err)
	}

	return &keyPair, nil
}

func (cm *InternalCertManager) GetRqliteClientTLSCert() (*tls.Certificate, error) {
	signedCert, err := signCert(cm.rqliteClientCert, cm.caCert, cm.caPrivKey, cm.rqliteClientKey.Public().(ed25519.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign rqlite cert: %v", err)
	}

	certPEM, err := certToPEM(signedCert)
	if err != nil {
		return nil, fmt.Errorf("failed to convert rqlite cert to PEM: %v", err)
	}

	certPrivKeyPEM, err := privKeyToPEM(cm.rqliteClientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert rqlite private key to PEM: %v", err)
	}

	keyPair, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to combine rqlite cert and key into key pair: %v", err)
	}

	return &keyPair, nil
}

func (cm *InternalCertManager) GetServerTLSCert() (*tls.Certificate, error) {
	signedCert, err := signCert(cm.serverCert, cm.caCert, cm.caPrivKey, cm.serverPrivKey.Public().(ed25519.PublicKey))
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
