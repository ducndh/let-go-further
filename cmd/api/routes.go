package main

import (
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", app.notFoundResponse)

	mux.HandleFunc("GET /v1/healthcheck", app.healthcheckHandler)

	mux.HandleFunc("GET /v1/movies", app.listMoviesHandler)
	mux.HandleFunc("POST /v1/movies", app.createMovieHandler)

	mux.HandleFunc("GET /v1/movies/{id}", app.showMovieHandler)
	mux.HandleFunc("PATCH /v1/movies/{id}", app.updateMovieHandler)
	mux.HandleFunc("DELETE /v1/movies/{id}", app.deleteMovieHandler)

	mux.HandleFunc("POST /v1/users", app.registerUserHandler)
	mux.HandleFunc("PUT /vi/users/activated", app.activateUserHandler)

	mux.HandleFunc("POST /v1/tokens/authentication", app.createAuthenticationTokenHandler)

	return app.recoverPanic(app.rateLimit(mux))
}
