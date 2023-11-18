package main

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type TestConfig struct {
	TestID                string `json:"test_id"`
	TestType              string `json:"test_type"`
	TestMessageDelay      int    `json:"test_message_delay"`
	MessageCountPerDriver int    `json:"message_count_per_driver"`
}

func main() {
	topic := "test_config"
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "group1",
		"auto.offset.reset": "smallest",
	})

	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}
	err = consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		fmt.Printf("Failed to subscribe to topic %s: %s\n", topic, err)
		os.Exit(1)
	}

	fmt.Printf("Subscribed to topic: %s\n", topic)

	for {
		ev := consumer.Poll(100)
		switch e := ev.(type) {
		case *kafka.Message:
			// Deserialize the JSON message into TestConfig
			var testConfig TestConfig
			err := json.Unmarshal(e.Value, &testConfig)
			if err != nil {
				fmt.Printf("Error decoding JSON: %v\n", err)
				continue
			}

			// Print or handle the received TestConfig as needed
			fmt.Printf("Received TestConfig: %+v\n", testConfig)

			// Optionally, you can store the TestConfig in a data structure or perform other actions
			// For example, you can create a slice to store all received TestConfig objects
			// testConfigs = append(testConfigs, testConfig)

		case kafka.Error:
			fmt.Printf("Kafka error: %v\n", e)
		}
	}
}