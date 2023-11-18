package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type TestConfig struct {
	TestID                string `json:"test_id"`
	TestType              string `json:"test_type"`
	TestMessageDelay      int    `json:"test_message_delay"`
	MessageCountPerDriver int    `json:"message_count_per_driver"`
}

type TriggerMessage struct {
	TestID  string `json:"test_id"`
	Trigger string `json:"trigger"`
}
type MetricMessage struct {
	NodeID   string      `json:"node_id"`
	TestID   string      `json:"test_id"`
	ReportID string      `json:"report_id"`
	NoOfReq  int         `json:"no_of_req"`
	Metrics  MetricsData `json:"metrics"`
}

// MetricsData structure
type MetricsData struct {
	MeanLatency   int `json:"mean_latency"`
	MedianLatency int `json:"median_latency"`
	MinLatency    int `json:"min_latency"`
	MaxLatency    int `json:"max_latency"`
}
type MetricProducer struct {
	producer        *kafka.Producer
	deliveryChannel chan kafka.Event
}

func NewMetricProducer() *MetricProducer {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"client.id":         "metric-producer",
	})

	if err != nil {
		fmt.Printf("Failed to create metric producer: %s\n", err)
		os.Exit(1)
	}

	return &MetricProducer{
		producer:        p,
		deliveryChannel: make(chan kafka.Event, 10000),
	}
}

// Close closes the MetricProducer
func (mp *MetricProducer) Close() {
	mp.producer.Close()
}

// PublishMetric sends a metric message to the metric_topic
func (mp *MetricProducer) PublishMetric(metricMessage MetricMessage) {
	jsonMessage, err := json.Marshal(metricMessage)
	if err != nil {
		fmt.Printf("Error encoding metric message: %v\n", err)
		return
	}

	topic := "metric_topic"
	err = mp.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: jsonMessage,
	}, mp.deliveryChannel)

	if err != nil {
		fmt.Printf("Failed to produce metric message: %v\n", err)
		return
	}
	<-mp.deliveryChannel

	fmt.Printf("Published metric message:\n%s\n", jsonMessage)
}
func main() {
	topics := []string{"test_config", "trigger"}

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "my-group",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	err = consumer.SubscribeTopics(topics, nil)
	if err != nil {
		fmt.Printf("Failed to subscribe to topics: %v\n", err)
		os.Exit(1)
	}

	run := true
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	var count int
	for run == true {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false
		default:
			ev := consumer.Poll(100)
			switch e := ev.(type) {
			case *kafka.Message:
				fmt.Printf("Received message on topic %s: %s\n", *e.TopicPartition.Topic, string(e.Value))

				switch *e.TopicPartition.Topic {
				case "test_config":
					var testConfig TestConfig
					// count := testConfig.MessageCountPerDriver
					if err := json.Unmarshal(e.Value, &testConfig); err == nil {
						storeTestConfig(testConfig)
						count = testConfig.MessageCountPerDriver
					}
				case "trigger":
					var triggerMessage TriggerMessage
					if err := json.Unmarshal(e.Value, &triggerMessage); err == nil {
						testID := triggerMessage.TestID
						testConfig := retrieveTestConfig(consumer, testID, count)
						performTsunamiTest(testConfig)
					}
				}
			case kafka.Error:
				fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
				run = false
			default:
				// Ignore other events
			}
		}
	}

	fmt.Println("Closing consumer")
	consumer.Close()
}

// Helper function to store test configurations
func storeTestConfig(testConfig TestConfig) {
	// Implement logic to store the test configuration
	fmt.Printf("Storing test configuration: %+v\n", testConfig)
}

// Helper function to retrieve test configuration for a given TestID
func retrieveTestConfig(consumer *kafka.Consumer, testID string, count int) TestConfig {
	// Implement logic to retrieve the test configuration
	// This might involve polling for messages with the specified TestID
	// and decoding the JSON content
	// For simplicity, this function returns a dummy TestConfig
	return TestConfig{
		TestID:                testID,
		TestType:              "Avalanche",
		TestMessageDelay:      0,
		MessageCountPerDriver: count,
	}
}

func performTsunamiTest(testConfig TestConfig) {
	fmt.Printf("Performing Tsunami test for TestID: %s\n", testConfig.TestID)

	// Implement logic to perform Tsunami tests with delays between requests
	interval := time.Duration(testConfig.TestMessageDelay) * time.Millisecond
	var req int = 0

	url := "https://github.com"
	numRequests := testConfig.MessageCountPerDriver
	fmt.Printf("%d requests will be sent\n", numRequests)

	var allLatencies []int     // Store all latencies for calculating cumulative metrics
	var currentLatencies []int // Store latencies for the current interval

	// Create metric producer
	metricProducer := NewMetricProducer()

	// Use a goroutine to periodically send metrics
	go func() {
		for {
			time.Sleep(interval)

			if len(currentLatencies) > 0 {
				allLatencies = append(allLatencies, currentLatencies...)

				meanLatency, medianLatency, minLatency, maxLatency := calculateLatencyMetrics(allLatencies)

				// Create and publish metric message
				metricMessage := MetricMessage{
					NodeID:   testConfig.TestID,
					TestID:   testConfig.TestID,
					ReportID: testConfig.TestID,
					NoOfReq:  req,
					Metrics: MetricsData{
						MeanLatency:   meanLatency,
						MedianLatency: medianLatency,
						MinLatency:    minLatency,
						MaxLatency:    maxLatency,
					},
				}
				metricProducer.PublishMetric(metricMessage)

				// Clear currentLatencies for the next interval
				currentLatencies = nil
				req = 0
			}
		}
	}()

	// Perform HTTP requests with delay
	for i := 1; i <= numRequests; i++ {
		startTime := time.Now()

		// Send HTTP GET request
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error sending HTTP request: %v\n", err)
			return
		}
		defer resp.Body.Close()

		// Measure latency
		latency := int(time.Since(startTime).Milliseconds())

		// Print results
		fmt.Printf("Request %d - Latency: %d ms\n", i, latency)
		req++
		currentLatencies = append(currentLatencies, latency)
	}

	// Close the MetricProducer when done
	// metricProducer.Close()

	fmt.Printf("Tsunami test completed for TestID: %s\n", testConfig.TestID)
}
// Helper function to perform HTTP tests and send metrics at a fixed interval
func performHTTPTest(testConfig TestConfig) {
	// Example: Report metrics every 5 seconds
	interval := 1 * time.Second
	// interval := 10 * time.Millisecond
	var req int = 0
	fmt.Printf("Performing HTTP test for TestID: %s\n", testConfig.TestID)

	url := "https://github.com"
	numRequests := testConfig.MessageCountPerDriver
	fmt.Printf("%d requests will be sent\n", numRequests)

	var allLatencies []int     // Store all latencies for calculating cumulative metrics
	var currentLatencies []int // Store latencies for the current interval

	// Create metric producer
	metricProducer := NewMetricProducer()

	// Use a goroutine to periodically send metrics
	go func() {
		for {
			time.Sleep(interval)

			if len(currentLatencies) > 0 {
				allLatencies = append(allLatencies, currentLatencies...)

				meanLatency, medianLatency, minLatency, maxLatency := calculateLatencyMetrics(allLatencies)

				// Create and publish metric message
				metricMessage := MetricMessage{
					NodeID:   testConfig.TestID,
					TestID:   testConfig.TestID,
					ReportID: testConfig.TestID,
					NoOfReq:  req,
					Metrics: MetricsData{
						MeanLatency:   meanLatency,
						MedianLatency: medianLatency,
						MinLatency:    minLatency,
						MaxLatency:    maxLatency,
					},
				}
				metricProducer.PublishMetric(metricMessage)

				// Clear currentLatencies for the next interval
				currentLatencies = nil
				req = 0
			}
		}
	}()

	// Perform HTTP requests
	for i := 1; i <= numRequests; i++ {
		startTime := time.Now()

		// Send HTTP GET request
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error sending HTTP request: %v\n", err)
			return
		}
		defer resp.Body.Close()

		// Measure latency
		latency := int(time.Since(startTime).Milliseconds())

		// Print results
		fmt.Printf("Request %d - Latency: %d ms\n", i, latency)
		req++
		currentLatencies = append(currentLatencies, latency)
	}

	// Close the MetricProducer when done
	// metricProducer.Close()

	fmt.Printf("HTTP test completed for TestID: %s\n", testConfig.TestID)
}

func calculateLatencyMetrics(latencies []int) (int, int, int, int) {
	// Calculate mean latency
	var sumLatency int
	for _, latency := range latencies {
		sumLatency += latency
	}
	meanLatency := sumLatency / len(latencies)

	// Calculate median latency
	medianLatency := calculateMedian(latencies)

	// Calculate min latency
	sort.Ints(latencies)
	minLatency := latencies[0]

	// Calculate max latency
	maxLatency := latencies[len(latencies)-1]

	return meanLatency, medianLatency, minLatency, maxLatency
}

// Helper function to calculate the median of a slice of integers
func calculateMedian(latencies []int) int {
	n := len(latencies)
	if n == 0 {
		return 0
	}

	sort.Ints(latencies)

	if n%2 == 0 {
		// If the number of latencies is even, calculate the average of the two middle values
		middle1 := latencies[n/2-1]
		middle2 := latencies[n/2]
		return (middle1 + middle2) / 2
	} else {
		// If the number of latencies is odd, return the middle value
		return latencies[n/2]
	}
}
