package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

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

type contextKey string

const UserClaimsKey contextKey = "claims"

func ExtractClaims(r *http.Request) (*Claims, error) {
	claims, ok := r.Context().Value(UserClaimsKey).(*Claims)
	if !ok {
		return nil, errors.New("Unauthorized Access")
	}
	return claims, nil
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
