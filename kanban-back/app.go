package main

import (
	"database/sql"
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type App struct {
	Port             string
	ConnectionString string
	DB               *sql.DB
	JWTKey           []byte
	SCHEMA_DIR       string
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

	return &App{ConnectionString: connString, Port: ":4000", JWTKey: jwt_secret, SCHEMA_DIR: schema_dir}, nil
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
	log.Printf("Loaded env from %s", file_path)
	return nil

}
