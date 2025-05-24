package broadcast

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/Ccccraz/cogmoteGO/internal/commonTypes"
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/gin-gonic/gin"
)

var (
	logKey = "broadcast"
)

// broadcast endpoint
type BroadcastEndpoint struct {
	mu          sync.Mutex    // mutex to protect subscribers
	subscribers []chan []byte // all subscribers
	history     [][]byte      // history of data
}

var (
	broadEndpoints   = make(map[string]*BroadcastEndpoint) // all broadcast endpoints
	broadEndpointsMu sync.RWMutex                          // mutex to protect broadcast endpoints
)

// add default data endpoint
func init() {
	broadEndpoints["default"] = &BroadcastEndpoint{
		subscribers: make([]chan []byte, 0),
	}
}

func GetBroadcasts(c *gin.Context) {
	broadEndpointsMu.RLock()
	defer broadEndpointsMu.RUnlock()

	keys := make([]string, 0, len(broadEndpoints))
	for k := range broadEndpoints {
		keys = append(keys, k)
	}

	c.JSON(http.StatusOK, gin.H{
		"broadcast_endpoints": keys,
	})
}

// create new data broadcast endpoint
func CreateBroadcast(c *gin.Context) {
	var request struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "without valid broadcast name",
			Detail: err.Error(),
		})
		return
	}

	broadEndpointsMu.Lock()
	defer broadEndpointsMu.Unlock()

	if _, exists := broadEndpoints[request.Name]; exists {
		c.JSON(http.StatusConflict, commonTypes.APIError{
			Error:  fmt.Sprintf("data broadcast endpoint: %s already exists", request.Name),
			Detail: "",
		})
		return
	}

	broadEndpoints[request.Name] = &BroadcastEndpoint{
		subscribers: make([]chan []byte, 0),
	}
	c.Status(http.StatusCreated)
}

// broadcast data to all subscribers when data update
func BroadcastData(c *gin.Context) {
	name := c.Param("name")

	broadEndpointsMu.RLock()
	endpoint, exists := broadEndpoints[name]
	broadEndpointsMu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, commonTypes.APIError{
			Error:  fmt.Sprintf("data broadcast endpoint: %s does not exist", name),
			Detail: "",
		})
		return
	}

	// read raw data
	data, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "invalid data format",
			Detail: err.Error(),
		})
		return
	}

	endpoint.mu.Lock()
	wg := sync.WaitGroup{}
	for _, ch := range endpoint.subscribers {
		wg.Add(1)
		go func(c chan []byte) {
			defer wg.Done()

			select {
			case c <- data:
			default:
				logger.Logger.Warn(
					"channel is full: ",
					slog.Group(
						logKey,
						slog.String("endpoint", name),
					))
			}
		}(ch)

	}
	wg.Wait()
	endpoint.mu.Unlock()

	endpoint.history = append(endpoint.history, data)

	c.Status(http.StatusOK)
}

// Subscribe to the broadcast endpoint and receive updates via Server-Sent Events (SSE)
func SubscribeBroadcast(c *gin.Context) {
	name := c.Param("name")

	broadEndpointsMu.RLock()
	endpoint, exists := broadEndpoints[name]
	broadEndpointsMu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, commonTypes.APIError{
			Error:  fmt.Sprintf("data broadcast endpoint: %s does not exist", name),
			Detail: "",
		})
		return
	}

	endpoint.mu.Lock()

	// send history data
	if len(endpoint.history) > 0 {
		for _, data := range endpoint.history {
			c.SSEvent("message", data)
			c.Writer.Flush()
		}
	}

	// create new subscriber channel
	ch := make(chan []byte, 10)

	// add subscriber to the endpoint
	endpoint.subscribers = append(endpoint.subscribers, ch)
	endpoint.mu.Unlock()

	// listen for data updates and send them to the subscriber
	defer func() {
		// remove subscriber from the endpoint
		endpoint.mu.Lock()
		defer endpoint.mu.Unlock()
		for i, sub := range endpoint.subscribers {
			if sub == ch {
				endpoint.subscribers = append(endpoint.subscribers[:i], endpoint.subscribers[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	for {
		select {
		case data := <-ch:
			// send data by SSE
			c.SSEvent("message", data)
			c.Writer.Flush()
		case <-c.Writer.CloseNotify():
			// close subscriber channel
			return
		}
	}
}

type MockTrialData struct {
	TrialId        uint   `json:"trial_id"`
	TrialStartTime int64  `json:"trial_start_time"`
	TrialStopTime  int64  `json:"trial_stop_time"`
	TrialResult    string `json:"trial_result"`
}

func GenMockTrialData(ctx context.Context, ch chan<- MockTrialData) {
	rand.NewSource(time.Now().UnixNano())

	results := []string{"correct", "incorrect", "timeout"}

	var trialId uint = 1
	for {
		select {
		case <-ctx.Done():
			return
		default:

			data := MockTrialData{
				TrialId:        trialId,
				TrialStartTime: time.Now().Unix(),
				TrialStopTime:  0,
				TrialResult:    results[rand.Intn(len(results))],
			}

			time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)
			data.TrialStopTime = time.Now().Unix()

			trialId++

			select {
			case ch <- data:
			case <-ctx.Done():
				return
			}
		}
	}
}

func GetMockData(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	mockDataChan := make(chan MockTrialData, 10)
	go GenMockTrialData(ctx, mockDataChan)

	for {
		select {
		case data, ok := <-mockDataChan:
			if !ok {
				c.JSON(http.StatusInternalServerError, commonTypes.APIError{
					Error:  "internal server error",
					Detail: "",
				})
				return
			}

			c.SSEvent("message", data)
			c.Writer.Flush()
		case <-c.Writer.CloseNotify():
			return
		case <-c.Done():
			return
		}
	}
}

func DeleteBroadcast(c *gin.Context) {
	name := c.Param("name")

	broadEndpointsMu.Lock()
	defer broadEndpointsMu.Unlock()

	if _, exists := broadEndpoints[name]; !exists {
		c.JSON(http.StatusNotFound, commonTypes.APIError{
			Error:  fmt.Sprintf("data broadcast endpoint: %s does not exist", name),
			Detail: "",
		})
		return
	}

	delete(broadEndpoints, name)

	c.Status(http.StatusOK)
}

func RegisterRoutes(r *gin.Engine) {
	r.GET("/broadcast/data", GetBroadcasts)
	r.POST("/broadcast/data", CreateBroadcast)

	r.GET("/broadcast/data/mock", headersMiddleware(), GetMockData)

	r.GET("/broadcast/data/default", headersMiddleware(), func(c *gin.Context) {
		c.Params = append(c.Params, gin.Param{Key: "name", Value: "default"})
		SubscribeBroadcast(c)
	})
	r.POST("/broadcast/data/default", func(c *gin.Context) {
		c.Params = append(c.Params, gin.Param{Key: "name", Value: "default"})
		BroadcastData(c)
	})

	r.GET("/broadcast/data/:name", headersMiddleware(), SubscribeBroadcast)
	r.POST("/broadcast/data/:name", BroadcastData)
	r.DELETE("/broadcast/data/:name", DeleteBroadcast)
}

func headersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Writer.Flush()
		c.Next()
	}
}
