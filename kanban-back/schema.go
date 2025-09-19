package main

import (
	"log"
	"os"
	"strings"
)

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
