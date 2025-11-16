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

const (
	baseURL     = "http://localhost:8080"
	numTeams    = 5
	usersPerTeam = 10
	numPRs      = 100
	concurrency = 10
	testDuration = 30 * time.Second
)

type Stats struct {
	totalRequests    int64
	successRequests  int64
	failedRequests   int64
	totalLatency     int64
	minLatency       int64
	maxLatency       int64
}

func (s *Stats) recordRequest(duration time.Duration, success bool) {
	atomic.AddInt64(&s.totalRequests, 1)
	if success {
		atomic.AddInt64(&s.successRequests, 1)
	} else {
		atomic.AddInt64(&s.failedRequests, 1)
	}
	
	latencyMs := duration.Milliseconds()
	atomic.AddInt64(&s.totalLatency, latencyMs)
	
	for {
		currentMin := atomic.LoadInt64(&s.minLatency)
		if currentMin == 0 || latencyMs < currentMin {
			if atomic.CompareAndSwapInt64(&s.minLatency, currentMin, latencyMs) {
				break
			}
		} else {
			break
		}
	}
	
	for {
		currentMax := atomic.LoadInt64(&s.maxLatency)
		if latencyMs > currentMax {
			if atomic.CompareAndSwapInt64(&s.maxLatency, currentMax, latencyMs) {
				break
			}
		} else {
			break
		}
	}
}

func (s *Stats) print() {
	total := atomic.LoadInt64(&s.totalRequests)
	success := atomic.LoadInt64(&s.successRequests)
	failed := atomic.LoadInt64(&s.failedRequests)
	avgLatency := float64(0)
	if total > 0 {
		avgLatency = float64(atomic.LoadInt64(&s.totalLatency)) / float64(total)
	}
	
	successRate := float64(0)
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}
	
	fmt.Printf("\n=== Load Test Results ===\n")
	fmt.Printf("Total Requests:    %d\n", total)
	fmt.Printf("Successful:        %d\n", success)
	fmt.Printf("Failed:            %d\n", failed)
	fmt.Printf("Success Rate:      %.2f%%\n", successRate)
	fmt.Printf("Avg Latency:       %.2f ms\n", avgLatency)
	fmt.Printf("Min Latency:       %d ms\n", atomic.LoadInt64(&s.minLatency))
	fmt.Printf("Max Latency:       %d ms\n", atomic.LoadInt64(&s.maxLatency))
	
	if total > 0 {
		rps := float64(total) / testDuration.Seconds()
		fmt.Printf("Requests/sec:      %.2f\n", rps)
	}
}

func makeRequest(method, path string, body interface{}) (int, time.Duration, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return 0, 0, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}
	
	start := time.Now()
	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return 0, 0, err
	}
	
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	duration := time.Since(start)
	
	if err != nil {
		return 0, duration, err
	}
	defer resp.Body.Close()
	
	io.ReadAll(resp.Body) 
	
	return resp.StatusCode, duration, nil
}

func setupTestData() error {
	fmt.Println("Setting up test data...")
	
	for i := 1; i <= numTeams; i++ {
		teamName := fmt.Sprintf("team-%d", i)
		members := make([]map[string]interface{}, usersPerTeam)
		
		for j := 1; j <= usersPerTeam; j++ {
			members[j-1] = map[string]interface{}{
				"user_id":   fmt.Sprintf("u%d-%d", i, j),
				"username":  fmt.Sprintf("User %d-%d", i, j),
				"is_active": true,
			}
		}
		
		body := map[string]interface{}{
			"team_name": teamName,
			"members":   members,
		}
		
		status, _, err := makeRequest("POST", "/team/add", body)
		if err != nil || (status != 201 && status != 400) {
			return fmt.Errorf("failed to create team %s: %v (status: %d)", teamName, err, status)
		}
	}
	
	fmt.Println("Test data setup completed")
	return nil
}

func runLoadTest(stats *Stats, stopCh <-chan struct{}, workerID int) {
	prCounter := workerID * 10000
	
	for {
		select {
		case <-stopCh:
			return
		default:
			operation := prCounter % 10
			prCounter++
			
			var status int
			var duration time.Duration
			var err error
			
			switch operation {
			case 0, 1, 2, 3, 4, 5: 
				teamID := (prCounter % numTeams) + 1
				userID := ((prCounter / numTeams) % usersPerTeam) + 1
				body := map[string]interface{}{
					"pull_request_id":   fmt.Sprintf("pr-%d-w%d-c%d", time.Now().UnixNano()/1000, workerID, prCounter),
					"pull_request_name": fmt.Sprintf("PR %d from worker %d", prCounter, workerID),
					"author_id":         fmt.Sprintf("u%d-%d", teamID, userID),
				}
				status, duration, err = makeRequest("POST", "/pullRequest/create", body)
				
			case 6, 7: 
				teamID := (prCounter % numTeams) + 1
				userID := ((prCounter / numTeams) % usersPerTeam) + 1
				path := fmt.Sprintf("/users/getReview?user_id=u%d-%d", teamID, userID)
				status, duration, err = makeRequest("GET", path, nil)
				
			case 8, 9: 
				teamID := (prCounter % numTeams) + 1
				path := fmt.Sprintf("/team/get?team_name=team-%d", teamID)
				status, duration, err = makeRequest("GET", path, nil)
			}
			
			success := err == nil && status >= 200 && status < 300
			stats.recordRequest(duration, success)
		}
	}
}

func main() {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		fmt.Printf("Service is not available at %s: %v\n", baseURL, err)
		return
	}
	resp.Body.Close()
	
	if resp.StatusCode != 200 {
		fmt.Printf("Service health check failed with status: %d\n", resp.StatusCode)
		return
	}
	
	fmt.Println("Service is available, starting load test...")
	
	if err := setupTestData(); err != nil {
		fmt.Printf("Failed to setup test data: %v\n", err)
		return
	}
	
	stats := &Stats{minLatency: 999999}
	var wg sync.WaitGroup
	stopCh := make(chan struct{})
	
	fmt.Printf("\nStarting load test with %d concurrent workers for %v...\n", concurrency, testDuration)
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		workerID := i
		go func(id int) {
			defer wg.Done()
			runLoadTest(stats, stopCh, id)
		}(workerID)
	}
	
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			total := atomic.LoadInt64(&stats.totalRequests)
			success := atomic.LoadInt64(&stats.successRequests)
			fmt.Printf("Progress: %d requests (%d successful)\n", total, success)
		}
	}()
	
	time.Sleep(testDuration)
	close(stopCh)
	ticker.Stop()
	
	wg.Wait()
	stats.print()

	fmt.Printf("\n=== SLI Requirements Check ===\n")
	
	avgLatency := float64(atomic.LoadInt64(&stats.totalLatency)) / float64(atomic.LoadInt64(&stats.totalRequests))
	if avgLatency <= 300 {
		fmt.Printf("GOOD: Avg latency (%.2f ms) meets SLI requirement (≤ 300 ms)\n", avgLatency)
	} else {
		fmt.Printf("BAD: Avg latency (%.2f ms) exceeds SLI requirement (≤ 300 ms)\n", avgLatency)
	}
	
	total := atomic.LoadInt64(&stats.totalRequests)
	success := atomic.LoadInt64(&stats.successRequests)
	successRate := float64(success) / float64(total) * 100
	
	if successRate >= 99.9 {
		fmt.Printf("GOOD: Success rate (%.2f%%) meets SLI requirement (≥ 99.9%%)\n", successRate)
	} else {
		fmt.Printf("BAD: Success rate (%.2f%%) below SLI requirement (≥ 99.9%%)\n", successRate)
		fmt.Printf("  Note: Some failures may be expected due to concurrent PR creation\n")
	}
	
	rps := float64(total) / testDuration.Seconds()
	if rps >= 5 {
		fmt.Printf("GOOD: RPS (%.2f) exceeds target (≥ 5)\n", rps)
	} else {
		fmt.Printf("ATTENTION: RPS (%.2f) below target (≥ 5)\n", rps)
	}
	
	fmt.Printf("\n=== Performance Summary ===\n")
	fmt.Printf("The service handled %.0f requests/second with average latency of %.2f ms\n", rps, avgLatency)
	if successRate >= 99.0 {
		fmt.Printf("Success rate of %.2f%% is excellent for high concurrency load testing\n", successRate)
	}
}
