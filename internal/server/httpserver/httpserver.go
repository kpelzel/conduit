// Copyright 2026. Triad National Security, LLC. All rights reserved.

package httpserver

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/lanl/conduit/internal/logger"
)

func CreateHTTPServer(log *logger.ConduitLogger, addr string, update chan bool) (*http.Server, chan []byte) {
	// change prefix for logger
	l := logger.NewConduitLogger(log.GetLevel(), fmt.Sprintf("%sHTTP server:", log.GetPrefix()))
	if log.GetPrefix() == "" {
		l = logger.NewConduitLogger(log.GetLevel(), "HTTP server:")
	}

	// flag.Parse()
	hub := newHub(l, update)
	go hub.run()

	router := mux.NewRouter()
	// disable until complete
	// router.HandleFunc("/ws", func(wr http.ResponseWriter, req *http.Request) {
	// 	serveWs(hub, wr, req)
	// })
	// router.HandleFunc("/status", func(wr http.ResponseWriter, req *http.Request) {
	// 	serveWs(hub, wr, req)
	// })
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)

	router.Handle("/debug/pprof/block", pprof.Handler("block"))
	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	server := &http.Server{
		Handler: handlers.CORS(
			handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"PUT", "GET", "POST", "DELETE"}),
		)(router),
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// go func() {
	// 	logrus.Infof("websocket is listening on: %s\n", server.Addr)
	// 	err := server.ListenAndServe()
	// 	if err != nil {
	// 		log.Fatal("ListenAndServe: ", err)
	// 	}
	// }()

	return server, hub.broadcast
}

func StartHTTPServer(s *http.Server, log *logger.ConduitLogger) {
	log.Infof("websocket is listening on: %s\n", s.Addr)
	err := s.ListenAndServe()
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
