package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/justinas/alice"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	Port   string
	DB     *sql.DB
	JWTKey []byte
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

type Claims struct {
	Username string `json:"user_name"`
	UserID   string `json:"user_id"`
	jwt.RegisteredClaims
}

type AuthResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"user_name"`
	Token    string `json:"token"`
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

func AuthMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		auth_header := r.Header.Get("Authorization")
		if len(auth_header) == 0 {
			RespondWithError(w, http.StatusUnauthorized, "Not Allowed To Access This Endpoint")
			return
		}
		token := auth_header[7:]
		log.Print(token)

		next.ServeHTTP(w, r)
	})
}
func LoadEnv(mode string) error {
	var file_path string

	switch mode {
	case "dev":
		file_path = ".env.dev"
	case "prod":
		file_path = ".env"
	case "test":
		file_path = ".env.test"
	default:
		file_path = ".env"
	}

	err := godotenv.Load(file_path)
	if err != nil {
		return err
	}
	return nil

}
func main() {

	log.Println(len(os.Args), os.Args)
	var ServerMode string
	flag.StringVar(&ServerMode, "mode", "dev", "determine the server mode running")

	log.Printf("Starting Server in %s mode", ServerMode)
	err := LoadEnv(ServerMode)
	if err != nil {
		log.Fatal("Error Loading .env file")
	}

	connString := os.Getenv("PSQL_URL")
	if len(connString) == 0 {
		log.Fatalf("No Environment Connection String Set")
	}

	DB, err := sql.Open("postgres", connString)
	if err != nil {
		log.Fatalf("Could Not Connect to Database\n %s", err)
	}
	app := &App{DB: DB, Port: ":4000", JWTKey: []byte(os.Getenv("JWT_KEY"))}
	defer DB.Close()

	router := mux.NewRouter()

	routeChain := alice.New(LoggingMiddleWare)

	routeChainAuthed := alice.New(LoggingMiddleWare, AuthMiddleWare)

	router.Handle("/", routeChain.ThenFunc(HandleHealth)).Methods("GET")

	router.Handle("/tax", routeChain.ThenFunc(HandleTax)).Methods("POST")

	router.Handle("/register", routeChain.ThenFunc(app.RegisterHandler)).Methods("POST")

	router.Handle("/login", routeChain.ThenFunc(app.LoginHandler)).Methods("POST")

	router.Handle("/projects", routeChainAuthed.ThenFunc(CreateProjectHandler)).Methods("POST")

	router.Handle("/projects/{id}", routeChain.ThenFunc(UpdateProjectHandler)).Methods("PUT")

	router.Handle("/projects/{id}", routeChain.ThenFunc(GetProjectHandler)).Methods("GET")

	router.Handle("/projects", routeChain.ThenFunc(GetProjectsHandler)).Methods("GET")

	router.Handle("/projects/{id}", routeChain.ThenFunc(DeleteProjectsHandler)).Methods("DELETE")

	log.Printf("Starter Server on port %s\n", app.Port[1:])
	log.Fatal(http.ListenAndServe(app.Port, router))
}
