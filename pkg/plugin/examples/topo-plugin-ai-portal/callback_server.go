package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

// Selection identifies the model chosen in AI Portal.
type Selection struct {
	ModelID string
	RepoID  string
}

type callbackResult struct {
	selection Selection
	err       error
}

// CallbackServer consumes the first request, returning its selection or validation error.
type CallbackServer struct {
	result   chan callbackResult
	claimed  atomic.Bool
	listener net.Listener
	server   *http.Server
}

func NewCallbackServer() (*CallbackServer, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for Portal callback: %w", err)
	}
	callbackServer := &CallbackServer{
		result:   make(chan callbackResult, 1),
		listener: listener,
	}
	callbackServer.server = &http.Server{
		Handler: callbackServer, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
		IdleTimeout: 5 * time.Second, MaxHeaderBytes: 8192,
	}
	return callbackServer, nil
}

func (s *CallbackServer) CallbackURL() string {
	return "http://" + s.listener.Addr().String() + "/"
}

func (s *CallbackServer) WaitForCallback(ctx context.Context) (Selection, error) {
	serveResult := make(chan error, 1)
	go func() {
		err := s.server.Serve(s.listener)
		serveResult <- fmt.Errorf("serve Portal callback: %w", err)
	}()
	select {
	case result := <-s.result:
		return result.selection, result.err
	case err := <-serveResult:
		return Selection{}, err
	case <-ctx.Done():
		return Selection{}, context.Cause(ctx)
	}
}

func (s *CallbackServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	shutdownErr := s.server.Shutdown(ctx)
	closeErr := s.server.Close()
	// Shutdown does not close the listener if Serve has not started yet.
	listenerErr := s.listener.Close()
	if errors.Is(listenerErr, net.ErrClosed) {
		listenerErr = nil
	}
	if err := errors.Join(shutdownErr, closeErr, listenerErr); err != nil {
		return fmt.Errorf("shut down Portal callback: %w", err)
	}
	return nil
}

func (s *CallbackServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.claimed.CompareAndSwap(false, true) {
		http.Error(w, "already handled", http.StatusServiceUnavailable)
		return
	}
	var result callbackResult
	defer func() { s.result <- result }()

	if r.Method != http.MethodGet {
		result.err = errors.New("use GET for the callback")
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, result.err.Error(), http.StatusMethodNotAllowed)
		return
	}
	result.selection, result.err = s.parseSelection(r.URL.RawQuery)
	if result.err != nil {
		http.Error(w, result.err.Error(), http.StatusBadRequest)
		return
	}
	_, _ = w.Write([]byte("Selection returned. Continue in your terminal.\n"))
}

func (s *CallbackServer) parseSelection(rawQuery string) (Selection, error) {
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return Selection{}, fmt.Errorf("malformed callback query")
	}
	for _, key := range []string{"version", "modelId", "repoId"} {
		if len(query[key]) != 1 || query.Get(key) == "" {
			return Selection{}, fmt.Errorf("callback requires exactly one %s", key)
		}
	}
	if query.Get("version") != "1" {
		return Selection{}, fmt.Errorf("unsupported callback version")
	}
	return Selection{ModelID: query.Get("modelId"), RepoID: query.Get("repoId")}, nil
}
