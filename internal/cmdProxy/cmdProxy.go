package cmdproxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	zmq "github.com/pebbe/zmq4"
)

// Endpoint is the request body for starting the command proxy
type Endpoint struct {
	NickName string `json:"nickname" binding:"required"`
	Hostname string `json:"hostname" binding:"required"`
	Port     uint   `json:"port" binding:"required"`
}

type HandshakeREP struct {
	Response string `json:"response"`
}

type HandshakeREQ struct {
	Request string `json:"request"`
}

// ReqClient is a client for sending requests to the Rep server
type ReqClient struct {
	hostname string
	port     uint
	context  *zmq.Context
	socket   *zmq.Socket
	mutex    sync.Mutex
}

var (
	reqClientMap      = make(map[string]*ReqClient)
	reqClientMapMutex sync.RWMutex
)

// Create a ZeroMQ REQ client
func createREQ(hostname string, port uint) (*ReqClient, error) {
	zctx, err := zmq.NewContext()

	if err != nil {
		return nil, fmt.Errorf("failed to create context: %v", err)
	}

	s, err := zctx.NewSocket(zmq.REQ)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket: %v", err)
	}

	err = s.Connect(fmt.Sprintf("tcp://%s:%d", hostname, port))

	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}

	return &ReqClient{
		hostname: hostname,
		port:     port,
		context:  zctx,
		socket:   s,
	}, nil
}

func (r *ReqClient) handShake() error {
	if err := r.socket.SetRcvtimeo(5 * time.Second); err != nil {
		return fmt.Errorf("failed to set receive timeout: %v", err)
	}

	request := HandshakeREQ{
		Request: "Hello",
	}
	requestJson, _ := json.Marshal(request)
	r.socket.SendBytes(requestJson, 0)

	msgJson, err := r.socket.RecvBytes(0)
	var msg HandshakeREP
	json.Unmarshal([]byte(msgJson), &msg)

	log.Printf("Received message from server: %s\n", msg.Response)

	if err != nil || msg.Response != "World" {
		return fmt.Errorf("failed to receive message from server: %v", err)
	}

	return nil
}

// Send a message to the server and return the response
func (r *ReqClient) Send(msg []byte) ([]byte, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, err := r.socket.SendBytes(msg, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %v", err)
	}

	response, err := r.socket.RecvBytes(0)
	if err != nil {
		return nil, fmt.Errorf("failed to receive message: %v", err)
	}

	return response, nil
}

// Close the REQ client
func (r *ReqClient) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	defer func() {
		r.socket = nil
		r.context = nil
	}()

	var errs []error

	if r.socket != nil {
		if err := r.socket.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close socket: %w", err))
		}
	}

	if r.context != nil {
		if err := r.context.Term(); err != nil {
			errs = append(errs, fmt.Errorf("failed to terminate context: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d errors occurred: %v", len(errs), errors.Join(errs...))
	}

	return nil
}

func GetAllCmdProxies(c *gin.Context) {
	reqClientMapMutex.RLock()
	defer reqClientMapMutex.RUnlock()

	if len(reqClientMap) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no command proxies found"})
		return
	}

	var reqClientInfos []Endpoint

	for nickname, reqClient := range reqClientMap {
		reqClientInfos = append(reqClientInfos, Endpoint{
			NickName: nickname,
			Hostname: reqClient.hostname,
			Port:     reqClient.port,
		})
	}

	c.JSON(http.StatusOK, reqClientInfos)
}

// Create a REQ client
func createCmdProxy(c *gin.Context) {
	var endpoint Endpoint

	if err := c.ShouldBindJSON(&endpoint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reqClientMapMutex.RLock()
	_, exist := reqClientMap[endpoint.NickName]
	reqClientMapMutex.RUnlock()

	if exist {
		c.JSON(http.StatusConflict, gin.H{"error": "Command proxy already started"})
		return
	}

	client, err := createREQ(endpoint.Hostname, endpoint.Port)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Starting command proxy for %s\n", endpoint.NickName)

	reqClientMapMutex.Lock()
	defer reqClientMapMutex.Unlock()

	if _, exist := reqClientMap[endpoint.NickName]; exist {
		client.Close()
		c.JSON(http.StatusConflict, gin.H{"error": "Command proxy already started"})
		return
	}
	reqClientMap[endpoint.NickName] = client

	go func() {
		if err := client.handShake(); err != nil {

			reqClientMapMutex.Lock()
			if _, exist := reqClientMap[endpoint.NickName]; exist {
				client.socket.Close()
				delete(reqClientMap, endpoint.NickName)
			}
			reqClientMapMutex.Unlock()
		}
	}()

	c.Status(http.StatusCreated)
}

// Forward a command to the server and return the response
func sendCmd(c *gin.Context) {
	reqClientMapMutex.RLock()
	reqClient, exist := reqClientMap[c.Param("nickname")]
	reqClientMapMutex.RUnlock()

	if !exist {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Command proxy not started"})
		return
	}

	cmd, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid command"})
		return
	}

	result, err := reqClient.Send(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusCreated, "application/json", result)
}

func DeleteAllCmdProxies(c *gin.Context) {
	reqClientMapMutex.Lock()

	if len(reqClientMap) == 0 {
		reqClientMapMutex.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "no command proxies found"})
		return
	}

	// Create a copy of clients to close them outside the lock
	clients := make([]*ReqClient, 0, len(reqClientMap))
	for _, client := range reqClientMap {
		clients = append(clients, client)
	}

	// Clear the map
	reqClientMap = make(map[string]*ReqClient)
	reqClientMapMutex.Unlock()

	// Close all clients outside the lock
	var wg sync.WaitGroup
	for _, client := range clients {
		wg.Add(1)
		go func(c *ReqClient) {
			defer wg.Done()
			if err := c.Close(); err != nil {
				log.Printf("Failed to close client: %v", err)
			}
		}(client)
	}

	c.Status(http.StatusOK)
}

func DeleteCmdProxy(c *gin.Context) {
	nickname := c.Param("nickname")
	if nickname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nickname is required"})
		return
	}

	reqClientMapMutex.RLock()
	client, exist := reqClientMap[nickname]
	reqClientMapMutex.RUnlock()

	if !exist {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("command proxy for '%s' not found", nickname)})
		return
	}

	reqClientMapMutex.Lock()
	defer reqClientMapMutex.Unlock()
	delete(reqClientMap, nickname)

	if err := client.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func RegisterRoutes(r *gin.Engine) {
	r.GET("/cmds/proxies", GetAllCmdProxies)
	r.POST("/cmds/proxies", createCmdProxy)
	r.POST("/cmds/proxies/:nickname", sendCmd)
	r.DELETE("/cmds/proxies", DeleteAllCmdProxies)
	r.DELETE("/cmds/proxies/:nickname", DeleteCmdProxy)
}
