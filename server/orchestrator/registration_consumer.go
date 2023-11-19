package main


import (
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"sync"
	"time"
)
// RegistrationMessage struct represents the structure of the registration message.
type RegistrationMessage struct {
	NodeID string `json:"node_id"`
}

// ConsumerConfig holds the configuration for the Kafka consumer.
var ConsumerConfig = &kafka.ConfigMap{
	"bootstrap.servers": "localhost:9092",
	"group.id":          "my-group",
	"auto.offset.reset": "earliest",
}

// NodeIDList holds the list of NodeIDs with associated timestamps.
var NodeIDList = struct {
	sync.RWMutex
	list map[string]time.Time
}{list: make(map[string]time.Time)}

// func main() {
// 	consumer, err := kafka.NewConsumer(ConsumerConfig)
// 	if err != nil {
// 		fmt.Printf("Error creating consumer: %v\n", err)
// 		return
// 	}
// 	defer consumer.Close()

// 	// Subscribe to the "registration" topic
// 	err = consumer.SubscribeTopics([]string{"registration"}, nil)
// 	if err != nil {
// 		fmt.Printf("Error subscribing to topic: %v\n", err)
// 		return
// 	}

// 	fmt.Println("Consumer started. Waiting for messages...")

// 	for {
// 		msg, err := consumer.ReadMessage(-1)
// 		if err == nil {
// 			// Process the received message
// 			err = processRegistrationMessage(msg.Value)
// 			if err != nil {
// 				fmt.Printf("Error processing message: %v\n", err)
// 			}
// 		} else {
// 			fmt.Printf("Error receiving message: %v\n", err)
// 		}
// 	}
// }

func HandleRegisterTopic() {
	consumer, err := kafka.NewConsumer(ConsumerConfig)
	if err != nil {
		fmt.Printf("Error creating consumer: %v\n", err)
		return
	}
	defer consumer.Close()

	// Subscribe to the "registration" topic
	err = consumer.SubscribeTopics([]string{"registration"}, nil)
	if err != nil {
		fmt.Printf("Error subscribing to topic: %v\n", err)
		return
	}

	fmt.Println("Consumer started. Waiting for messages...")
	for {
		msg, err := consumer.ReadMessage(-1)
		if err == nil {
			// Process the received message
			err = processRegistrationMessage(msg.Value)
			if err != nil {
				fmt.Printf("Error processing message: %v\n", err)
			}
		} else {
			fmt.Printf("Error receiving message: %v\n", err)
		}
	}
}

func processRegistrationMessage(message []byte) error {
	var regMessage RegistrationMessage
	err := json.Unmarshal(message, &regMessage)
	if err != nil {
		return err
	}

	// Add the NodeID to the list
	NodeIDList.Lock()
	NodeIDList.list[regMessage.NodeID] = time.Now()
	fmt.Printf("Node ID list: %v\n", NodeIDList.list)
	NodeIDList.Unlock()

	fmt.Printf("Received registration message. Node ID: %s\n", regMessage.NodeID)

	return nil
}



