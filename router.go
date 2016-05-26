package main

import (
	"github.com/gorilla/mux"
	"github.com/unrolled/secure"
	"net/http"
)

func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	secureMiddleware := secure.New(secure.Options{
		AllowedHosts:          []string{"doesntexist.com"},
		SSLRedirect:           true,
		SSLHost:               "doesntexist.com",
		SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
		STSSeconds:            315360000,
		STSIncludeSubdomains:  true,
		STSPreload:            true,
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		BrowserXssFilter:      true,
		ContentSecurityPolicy: "default-src 'self' https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css https://ajax.googleapis.com/ajax/libs/jquery/1.12.2/jquery.min.js 'unsafe-inline'",
		PublicKey:             `pin-sha256="base64+primary=="; pin-sha256="base64+backup=="; max-age=5184000; includeSubdomains; report-uri="https://www.example.com/hpkp-report"`,

		IsDevelopment: true,
	})

	for _, route := range routes {
		var handler http.Handler

		handler = route.HandlerFunc
		handler = Logger(handler, route.Name)
		handler = secureMiddleware.Handler(handler)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}
