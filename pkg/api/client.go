package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) Register(agentInfo interface{}) error {
	return c.post("/register", agentInfo)
}

func (c *Client) Heartbeat(agentID string) error {
	data := map[string]string{
		"agent_id":  agentID,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	return c.post("/heartbeat", data)
}

func (c *Client) SendAssets(assets interface{}) error {
	return c.post("/assets", assets)
}

func (c *Client) post(endpoint string, data interface{}) error {
	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("backend returned status: %d", resp.StatusCode)
	}

	return nil
}
