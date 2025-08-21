package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/justinas/alice"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	Port string
	DB   *sql.DB
}

type RouteResponse struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	ID      string `json:"id,omitempty"`
}

type AuthRequest struct {
	Username string `json:"user_name"`
	Password string `json:"password"`
}

type AuthResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"user_name"`
}

type ErrorResponse struct {
	Messsage   string `json:"message"`
	StatusCode int    `json:"status_code"`
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

func ValidateEmail(email string) error {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !re.MatchString(email) {
		return errors.New("email dose not match email fromate")
	} else {
		return nil
	}
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
	err = ValidateEmail(auth_req.Username)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "email not in correct formate")
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
		RespondWithError(w, http.StatusInternalServerError, "Error Creating User")
		log.Print(err)
		return
	}
	auth_res := AuthResponse{UserID: user_id, Username: auth_req.Username}
	json.NewEncoder(w).Encode(&auth_res)
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
	var user_name string
	var password string
	err = app.DB.QueryRow("SELECT user_name,password FROM users WHERE user_name = ?;", auth_req.Username).Scan(user_name, password)
	if err != nil {
		log.Printf("User with User Name %s not Found", auth_req.Username)
		log.Print(err)
		RespondWithError(w, http.StatusBadRequest, "User Not Found")
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(password), []byte(auth_req.Password))
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Wrong Password Or User")
		return
	}
	auth_res := AuthResponse{UserID: user_name, Username: auth_req.Username}
	json.NewEncoder(w).Encode(&auth_res)
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

	connString := os.Getenv("PSQL_URL")
	if len(connString) == 0 {
		log.Fatalf("No Environment Connection String Set")
	}

	DB, err := sql.Open("postgres", connString)
	if err != nil {
		log.Fatalf("Could Not Connect to Database\n %s", err)
	}
	app := &App{DB: DB, Port: ":4000"}
	defer DB.Close()

	router := mux.NewRouter()

	routeChain := alice.New(LoggingMiddleWare)

	router.Handle("/", routeChain.ThenFunc(HandleHealth)).Methods("GET")

	router.Handle("/tax", routeChain.ThenFunc(HandleTax)).Methods("POST")

	router.Handle("/register", routeChain.ThenFunc(app.RegisterHandler)).Methods("POST")

	router.Handle("/login", routeChain.ThenFunc(app.LoginHandler)).Methods("POST")

	router.Handle("/projects", routeChain.ThenFunc(CreateProjectHandler)).Methods("POST")

	router.Handle("/projects/{id}", routeChain.ThenFunc(UpdateProjectHandler)).Methods("PUT")

	router.Handle("/projects/{id}", routeChain.ThenFunc(GetProjectHandler)).Methods("GET")

	router.Handle("/projects", routeChain.ThenFunc(GetProjectsHandler)).Methods("GET")

	router.Handle("/projects/{id}", routeChain.ThenFunc(DeleteProjectsHandler)).Methods("DELETE")

	log.Printf("Starter Server on port %s\n", app.Port[1:])
	log.Fatal(http.ListenAndServe(app.Port, router))
}
