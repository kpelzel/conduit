// Copyright 2026. Triad National Security, LLC. All rights reserved.

package httpserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/lanl/conduit/api"
	"github.com/lanl/conduit/defaults"
	"github.com/lanl/conduit/internal/logger"
	"github.com/lanl/conduit/internal/server/httpserver/auth"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type HTTPServer struct {
	log *logger.ConduitLogger

	addr      string
	server    *http.Server
	router    http.Handler
	validator *auth.Introspector

	conduitClient     api.ConduitApiClient
	conduitClientConn *grpc.ClientConn

	originPolicy *OriginPolicy
}

func CreateHTTPServer(
	log *logger.ConduitLogger,
	addr string,
	clientCert *tls.Certificate,
	certPool *x509.CertPool,
	grpcAddr string,
) (*HTTPServer, error) {
	l := logger.NewConduitLogger(log.GetLevel(), fmt.Sprintf("%sHTTP server:", log.GetPrefix()))
	if log.GetPrefix() == "" {
		l = logger.NewConduitLogger(log.GetLevel(), "HTTP server:")
	}

	rawAllowedOrigins := viper.GetStringSlice(defaults.ConfigServerHTTPAllowedOriginsKey)

	originPolicy, err := NewOriginPolicy(l, rawAllowedOrigins)
	if err != nil {
		return nil, err
	}

	router := http.NewServeMux()

	discoveryURL := viper.GetString(defaults.ConfigOAuthDiscoveryKey)
	clientID := viper.GetString(defaults.ConfigOAuthclientIDKey)
	clientSecret := viper.GetString(defaults.ConfigOAuthclientSecretKey)
	userInfoFallback := viper.GetBool(defaults.ConfigOAuthUserFallbackKey)
	usernameClaims := viper.GetStringSlice(defaults.ConfigOAuthUserClaimsKey)
	introspectionAuthMethod := viper.GetString(defaults.ConfigOAuthIntrospectionAuthMethodKey)

	if discoveryURL == "" {
		return nil, fmt.Errorf("OAuth discovery URL is required")
	}
	if clientID == "" {
		return nil, fmt.Errorf("OAuth client ID is required")
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("OAuth client secret is required")
	}

	authCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	validator, err := auth.NewIntrospector(authCtx, auth.Config{
		DiscoveryURL: discoveryURL,

		ClientID:     clientID,
		ClientSecret: clientSecret,

		UsernameClaims: usernameClaims,

		UseUserInfoFallback: userInfoFallback,

		ViperIntrospectionAuthMethod: introspectionAuthMethod,

		// does the IdP return aud?
		// ExpectedAudience: "conduit",
	}, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create http validator: %v", err)
	}

	conn, conduitClient, err := GetGRPCClient(l, clientCert, certPool, grpcAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get grpc client: %v", err)
	}

	h := &HTTPServer{
		addr:              addr,
		log:               l,
		conduitClient:     conduitClient,
		conduitClientConn: conn,
		router:            router,
		validator:         validator,
		originPolicy:      originPolicy,
	}

	h.registerHTTPRoutes(router)

	corsHandler := handlers.CORS(
		handlers.AllowedOrigins(originPolicy.CORSOrigins()),
		handlers.AllowedHeaders([]string{
			"Authorization",
			"Content-Type",
		}),
		handlers.AllowedMethods([]string{
			http.MethodGet,
			http.MethodPost,
			http.MethodOptions,
		}),
	)(router)

	h.server = &http.Server{
		Handler: corsHandler,
		Addr:    addr,

		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return h, nil
}

func (h *HTTPServer) StartHTTPServer() error {
	defer h.conduitClientConn.Close()
	h.log.Infof("HTTP server listening on %s", h.addr)

	if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func (h *HTTPServer) registerHTTPRoutes(router *http.ServeMux) {
	router.Handle("GET /ws", h.authedHandler(h.serveWs))

	router.Handle("GET /transfers", h.authedHandler(h.getTransfers))
	router.Handle("POST /transfers", h.authedHandler(h.startTransfer))

	router.Handle("GET /transfers/{transferID}", h.authedHandler(h.getTransferByID))
	router.Handle("POST /transfers/query", h.authedHandler(h.queryTransfers))
	router.Handle("POST /transfers/abort", h.authedHandler(h.abortTransfers))
	router.Handle("POST /transfers/{transferID}/abort", h.authedHandler(h.abortTransfer))
}

func (h *HTTPServer) authedHandler(next func(http.ResponseWriter, *http.Request, string)) http.Handler {
	return auth.RequireBearer(
		h.validator,
		auth.ValidateOptions{},
		http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {
			req.Body = http.MaxBytesReader(wr, req.Body, 1<<20) // 1 MiB
			defer req.Body.Close()

			username, ok := auth.UsernameFromRequest(req)
			if !ok {
				http.Error(wr, "missing authenticated user", http.StatusInternalServerError)
				return
			}

			next(wr, req, username)
		}),
	)

}

// getGRPCClient authenticates with kerberos, dials into the conduit server, and returns a ConduitApiClient
func GetGRPCClient(log *logger.ConduitLogger, clientCert *tls.Certificate, certPool *x509.CertPool, conduitAddr string) (*grpc.ClientConn, api.ConduitApiClient, error) {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*clientCert},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
	}

	creds := credentials.NewTLS(tlsConfig)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	log.Debugf("dialing to grpc server at: %v\n", conduitAddr)
	conn, err := grpc.NewClient(conduitAddr, opts...)
	// conn, err := grpc.DialContext(ctx, conduitAddr, opts...)
	if err != nil {
		// ctxCancel()
		return nil, nil, fmt.Errorf("failed to dial into conduit server: %v", err)
	}

	log.Debugf("creating grpc client")
	client := api.NewConduitApiClient(conn)

	return conn, client, nil
}
