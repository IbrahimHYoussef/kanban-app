package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/justinas/alice"
)

var port string = ":4000"

type RouteResponse struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	ID      string `json:"id,omitempty"`
}

type HealthCheck struct {
	IsHealthy bool   `json:"is_healthy"`
	Time      string `json:"time"`
}

type TaxCalculaterRequest struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
}

type TaxCalculaterResponse struct {
	Type        string  `json:"type"`
	TaxAmount   float64 `json:"tax_amount"`
	TotalAmount float64 `json:"total_amount"`
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	w.Header().Set("Content-type", "application/json")
	now := time.Now().String()
	json.NewEncoder(w).Encode(HealthCheck{IsHealthy: true, Time: now})
	end := time.Now()
	total := end.Sub(start).String()
	log.Printf("request took : %s", total)
}

func HandleTax(w http.ResponseWriter, r *http.Request) {
	// decoder := json.NewDecoder(w)
	// encoder := json.Encoder(w)

	var body TaxCalculaterRequest

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "Invalid Request Body", http.StatusUnprocessableEntity)
		log.Println("Invalid Request Body")
		return
	}

	var response TaxCalculaterResponse

	tax_type := strings.ToLower(body.Type)
	var tax_mult float64
	switch tax_type {
	case "vat":
		tax_mult = 0.14
	case "cgt":
		tax_mult = 0.20
	default:
		http.Error(w, "Not Supported Tax Type", http.StatusBadRequest)
		log.Printf("Invalid Tax Type %s\n", body.Type)
		return
	}
	response.TaxAmount = body.Amount * tax_mult
	response.TotalAmount = body.Amount + response.TaxAmount
	response.Type = body.Type

	json.NewEncoder(w).Encode(response)
	log.Printf("Calculated Tax with Type %s and Amount %f Total Amount %f\n", tax_type, response.TaxAmount, response.TotalAmount)

}

// register
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
}

// login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
}

// create project
func CreateProjectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")

}

// update project
func UpdateProjectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	vars := mux.Vars(r)
	projectId := vars["id"]
	log.Printf("updated project with id %s", projectId)
	json.NewEncoder(w).Encode(RouteResponse{ID: projectId, Message: "Hello from create projects"})
}

// get projects

func GetProjectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")

}

// get project
func GetProjectsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
}

// delete project
func DeleteProjectsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
}

func LoggingMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error Loading .env file")
	}
	log.Println("Hello server")

	router := mux.NewRouter()

	routeChain := alice.New(LoggingMiddleWare)

	router.Handle("/", routeChain.ThenFunc(HandleHealth)).Methods("GET")

	router.Handle("/tax", routeChain.ThenFunc(HandleTax)).Methods("POST")

	router.Handle("/register", routeChain.ThenFunc(RegisterHandler)).Methods("POST")

	router.Handle("/login", routeChain.ThenFunc(LoginHandler)).Methods("POST")

	router.Handle("/projects", routeChain.ThenFunc(CreateProjectHandler)).Methods("POST")

	router.Handle("/projects/{id}", routeChain.ThenFunc(UpdateProjectHandler)).Methods("PUT")

	router.Handle("/projects/{id}", routeChain.ThenFunc(GetProjectHandler)).Methods("GET")

	router.Handle("/projects", routeChain.ThenFunc(GetProjectsHandler)).Methods("GET")

	router.Handle("/projects/{id}", routeChain.ThenFunc(DeleteProjectsHandler)).Methods("DELETE")

	log.Printf("Starter Server on port %s\n", port[1:])
	log.Fatal(http.ListenAndServe(port, router))
}
