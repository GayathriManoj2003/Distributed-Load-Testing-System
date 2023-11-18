package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// TestConfig represents the configuration for a test.
type TestConfig struct {
	TestID                string `json:"test_id"`
	TestType              string `json:"test_type"`
	TestMessageDelay      int    `json:"test_message_delay"`
	MessageCountPerDriver int    `json:"message_count_per_driver"`
}

// TriggerMessage represents a trigger message for a test.
type TriggerMessage struct {
	TestID  string `json:"test_id"`
	Trigger string `json:"trigger"`
}

// makeRequest performs an HTTP GET request to the given URL.
func makeRequest(url string, wg *sync.WaitGroup) {
	defer wg.Done()

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error making request: %s\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
}

// startLoadTest initiates an HTTP load test for the given URL and number of concurrent requests.
func startLoadTest(url string, numRequests int) {
	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go makeRequest(url, &wg)
	}

	wg.Wait()

	duration := time.Since(startTime)
	fmt.Printf("Total time taken for load test: %s\n", duration)
}

// consumeTestConfigMessages subscribes to the specified Kafka topic and processes TestConfig messages.
func consumeTestConfigMessages(consumer *kafka.Consumer, topic string) {
	for {
		ev := consumer.Poll(100)
		switch e := ev.(type) {
		case *kafka.Message:
			var testConfig TestConfig
			err := json.Unmarshal(e.Value, &testConfig)
			if err != nil {
				fmt.Printf("Error decoding JSON: %v\n", err)
				continue
			}

			fmt.Printf("Received TestConfig: %+v\n", testConfig)
			// Optionally, you can store the TestConfig or perform other actions

		case kafka.Error:
			fmt.Printf("Kafka error: %v\n", e)
		}
	}
}

// consumeTriggerMessages subscribes to the specified Kafka topic and processes TriggerMessage messages.
func consumeTriggerMessages(consumer *kafka.Consumer, topic string) {
	for {
		ev := consumer.Poll(100)
		switch e := ev.(type) {
		case *kafka.Message:
			var triggerMsg TriggerMessage
			err := json.Unmarshal(e.Value, &triggerMsg)
			if err != nil {
				fmt.Printf("Error decoding trigger message: %s\n", err)
				continue
			}

			if triggerMsg.Trigger == "YES" {
				fmt.Printf("Received trigger message. Starting load test for TestID: %s\n", triggerMsg.TestID)

				// Replace the URL with the actual URL for your load test
				url := "https://api.github.com/"
				numRequests := 10 // Number of concurrent requests

				startLoadTest(url, numRequests)
			}

		case *kafka.Error:
			fmt.Printf("Kafka consumer error: %v\n", e)
		}
	}
}

// main function for handling test configuration and trigger messages.
func main() {
	testConfigTopic := "test_config"
	triggerTopic := "test_trigger"

	consumerConfig := &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "group1",
		"auto.offset.reset": "earliest",
	}

	testConfigConsumer, err := kafka.NewConsumer(consumerConfig)
	if err != nil {
		fmt.Printf("Failed to create consumer for test config: %s\n", err)
		os.Exit(1)
	}

	triggerConsumer, err := kafka.NewConsumer(consumerConfig)
	if err != nil {
		fmt.Printf("Failed to create consumer for trigger messages: %s\n", err)
		os.Exit(1)
	}

	// Subscribe to test config and trigger topics
	err = testConfigConsumer.SubscribeTopics([]string{testConfigTopic}, nil)
	if err != nil {
		fmt.Printf("Failed to subscribe to topic %s: %s\n", testConfigTopic, err)
		os.Exit(1)
	}

	err = triggerConsumer.Subscribe([]string{triggerTopic}, nil)
	if err != nil {
		fmt.Printf("Failed to subscribe to topic %s: %s\n", triggerTopic, err)
		os.Exit(1)
	}

	fmt.Printf("Subscribed to topics: %s, %s\n", testConfigTopic, triggerTopic)

	// Start goroutines to consume messages for test config and trigger
	go consumeTestConfigMessages(testConfigConsumer, testConfigTopic)
	go consumeTriggerMessages(triggerConsumer, triggerTopic)

	// Keep the main thread running
	select {}
}
