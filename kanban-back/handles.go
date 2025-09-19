package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

type RouteResponse struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	ID      string `json:"id,omitempty"`
}

type ErrorResponse struct {
	Messsage   string `json:"message"`
	StatusCode int    `json:"status_code"`
}

type Project struct {
	ProjectID       int      `json:"project_id,omitempty"`
	Name            string   `json:"name,omitempty"`
	ReboUrl         string   `json:"rebo_url,omitempty"`
	SiteUrl         string   `json:"site_url,omitempty"`
	Description     string   `json:"description,omitempty"`
	Dependencies    []string `json:"dependencies,omitempty"`
	DevDependencies []string `json:"dev_dependencies,omitempty"`
	State           string   `json:"status,omitempty"`
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

func RespondWithError(w http.ResponseWriter, status_code int, message string) {
	res := ErrorResponse{Messsage: message, StatusCode: status_code}
	w.WriteHeader(res.StatusCode)
	json.NewEncoder(w).Encode(&res)
}

// create project
func (app *App) CreateProjectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")

	claims, err := ExtractClaims(r)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "UnAuthorized")
		return
	}
	user_id := claims.UserID
	log.Printf("User accessing is %s", claims.UserID)

	var req Project
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("")
		RespondWithError(w, http.StatusBadRequest, "Invalid Request")
		return
	}

	query := `INSERT INTO projects
        (
        name,
        rebo_url,
        site_url,
        description,
        dependencies,
        dev_dependencies,
        status,
        user_id
        )
        VALUES
        ($1,$2,$3,$4,$5,$6,$7,$8)
        returning project_id
        `
	var project_id int
	err = app.DB.QueryRow(query,
		req.Name,
		req.ReboUrl,
		req.SiteUrl,
		req.Description,
		pq.Array(req.Dependencies),
		pq.Array(req.DevDependencies),
		req.State,
		user_id).Scan(&project_id)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid Request")
		log.Print(err)
		return
	}
	req.ProjectID = project_id

	json.NewEncoder(w).Encode(&req)

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
	log.Printf("User accessing is %s", r.Context())
}

// get project
func GetProjectsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	claims, err := ExtractClaims(r)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "UnAuthorized")
		return
	}
	log.Printf("User accessing is %s", claims.UserID)
}

// delete project
func DeleteProjectsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
}
