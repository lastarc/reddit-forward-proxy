package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/exp/slog"

	"github.com/zitadel/zitadel-go/v3/pkg/authorization"
	"github.com/zitadel/zitadel-go/v3/pkg/authorization/oauth"
	"github.com/zitadel/zitadel-go/v3/pkg/http/middleware"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
)

var (
	domain = flag.String("domain", "", "your ZITADEL instance domain (in the form: <instance>.zitadel.cloud or <yourdomain>)")
	key    = flag.String("key", "", "path to your key.json")
	port   = flag.String("port", "8089", "port to run the server on (default is 8089)")
)

type ApiKeyRewriteHandler struct {
	handler http.Handler
}

func (h *ApiKeyRewriteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if apiKey := r.URL.Query().Get("apiKey"); apiKey != "" {
		slog.Debug("Rewriting authorization with %s", apiKey)
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	}
	h.handler.ServeHTTP(w, r)
}

func NewApiKeyRewriteHandler(handlerToWrap http.Handler) *ApiKeyRewriteHandler {
	return &ApiKeyRewriteHandler{
		handler: handlerToWrap,
	}
}

func main() {
	flag.Parse()

	ctx := context.Background()

	authZ, err := authorization.New(ctx, zitadel.New(*domain), oauth.DefaultAuthorization(*key))
	if err != nil {
		slog.Error("zitadel sdk could not initialize", "error", err)
		os.Exit(1)
	}

	mw := middleware.New(authZ)

	router := http.NewServeMux()

	router.Handle("/api/healthz", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			err = jsonResponse(w, "OK", http.StatusOK)
			if err != nil {
				slog.Error("error writing response", "error", err)
			}
		}))

	router.Handle("/api/proxy", mw.RequireAuthorization(authorization.WithRole("reddit-forward-proxy-access"))(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			url := r.URL.Query().Get("url")
			if url == "" {
				err = jsonResponse(w, map[string]string{
					"error": "missing url parameter",
				}, http.StatusBadRequest)
				if err != nil {
					slog.Error("error writing response", "error", err)
				}
				return
			}

			slog.Info("proxying", "url", url)

			// fetch url with header
			client := &http.Client{}
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				_ = jsonResponse(w, map[string]string{
					"error": err.Error(),
				}, http.StatusInternalServerError)
			}
			req.Header.Set("User-Agent", "reddit-forward-proxy/0.1")
			resp, err := client.Do(req)
			if err != nil {
				err = jsonResponse(w, map[string]string{
					"error": err.Error(),
				}, http.StatusInternalServerError)
				if err != nil {
					slog.Error("error writing response", "error", err)
				}
				return
			}
			defer resp.Body.Close()

			_, err = io.Copy(w, resp.Body)
			if err != nil {
				err = jsonResponse(w, map[string]string{
					"error": err.Error(),
				}, http.StatusInternalServerError)
				if err != nil {
					slog.Error("error writing response", "error", err)
				}
				return
			}
		})))

	lis := fmt.Sprintf(":%s", *port)
	slog.Info("server listening, press ctrl+c to stop", "addr", "http://localhost"+lis)
	err = http.ListenAndServe(lis, NewApiKeyRewriteHandler(router))
	if !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server terminated", "error", err)
		os.Exit(1)
	}
}

func jsonResponse(w http.ResponseWriter, resp any, status int) error {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
