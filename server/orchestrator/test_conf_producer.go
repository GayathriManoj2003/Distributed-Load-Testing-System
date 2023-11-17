package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	// "time"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type TestConfig struct {
	TestID             string `json:"test_id"`
	TestType           string `json:"test_type"`
	TestMessageDelay   int    `json:"test_message_delay"`
}

type Producer struct {
	producer          *kafka.Producer
	topic             string
	deliveryChannel   chan kafka.Event
}

func CreateProducer(p *kafka.Producer, topic string) *Producer {
	return &Producer{
		producer:         p,
		topic:            topic,
		deliveryChannel:  make(chan kafka.Event, 10000),
	}
}

func (prod *Producer) sendTestConfig(testConfiguration TestConfig) error {
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

	testConfiguration := TestConfig{
		TestID:           "04bbe3c605e6d2b0bcb274d5ffb9b5ee",
		TestType:         "avalanche",
		TestMessageDelay: 1000,
	}

	err = prod.sendTestConfig(testConfiguration)
	if err != nil {
		log.Fatal(err)
	}
}
