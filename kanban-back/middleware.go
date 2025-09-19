package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xeipuuv/gojsonschema"
)

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

		ctx := context.WithValue(r.Context(), UserClaimsKey, claims)

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
