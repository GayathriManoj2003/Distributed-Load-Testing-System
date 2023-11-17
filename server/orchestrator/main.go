package main

import (
	"encoding/json"
	"fmt"
	"os"
	"net/http"
	"crypto/rand"
	"encoding/hex"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type RequestBody struct {
	TestType         string `json:"TestType"`
	TestMessageDelay int    `json:"TestMessageDelay"`
	NumRequests      int    `json:"NumRequests"`
}

type TestConfig struct {
	TestID         string `json:"TestID"`
	TestType         string `json:"TestType"`
	TestMessageDelay int    `json:"TestMessageDelay"`
	NumRequests      int    `json:"NumRequests"`
}

type Producer struct {
	producer          *kafka.Producer
	topic             string
	deliveryChannel   chan kafka.Event
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

func CreateProducer(p *kafka.Producer, topic string) *Producer {
	return &Producer{
		producer:         p,
		topic:            topic,
		deliveryChannel:  make(chan kafka.Event, 10000),
	}
}

func sendTestConfig(testConfiguration TestConfig) error {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"client.id":         "something",
		"acks":              "all",
	})

	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	prod := CreateProducer(p, "test_config")

	jsonMessage, err := json.Marshal(testConfiguration)
	if err != nil {
		return err
	}

	err = prod.producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &prod.topic,
				Partition: kafka.PartitionAny,
			},
			Value: jsonMessage,
		}, prod.deliveryChannel)

	if err != nil {
		return err
	}
	<-prod.deliveryChannel

	fmt.Printf("Placed message on the queue: %s\n", jsonMessage)
	return nil
}


func main() {
	http.HandleFunc("/ping", handlePostTest)
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

	testConfig := TestConfig{
		TestID:           Test_ID,
		TestType:         requestBody.TestType,
		TestMessageDelay: requestBody.TestMessageDelay,
		NumRequests:      requestBody.NumRequests,
	}
	
	fmt.Printf("Test Config: %+v\n", testConfig)

	sendTestConfig(testConfig)
	
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
