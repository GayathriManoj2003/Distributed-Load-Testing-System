package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type AggregatedMetrics struct {
	TotalRequests      int
	MeanLatency        float64
	MinLatency         float64
	MaxLatency         float64
	TotalNumRequests   int // New field to store the total number of requests
	TestID             string
	NodeID             string
}

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

type MetricsConsumer struct {
	consumer           *kafka.Consumer
	aggregatedMetrics  map[string]*AggregatedMetrics // Dictionary to store aggregated metrics by TestID and NodeID
	aggregatedRequests int
	totalNumRequests   int // New field to store the total number of requests across all messages
	metricsLock        sync.Mutex
}

type Metrics struct {
	TotalRequests int     `json:"total_requests"`
	MeanLatency   float64 `json:"mean_latency"`
	MinLatency    float64 `json:"min_latency"`
	MaxLatency    float64 `json:"max_latency"`
}

func CreateMetricsConsumer() *MetricsConsumer {
	return &MetricsConsumer{
		aggregatedMetrics: make(map[string]*AggregatedMetrics),
	}
}

func (mc *MetricsConsumer) consumeMetrics() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			mc.consumer.Close()
			os.Exit(0)
		default:
			ev := mc.consumer.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				var metrics MetricsMessage
				err := json.Unmarshal(e.Value, &metrics)
				if err != nil {
					fmt.Printf("Error decoding metrics JSON: %v\n", err)
					continue
				}

				// Process metrics data
				mc.processMetrics(metrics)
			case kafka.Error:
				fmt.Fprintf(os.Stderr, "Error: %v\n", e)
			}
		}
	}
}

func (mc *MetricsConsumer) processMetrics(metrics MetricsMessage) {
	mc.metricsLock.Lock()
	defer mc.metricsLock.Unlock()

	// Increment the total number of requests
	mc.totalNumRequests += metrics.NumRequests

	// Check if TestID and NodeID are present in the dictionary
	key := metrics.TestID + "_" + metrics.NodeID
	if _, ok := mc.aggregatedMetrics[key]; !ok {
		mc.aggregatedMetrics[key] = &AggregatedMetrics{
			TestID: metrics.TestID,
			NodeID: metrics.NodeID,
		}
	}
	fmt.Printf("Aggregated metrics : Test ID: %s, Node ID: %s, Total Requests: %d, Mean Latency: %f, Min Latency: %f, Max Latency: %f, Total Num Requests: %d\n",
		mc.aggregatedMetrics[key].TestID, mc.aggregatedMetrics[key].NodeID,
		mc.aggregatedMetrics[key].TotalRequests, mc.aggregatedMetrics[key].MeanLatency,
		mc.aggregatedMetrics[key].MinLatency, mc.aggregatedMetrics[key].MaxLatency,
		mc.totalNumRequests)

	// Add metrics values to the aggregated metrics
	mc.aggregatedMetrics[key].TotalRequests += 1
	meanLatency, _ := strconv.ParseFloat(metrics.Metrics.MeanLatency, 64)
	mc.aggregatedMetrics[key].MeanLatency += (meanLatency)
	minLatency, _ := strconv.ParseFloat(metrics.Metrics.MinLatency, 64)
	mc.aggregatedMetrics[key].MinLatency = math.Max(mc.aggregatedMetrics[key].MinLatency, minLatency)
	maxLatency, _ := strconv.ParseFloat(metrics.Metrics.MaxLatency, 64)
	mc.aggregatedMetrics[key].MaxLatency = math.Max(mc.aggregatedMetrics[key].MaxLatency, maxLatency)

	fmt.Printf("Received metrics for Test ID: %s, Node ID: %s, No of Requests: %d\n", metrics.TestID, metrics.NodeID, metrics.NumRequests)

	// Check if the desired number of requests is met
	if mc.totalNumRequests == 10 {
		mc.calculateAndStoreAggregatedMetrics(key)
	}
}

func (mc *MetricsConsumer) calculateAndStoreAggregatedMetrics(key string) {
	// Calculate mean values
	mc.aggregatedMetrics[key].MeanLatency /= float64(mc.aggregatedMetrics[key].TotalRequests)
	// Create a struct to hold the aggregated metrics
	aggregatedData := struct {
		TestID       string  `json:"test_id"`
		Metrics      Metrics `json:"metrics"`
	}{
		TestID: mc.aggregatedMetrics[key].TestID,
		Metrics: Metrics{
			TotalRequests: mc.aggregatedMetrics[key].TotalRequests,
			MeanLatency:   mc.aggregatedMetrics[key].MeanLatency,
			MinLatency:    mc.aggregatedMetrics[key].MinLatency,
			MaxLatency:    mc.aggregatedMetrics[key].MaxLatency,
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(aggregatedData)
	if err != nil {
		fmt.Printf("Error encoding JSON: %v\n", err)
		return
	}

	// Save to a JSON file
	fileName := fmt.Sprintf("%s_%s_metrics.json", mc.aggregatedMetrics[key].TestID, mc.aggregatedMetrics[key].NodeID)
	err = ioutil.WriteFile(fileName, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return
	}

	fmt.Printf("Aggregated metrics saved to %s\n", fileName)
}

func main() {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "metrics-consumer-group",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	metricsConsumer := CreateMetricsConsumer()
	metricsConsumer.consumer = consumer

	topics := []string{"metrics"}
	consumer.SubscribeTopics(topics, nil)

	fmt.Println("Metrics Consumer started...")
	metricsConsumer.consumeMetrics()
}
