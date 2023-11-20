package main

import (
	"encoding/json"
	"fmt"
	"os"
	"net/http"
	"crypto/rand"
	"time"
	"encoding/hex"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gorilla/websocket"
	"sync"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var p *kafka.Producer
var prod *Producer
var consumer *kafka.Consumer
var metricsConsumer *MetricsConsumer
var wsMutex sync.Mutex

func init() {
	var err error

	// Initialize Kafka producer
	p, err = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"client.id":         "something",
		"acks":              "all",
	})
	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	// Initialize Kafka consumer
	consumer, err = kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "metrics-consumer-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	// Create instances of Producer and MetricsConsumer
	prod = CreateProducer(p)
	metricsConsumer = CreateMetricsConsumer()
	metricsConsumer.consumer = consumer

	topics := []string{"metrics"}
	err = consumer.SubscribeTopics(topics, nil)
	if err != nil {
		fmt.Printf("Failed to subscribe to topics: %v\n", err)
		os.Exit(1)
	}
	go HandleHeartbeatRegisterTopics()
}

// WebSocketClients stores connected clients
var WebSocketClients = make(map[*websocket.Conn]bool)

func SendMessageToClients(message string) {
	wsMutex.Lock()
	defer wsMutex.Unlock()
	for client := range WebSocketClients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			fmt.Println("Error writing to WebSocket:", err)
			delete(WebSocketClients, client)
			client.Close()
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	// Add the client to the WebSocketClients map
	wsMutex.Lock()
	WebSocketClients[conn] = true
	wsMutex.Unlock()

	for {
		time.Sleep(time.Millisecond*50)
	}
}
type RequestBody struct {
	TestType         string `json:"TestType"`
	TestMessageDelay int    `json:"TestMessageDelay"`
	NumRequests      int    `json:"NumRequests"`
}
type RequestBodyTrigger struct {
	TestID         string `json:"TestID"`
}

type TestConfig struct {
	TestID         			string `json:"test_id"`
	TestType         		string `json:"test_type"`
	TestMessageDelay 		int    `json:"test_message_delay"`
	MessageCountPerDriver	int    `json:"message_count_per_driver"`
}


type TriggerMessage struct {
	TestID  string `json:"test_id"`
	Trigger string `json:"trigger"`
}

type Producer struct {
	producer          *kafka.Producer
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

func CreateProducer(p *kafka.Producer) *Producer {
	return &Producer{
		producer:         p,
		deliveryChannel:  make(chan kafka.Event, 10000),
	}
}

func (prod *Producer) sendTestConfig(testConfiguration TestConfig) error {
	jsonMessage, err := json.Marshal(testConfiguration)
	if err != nil {
		return err
	}

	topic := "test_config"
	err = prod.producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
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

func (prod *Producer) sendTriggerMessage(TestID string) error {
	triggerMessage := TriggerMessage{
		TestID:  TestID,
		Trigger: "YES",
	}

	jsonMessage, err := json.Marshal(triggerMessage)
	if err != nil {
		return err
	}

	topic := "trigger"
	err = prod.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: jsonMessage,
	}, prod.deliveryChannel)

	if err != nil {
		return err
	}
	<-prod.deliveryChannel

	fmt.Printf("Trigger message sent for Test ID: %s\n", TestID)
	return nil
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/ping", handlePostTest)
	http.HandleFunc("/trigger", handleClientTrigger)
	fmt.Println("Starting server on :8080...")
	http.ListenAndServe(":8080", addCorsHeaders(http.DefaultServeMux))
}

func handleClientTrigger(w http.ResponseWriter, r *http.Request) {
	var requestBody RequestBodyTrigger
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		fmt.Printf("Error decoding JSON: %v\n", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	fmt.Printf("Received POST request with body: %+v\n", requestBody)
	w.WriteHeader(http.StatusOK)
	time.Sleep(time.Second * 3)
	prod.sendTriggerMessage(requestBody.TestID)
}

func handlePostTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

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
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Received POST request with body: %+v\n", requestBody)
	fmt.Println("Test_ID:", Test_ID)

	// Send the response to the client
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(Test_ID))

	// Start Kafka-related operations asynchronously

	go startKafkaStuff(Test_ID, requestBody)
	go metricsConsumer.HandleMetricsMessage(requestBody)
}


func startKafkaStuff(Test_ID string, requestBody RequestBody) {
	testConfig := TestConfig{
		TestID:                Test_ID,
		TestType:              requestBody.TestType,
		TestMessageDelay:      requestBody.TestMessageDelay,
		MessageCountPerDriver: requestBody.NumRequests,
	}

	fmt.Printf("Test Config: %+v\n", testConfig)

	prod.sendTestConfig(testConfig)
	// time.Sleep(time.Second * 1)
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
