package biz

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter(middleware ...Middleware) *Router {
	return &Router{
		Mux:        mux.NewRouter(),
		middleware: middleware,
	}
}

type Middleware func(http.Handler) http.Handler

type Router struct {
	Mux        *mux.Router
	middleware []Middleware
}

func (r *Router) GET(pattern string, handler http.Handler) *mux.Route {
	return r.Handle(pattern, handler).Methods("GET")
}

func (r *Router) POST(pattern string, handler http.Handler) *mux.Route {
	return r.Handle(pattern, handler).Methods("POST")
}

func (r *Router) PUT(pattern string, handler http.Handler) *mux.Route {
	return r.Handle(pattern, handler).Methods("PUT")
}

func (r *Router) PATCH(pattern string, handler http.Handler) *mux.Route {
	return r.Handle(pattern, handler).Methods("PATCH")
}

func (r *Router) DELETE(pattern string, handler http.Handler) *mux.Route {
	return r.Handle(pattern, handler).Methods("DELETE")
}

func (r *Router) Handle(pattern string, h http.Handler) *mux.Route {
	return r.Mux.Handle(pattern, call(r.middleware, h))
}

func (r *Router) Use(m ...Middleware) {
	r.middleware = append(r.middleware, m...)
}

func (r *Router) UseFunc(middleware ...Func) {
	for _, m := range middleware {
		r.Use(UseFunc(m))
	}
}

// Group will create a new sub-router with the previous middleware included
//
// Passing nil as the second argument will clear any previous middleware
func (r *Router) Group(pattern string, middleware ...Middleware) *Router {
	nr := &Router{
		Mux: r.Mux.PathPrefix(pattern).Subrouter(),
	}
	if len(middleware) == 1 && middleware[0] == nil {
		return nr
	}
	nr.middleware = append(r.middleware, middleware...)

	return nr
}

// With is used to apply middleware to the next handler by creating a new Router
//
// Example:
//      r.With(middlewareOne, middlewareTwo).Get("/foo", getFoo)
//
// middlewareOne and middlewateTwo will only be applied to getFoo
func (r *Router) With(middleware ...Middleware) *Router {
	// combine middleware
	middleware = append(r.middleware, middleware...)

	return &Router{
		Mux:        r.Mux,
		middleware: middleware,
	}
}

func (r *Router) WithFunc(funcs ...Func) *Router {
	md := make([]Middleware, len(funcs))
	for i, f := range funcs {
		md[i] = UseFunc(f)
	}
	return r.With(md...)
}

// Skip will return a Router that removes the passed in middleware
//
// Example:
//		r.Use(authMiddleware)
//		r.Skip(authMiddleware).Get("/info", infoHandler)
//
// In the above example authMiddleware will not get run on infoHandler
func (r *Router) Skip(middleware ...Middleware) *Router {
	md := []Middleware{}
	for i := 0; i < len(r.middleware); i++ {
		for j := 0; j < len(middleware); j++ {
			if fmt.Sprintf("%p", middleware[i]) == fmt.Sprintf("%p", r.middleware[j]) {
				continue
			}
		}
		md = append(md, r.middleware[i])
	}

	return &Router{
		Mux:        r.Mux,
		middleware: md,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Mux.ServeHTTP(w, req)
}

type Func func(next http.Handler, w http.ResponseWriter, r *http.Request)

// UseFunc is a helper function that can be used to create middleware and reduce boiler plate code
func UseFunc(f Func) Middleware {
	return func(n http.Handler) http.Handler {
		nf := func(w http.ResponseWriter, r *http.Request) {
			f(n, w, r)
		}
		return http.HandlerFunc(nf)
	}
}

// call creates a handler that calls all of the passed in middleware with the handler
func call(middleware []Middleware, handler http.Handler) http.Handler {
	if len(middleware) == 0 {
		return handler
	}

	combined := middleware[len(middleware)-1](handler)
	for i := len(middleware) - 2; i >= 0; i-- {
		combined = middleware[i](combined)
	}

	return combined
}
