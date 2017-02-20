package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/biz/biz"
	"github.com/edataforms/pkg/log"
	"github.com/edataforms/pkg/session"
	"github.com/gorilla/mux"
)

func main() {
	r := biz.NewRouter()

	fooMd := func(next http.Handler, w http.ResponseWriter, r *http.Request) {
		fmt.Println("In foo middleware")
		next.ServeHTTP(w, r)
		fmt.Println("after Next")
	}

	r.UseFunc(fooMd)

	api := r.Group("/", nil)
	api.UseFunc(func(next http.Handler, w http.ResponseWriter, r *http.Request) {
		fmt.Println("api only")
		next.ServeHTTP(w, r)
	})

	r.WithFunc(func(next http.Handler, w http.ResponseWriter, r *http.Request) {
		fmt.Println("With middleware")
		next.ServeHTTP(w, r)
	}).GET("/base/{name}", UseBase(GetUser))

	api.GET("/foo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "foo")
	}))

	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Println(err)
	}
}

type Ctx struct {
	Vars map[string]string
	Keys map[string]interface{}
	R    *http.Request
	W    http.ResponseWriter
}

func (ctx *Ctx) Get(key string) interface{} {
	v, ok := ctx.Keys[key]
	if !ok {
		return nil
	}
	return v
}

func (ctx *Ctx) Set(key string, value interface{}) {
	ctx.Keys[key] = value
}

func (ctx *Ctx) Param(key string) string {
	v, ok := ctx.Vars[key]
	if !ok {
		return ""
	}
	return v
}

func NewCtx(w http.ResponseWriter, r *http.Request) *Ctx {
	return &Ctx{
		Vars: mux.Vars(r),
		Keys: map[string]interface{}{},
		R:    r,
		W:    w,
	}
}

type Base struct {
	Session *session.Session
	Log     *log.Log
	DB      *sql.DB
	call    func(*Base, http.ResponseWriter, *http.Request)
	*Ctx
}

func (b *Base) Param(key string) string {
	s, ok := b.Vars[key]
	if !ok {
		return ""
	}
	return s
}

func (b *Base) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	nb := NewBase(w, r)

	b.call(nb, w, r)
}

func UseBase(f func(b *Base, w http.ResponseWriter, r *http.Request)) http.Handler {
	return &Base{
		call: f,
	}
}

func NewBase(w http.ResponseWriter, r *http.Request) *Base {
	b := &Base{
		Log:     log.Empty(),
		Session: session.New(),
		DB:      &sql.DB{},
	}

	b.Ctx = NewCtx(w, r)

	return b
}

func GetUser(base *Base, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s", base.Param("name"))
}
