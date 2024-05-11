package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Heartbeat struct {
	NodeID    string `json:"node_id"`
	Heartbeat string `json:"heartbeat"`
	Timestamp string `json:"timestamp"`
}

type FailureMessage struct {
	NodeID string `json:"NodeID"`
}

func HandleHeartbeatRegisterTopics() {

	var err error

	consumer, err = kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": Kafka_URL,
		"group.id":          "heartbeat-register-consumer-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	topics := []string{"heartbeat", "registration"}
	err = consumer.SubscribeTopics(topics, nil)
	if err != nil {
		fmt.Printf("Failed to subscribe to topics: %v\n", err)
		os.Exit(1)
	}
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Periodically check and remove nodes that haven't sent a heartbeat in 15 seconds
	go func() {
		for {
			time.Sleep(5 * time.Second) // Adjust the frequency based on your requirements
			NodeIDList.RLock()
			for nodeID, lastHeartbeat := range NodeIDList.list {
				if time.Since(lastHeartbeat) > 15*time.Second {
					// Remove the nodeID from the list
					NodeIDList.RUnlock()
					NodeIDList.Lock()
					delete(NodeIDList.list, nodeID)
					NodeIDList.Unlock()
					NodeIDList.RLock()

					nodeIDMessage := FailureMessage{NodeID: nodeID}

					// Marshal the struct into a JSON-formatted string
					jsonData, err := json.Marshal(nodeIDMessage)
					if err != nil {
						fmt.Printf("Error encoding JSON: %v\n", err)
						return
					}
					message := string(jsonData)
					// formattedString := fmt.Sprintf("{'NodeID': '%s'}", nodeID)
					go SendMessageToClients(message)
					fmt.Printf("\n~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
					fmt.Printf("NodeID %s removed from the list\n", nodeID)
					fmt.Printf("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
				}
			}
			NodeIDList.RUnlock()
		}
	}()

	run := true
	for run {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false
		default:
			ev := consumer.Poll(100)
			switch e := ev.(type) {
				case *kafka.Message:
					switch *e.TopicPartition.Topic {
						case "registration":
							ProcessRegistrationMessage(e.Value)
						case "heartbeat":
							var heartbeat Heartbeat
							if err := json.Unmarshal(e.Value, &heartbeat); err == nil {
								processHeartbeat(heartbeat)
							}
					}

				case kafka.Error:
					fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
					run = false
			}
		}
	}

	fmt.Println("Closing consumer")
	consumer.Close()
}

func processHeartbeat(heartbeat Heartbeat) {
	// Implement logic to process the received heartbeat

	NodeIDList.Lock()
	NodeIDList.list[heartbeat.NodeID] = time.Now()
	NodeIDList.Unlock()
}

func GetNumActiveNodes() int {
	NodeIDList.RLock()
	defer NodeIDList.RUnlock()
	return len(NodeIDList.list)
}
