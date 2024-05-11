package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gorilla/websocket"
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
var Kafka_URL string = os.Getenv("BOOTSTRAP_SERVER")

func init() {
	var err error

	ensureDirectoriesExist()

	// Initialize Kafka producer
	p, err = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": Kafka_URL,
		"client.id":         "something",
		"acks":              "all",
	})
	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	// Initialize Kafka consumer
	consumer, err = kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": Kafka_URL,
		"group.id":          "metrics-consumer-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	// Create Kafka AdminClient
    adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
        "bootstrap.servers": Kafka_URL,
    })
    if err != nil {
        fmt.Printf("Failed to create admin client: %s\n", err)
        os.Exit(1)
    }

    defer adminClient.Close()

    // Define topic configurations
    topics := []kafka.TopicSpecification{
        {Topic: "heartbeat", NumPartitions: 1, ReplicationFactor: 1},
        {Topic: "registration", NumPartitions: 1, ReplicationFactor: 1},
        {Topic: "test_config", NumPartitions: 1, ReplicationFactor: 1},
        {Topic: "trigger", NumPartitions: 1, ReplicationFactor: 1},
        {Topic: "metrics", NumPartitions: 1, ReplicationFactor: 1},
    }

    // Create topics
    _, err = adminClient.CreateTopics(context.Background(), topics)
    if err != nil {
        fmt.Printf("Failed to create topics: %v\n", err)
        os.Exit(1)
    }

	// Create instances of Producer and MetricsConsumer
	prod = CreateProducer(p)
	metricsConsumer = CreateMetricsConsumer()
	metricsConsumer.consumer = consumer

	topics1 := []string{"metrics"}
	err = consumer.SubscribeTopics(topics1, nil)
	if err != nil {
		fmt.Printf("Failed to subscribe to topics: %v\n", err)
		os.Exit(1)
	}
	go HandleHeartbeatRegisterTopics()
}

func ensureDirectoriesExist() {
    directories := []string{"Test_Reports", "Final_Test_Reports"}

    for _, dir := range directories {
        if _, err := os.Stat(dir); os.IsNotExist(err) {
            err := os.MkdirAll(dir, 0755)
            if err != nil {
                fmt.Printf("Error creating directory %s: %v\n", dir, err)
            } else {
                fmt.Printf("Directory %s created successfully\n", dir)
            }
        }
    }
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

	fmt.Printf("\n**********************************************************************\n")
	fmt.Printf("Trigger message sent for Test ID: %s\n", TestID)
	fmt.Printf("**********************************************************************\n")
	return nil
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/tests", handlePostTest)
	http.HandleFunc("/trigger", handleClientTrigger)
	http.HandleFunc("/ping", handlePing)
	fmt.Println("Starting server on :8080...")
	http.ListenAndServe(":8080", addCorsHeaders(http.DefaultServeMux))
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleClientTrigger(w http.ResponseWriter, r *http.Request) {
	var requestBody RequestBodyTrigger
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		fmt.Printf("Error decoding JSON: %v\n", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	// fmt.Printf("Received POST request with body: %+v\n", requestBody)
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

	fmt.Printf("\n~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
	fmt.Printf("Received POST request with body: %+v\n", requestBody)
	fmt.Println("Test_ID:", Test_ID)
	fmt.Printf("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")

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

	fmt.Printf("\n***************************************************************************************************************\n")
	fmt.Printf("Test Config: %+v\n", testConfig)
	fmt.Printf("*****************************************************************************************************************\n")


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
