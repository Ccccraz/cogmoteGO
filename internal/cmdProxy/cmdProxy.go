package cmdproxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Ccccraz/cogmoteGO/internal/commonTypes"
	"github.com/Ccccraz/cogmoteGO/internal/logger"
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
	hostname  string
	port      uint
	available bool
	context   *zmq.Context
	socket    *zmq.Socket
	mutex     sync.Mutex
}

var (
	reqClientMap      = make(map[string]*ReqClient)
	reqClientMapMutex sync.RWMutex
	logKey            = "cmdProxies"
)

// Create a ZeroMQ REQ client
func createREQ(hostname string, port uint) (*ReqClient, error) {
	zctx, err := zmq.NewContext()

	if err != nil {
		return nil, fmt.Errorf("failed to create context: %w", err)
	}

	s, err := zctx.NewSocket(zmq.REQ)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket: %w", err)
	}

	err = s.Connect(fmt.Sprintf("tcp://%s:%d", hostname, port))

	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &ReqClient{
		hostname: hostname,
		port:     port,
		context:  zctx,
		socket:   s,
	}, nil
}

func (r *ReqClient) handShake() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	const (
		expectedRequest  = "Hello"
		expectedResponse = "World"
		handshakeTimeout = 5 * time.Second
		proxyTimeout     = -1
	)

	if err := r.socket.SetSndtimeo(handshakeTimeout); err != nil {
		return fmt.Errorf("failed to set send timeout: %w", err)
	}

	if err := r.socket.SetRcvtimeo(proxyTimeout); err != nil {
		return fmt.Errorf("failed to set receive timeout: %w", err)
	}

	defer func() {
		if err := r.socket.SetSndtimeo(proxyTimeout); err != nil {
			logger.Logger.Error(
				"Failed to set send timeout",
				slog.Group(
					logKey,
					slog.String("error", err.Error()),
				))
		}
		if err := r.socket.SetRcvtimeo(proxyTimeout); err != nil {
			logger.Logger.Error(
				"Failed to set receive timeout",
				slog.Group(
					logKey,
					slog.String("error", err.Error()),
				))
		}
	}()

	// time cost benchmark start mark
	start := time.Now()

	request := HandshakeREQ{
		Request: expectedRequest,
	}

	requestJson, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal handshake request: %w", err)
	}

	if _, err := r.socket.SendBytes(requestJson, 0); err != nil {
		return fmt.Errorf("failed to send handshake request: %w", err)
	}

	msgJson, err := r.socket.RecvBytes(0)
	if len(msgJson) == 0 {
		return fmt.Errorf("empty handshake response...")
	} else if err != nil {
		return fmt.Errorf("failed to receive handshake response: %w", err)
	}

	var msg HandshakeREP
	if err := json.Unmarshal(msgJson, &msg); err != nil {
		return fmt.Errorf("invalid handshake response: %w", err)
	}

	if msg.Response != expectedResponse {
		return fmt.Errorf("wrong handshake response: %s", msg.Response)
	}

	r.available = true

	// time cost benchmark end mark
	elapsed := time.Since(start)
	logger.Logger.Debug("Handshake completed",
		slog.Group(
			logKey,
			slog.String("response", msg.Response),
			slog.String("duration", elapsed.String()),
		))

	return nil
}

// Send a message to the server and return the response
func (r *ReqClient) Send(msg []byte) ([]byte, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, err := r.socket.SendBytes(msg, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	response, err := r.socket.RecvBytes(0)
	if err != nil {
		return nil, fmt.Errorf("failed to receive message: %w", err)
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
		return errors.Join(errs...)
	}

	return nil
}

func GetAllCmdProxies(c *gin.Context) {
	reqClientMapMutex.RLock()
	defer reqClientMapMutex.RUnlock()

	if len(reqClientMap) == 0 {
		c.JSON(http.StatusNotFound, commonTypes.APIError{
			Error:  "no command proxies found",
			Detail: "",
		})
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
		c.JSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "invalid proxy endpoint",
			Detail: err.Error(),
		})
		return
	}

	reqClientMapMutex.RLock()
	_, exist := reqClientMap[endpoint.NickName]
	reqClientMapMutex.RUnlock()

	if exist {
		c.JSON(http.StatusConflict, commonTypes.APIError{
			Error:  fmt.Sprintf("command proxy %s already started", endpoint.NickName),
			Detail: "",
		})
		return
	}

	client, err := createREQ(endpoint.Hostname, endpoint.Port)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  fmt.Sprintf("failed to create command proxy %s", endpoint.NickName),
			Detail: err.Error(),
		})
		return
	}

	logger.Logger.Info(
		"starting command proxy: ",
		slog.Group(
			logKey,
			slog.String("nickname", endpoint.NickName),
			slog.String("hostname", endpoint.Hostname),
			slog.Int("port", int(endpoint.Port)),
		),
	)

	reqClientMapMutex.Lock()
	defer reqClientMapMutex.Unlock()

	if _, exist := reqClientMap[endpoint.NickName]; exist {
		client.Close()
		c.JSON(http.StatusConflict, commonTypes.APIError{
			Error:  fmt.Sprintf("command proxy %s already started", endpoint.NickName),
			Detail: "",
		})
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

			logger.Logger.Error(
				"handshake failed: ",
				slog.Group(
					logKey,
					slog.String("nickname", endpoint.NickName),
					slog.String("error", err.Error()),
				))
		}
	}()

	c.Status(http.StatusCreated)
}

// Forward a command to the server and return the response
func sendCmd(c *gin.Context) {
	// time cost benchmark start mark
	handleStart := time.Now()

	nickname := c.Param("nickname")

	reqClientMapMutex.RLock()
	reqClient, exist := reqClientMap[nickname]
	reqClientMapMutex.RUnlock()

	if !exist {
		c.JSON(http.StatusNotFound, commonTypes.APIError{
			Error:  fmt.Sprintf("command proxy %s not found", nickname),
			Detail: "",
		})
		logger.Logger.Error(
			"command proxy not found: ",
			slog.Group(
				logKey,
				slog.String("nickname", nickname),
			))
		return
	}

	if !reqClient.available {
		c.JSON(http.StatusServiceUnavailable, commonTypes.APIError{
			Error:  fmt.Sprintf("command proxy %s is not available", nickname),
			Detail: "",
		})
		logger.Logger.Error(
			"command proxy not available: ",
			slog.Group(
				logKey,
				slog.String("nickname", nickname),
				slog.String("error", "command proxy is not available"),
			))
		return
	}

	cmd, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "cannot get command data from request body",
			Detail: err.Error(),
		})
		logger.Logger.Error(
			"cannot get command data from request body: ",
			slog.Group(
				logKey,
				slog.String("nickname", nickname),
			))
		return
	}
	handleElapsed := time.Since(handleStart)

	sendStart := time.Now()
	result, err := reqClient.Send(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  fmt.Sprintf("failed to send command to command proxy %s", nickname),
			Detail: err.Error(),
		})
		logger.Logger.Error(
			"failed to send command to command proxy: ",
			slog.Group(
				logKey,
				slog.String("nickname", nickname),
				slog.String("detail", err.Error()),
			))
		return
	}

	sendElapsed := time.Since(sendStart)
	logger.Logger.Debug("command send success: ",
		slog.Group(
			logKey,
			slog.String("nickname", nickname),
			slog.String("handleDuration", handleElapsed.String()),
			slog.String("sendDuration", sendElapsed.String()),
		))

	c.Data(http.StatusCreated, "application/json", result)
}

func DeleteAllCmdProxies(c *gin.Context) {
	reqClientMapMutex.Lock()
	defer reqClientMapMutex.Unlock()

	// if reqClientMap is empty, return directly
	if len(reqClientMap) == 0 {
		c.Status(http.StatusOK)
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
	var (
		wg     sync.WaitGroup
		errs   []error
		errMux sync.Mutex
	)

	for _, client := range clients {
		wg.Add(1)
		go func(c *ReqClient) {
			defer wg.Done()
			if err := c.Close(); err != nil {
				logger.Logger.Error(
					"failed to close client: ",
					slog.Group(
						logKey,
						slog.String("error", err.Error()),
					),
				)
				errMux.Lock()
				errs = append(errs, err)
				errMux.Unlock()
			}
		}(client)
	}
	wg.Wait()

	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to close some clients",
			Detail: errors.Join(errs...).Error(),
		})
	}

	c.Status(http.StatusOK)
}

func DeleteCmdProxy(c *gin.Context) {
	nickname := c.Param("nickname")
	if nickname == "" {
		c.JSON(http.StatusBadRequest, commonTypes.APIError{
			Error:  "nickname is required",
			Detail: "",
		})
		return
	}

	reqClientMapMutex.RLock()
	client, exist := reqClientMap[nickname]
	reqClientMapMutex.RUnlock()

	if !exist {
		c.JSON(http.StatusNotFound, commonTypes.APIError{
			Error:  fmt.Sprintf("command proxy for '%s' not found", nickname),
			Detail: "",
		})
		return
	}

	reqClientMapMutex.Lock()
	defer reqClientMapMutex.Unlock()
	delete(reqClientMap, nickname)

	if err := client.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  fmt.Sprintf("failed to close command proxy %s", nickname),
			Detail: err.Error(),
		})
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
