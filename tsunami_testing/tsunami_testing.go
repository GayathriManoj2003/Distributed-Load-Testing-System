// consuming-api/request-building/main.go
package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func main() {
	// Set the delay interval for Tsunami testing
	delayInterval := 500 * time.Millisecond

	// Example: Simulate Tsunami testing with a delay of 500 milliseconds between each request
	for i := 1; i <= 10; i++ {
		go makeRequest(i, delayInterval)
		time.Sleep(delayInterval)
	}

	// Keep the program running for Goroutines to finish
	select {}
}

func makeRequest(requestNumber int, delayInterval time.Duration) {
	c := http.Client{Timeout: time.Duration(1) * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/", nil)
	if err != nil {
		fmt.Printf("error %s", err)
		return
	}
	req.Header.Add("Accept", `application/json`)
	resp, err := c.Do(req)
	if err != nil {
		fmt.Printf("error %s", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error %s", err)
		return
	}
	fmt.Printf("Request %d - Body: %s \n", requestNumber, body)
	fmt.Printf("Request %d - Response status: %s \n", requestNumber, resp.Status)
	time.Sleep(delayInterval) // Introduce delay between requests
}
