package client

import (
	"fmt"
	"io"
	"net/http"
)

// CheckPublicIP 使用当前 HTTPClient 的本地出口 IP 访问 https://api.ipify.org
// 并打印返回的公网 IP 地址（确认 localAddr 是否生效）
func (c *HTTPClient) CheckPublicIP() (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("HTTP client is not initialized")
	}

	url := "https://api.ipify.org" // 返回请求方公网 IP 的公共 API
	resp, err := c.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get public IP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	publicIP := string(body)
	fmt.Printf("🌐 Public IP detected (via local %v): %s\n", c.localAddr, publicIP)
	return publicIP, nil
}
