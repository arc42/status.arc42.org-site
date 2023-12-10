package main

import (
	"fmt"
	"sync"
)

var mutex sync.Mutex

// ApiResponse represents the structure of the API response
type ApiResponse struct {
	Value1 string
	Type   string
}

type ApiCalls struct {
	MyValue string
	MyType  string
}

// this func called in parallel channels
func callAPI(url string, myType string, res *ApiResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	mutex.Lock()
	defer mutex.Unlock()

	res.Value1 = url
	res.Type = myType

	println("API called with type", myType)
}

// try out Goroutines
func main() {
	var wg sync.WaitGroup
	results := make([]ApiResponse, 3)

	urls := []ApiCalls{
		{"endpoint1", "D7"},
		{"endpoint2", "D30"},
		{"endpoint3", "M12"}}

	for index, url := range urls {
		wg.Add(1)
		/*		results[index].Value1 = ""
				results[index].Type = ""
		*/
		go callAPI(url.MyValue, url.MyType, &results[index], &wg)

	}

	// wait for goroutines to finish
	wg.Wait()

	fmt.Println("Results:")
	fmt.Println("==========")

	for _, r := range results {
		println(r.Value1, r.Type)
	}
}
