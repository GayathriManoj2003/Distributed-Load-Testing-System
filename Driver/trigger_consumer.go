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

type TriggerMessage struct {
	TestID  string `json:"test_id"`
	Trigger string `json:"trigger"`
}

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

func main() {
	topic := "test_trigger"
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "group1",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	err = consumer.Subscribe(topic, nil)
	if err != nil {
		fmt.Printf("Failed to subscribe to topic: %s\n", err)
		os.Exit(1)
	}

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

				// HTTP Load Testing
				url := "https://api.github.com/" // Replace with the actual URL
				numRequests := 10                // Number of concurrent requests

				var wg sync.WaitGroup
				startTime := time.Now()

				for i := 0; i < numRequests; i++ {
					wg.Add(1)
					go makeRequest(url, &wg)
				}

				wg.Wait()

				duration := time.Since(startTime)
				fmt.Printf("Total time taken for load test: %s\n", duration)

				// End of HTTP Load Testing
			}
		case *kafka.Error:
			fmt.Printf("Kafka consumer error: %v\n", e)
		}
	}
}