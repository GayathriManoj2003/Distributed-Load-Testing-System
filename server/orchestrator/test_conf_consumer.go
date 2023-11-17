package main

import (
	"fmt"
	"os"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func main() {
	topic := "test_config"
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":    "localhost:9092",
		"group.id":             "group1",
		"auto.offset.reset":    "smallest",
	})

	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}
	err = consumer.Subscribe(topic, nil)
	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	for {
		ev := consumer.Poll(100)
		switch e := ev.(type){
			case *kafka.Message:
				fmt.Printf("consumed msg: %s\n", string(e.Value))
			case *kafka.Error:
				fmt.Printf("%v\n", e)
		}
	}
}