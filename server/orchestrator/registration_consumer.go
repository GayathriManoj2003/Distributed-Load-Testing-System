package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// RegistrationMessage struct represents the structure of the registration message.
type RegistrationMessage struct {
	NodeID string `json:"node_id"`
}

// NodeIDList holds the list of NodeIDs with associated timestamps.
var NodeIDList = struct {
	sync.RWMutex
	list map[string]time.Time
}{list: make(map[string]time.Time)}

func ProcessRegistrationMessage(message []byte) error {
	var regMessage RegistrationMessage
	err := json.Unmarshal(message, &regMessage)
	if err != nil {
		return err
	}

	// Add the NodeID to the list
	NodeIDList.Lock()
	NodeIDList.list[regMessage.NodeID] = time.Now()
	// fmt.Printf("Node ID list: %v\n", NodeIDList.list)
	NodeIDList.Unlock()

	fmt.Printf("\n~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
	fmt.Printf("Received registration message. Node ID: %s\n", regMessage.NodeID)
	fmt.Printf("The Driver Nodes Registered Currents Are : ")
	fmt.Printf("%v\n", NodeIDList.list)
	fmt.Printf("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")


	return nil
}



