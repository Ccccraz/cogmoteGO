package cmdproxy

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	zmq "github.com/pebbe/zmq4"
)

// StartCmdProxyReq is the request body for starting the command proxy
type StartCmdProxyReq struct {
	Hostname string `json:"hostname" binding:"required"`
	Port     uint   `json:"port" binding:"required"`
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
	reqClient     *ReqClient // global ReqClient instance
	reqClientOnce sync.Once  // sync.Once for initializing reqClient
	reqClientErr  error      // error from creating reqClient
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

	s.Send("Hello", 0)

	msg, err := s.Recv(0)

	if err != nil && msg != "World" {
		return nil, fmt.Errorf("failed to receive message from server: %v", err)
	}

	return &ReqClient{
		hostname: hostname,
		port:     port,
		context:  zctx,
		socket:   s,
	}, nil
}

// Send a message to the server and return the response
func (r *ReqClient) Send(msg []byte) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, err := r.socket.SendBytes(msg, 0)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %v", err)
	}

	result, err := r.socket.Recv(0)
	if err != nil {
		return "", fmt.Errorf("failed to receive message: %v", err)
	}

	return result, nil
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
func startCmdProxy(c *gin.Context) {
	var req StartCmdProxyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reqClientOnce.Do(func() {
		reqClient, reqClientErr = createREQ(req.Hostname, req.Port)
	})

	if reqClientErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": reqClientErr.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"result": "Command proxy started"})
}

// Forward a command to the server and return the response
func cmd(c *gin.Context) {
	if reqClient == nil {
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

	c.JSON(http.StatusCreated, gin.H{"repsonse": result})
}

func RegisterRoutes(r *gin.Engine) {
	r.POST("/StartCmdProxy", startCmdProxy)
	r.POST("/Cmd", cmd)
}
