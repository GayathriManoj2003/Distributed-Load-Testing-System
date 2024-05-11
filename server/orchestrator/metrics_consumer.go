package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
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
	totalNumRequests   int
	metricsLock        sync.Mutex
	requestBody        RequestBody
	storedAggregatedMetrics map[string]*StoredAggregatedMetrics
}

type StoredAggregatedMetrics struct {
    TotalRequests int     
    MeanLatency   float64 
    MinLatency    float64 
    MaxLatency    float64 
	NumNodes      int
}

func CreateMetricsConsumer() *MetricsConsumer {
	return &MetricsConsumer{
		aggregatedMetrics: make(map[string]*AggregatedMetrics),
		storedAggregatedMetrics:  make(map[string]*StoredAggregatedMetrics),
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
					switch *e.TopicPartition.Topic {
						case "metrics":
							var metrics MetricsMessage
							err := json.Unmarshal(e.Value, &metrics)
							if err != nil {
								fmt.Printf("Error decoding metrics JSON: %v\n", err)
								continue
							}
							go mc.processMetrics(metrics) 
					}

					// Process metrics data
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

	fmt.Printf("\n~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
	fmt.Printf("Received metrics for Test ID: %s, Node ID: %s, No of Requests: %d\n", metrics.TestID, metrics.NodeID, metrics.NoOfReq)
	fmt.Printf("Total Requests: %d\n", mc.aggregatedMetrics[key].TotalRequests)
	fmt.Printf("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")

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
	mc.metricsLock.Lock()
	defer mc.metricsLock.Unlock()

	if mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID] == nil {
		mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID] = &StoredAggregatedMetrics{
			TotalRequests: mc.aggregatedMetrics[key].TotalRequests,
			MeanLatency:   mc.aggregatedMetrics[key].MeanLatency,
			MinLatency:    mc.aggregatedMetrics[key].MinLatency,
			MaxLatency:    mc.aggregatedMetrics[key].MaxLatency,
			NumNodes:      1,
		}
	} else {
        // Update the stored aggregated metrics
        mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].TotalRequests += mc.aggregatedMetrics[key].TotalRequests
		mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].MeanLatency += mc.aggregatedMetrics[key].MeanLatency
		mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].MinLatency = math.Min(mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].MinLatency, mc.aggregatedMetrics[key].MinLatency)
		mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].MaxLatency = math.Max(mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].MaxLatency, mc.aggregatedMetrics[key].MaxLatency)
		mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].NumNodes += 1
	}

	fmt.Printf("\n*****************************************************************************************************************************************\n")
	fmt.Printf("Final Metrics : \n")
	fmt.Printf(" Test ID : %s\n",mc.aggregatedMetrics[key].TestID)
	fmt.Printf(" Node ID : %s\n",mc.aggregatedMetrics[key].NodeID)
	fmt.Printf("  Total Requests: %d\n", mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].TotalRequests)
	fmt.Printf("  Mean Latency: %.2f ms\n", mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].MeanLatency)
	fmt.Printf("  Min Latency: %.2f ms\n", mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].MinLatency)
	fmt.Printf("  Max Latency: %.2f ms\n", mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].MaxLatency)
	fmt.Printf("  Num Nodes: %d\n", mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].NumNodes)
	fmt.Printf("  Max Nodes: %d\n", GetNumActiveNodes())
	fmt.Printf("*****************************************************************************************************************************************\n")


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
	fileName := fmt.Sprintf("Test_Reports/Node_Test_%s_%s_metrics.json", mc.aggregatedMetrics[key].TestID,mc.aggregatedMetrics[key].NodeID)
	err = os.WriteFile(fileName, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return
	}

	fmt.Printf("Aggregated metrics saved to %s\n", fileName)

	if(mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].NumNodes >= GetNumActiveNodes()) {
		mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].MeanLatency /= float64(GetNumActiveNodes())

		storedAggregatedData := struct {
			TestID  string                `json:"test_id"`
			Metrics StoredAggregatedMetrics `json:"metrics"`
		}{
			TestID: mc.aggregatedMetrics[key].TestID,
			Metrics: StoredAggregatedMetrics{
				TotalRequests: mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].TotalRequests,
				MeanLatency:   mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].MeanLatency,
				MinLatency:    mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].MinLatency,
				MaxLatency:    mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].MaxLatency,
				NumNodes:      mc.storedAggregatedMetrics[mc.aggregatedMetrics[key].TestID].NumNodes,
			},
		}
	
		// Convert to JSON
		storedJsonData, err := json.Marshal(storedAggregatedData)
		if err != nil {
			fmt.Printf("Error encoding JSON: %v\n", err)
			return
		}
		message := string(storedJsonData)
		go SendMessageToClients(message)
	
		// Save to a JSON file
		storedFileName := fmt.Sprintf("Final_Test_Reports/%s_metrics.json", mc.aggregatedMetrics[key].TestID)
		err = os.WriteFile(storedFileName, storedJsonData, 0644)
		if err != nil {
			fmt.Printf("Error writing to file: %v\n", err)
			return
		}
	
		fmt.Printf("Stored aggregated metrics saved to %s\n", storedFileName)
	}
}

func (cons *MetricsConsumer) HandleMetricsMessage(requestBody RequestBody) {
	fmt.Println("Metrics Consumer started...")
	cons.requestBody=requestBody
	cons.consumeMetrics()
}