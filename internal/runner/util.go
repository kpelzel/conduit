// Copyright 2026. Triad National Security, LLC. All rights reserved.

package internal

import (
	"context"
	"crypto/x509"
	"fmt"
	"net"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

func clientCertInfo(ctx context.Context) (cert *x509.Certificate, ip net.IP, dns []string, uris []string, cn string, err error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, nil, nil, nil, "", fmt.Errorf("no peer in context")
	}

	// Remote addr (last hop)
	if ta, ok := p.Addr.(*net.TCPAddr); ok {
		ip = ta.IP
	}

	// Auth info must be TLS for mTLS
	ti, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, ip, nil, nil, "", fmt.Errorf("not a TLS connection")
	}

	// Prefer the verified chain if present; fall back to raw peer certs
	if len(ti.State.VerifiedChains) > 0 && len(ti.State.VerifiedChains[0]) > 0 {
		cert = ti.State.VerifiedChains[0][0]
	} else if len(ti.State.PeerCertificates) > 0 {
		cert = ti.State.PeerCertificates[0]
	} else {
		return nil, ip, nil, nil, "", fmt.Errorf("no client certificate")
	}

	dns = cert.DNSNames
	uris = make([]string, 0, len(cert.URIs))
	for _, u := range cert.URIs {
		uris = append(uris, u.String()) // e.g., "spiffe://trust-domain/ns/namespace/sa/name"
	}
	cn = cert.Subject.CommonName // deprecated for identity; prefer SANs/URIs
	return cert, ip, dns, uris, cn, nil
}
