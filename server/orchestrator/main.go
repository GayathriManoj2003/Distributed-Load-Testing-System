package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"crypto/rand"
	"encoding/hex"
)

type RequestBody struct {
	NumberNodes      int    `json:"NumberNodes"`
	TestType         string `json:"TestType"`
	TestMessageDelay int    `json:"TestMessageDelay"`
}

func generateUniqueToken() (string, error) {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	token := hex.EncodeToString(randomBytes)
	return token, nil
}


func main() {
	http.HandleFunc("/test_config", handlePostTest)
	fmt.Println("Starting server on :8080...")
	http.ListenAndServe(":8080", addCorsHeaders(http.DefaultServeMux))
}

func handlePostTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Printf("here")

	var requestBody RequestBody
    err := json.NewDecoder(r.Body).Decode(&requestBody)
    if err != nil {
        fmt.Printf("Error decoding JSON: %v\n", err)
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
	Test_ID, err := generateUniqueToken()
	if err != nil {
		fmt.Println("Error generating unique token:", err)
		return
	}
	fmt.Printf("Received POST request with body: %+v\n", requestBody)
	fmt.Println("Test_ID:", Test_ID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("POST request received successfully"))
}

func addCorsHeaders(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
