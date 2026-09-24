package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"
)

// throughRegistrar serves h as the only route of a registrar built with niac's
// error envelope, so a handler test gets the same method gate and panic
// recovery the daemon applies. Empty methods leaves dispatch to h.
func throughRegistrar(methods []string, h http.HandlerFunc) http.HandlerFunc {
	reg := route.New(route.Config{Error: simpleErr, MaxBodyBytes: MaxRequestBodySize})
	reg.Register(route.Route{Path: "/", Handler: h, Methods: methods})
	return reg.Handler().ServeHTTP
}
