package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Payload struct {
	UserID    string `json:"userId"`
	ProductID string `json:"productId"`
	Amount    int    `json:"amount"`
}

func postPayload(payload Payload, wg *sync.WaitGroup) {
	defer wg.Done()
	url := "http://localhost:8080/payment"

	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Erro ao converter payload para JSON:", err)
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		fmt.Println("Erro ao fazer a requisição:", err)
		return
	}

	fmt.Printf("URL: %s, STATUS CODE: %d, PAYLOAD: %v\n", url, resp.StatusCode, payload)

	resp.Body.Close()
}

func main() {

	payloads := []Payload{
		{"123", "123", 12},
		{"123", "123", 10},
		{"123", "445", 10},
		{"123", "445", 12},
		{"123", "789", 10},
		{"123", "789", 12},

		{"123", "", 12},
		{"124", "789", 10},
		{"123", "123", 0},
		{"", "123", 12},
	}

	for {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				payload := payloads[rand.Intn(len(payloads))]
				postPayload(payload, &wg)
			}()
		}
		fmt.Println("")
		time.Sleep(5 * time.Second)
	}
}
