package client

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

type HTTPClient struct {
	endpoint            string
	channelName         string
	fatFingerProtection bool
	client              *http.Client
	localAddr           net.Addr
}

// HTTPClientOption 定义 HTTP 客户端配置选项
type HTTPClientOption func(*HTTPClient)

// WithLocalAddress 设置本地地址
func WithLocalAddress(localAddr string) HTTPClientOption {
	return func(c *HTTPClient) {
		if localAddr != "" {
			if addr, err := net.ResolveTCPAddr("tcp", localAddr+":0"); err == nil {
				c.localAddr = addr
			}
		}
	}
}

// WithLocalIP 设置本地IP地址（端口由系统自动分配）
func WithLocalIP(localIP string) HTTPClientOption {
	return func(c *HTTPClient) {
		if localIP != "" {
			if ip := net.ParseIP(localIP); ip != nil {
				c.localAddr = &net.TCPAddr{IP: ip, Port: 0}
			}
		}
	}
}

func NewHTTPClient(baseUrl string, opts ...HTTPClientOption) *HTTPClient {
	if baseUrl == "" {
		return nil
	}

	client := &HTTPClient{
		endpoint:            baseUrl,
		channelName:         "",
		fatFingerProtection: true,
	}

	// 应用选项
	for _, opt := range opts {
		opt(client)
	}

	// 创建自定义的 HTTP 客户端
	client.setupHTTPClient()

	return client
}

// setupHTTPClient 根据配置创建 HTTP 客户端
func (c *HTTPClient) setupHTTPClient() {

	var dialer *net.Dialer

	if c.localAddr != nil {
		fmt.Println("has local address")
		dialer = &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 60 * time.Second,
			LocalAddr: c.localAddr,
		}
	} else {
		dialer = &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 60 * time.Second,
		}
	}

	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		MaxConnsPerHost:     1000,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     10 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}

	c.client = &http.Client{
		Timeout:   time.Second * 30,
		Transport: transport,
	}
}

func (c *HTTPClient) SetFatFingerProtection(enabled bool) {
	c.fatFingerProtection = enabled
}
