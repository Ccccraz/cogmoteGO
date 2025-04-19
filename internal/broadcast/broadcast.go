package broadcast

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
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
	broadEndpoints["data"] = &BroadcastEndpoint{
		subscribers: make([]chan []byte, 0),
	}
}

// create new data broadcast endpoint
func CreateBroadcastEndpoint(c *gin.Context) {
	var request struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	broadEndpointsMu.Lock()
	defer broadEndpointsMu.Unlock()

	if _, exists := broadEndpoints[request.Name]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": "endpoint already exists"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return
	}

	// read raw data
	data, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data"})
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
				log.Println("channel is full")
			}
		}(ch)

	}
	wg.Wait()
	endpoint.mu.Unlock()

	endpoint.history = append(endpoint.history, data)

	c.Status(http.StatusOK)
}

// Subscribe to the broadcast endpoint and receive updates via Server-Sent Events (SSE)
func SubscribeDataEndpoint(c *gin.Context) {
	name := c.Param("name")

	broadEndpointsMu.RLock()
	endpoint, exists := broadEndpoints[name]
	broadEndpointsMu.RUnlock()

	if !exists {
		c.JSON(404, gin.H{"error": "endpoint not found"})
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

func RegisterRoutes(r *gin.Engine) {
	r.POST("/create", CreateBroadcastEndpoint)
	r.POST("/:name", BroadcastData)
	r.GET("/:name", headersMiddleware(), SubscribeDataEndpoint)
}

func headersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()
		c.Next()
	}
}
