package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var port string = ":4000"

type RouteResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"`
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
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
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

func main() {
	log.Println("Hello server")

	router := mux.NewRouter()

	router.HandleFunc("/", HandleHealth).Methods("GET")

	router.HandleFunc("/tax", HandleTax).Methods("POST")

	router.HandleFunc("/register", RegisterHandler).Methods("POST")

	router.HandleFunc("/login", LoginHandler).Methods("POST")

	router.HandleFunc("/projects", CreateProjectHandler).Methods("POST")

	router.HandleFunc("/projects/{id}", UpdateHandler).Methods("PUT")

	router.HandleFunc("/projects/{id}", GetProjectHandler).Methods("GET")

	router.HandleFunc("/projects", GetProjectsHandler).Methods("GET")

	router.HandleFunc("/projects/{id}", DeleteProjectsHandler).Methods("DELETE")

	log.Printf("Starter Server on port %s\n", port[1:])
	log.Fatal(http.ListenAndServe(port, router))
}
