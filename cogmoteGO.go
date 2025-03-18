package main

import (
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

type Stream struct {
	mu          sync.Mutex    // 保证并发安全的互斥锁
	subscribers []chan []byte // 所有订阅者的通道列表
	history     [][]byte      // 历史数据
}

var (
	streams   = make(map[string]*Stream) // 全局流注册表
	streamsMu sync.RWMutex               // 保护streams的读写锁
)

// TODO: Add TrustedProxies
func main() {
	if envMode := os.Getenv("GIN_MODE"); envMode == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Default data endpoint
	streams["data"] = &Stream{
		subscribers: make([]chan []byte, 0),
	}

	// 创建新的数据端点
	r.POST("/create/:name", func(c *gin.Context) {
		name := c.Param("name")

		streamsMu.Lock()
		defer streamsMu.Unlock()

		if _, exists := streams[name]; exists {
			c.JSON(http.StatusConflict, gin.H{"error": "stream already exists"})
			return
		}

		streams[name] = &Stream{
			subscribers: make([]chan []byte, 0),
		}
		c.Status(http.StatusCreated)
	})

	// 接收数据端点
	r.POST("/:name", func(c *gin.Context) {
		name := c.Param("name")

		streamsMu.RLock()
		stream, exists := streams[name]
		streamsMu.RUnlock()

		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "stream not found"})
			return
		}

		// 读取原始数据
		data, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data"})
			return
		}

		stream.mu.Lock()
		for _, ch := range stream.subscribers {
			ch <- data
		}
		stream.mu.Unlock()
		stream.history = append(stream.history, data)

		c.Status(http.StatusOK)
	})

	// 客户端订阅端点（SSE）
	r.GET("/:name", func(c *gin.Context) {
		name := c.Param("name")

		streamsMu.RLock()
		stream, exists := streams[name]
		streamsMu.RUnlock()

		if !exists {
			c.JSON(404, gin.H{"error": "stream not found"})
			return
		}

		// 设置SSE响应头
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()

		// 创建新订阅者通道
		ch := make(chan []byte)
		stream.mu.Lock()
		// 发送历史数据
		if len(stream.history) > 0 {
			for _, data := range stream.history {
				c.SSEvent("message", data)
				c.Writer.Flush()
			}
		}
		// 注册订阅者
		stream.subscribers = append(stream.subscribers, ch)
		stream.mu.Unlock()

		// 保持连接打开，监听数据更新
		defer func() {
			// 客户端断开时移除订阅者
			stream.mu.Lock()
			defer stream.mu.Unlock()
			for i, sub := range stream.subscribers {
				if sub == ch {
					stream.subscribers = append(stream.subscribers[:i], stream.subscribers[i+1:]...)
					break
				}
			}
			close(ch)
		}()

		for {
			select {
			case data := <-ch:
				// 将数据以SSE格式发送
				c.SSEvent("message", data)
				c.Writer.Flush()
			case <-c.Writer.CloseNotify():
				// 客户端断开连接
				return
			}
		}
	})

	r.Run(":9012")
}
