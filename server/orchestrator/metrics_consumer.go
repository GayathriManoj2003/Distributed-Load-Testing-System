package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gorilla/websocket"
)

type MetricsData struct {
	MeanLatency float64 `json:"mean_latency"`
	MinLatency  float64 `json:"min_latency"`
	MaxLatency  float64 `json:"max_latency"`
}

type AggregatedMetrics struct {
	TotalRequests    int
	MeanLatency      float64
	MinLatency       float64
	MaxLatency       float64
	TotalNumRequests int
	TestID           string
	NodeID           string
}

type MetricsMessage struct {
	NodeID   string      `json:"node_id"`
	TestID   string      `json:"test_id"`
	ReportID string      `json:"report_id"`
	NoOfReq  int         `json:"no_of_req"`
	Metrics  MetricsData `json:"metrics"`
}

type MetricsConsumer struct {
	consumer           *kafka.Consumer
	aggregatedMetrics  map[string]*AggregatedMetrics
	aggregatedRequests int
	totalNumRequests   int
	metricsLock        sync.Mutex
	wsConn             *websocket.Conn 
	requestBody        RequestBody
	storedAggregatedMetrics *StoredAggregatedMetrics
}

type StoredAggregatedMetrics struct {
    TotalRequests int     
    MeanLatency   float64 
    MinLatency    float64 
    MaxLatency    float64 
}

func CreateMetricsConsumer() *MetricsConsumer {
	return &MetricsConsumer{
		aggregatedMetrics: make(map[string]*AggregatedMetrics),
		wsConn:            nil,
		storedAggregatedMetrics:  nil,
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
			ev := mc.consumer.Poll(50)
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
				go mc.processMetrics(metrics)
			case kafka.Error:
				fmt.Fprintf(os.Stderr, "Error: %v\n", e)
			}
		}
	}
}

func (mc *MetricsConsumer) setWebSocketConnection(conn *websocket.Conn) {
    mc.metricsLock.Lock()
    defer mc.metricsLock.Unlock()

    mc.wsConn = conn
}

func (mc *MetricsConsumer) processMetrics(metrics MetricsMessage) {
	mc.metricsLock.Lock()
	defer mc.metricsLock.Unlock()
	// Increment the total number of requests
	mc.totalNumRequests += metrics.NoOfReq

	// Check if TestID and NodeID are present in the dictionary
	key := metrics.TestID + "_" + metrics.NodeID
	if _, ok := mc.aggregatedMetrics[key]; !ok {
		mc.aggregatedMetrics[key] = &AggregatedMetrics{
			TestID: metrics.TestID,
			NodeID: metrics.NodeID,
			TotalRequests: 0,
			MinLatency: metrics.Metrics.MinLatency,
			MaxLatency:	metrics.Metrics.MaxLatency,
			TotalNumRequests: 0,
		}
	}

	// Add metrics values to the aggregated metrics
	mc.aggregatedMetrics[key].TotalNumRequests += metrics.NoOfReq
	mc.aggregatedMetrics[key].TotalRequests += metrics.NoOfReq
	mc.aggregatedMetrics[key].MeanLatency = metrics.Metrics.MeanLatency
	mc.aggregatedMetrics[key].MinLatency = metrics.Metrics.MinLatency
	mc.aggregatedMetrics[key].MaxLatency = metrics.Metrics.MaxLatency

	fmt.Printf("Received metrics for Test ID: %s, Node ID: %s, No of Requests: %d\n", metrics.TestID, metrics.NodeID, metrics.NoOfReq)
	fmt.Printf("Aggregated Metrics:\n")
	fmt.Printf("Updated Aggregated Metrics:\n")
	fmt.Printf("  Test ID: %s\n", key)
	fmt.Printf("  Total Requests: %d\n", mc.aggregatedMetrics[key].TotalRequests)
	fmt.Printf("  Mean Latency: %.2f ms\n", mc.aggregatedMetrics[key].MeanLatency)
	fmt.Printf("  Min Latency: %.2f ms\n", mc.aggregatedMetrics[key].MinLatency)
	fmt.Printf("  Max Latency: %.2f ms\n", mc.aggregatedMetrics[key].MaxLatency)

	json_bytes, err := json.Marshal(mc.aggregatedMetrics[key])
	if err != nil {
		fmt.Printf("Error encoding JSON: %v\n", err)
		return
	}
	message := string(json_bytes)
	go SendMessageToClients(message)
	// Check if the desired number of requests is met
	if mc.aggregatedMetrics[key].TotalNumRequests == mc.requestBody.NumRequests {
		go mc.calculateAndStoreAggregatedMetrics(key)
	}
}

func (mc *MetricsConsumer) calculateAndStoreAggregatedMetrics(key string) {
	if mc.storedAggregatedMetrics == nil {
        mc.storedAggregatedMetrics = &StoredAggregatedMetrics{
            TotalRequests:   mc.aggregatedMetrics[key].TotalRequests,
            MeanLatency:     mc.aggregatedMetrics[key].MeanLatency,
            MinLatency:      mc.aggregatedMetrics[key].MinLatency,
            MaxLatency:      mc.aggregatedMetrics[key].MaxLatency,
        }
    } else {
        // Update the stored aggregated metrics
        mc.storedAggregatedMetrics.TotalRequests += mc.aggregatedMetrics[key].TotalRequests
        mc.storedAggregatedMetrics.MeanLatency += mc.aggregatedMetrics[key].MeanLatency
        mc.storedAggregatedMetrics.MinLatency = math.Min(mc.storedAggregatedMetrics.MinLatency, mc.aggregatedMetrics[key].MinLatency)
        mc.storedAggregatedMetrics.MaxLatency = math.Max(mc.storedAggregatedMetrics.MaxLatency, mc.aggregatedMetrics[key].MaxLatency)
    }
		// Print the updated stored aggregated metrics
		fmt.Printf("  Total Requests: %d\n", mc.storedAggregatedMetrics.TotalRequests)
		fmt.Printf("  Mean Latency: %.2f ms\n", mc.storedAggregatedMetrics.MeanLatency)
		fmt.Printf("  Min Latency: %.2f ms\n", mc.storedAggregatedMetrics.MinLatency)
		fmt.Printf("  Max Latency: %.2f ms\n", mc.storedAggregatedMetrics.MaxLatency)

	// Calculate mean values
	mc.aggregatedMetrics[key].MeanLatency /= float64(mc.aggregatedMetrics[key].TotalRequests)

	// Create a struct to hold the aggregated metrics
	aggregatedData := struct {
		TestID  string      `json:"test_id"`
		Metrics AggregatedMetrics `json:"metrics"`
	}{
		TestID: mc.aggregatedMetrics[key].TestID,
		Metrics: AggregatedMetrics{
			TotalRequests: 		mc.aggregatedMetrics[key].TotalRequests,
			MeanLatency:   		mc.aggregatedMetrics[key].MeanLatency,
			MinLatency:    		mc.aggregatedMetrics[key].MinLatency,
			MaxLatency:    		mc.aggregatedMetrics[key].MaxLatency,
			TotalNumRequests: 	mc.aggregatedMetrics[key].TotalNumRequests,
			TestID:        		mc.aggregatedMetrics[key].TestID,
			NodeID:        		mc.aggregatedMetrics[key].NodeID,
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(aggregatedData)
	if err != nil {
		fmt.Printf("Error encoding JSON: %v\n", err)
		return
	}

	// Save to a JSON file
	fileName := fmt.Sprintf("Test_Reports/%s_metrics.json", mc.aggregatedMetrics[key].TestID)
	err = ioutil.WriteFile(fileName, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return
	}

	fmt.Printf("Aggregated metrics saved to %s\n", fileName)
}

func (cons *MetricsConsumer) HandleMetricsMessage(requestBody RequestBody) {
	fmt.Println("Metrics Consumer started...")
	cons.requestBody=requestBody
	cons.consumeMetrics()
}