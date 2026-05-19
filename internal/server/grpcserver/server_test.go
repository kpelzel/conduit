// Copyright 2026. Triad National Security, LLC. All rights reserved.

package grpcserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/jcmturner/goidentity/v6"
	krbCred "github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/lanl/conduit/internal/logger"
	cert "github.com/lanl/conduit/internal/pki"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

type getUserTestCase struct {
	context              *context.Context
	requestedUser        string
	expectedUser         string
	expectedReqPrivLevel privLevel
	expectedErr          bool
}

var (
	testServer      *ConduitServer
	adminContext    context.Context
	serviceContext  context.Context
	userKrbContext  context.Context
	userCertContext context.Context
	noContext       context.Context
	testUsername    = "testuser"
	testRealm       = "testRealm"

	getUserTestCases = []getUserTestCase{
		// case 1: admin cert used, no user provided
		{
			context:              &adminContext,
			requestedUser:        "",
			expectedUser:         "",
			expectedReqPrivLevel: privilegedAdmin,
			expectedErr:          false,
		},
		// case 2: admin cert used, user provided
		{
			context:              &adminContext,
			requestedUser:        testUsername,
			expectedUser:         testUsername,
			expectedReqPrivLevel: privilegedAdmin,
			expectedErr:          false,
		},
		// case 3: service cert used, no user provided
		{
			context:              &serviceContext,
			requestedUser:        "",
			expectedUser:         "",
			expectedReqPrivLevel: privilegedService,
			expectedErr:          false,
		},
		// case 4: service cert used, no user provided
		{
			context:              &serviceContext,
			requestedUser:        testUsername,
			expectedUser:         testUsername,
			expectedReqPrivLevel: privilegedService,
			expectedErr:          false,
		},
		// case 5: user cert used, no user provided
		{
			context:              &userCertContext,
			requestedUser:        "",
			expectedUser:         testUsername,
			expectedReqPrivLevel: unprivileged,
			expectedErr:          false,
		},
		// case 6: user cert used, user provided
		{
			context:              &userCertContext,
			requestedUser:        privilegedAdmins[0],
			expectedUser:         testUsername,
			expectedReqPrivLevel: unprivileged,
			expectedErr:          false,
		},
		// case 7: kerb used, no user provided
		{
			context:              &userKrbContext,
			requestedUser:        "",
			expectedUser:         testUsername,
			expectedReqPrivLevel: unprivileged,
			expectedErr:          false,
		},
		// case 8: kerb used, user provided
		{
			context:              &userKrbContext,
			requestedUser:        privilegedAdmins[0],
			expectedUser:         testUsername,
			expectedReqPrivLevel: unprivileged,
			expectedErr:          false,
		},
		// case 9: no kerb or cert used, no user provided
		{
			context:              &noContext,
			requestedUser:        "",
			expectedUser:         "",
			expectedReqPrivLevel: unprivileged,
			expectedErr:          true,
		},
		// case 10: no kerb or cert used, user provided
		{
			context:              &noContext,
			requestedUser:        testUsername,
			expectedUser:         "",
			expectedReqPrivLevel: unprivileged,
			expectedErr:          true,
		},
	}
)

// creates fake certs and server
func init() {
	testServer = &ConduitServer{
		log: logger.NewConduitLogger(logrus.ErrorLevel, ""),
	}

	// create user kerberos context
	userIdentity := krbCred.New(testUsername, testRealm)
	userKrbContext = context.Background()
	userKrbContext = context.WithValue(userKrbContext, goidentity.CTXKey, userIdentity)

	// create null context
	noTLSInfo := credentials.TLSInfo{
		State: tls.ConnectionState{
			VerifiedChains: [][]*x509.Certificate{},
		},
		CommonAuthInfo: credentials.CommonAuthInfo{
			SecurityLevel: credentials.PrivacyAndIntegrity,
		},
	}

	noPeer := &peer.Peer{
		AuthInfo: noTLSInfo,
	}
	noContext = peer.NewContext(context.Background(), noPeer)

	// create user cert context
	userCert, err := cert.GenerateClientCert(testUsername, time.Now().AddDate(10, 0, 0))
	if err != nil {
		logrus.Errorf("failed to generate client cert: %v", err)
	}

	userTLSInfo := credentials.TLSInfo{
		State: tls.ConnectionState{
			VerifiedChains: [][]*x509.Certificate{{
				userCert,
			}},
		},
		CommonAuthInfo: credentials.CommonAuthInfo{
			SecurityLevel: credentials.PrivacyAndIntegrity,
		},
	}

	userPeer := &peer.Peer{
		AuthInfo: userTLSInfo,
	}
	userCertContext = peer.NewContext(context.Background(), userPeer)

	// create admin context
	adminCert, err := cert.GenerateClientCert(privilegedAdmins[0], time.Now().AddDate(10, 0, 0))
	if err != nil {
		logrus.Errorf("failed to generate client cert: %v", err)
	}

	adminTLSInfo := credentials.TLSInfo{
		State: tls.ConnectionState{
			VerifiedChains: [][]*x509.Certificate{{
				adminCert,
			}},
		},
		CommonAuthInfo: credentials.CommonAuthInfo{
			SecurityLevel: credentials.PrivacyAndIntegrity,
		},
	}

	adminPeer := &peer.Peer{
		AuthInfo: adminTLSInfo,
	}
	adminContext = peer.NewContext(context.Background(), adminPeer)

	// create service context
	serviceCert, err := cert.GenerateClientCert(privilegedServices[0], time.Now().AddDate(10, 0, 0))
	if err != nil {
		logrus.Errorf("failed to generate client cert: %v", err)
	}

	serviceTLSInfo := credentials.TLSInfo{
		State: tls.ConnectionState{
			VerifiedChains: [][]*x509.Certificate{{
				serviceCert,
			}},
		},
		CommonAuthInfo: credentials.CommonAuthInfo{
			SecurityLevel: credentials.PrivacyAndIntegrity,
		},
	}

	servicePeer := &peer.Peer{
		AuthInfo: serviceTLSInfo,
	}
	serviceContext = peer.NewContext(context.Background(), servicePeer)

}

func TestGetUserFromRequest(t *testing.T) {
	for tcn, tc := range getUserTestCases {
		user, reqPrivLevel, err := testServer.getUserFromRequest(*tc.context, &tc.requestedUser)

		if user != tc.expectedUser {
			t.Errorf("failed test case %d: incorrect user returned. expected: [%s], got: [%s]", tcn+1, tc.expectedUser, user)
		}

		if reqPrivLevel != tc.expectedReqPrivLevel {
			t.Errorf("failed test case %d: incorrect reqPrivLevel returned. expected: [%v], got: [%v]", tcn+1, tc.expectedReqPrivLevel, reqPrivLevel)
		}

		if (err == nil && tc.expectedErr) || (err != nil && !tc.expectedErr) {
			t.Errorf("failed test case %d: incorrect error returned. expected: [%v], got: [%v]", tcn+1, tc.expectedErr, err)
		}

	}

}
