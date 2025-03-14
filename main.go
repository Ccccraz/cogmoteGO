package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// Stream 结构体用于管理每个数据流
type Stream struct {
	mu          sync.Mutex    // 保证并发安全的互斥锁
	subscribers []chan []byte // 所有订阅者的通道列表
	data        []byte        // 保存最新的数据（原始字节）
}

type Event struct {
	// Events are pushed to this channel by the main events-gathering routine
	Message chan []byte

	// New client connections
	NewClients chan chan []byte

	// Closed client connections
	ClosedClients chan chan []byte

	// Total client connections
	TotalClients map[chan []byte]bool
}

var (
	streams   = make(map[string]*Stream) // 全局流注册表
	streamsMu sync.RWMutex               // 保护streams的读写锁
)

func main() {
	r := gin.Default()

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

	// 接收数据推送
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

		// 更新数据并通知所有订阅者
		stream.mu.Lock()
		stream.data = data
		// 向所有订阅者广播
		for _, ch := range stream.subscribers {
			ch <- data
		}
		stream.mu.Unlock()

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
		// 发送当前最新数据
		if len(stream.data) > 0 {
			c.SSEvent("message", stream.data)
			c.Writer.Flush()
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

	r.Run()
}
