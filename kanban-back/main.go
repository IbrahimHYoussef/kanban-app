package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/justinas/alice"
	_ "github.com/lib/pq"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	Port       string
	DB         *sql.DB
	JWTKey     []byte
	SCHEMA_DIR string
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

func LoggingMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func (app *App) AuthMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		auth_header := r.Header.Get("Authorization")
		if len(auth_header) == 0 {
			RespondWithError(w, http.StatusUnauthorized, "Not Allowed To Access This Endpoint")
			return
		}
		tokenString := auth_header[7:]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return app.JWTKey, nil
		})
		if err != nil {
			if err == jwt.ErrTokenSignatureInvalid {
				RespondWithError(w, http.StatusUnauthorized, "Not Allowed To Access This Endpoint")
				return
			}
			RespondWithError(w, http.StatusBadRequest, "Not Allowed To Access This Endpoint")
			return
		}
		if !token.Valid {
			RespondWithError(w, http.StatusUnauthorized, "Not Allowed To Access This Endpoint")
			return
		}

		ctx := context.WithValue(r.Context(), claims, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ValidationMiddelWare(schema string) func(http.Handler) http.Handler {
	if len(schema) == 0 {
		log.Println("No Schema String was added Reverting to Default")
		schema = "{}"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]interface{}
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				log.Printf("Error no body to read in ValidationMiddelWare")
				RespondWithError(w, http.StatusBadRequest, "Invalid Request Payload")
				return
			}
			err = json.Unmarshal(bodyBytes, &body)
			if err != nil {
				log.Printf("Failed to Unmarshal in ValidationMiddelWare")
				RespondWithError(w, http.StatusBadRequest, "Invalid Request Payload")
				return
			}
			schemaLoader := gojsonschema.NewStringLoader(schema)

			documentLoader := gojsonschema.NewGoLoader(body)
			result, err := gojsonschema.Validate(schemaLoader, documentLoader)
			if err != nil {
				RespondWithError(w, http.StatusInternalServerError, "Error Validating json")
				return
			}
			if !result.Valid() {
				var errs []string
				for _, err := range result.Errors() {
					errs = append(errs, err.String())
				}
				RespondWithError(w, http.StatusBadRequest, strings.Join(errs, ", "))
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			next.ServeHTTP(w, r)
		})
	}
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
		file_path = ".env.dev"
	}

	err := godotenv.Load(file_path)
	if err != nil {
		return err
	}
	return nil

}

// load schema to load json schema
func LoadSchema(filePath string) (string, error) {
	log.Printf("loading %s", filePath)
	date, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(date), nil
}

func LoadSchemas(dirPath string) (map[string]string, error) {
	if dirPath[len(dirPath)-1] != '/' {
		dirPath = dirPath + "/"
	}
	log.Printf("loading json schemas from %s", dirPath)
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, file := range files {
		sliced := strings.Split(file.Name(), ".")
		if sliced[len(sliced)-1] == "json" {
			fcontent, err := LoadSchema(dirPath + file.Name())
			if err != nil {
				return nil, err
			}
			result[file.Name()] = fcontent
		}
	}
	return result, nil
}

func CreateApp(server_mode string) (*App, error) {
	err := LoadEnv(server_mode)
	if err != nil {
		// log.Fatal("Error Loading .env file")
		return nil, errors.New("Error Loading .env file")
	}

	connString := os.Getenv("PSQL_URL")
	if len(connString) == 0 {
		// log.Fatalf("No Environment Connection String Set")
		return nil, errors.New("No Environment Connection String Set")
	}

	DB, err := sql.Open("postgres", connString)
	if err != nil {
		// log.Fatalf("Could Not Connect to Database\n %s", err)
		return nil, fmt.Errorf("Could Not Connect to Database\n %s", err)
	}
	defer DB.Close()

	jwt_secret := []byte(os.Getenv("JWT_KEY"))
	if len(jwt_secret) == 0 {
		// log.Fatalf("JWT secret is not added in the environment varibles")
		return nil, errors.New("JWT secret is not added in the environment varibles")
	}
	schema_dir := os.Getenv("SCHEMA_PATH")
	if len(schema_dir) == 0 {
		// log.Fatal("Schema Path not added in the environment varibles")
		return nil, errors.New("Schema Path not added in the environment varibles")
	}

	return &App{DB: DB, Port: ":4000", JWTKey: jwt_secret, SCHEMA_DIR: schema_dir}, nil
}

func main() {

	var ServerMode string
	flag.StringVar(&ServerMode, "mode", "dev", "determine the server mode running")

	log.Printf("Starting Server in %s mode", ServerMode)

	app, err := CreateApp(ServerMode)
	if err != nil {
		log.Fatal(err)
	}

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
