package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type BroadcastRequest struct {
	Name       string   `json:"name"`
	Body       string   `json:"body"`
	Recipients []string `json:"recipients"`
}

type BroadcastResponse struct {
	BroadcastID string `json:"broadcast_id"`
	Queued      int    `json:"queued"`
}

type LoadTestResult struct {
	TotalRequests   int
	SuccessCount    int32
	FailureCount    int32
	TotalDuration   time.Duration
	RequestsPerSec  float64
	AvgResponseTime time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration
	Errors          map[string]int
}

func runLoadTest(url string, numRequests int, concurrency int) *LoadTestResult {
	var (
		successCount  int32
		failureCount  int32
		totalRespTime int64
		minRespTime   int64 = int64(^uint64(0) >> 1) // Max int64
		maxRespTime   int64
		errorsMu      sync.Mutex
		errors        = make(map[string]int)
		wg            sync.WaitGroup
		semaphore     = make(chan struct{}, concurrency)
	)

	startTime := time.Now()

	fmt.Printf("\n🚀 Starting load test: %d requests with concurrency %d\n", numRequests, concurrency)
	fmt.Printf("Target: %s\n", url)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(reqNum int) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			reqStart := time.Now()

			// Create request payload
			payload := BroadcastRequest{
				Name: fmt.Sprintf("Load Test Broadcast %d", reqNum),
				Body: fmt.Sprintf("Test message from load test request #%d", reqNum),
				Recipients: []string{
					fmt.Sprintf("+6681234%04d", reqNum%10000),
					fmt.Sprintf("+6689876%04d", reqNum%10000),
				},
			}

			jsonData, _ := json.Marshal(payload)

			// Send HTTP request
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
			reqDuration := time.Since(reqStart)

			// Track response time
			respTimeNs := reqDuration.Nanoseconds()
			atomic.AddInt64(&totalRespTime, respTimeNs)

			// Update min/max response times
			for {
				oldMin := atomic.LoadInt64(&minRespTime)
				if respTimeNs >= oldMin || atomic.CompareAndSwapInt64(&minRespTime, oldMin, respTimeNs) {
					break
				}
			}
			for {
				oldMax := atomic.LoadInt64(&maxRespTime)
				if respTimeNs <= oldMax || atomic.CompareAndSwapInt64(&maxRespTime, oldMax, respTimeNs) {
					break
				}
			}

			if err != nil {
				atomic.AddInt32(&failureCount, 1)
				errorsMu.Lock()
				errors[err.Error()]++
				errorsMu.Unlock()
				return
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				atomic.AddInt32(&failureCount, 1)
				errorsMu.Lock()
				errMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
				errors[errMsg]++
				errorsMu.Unlock()
				return
			}

			// Parse response
			var broadcastResp BroadcastResponse
			if err := json.Unmarshal(body, &broadcastResp); err != nil {
				atomic.AddInt32(&failureCount, 1)
				errorsMu.Lock()
				errors["JSON parse error"]++
				errorsMu.Unlock()
				return
			}

			atomic.AddInt32(&successCount, 1)

			// Progress indicator
			if reqNum%10 == 0 {
				fmt.Print(".")
			}
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return &LoadTestResult{
		TotalRequests:   numRequests,
		SuccessCount:    successCount,
		FailureCount:    failureCount,
		TotalDuration:   totalDuration,
		RequestsPerSec:  float64(numRequests) / totalDuration.Seconds(),
		AvgResponseTime: time.Duration(totalRespTime / int64(numRequests)),
		MinResponseTime: time.Duration(minRespTime),
		MaxResponseTime: time.Duration(maxRespTime),
		Errors:          errors,
	}
}

func printResults(result *LoadTestResult) {
	fmt.Printf("\n📊 Load Test Results\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total Requests:      %d\n", result.TotalRequests)
	fmt.Printf("✅ Success:           %d (%.2f%%)\n", result.SuccessCount, float64(result.SuccessCount)/float64(result.TotalRequests)*100)
	fmt.Printf("❌ Failed:            %d (%.2f%%)\n", result.FailureCount, float64(result.FailureCount)/float64(result.TotalRequests)*100)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("⏱️  Total Duration:    %v\n", result.TotalDuration)
	fmt.Printf("⚡ Requests/sec:      %.2f\n", result.RequestsPerSec)
	fmt.Printf("📈 Avg Response Time: %v\n", result.AvgResponseTime)
	fmt.Printf("⬇️  Min Response Time: %v\n", result.MinResponseTime)
	fmt.Printf("⬆️  Max Response Time: %v\n", result.MaxResponseTime)

	if len(result.Errors) > 0 {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("❌ Errors:")
		for errMsg, count := range result.Errors {
			fmt.Printf("   • %s: %d times\n", errMsg, count)
		}
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

func main() {
	baseURL := "http://localhost:8080/api/broadcasts"

	// Check if server is running
	fmt.Println("🔍 Checking if server is running...")
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		fmt.Printf("❌ Error: Cannot connect to server at %s\n", baseURL)
		fmt.Println("💡 Make sure the server is running: make run-broadcast")
		return
	}
	resp.Body.Close()
	fmt.Println("✅ Server is running\n")

	// Test 1: 100 requests with 10 concurrent connections
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("TEST 1: 100 Requests (Concurrency: 10)")
	fmt.Println("═══════════════════════════════════════════════════════")
	result100 := runLoadTest(baseURL, 100, 10)
	printResults(result100)

	// Wait a bit between tests
	fmt.Println("⏳ Waiting 3 seconds before next test...")
	time.Sleep(3 * time.Second)

	// Test 2: 1000 requests with 50 concurrent connections
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("TEST 2: 1000 Requests (Concurrency: 50)")
	fmt.Println("═══════════════════════════════════════════════════════")
	result1000 := runLoadTest(baseURL, 1000, 50)
	printResults(result1000)

	// Summary comparison
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("📊 COMPARISON SUMMARY")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("100 Requests:  %.2f req/sec | Avg: %v\n", result100.RequestsPerSec, result100.AvgResponseTime)
	fmt.Printf("1000 Requests: %.2f req/sec | Avg: %v\n", result1000.RequestsPerSec, result1000.AvgResponseTime)
	fmt.Println("═══════════════════════════════════════════════════════")
}
