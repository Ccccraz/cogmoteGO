package cmdproxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Ccccraz/cogmoteGO/internal/commonTypes"
	"github.com/Ccccraz/cogmoteGO/internal/config"
	"github.com/Ccccraz/cogmoteGO/internal/logger"
	"github.com/gin-gonic/gin"
	zmq "github.com/pebbe/zmq4"
)

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

type ReqClient struct {
	hostname string
	port     uint

	available atomic.Bool
	closed    atomic.Bool

	context *zmq.Context
	socket  *zmq.Socket

	mutex sync.Mutex
}

var (
	reqClientMap      = make(map[string]*ReqClient)
	reqClientMapMutex sync.RWMutex
	logKey            = "cmdProxies"
	cfg               config.Config
)

var (
	ErrClientClosed       = errors.New("req client closed")
	ErrClientUnavailable  = errors.New("req client unavailable")
	ErrMaxRetriesExceeded = errors.New("lazy pirate: max retries exceeded")
)

func lazyPirateMaxRetries() int {
	if cfg.Proxy.MaxRetries <= 0 {
		return 3
	}
	return cfg.Proxy.MaxRetries
}

func lazyPirateRetryInterval() time.Duration {
	d := time.Duration(cfg.Proxy.RetryInterval)
	if d <= 0 {
		return 200 * time.Millisecond
	}
	return d
}

func applyHandshakeTimeouts(s *zmq.Socket) error {
	if s == nil {
		return nil
	}
	if err := s.SetSndtimeo(time.Duration(cfg.Proxy.HandshakeTimeout) * time.Millisecond); err != nil {
		return fmt.Errorf("failed to set handshake send timeout: %w", err)
	}
	if err := s.SetRcvtimeo(time.Duration(cfg.Proxy.HandshakeTimeout) * time.Millisecond); err != nil {
		return fmt.Errorf("failed to set handshake recv timeout: %w", err)
	}
	return nil
}

func applyMsgTimeouts(s *zmq.Socket) error {
	if s == nil {
		return nil
	}
	if err := s.SetSndtimeo(time.Duration(cfg.Proxy.MsgTimeout) * time.Millisecond); err != nil {
		return fmt.Errorf("failed to set msg send timeout: %w", err)
	}
	if err := s.SetRcvtimeo(time.Duration(cfg.Proxy.MsgTimeout) * time.Millisecond); err != nil {
		return fmt.Errorf("failed to set msg recv timeout: %w", err)
	}
	return nil
}

func createREQ(hostname string, port uint) (*ReqClient, error) {
	zctx, err := zmq.NewContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create context: %w", err)
	}

	s, err := zctx.NewSocket(zmq.REQ)
	if err != nil {
		_ = zctx.Term()
		return nil, fmt.Errorf("failed to create socket: %w", err)
	}

	if err := s.Connect(fmt.Sprintf("tcp://%s:%d", hostname, port)); err != nil {
		_ = s.Close()
		_ = zctx.Term()
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &ReqClient{
		hostname: hostname,
		port:     port,
		context:  zctx,
		socket:   s,
	}, nil
}

// Lazy Pirate: on timeout/EFSM, discard REQ socket and recreate it, then retry.
func (r *ReqClient) recreateSocketLocked() error {
	if r.closed.Load() {
		return ErrClientClosed
	}
	if r.context == nil {
		return ErrClientClosed
	}

	if r.socket != nil {
		_ = r.socket.Close()
		r.socket = nil
	}

	s, err := r.context.NewSocket(zmq.REQ)
	if err != nil {
		return fmt.Errorf("failed to recreate socket: %w", err)
	}

	if err := s.Connect(fmt.Sprintf("tcp://%s:%d", r.hostname, r.port)); err != nil {
		_ = s.Close()
		return fmt.Errorf("failed to reconnect to server: %w", err)
	}

	// Msg timeouts are applied after the initial handshake.
	if err := applyMsgTimeouts(s); err != nil {
		_ = s.Close()
		return err
	}

	r.socket = s
	return nil
}

func (r *ReqClient) handShake() error {
	if r.closed.Load() {
		return ErrClientClosed
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.closed.Load() || r.socket == nil {
		return ErrClientClosed
	}

	const (
		expectedRequest  = "Hello"
		expectedResponse = "World"
	)

	if err := applyHandshakeTimeouts(r.socket); err != nil {
		return err
	}

	start := time.Now()

	request := HandshakeREQ{Request: expectedRequest}
	requestJson, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal handshake request: %w", err)
	}

	if _, err := r.socket.SendBytes(requestJson, 0); err != nil {
		return fmt.Errorf("failed to send handshake request: %w", err)
	}

	msgJson, err := r.socket.RecvBytes(0)
	if len(msgJson) == 0 {
		return fmt.Errorf("empty handshake response")
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

	r.available.Store(true)

	if err := applyMsgTimeouts(r.socket); err != nil {
		r.available.Store(false)
		return err
	}

	elapsed := time.Since(start)
	logger.Logger.Debug(
		"Handshake completed",
		slog.Group(
			logKey,
			slog.String("response", msg.Response),
			slog.String("duration", elapsed.String()),
		),
	)

	return nil
}

func (r *ReqClient) Send(msg []byte) ([]byte, error) {
	if r.closed.Load() {
		return nil, ErrClientClosed
	}
	if !r.available.Load() {
		return nil, ErrClientUnavailable
	}

	maxRetries := lazyPirateMaxRetries()
	retryInterval := lazyPirateRetryInterval()
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		r.mutex.Lock()

		if r.closed.Load() || r.socket == nil {
			r.mutex.Unlock()
			return nil, ErrClientClosed
		}
		if !r.available.Load() {
			r.mutex.Unlock()
			return nil, ErrClientUnavailable
		}

		_, err := r.socket.SendBytes(msg, 0)
		if err != nil {
			lastErr = fmt.Errorf("failed to send message: %w", err)
			recoverable := isRecoverableZmqError(err)
			if recoverable {
				_ = r.recreateSocketLocked()
			}
			r.mutex.Unlock()

			if !recoverable {
				return nil, lastErr
			}
			if attempt < maxRetries-1 {
				time.Sleep(retryInterval)
				continue
			}
			return nil, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
		}

		resp, err := r.socket.RecvBytes(0)
		if err == nil {
			r.mutex.Unlock()
			return resp, nil
		}

		lastErr = fmt.Errorf("failed to receive message: %w", err)
		recoverable := isRecoverableZmqError(err)
		if recoverable {
			_ = r.recreateSocketLocked()
		}
		r.mutex.Unlock()

		if !recoverable {
			return nil, lastErr
		}
		if attempt < maxRetries-1 {
			time.Sleep(retryInterval)
			continue
		}
		return nil, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
	}
	return nil, ErrMaxRetriesExceeded
}

func isRecoverableZmqError(err error) bool {
	if err == nil {
		return false
	}
	for e := err; e != nil; e = errors.Unwrap(e) {
		switch zmq.AsErrno(e) {
		case zmq.Errno(syscall.EAGAIN), zmq.ETIMEDOUT:
			return true
		case zmq.EFSM:
			return true
		case zmq.ETERM:
			return false
		default:
		}
	}
	return false
}

func (r *ReqClient) Close() error {
	if !r.closed.CompareAndSwap(false, true) {
		return nil
	}

	r.available.Store(false)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	var errs []error

	if r.socket != nil {
		if err := r.socket.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close socket: %w", err))
		}
		r.socket = nil
	}

	if r.context != nil {
		if err := r.context.Term(); err != nil {
			errs = append(errs, fmt.Errorf("failed to terminate context: %w", err))
		}
		r.context = nil
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
	if _, exist := reqClientMap[endpoint.NickName]; exist {
		reqClientMapMutex.Unlock()
		_ = client.Close()
		c.JSON(http.StatusConflict, commonTypes.APIError{
			Error:  fmt.Sprintf("command proxy %s already started", endpoint.NickName),
			Detail: "",
		})
		return
	}
	reqClientMap[endpoint.NickName] = client
	reqClientMapMutex.Unlock()

	go func(nick string, cl *ReqClient) {
		if err := cl.handShake(); err != nil {
			logger.Logger.Error(
				"handshake failed: ",
				slog.Group(
					logKey,
					slog.String("nickname", nick),
					slog.String("error", err.Error()),
				),
			)
			_ = destroyReqClient(nick, cl)
		}
	}(endpoint.NickName, client)

	c.Status(http.StatusCreated)
}

func sendCmd(c *gin.Context) {
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
			slog.Group(logKey, slog.String("nickname", nickname)),
		)
		return
	}

	if !reqClient.available.Load() || reqClient.closed.Load() {
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
			),
		)
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
			slog.Group(logKey, slog.String("nickname", nickname)),
		)
		return
	}
	handleElapsed := time.Since(handleStart)

	sendStart := time.Now()
	result, err := reqClient.Send(cmd)
	if err != nil {
		if errors.Is(err, ErrClientClosed) || errors.Is(err, ErrClientUnavailable) {
			c.JSON(http.StatusServiceUnavailable, commonTypes.APIError{
				Error:  fmt.Sprintf("command proxy %s is not available", nickname),
				Detail: err.Error(),
			})
			return
		}

		if errors.Is(err, ErrMaxRetriesExceeded) || isTimeoutError(err) {
			logger.Logger.Error(
				"command proxy timed out (lazy pirate retries exhausted), destroying req client",
				slog.Group(
					logKey,
					slog.String("nickname", nickname),
					slog.String("detail", err.Error()),
				),
			)
			_ = destroyReqClient(nickname, reqClient)
			c.JSON(http.StatusGatewayTimeout, commonTypes.APIError{
				Error:  fmt.Sprintf("command proxy %s timed out", nickname),
				Detail: err.Error(),
			})
			return
		}

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
			),
		)
		return
	}

	sendElapsed := time.Since(sendStart)
	logger.Logger.Debug(
		"command send success: ",
		slog.Group(
			logKey,
			slog.String("nickname", nickname),
			slog.String("handleDuration", handleElapsed.String()),
			slog.String("sendDuration", sendElapsed.String()),
		),
	)

	c.Data(http.StatusCreated, "application/json", result)
}

func DeleteAllCmdProxies(c *gin.Context) {
	reqClientMapMutex.Lock()

	if len(reqClientMap) == 0 {
		reqClientMapMutex.Unlock()
		c.Status(http.StatusOK)
		return
	}

	snapshot := make(map[string]*ReqClient, len(reqClientMap))
	for nick, client := range reqClientMap {
		snapshot[nick] = client
	}

	reqClientMap = make(map[string]*ReqClient)
	reqClientMapMutex.Unlock()

	var (
		wg     sync.WaitGroup
		errs   []error
		errMux sync.Mutex
	)

	for nick, client := range snapshot {
		wg.Add(1)
		go func(n string, cl *ReqClient) {
			defer wg.Done()
			if err := destroyReqClient(n, cl); err != nil {
				errMux.Lock()
				errs = append(errs, err)
				errMux.Unlock()
			}
		}(nick, client)
	}
	wg.Wait()

	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  "failed to close some clients",
			Detail: errors.Join(errs...).Error(),
		})
		return
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

	if err := destroyReqClient(nickname, client); err != nil {
		c.JSON(http.StatusInternalServerError, commonTypes.APIError{
			Error:  fmt.Sprintf("failed to close command proxy %s", nickname),
			Detail: err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	for e := err; e != nil; e = errors.Unwrap(e) {
		switch zmq.AsErrno(e) {
		case zmq.Errno(syscall.EAGAIN), zmq.ETIMEDOUT:
			return true
		default:
		}
	}
	return false
}

func destroyReqClient(nickname string, client *ReqClient) error {
	if client == nil {
		return nil
	}

	client.available.Store(false)

	reqClientMapMutex.Lock()
	if stored, exist := reqClientMap[nickname]; exist && stored == client {
		delete(reqClientMap, nickname)
	}
	reqClientMapMutex.Unlock()

	if err := client.Close(); err != nil {
		logger.Logger.Error(
			"failed to close command proxy client",
			slog.Group(
				logKey,
				slog.String("nickname", nickname),
				slog.String("error", err.Error()),
			),
		)
		return err
	}
	return nil
}

func RegisterRoutes(r gin.IRouter, config config.Config) {
	cfg = config
	r.GET("/cmds/proxies", GetAllCmdProxies)
	r.POST("/cmds/proxies", createCmdProxy)
	r.POST("/cmds/proxies/:nickname", sendCmd)
	r.DELETE("/cmds/proxies", DeleteAllCmdProxies)
	r.DELETE("/cmds/proxies/:nickname", DeleteCmdProxy)
}
