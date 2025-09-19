package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type RouteResponse struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	ID      string `json:"id,omitempty"`
}

type AuthRequest struct {
	Username string `json:"user_name"`
	Password string `json:"password"`
}

type Claims struct {
	Username string `json:"user_name"`
	UserID   string `json:"user_id"`
	jwt.RegisteredClaims
}

type AuthResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"user_name"`
	Token    string `json:"token,omitempty"`
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

// register
func (app *App) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	var auth_req AuthRequest
	err := json.NewDecoder(r.Body).Decode(&auth_req)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid Request")
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(auth_req.Password), bcrypt.DefaultCost)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	var user_id string
	err = app.DB.QueryRow("INSERT INTO \"users\" (user_name,password) VALUES ($1,$2) RETURNING user_id", auth_req.Username, string(hashedPassword)).Scan(&user_id)
	if err != nil {
		// TODO HANDLE USER ALERADY EXITS ERROR
		RespondWithError(w, http.StatusInternalServerError, "Error Creating User")
		log.Print(err)
		return
	}
	auth_res := AuthResponse{UserID: user_id, Username: auth_req.Username}
	json.NewEncoder(w).Encode(&auth_res)
}

func (app *App) GenerateToken(user_name string, user_id int64) (string, error) {
	expirationTime := time.Now().Add(time.Minute * 20)
	claims := &Claims{Username: user_name, UserID: strconv.FormatInt(user_id, 10), RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expirationTime)}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(app.JWTKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// login
func (app *App) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	var auth_req AuthRequest
	err := json.NewDecoder(r.Body).Decode(&auth_req)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid Request")
		return
	}
	var user_id int64
	var user_stored AuthRequest
	err = app.DB.QueryRow("SELECT user_id,user_name,password FROM \"users\" WHERE user_name = $1", auth_req.Username).Scan(&user_id, &user_stored.Username, &user_stored.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User with User Name %s not Found", auth_req.Username)
			RespondWithError(w, http.StatusBadRequest, "User Not Found")
		}
		log.Print(err)
		RespondWithError(w, http.StatusInternalServerError, "InternalServerError")
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(user_stored.Password), []byte(auth_req.Password))
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Wrong Password Or User")
		return
	}

	tokenString, err := app.GenerateToken(user_stored.Username, user_id)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Wrong Password Or User")
		return
	}
	auth_res := AuthResponse{UserID: strconv.FormatInt(user_id, 10), Username: auth_req.Username, Token: tokenString}
	json.NewEncoder(w).Encode(&auth_res)
}

// create project
func (app *App) CreateProjectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	log.Printf("User accessing is %s", r.Context())

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
