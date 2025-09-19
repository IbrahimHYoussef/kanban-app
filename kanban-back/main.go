package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	_ "github.com/lib/pq"
)

// load schema to load json schema

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	var ServerMode string
	flag.StringVar(&ServerMode, "mode", "dev", "determine the server mode running")
	flag.Parse()

	log.Printf("Starting Server in %s mode", ServerMode)

	app, err := CreateApp(ServerMode)
	if err != nil {
		log.Fatal(err)
	}
	DB, err := sql.Open("postgres", app.ConnectionString)
	if err != nil {
		log.Fatalf("Could Not Connect to Database\n %s", err)
	}
	defer DB.Close()
	app.DB = DB
	schemas, err := LoadSchemas(app.SCHEMA_DIR)
	if err != nil {
		log.Fatalf("Could Not load schemas")
	}

	router := mux.NewRouter()

	routeChain := alice.New(LoggingMiddleWare)

	routeChainAuthed := alice.New(LoggingMiddleWare, app.AuthMiddleWare)

	loginMiddleWare := alice.New(LoggingMiddleWare, ValidationMiddelWare(schemas["loginuser.json"]))
	projectMiddleWare := alice.New(LoggingMiddleWare, app.AuthMiddleWare, ValidationMiddelWare(schemas["projects.json"]))

	router.Handle("/", routeChain.ThenFunc(HandleHealth)).Methods("GET")
	router.Handle("/tax", routeChain.ThenFunc(HandleTax)).Methods("POST")

	router.Handle("/api/v1/auth/register", loginMiddleWare.ThenFunc(app.RegisterHandler)).Methods("POST")
	router.Handle("/api/v1/auth/login", loginMiddleWare.ThenFunc(app.LoginHandler)).Methods("POST")

	router.Handle("/api/v1/projects/{id}", routeChainAuthed.ThenFunc(GetProjectHandler)).Methods("GET")
	router.Handle("/api/v1/projects", routeChainAuthed.ThenFunc(GetProjectsHandler)).Methods("GET")
	router.Handle("/api/v1/projects", projectMiddleWare.ThenFunc(app.CreateProjectHandler)).Methods("POST")
	router.Handle("/api/v1/projects/{id}", projectMiddleWare.ThenFunc(UpdateProjectHandler)).Methods("PUT")
	router.Handle("/api/v1/projects/{id}", routeChainAuthed.ThenFunc(DeleteProjectsHandler)).Methods("DELETE")
	// TODO need to create a Task api as projects have tasks
	log.Printf("Starter Server on port %s\n", app.Port[1:])
	log.Fatal(http.ListenAndServe(app.Port, router))
}
