package cmdproxy

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

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
	reqClientOnce     sync.Once // sync.Once for initializing reqClient
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

	if r.socket != nil {
		r.socket.Close()
		r.socket = nil
	}

	if r.context != nil {
		r.context.Term()
		r.context = nil
	}

	return nil
}

// Create a REQ client
func createCmdProxy(c *gin.Context) {
	var endpoint Endpoint

	if err := c.ShouldBindJSON(&endpoint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var reqClient *ReqClient
	var reqInitErr error
	reqClientOnce.Do(func() {
		reqClient, reqInitErr = createREQ(endpoint.Hostname, endpoint.Port)
	})

	if reqInitErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": reqInitErr.Error()})
		return
	}

	reqClientMapMutex.Lock()
	if _, exist := reqClientMap[endpoint.NickName]; exist {
		c.JSON(http.StatusConflict, gin.H{"error": "Command proxy already started"})
		return
	} else {
		log.Printf("Starting command proxy for %s\n", endpoint.NickName)
		reqClientMap[endpoint.NickName] = reqClient
	}
	reqClientMapMutex.Unlock()

	go func() {
		if err := reqClient.handShake(); err != nil {

			reqClientMapMutex.Lock()
			delete(reqClientMap, endpoint.NickName)
			reqClientMapMutex.Unlock()

			reqClient.socket.Close()
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

func RegisterRoutes(r *gin.Engine) {
	r.POST("/cmds/proxies", createCmdProxy)
	r.POST("/cmds/proxies/:nickname", sendCmd)
}
