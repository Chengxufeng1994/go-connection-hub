package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

type HTTPServer struct {
	handler http.Handler
	srv     *http.Server
}

var _ Server = (*HTTPServer)(nil)

func NewHTTPServer(handler http.Handler) *HTTPServer {
	srv := &HTTPServer{
		handler: handler,
	}
	return srv
}

func (h *HTTPServer) Start(ctx context.Context) error {
	h.srv = &http.Server{
		Addr:         ":8080",
		Handler:      h.handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	var eg errgroup.Group
	eg.Go(func() error {
		err := h.srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil
	})

	return eg.Wait()
}

func (h *HTTPServer) Stop(ctx context.Context) error {
	return h.srv.Shutdown(ctx)
}
