package main

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type MetricsMessage struct {
	NodeID   string `json:"node_id"`
	TestID   string `json:"test_id"`
	NumRequests int `json:"num_requests"`
	Metrics  struct {
		MeanLatency string `json:"mean_latency"`
		MinLatency  string `json:"min_latency"`
		MaxLatency  string `json:"max_latency"`
	} `json:"metrics"`
}

func main() {
	// Create Kafka producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"client.id":         "metrics-producer",
		"acks":              "all",
	})

	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	// Produce a single metrics message
	metricsMessage := MetricsMessage{
		NodeID:   "node-1",
		TestID:   "test-1",
		NumRequests: 5,
		Metrics: struct {
			MeanLatency string `json:"mean_latency"`
			MinLatency  string `json:"min_latency"`
			MaxLatency  string `json:"max_latency"`
		}{
			MeanLatency: "10",
			MinLatency:  "1",
			MaxLatency:  "20",
		},
	}

	jsonMessage, err := json.Marshal(metricsMessage)
	if err != nil {
		fmt.Printf("Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	topic := "metrics"
	err = producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: jsonMessage,
	}, nil)

	if err != nil {
		fmt.Printf("Failed to produce message: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Metrics message produced successfully")

	// Wait for the producer to flush messages before exiting
	producer.Flush(5000)
}
