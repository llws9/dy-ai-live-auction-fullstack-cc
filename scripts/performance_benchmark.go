package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// PerformanceTestResult 性能测试结果
type PerformanceTestResult struct {
	TotalRequests   int
	SuccessCount    int
	FailCount       int
	AvgResponseTime time.Duration
	MaxResponseTime time.Duration
	MinResponseTime time.Duration
	QPS             float64
}

// ConcurrentBidTest 并发出价测试
func ConcurrentBidTest(auctionID int64, concurrency int, requestsPerUser int) *PerformanceTestResult {
	fmt.Printf("开始并发出价测试: %d 并发用户, 每用户 %d 次请求\n", concurrency, requestsPerUser)

	result := &PerformanceTestResult{
		TotalRequests:   concurrency * requestsPerUser,
		MinResponseTime: time.Hour, // 初始化为最大值
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for j := 0; j < requestsPerUser; j++ {
				reqStart := time.Now()

				// 构造出价请求
				amount := float64(100 + userID*10 + j)
				url := fmt.Sprintf("http://localhost:8082/api/v1/auctions/%d/bids", auctionID)

				reqBody := map[string]interface{}{
					"amount": amount,
				}
				bodyBytes, _ := json.Marshal(reqBody)

				req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
				if err != nil {
					mu.Lock()
					result.FailCount++
					mu.Unlock()
					continue
				}

				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID+1000))

				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)
				if err != nil {
					mu.Lock()
					result.FailCount++
					mu.Unlock()
					continue
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				reqDuration := time.Since(reqStart)

				mu.Lock()
				if resp.StatusCode == 200 || resp.StatusCode == 201 {
					result.SuccessCount++
				} else {
					result.FailCount++
				}

				if reqDuration > result.MaxResponseTime {
					result.MaxResponseTime = reqDuration
				}
				if reqDuration < result.MinResponseTime {
					result.MinResponseTime = reqDuration
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	totalDuration := time.Since(startTime)
	result.AvgResponseTime = totalDuration / time.Duration(result.TotalRequests)
	result.QPS = float64(result.TotalRequests) / totalDuration.Seconds()

	return result
}

// WebSocketConnectionTest WebSocket连接测试
func WebSocketConnectionTest(auctionID int64, connectionCount int) {
	fmt.Printf("开始WebSocket连接测试: %d 个并发连接\n", connectionCount)

	// 注意：这个测试需要实际的WebSocket客户端库
	// 这里只是一个框架示例

	fmt.Printf("WebSocket连接测试需要使用专门的工具，如:\n")
	fmt.Printf("1. wscat -c 'ws://localhost:8083/ws?auction_id=%d'\n", auctionID)
	fmt.Printf("2. 或者使用Artillery、k6等专业工具\n")
}

// APIResponseTimeTest API响应时间测试
func APIResponseTimeTest(auctionID int64, iterations int) {
	fmt.Printf("开始API响应时间测试: %d 次请求\n", iterations)

	urls := []string{
		fmt.Sprintf("http://localhost:8082/api/v1/auctions/%d", auctionID),
		fmt.Sprintf("http://localhost:8082/api/v1/auctions/%d/ranking", auctionID),
		fmt.Sprintf("http://localhost:8081/api/v1/products"),
	}

	for _, url := range urls {
		totalDuration := time.Duration(0)
		successCount := 0

		for i := 0; i < iterations; i++ {
			start := time.Now()
			resp, err := http.Get(url)
			if err != nil {
				continue
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			if resp.StatusCode == 200 {
				successCount++
				totalDuration += time.Since(start)
			}
		}

		if successCount > 0 {
			avgTime := totalDuration / time.Duration(successCount)
			fmt.Printf("URL: %s\n", url)
			fmt.Printf("  成功率: %.2f%%\n", float64(successCount)/float64(iterations)*100)
			fmt.Printf("  平均响应时间: %v\n", avgTime)
		}
	}
}

func main() {
	fmt.Println("=====================================")
	fmt.Println("   直播竞拍系统性能测试")
	fmt.Println("=====================================")
	fmt.Println()

	// 使用已存在的竞拍ID (从之前的测试中知道是3)
	auctionID := int64(3)

	// 测试1: 并发出价测试
	fmt.Println("\n【测试1】并发出价测试")
	fmt.Println("-------------------------------------")
	result := ConcurrentBidTest(auctionID, 50, 5) // 50个用户，每个用户5次出价

	fmt.Printf("\n测试结果:\n")
	fmt.Printf("  总请求数: %d\n", result.TotalRequests)
	fmt.Printf("  成功: %d\n", result.SuccessCount)
	fmt.Printf("  失败: %d\n", result.FailCount)
	fmt.Printf("  成功率: %.2f%%\n", float64(result.SuccessCount)/float64(result.TotalRequests)*100)
	fmt.Printf("  平均响应时间: %v\n", result.AvgResponseTime)
	fmt.Printf("  最大响应时间: %v\n", result.MaxResponseTime)
	fmt.Printf("  最小响应时间: %v\n", result.MinResponseTime)
	fmt.Printf("  QPS: %.2f\n", result.QPS)

	// 测试2: API响应时间测试
	fmt.Println("\n【测试2】API响应时间测试")
	fmt.Println("-------------------------------------")
	APIResponseTimeTest(auctionID, 100)

	// 测试3: WebSocket连接测试（说明）
	fmt.Println("\n【测试3】WebSocket连接测试")
	fmt.Println("-------------------------------------")
	WebSocketConnectionTest(auctionID, 100)

	fmt.Println("\n=====================================")
	fmt.Println("   性能测试完成")
	fmt.Println("=====================================")
}
