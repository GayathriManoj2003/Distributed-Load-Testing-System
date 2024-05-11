package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

var Kafka_URL string = os.Getenv("BOOTSTRAP_SERVER")
type RegistrationMessage struct {
	NodeID string `json:"node_id"`
}

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

// Heartbeat structure
type Heartbeat struct {
	NodeID    string `json:"node_id"`
	Heartbeat string `json:"heartbeat"`
	Timestamp string `json:"timestamp"`
}

// MetricsData structure
type MetricsData struct {
	MeanLatency float64 `json:"mean_latency"`
	MinLatency  float64 `json:"min_latency"`
	MaxLatency  float64 `json:"max_latency"`
}
type MetricProducer struct {
	producer        *kafka.Producer
	deliveryChannel chan kafka.Event
}
type HeartbeatProducer struct {
	producer        *kafka.Producer
	deliveryChannel chan kafka.Event
}

var current_latency_mutex sync.Mutex

func generateUniqueToken() (string, error) {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	token := hex.EncodeToString(randomBytes)
	return token, nil
}

func NewMetricProducer() *MetricProducer {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": Kafka_URL,
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
func NewHeartbeatProducer() *HeartbeatProducer {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": Kafka_URL,
		"client.id":         "heartbeat-producer",
	})

	if err != nil {
		fmt.Printf("Failed to create metric producer: %s\n", err)
		os.Exit(1)
	}

	return &HeartbeatProducer{
		producer:        p,
		deliveryChannel: make(chan kafka.Event, 10000),
	}
}

// Close closes the MetricProducer
func (mp *MetricProducer) Close() {
	mp.producer.Close()
}

// Close closes the HeartbeatProducer
func (mp *HeartbeatProducer) Close() {
	mp.producer.Close()
}

// PublishMetric sends a metric message to the metric_topic
func (mp *MetricProducer) PublishMetric(metricMessage MetricMessage) {
	jsonMessage, err := json.Marshal(metricMessage)
	if err != nil {
		fmt.Printf("Error encoding metric message: %v\n", err)
		return
	}

	topic := "metrics"
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
func (hp *HeartbeatProducer) PublishHeartbeat(heartbeatMessage Heartbeat) {

	jsonMessage, err := json.Marshal(heartbeatMessage)
	if err != nil {
		fmt.Printf("Error encoding heartbeat message: %v\n", err)
		return
	}

	topic := "heartbeat"
	err = hp.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: jsonMessage,
	}, hp.deliveryChannel)

	if err != nil {
		fmt.Printf("Failed to produce heartbeat message: %v\n", err)
		return
	}
	<-hp.deliveryChannel

	// fmt.Printf("Published heartbeat message:\n%s\n", jsonMessage)
}
func produceHeartbeat(done chan bool, nodeID string) {
	// Infinite loop for continuous heartbeat production
	heartbeatProducer := NewHeartbeatProducer()
	for {
		select {
		case <-time.After(time.Second): // Produce heartbeat every second
			heartbeat := Heartbeat{
				NodeID:    nodeID,
				Heartbeat: "YES",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			// heartbeatJSON, err := json.Marshal(heartbeat)
			// if err != nil {
			// 	fmt.Println("Error encoding heartbeat:", err)
			// 	continue
			// }
			// fmt.Println(string(heartbeatJSON)) // Replace this with your logic to send the heartbeat
			heartbeatProducer.PublishHeartbeat(heartbeat)
		case <-done: // Signal to stop the goroutine
			return
		}
	}
}
func main() {
	done := make(chan bool)
	// Generate nodeID
	nodeID, err := generateUniqueToken()
	go produceHeartbeat(done, nodeID)
	topics := []string{"test_config", "trigger"}

	// Create the producer for metric messages
	metricProducer := NewMetricProducer()

	if err != nil {
		fmt.Println("Error generating unique token:", err)
		os.Exit(1)
	}

	// Send registration message to orchestrator
	err = sendRegistrationMessage(metricProducer, nodeID)
	if err != nil {
		fmt.Printf("Failed to send registration message: %s\n", err)
		os.Exit(1)
	}

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": Kafka_URL,
		"group.id":          nodeID,
		"auto.offset.reset": "latest",
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
	var test TestConfig
	for run {
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
					if err := json.Unmarshal(e.Value, &testConfig); err == nil {
						test = testConfig
					}
				case "trigger":
					var triggerMessage TriggerMessage
					if err := json.Unmarshal(e.Value, &triggerMessage); err == nil {
						if triggerMessage.Trigger == "YES" && triggerMessage.TestID == test.TestID {
							if test.TestType == "Tsunami" {
								performTsunamiTest(test, nodeID)
							}
							if test.TestType == "Avalanche" {
								performAvalancheTest(test, nodeID)
							}

						}
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
	<-done
	fmt.Println("Producer stopped.")
}

// Helper function to send registration messages
func sendRegistrationMessage(prod *MetricProducer, nodeID string) error {
	regMessage := RegistrationMessage{
		NodeID: nodeID,
	}

	jsonMessage, err := json.Marshal(regMessage)
	if err != nil {
		return err
	}

	topic := "registration"
	err = prod.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: jsonMessage,
	}, prod.deliveryChannel)

	if err != nil {
		return err
	}
	<-prod.deliveryChannel

	fmt.Printf("Registration message sent. Node ID: %s\n", nodeID)
	return nil
}

func performTsunamiTest(testConfig TestConfig, nodeID string) {
	fmt.Printf("Performing Tsunami test for TestID: %s\n", testConfig.TestID)

	// Implement logic to perform Tsunami tests with delays between requests
	interval := time.Duration(testConfig.TestMessageDelay) * time.Millisecond
	var req int = 0

	url := "https://github.com"
	numRequests := testConfig.MessageCountPerDriver
	fmt.Printf("%d requests will be sent\n", numRequests)

	var allLatencies []float64     // Store all latencies for calculating cumulative metrics
	var currentLatencies []float64 // Store latencies for the current interval

	// Create metric producer
	metricProducer := NewMetricProducer()

	// Use a goroutine to periodically send metrics
	go func() {
		for {

			if len(currentLatencies) > 0 {
				current_latency_mutex.Lock()
				allLatencies = append(allLatencies, currentLatencies...)

				meanLatency, minLatency, maxLatency := calculateLatencyMetrics(allLatencies)

				// Create and publish metric message
				metricMessage := MetricMessage{
					NodeID:   nodeID,
					TestID:   testConfig.TestID,
					ReportID: testConfig.TestID,
					NoOfReq:  req,
					Metrics: MetricsData{
						MeanLatency: meanLatency,
						MinLatency:  minLatency,
						MaxLatency:  maxLatency,
					},
				}
				metricProducer.PublishMetric(metricMessage)

				// Clear currentLatencies for the next interval
				currentLatencies = nil
				req = 0
				current_latency_mutex.Unlock()
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
		latency := float64(time.Since(startTime).Milliseconds())

		// Print results
		fmt.Printf("Request %d - Latency: %0.3f ms\n", i, latency)
		current_latency_mutex.Lock()
		req++
		currentLatencies = append(currentLatencies, latency)
		current_latency_mutex.Unlock()
		timer := time.After(interval)
		<-timer // block until timer channel sends a value
	}

	// Close the MetricProducer when done
	// metricProducer.Close()

	fmt.Printf("Tsunami test completed for TestID: %s\n", testConfig.TestID)
}

// Helper function to perform HTTP tests and send metrics at a fixed interval
func performAvalancheTest(testConfig TestConfig, nodeID string) {
	// Example: Report metrics every 5 seconds

	// interval := 10 * time.Millisecond
	var req int = 0
	fmt.Printf("Performing HTTP test for TestID: %s\n", testConfig.TestID)

	url := "https://github.com"
	numRequests := testConfig.MessageCountPerDriver
	fmt.Printf("%d requests will be sent\n", numRequests)

	var allLatencies []float64     // Store all latencies for calculating cumulative metrics
	var currentLatencies []float64 // Store latencies for the current interval

	// Create metric producer
	metricProducer := NewMetricProducer()

	// Use a goroutine to periodically send metrics
	go func() {
		for {

			if len(currentLatencies) > 0 {
				current_latency_mutex.Lock()

				allLatencies = append(allLatencies, currentLatencies...)

				meanLatency, minLatency, maxLatency := calculateLatencyMetrics(allLatencies)

				// Create and publish metric message
				metricMessage := MetricMessage{
					NodeID:   nodeID,
					TestID:   testConfig.TestID,
					ReportID: testConfig.TestID,
					NoOfReq:  req,
					Metrics: MetricsData{
						MeanLatency: meanLatency,
						MinLatency:  minLatency,
						MaxLatency:  maxLatency,
					},
				}
				metricProducer.PublishMetric(metricMessage)

				// Clear currentLatencies for the next interval
				currentLatencies = nil
				req = 0
				current_latency_mutex.Unlock()

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
		latency := float64(time.Since(startTime).Milliseconds())

		// Print results
		fmt.Printf("Request %d - Latency: %0.02f ms\n", i, latency)
		current_latency_mutex.Lock()
		req++
		currentLatencies = append(currentLatencies, latency)
		current_latency_mutex.Unlock()

	}

	// Close the MetricProducer when done
	// metricProducer.Close()

	fmt.Printf("HTTP test completed for TestID: %s\n", testConfig.TestID)
}

func calculateLatencyMetrics(latencies []float64) (float64, float64, float64) {
	// Calculate mean latency
	var sumLatency float64
	for _, latency := range latencies {
		sumLatency += latency
	}
	meanLatency := sumLatency / float64(len(latencies))

	// Calculate median latency
	// medianLatency := calculateMedian(latencies)

	// Calculate min latency
	sort.Float64s(latencies)
	minLatency := latencies[0]

	// Calculate max latency
	maxLatency := latencies[len(latencies)-1]

	return meanLatency, minLatency, maxLatency
}