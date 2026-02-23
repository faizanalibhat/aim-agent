package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

type RegisterResponse struct {
	AgentID string `json:"agent_id"`
}

func (c *Client) Register(hostname, os, version, ipAddress, architecture, arch string) (string, error) {
	data := map[string]string{
		"hostname":     hostname,
		"os":           os,
		"version":      version,
		"ipAddress":    ipAddress,
		"architecture": architecture,
		"arch":         arch,
		"api_key":      c.APIKey,
	}

	respBody, err := c.postWithResponse("/register", data)
	if err != nil {
		return "", err
	}

	var regResp RegisterResponse
	if err := json.Unmarshal(respBody, &regResp); err != nil {
		return "", err
	}

	if regResp.AgentID == "" {
		return "", fmt.Errorf("registration failed: agent_id not returned (body: %s)", string(respBody))
	}

	return regResp.AgentID, nil
}

func (c *Client) Heartbeat(agentID, version string) (*ResultsResponse, error) {
	data := map[string]string{
		"agent_id":  agentID,
		"version":   version,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	respBody, err := c.postWithResponse("/heartbeat", data)
	if err != nil {
		return nil, err
	}

	var res ResultsResponse
	if err := json.Unmarshal(respBody, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

type AgentConfiguration struct {
	Kill              bool   `json:"kill"`
	HeartbeatInterval int    `json:"heartbeat_interval"` // in seconds
	AssetPushInterval int    `json:"asset_push_interval"` // in seconds
	LatestVersion     string `json:"latest_version"`
	DownloadURL       string `json:"download_url"`
}

type ResultsResponse struct {
	Configuration AgentConfiguration `json:"configuration"`
}

func (c *Client) SendResults(agentID string, results interface{}) (*ResultsResponse, error) {
	data := map[string]interface{}{
		"agent_id": agentID,
		"data":     results,
	}

	respBody, err := c.postWithResponse("/results", data)
	if err != nil {
		return nil, err
	}

	var res ResultsResponse
	if err := json.Unmarshal(respBody, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) post(endpoint string, data interface{}) error {
	_, err := c.postWithResponse(endpoint, data)
	return err
}

func (c *Client) postWithResponse(endpoint string, data interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNotFound {
		return nil, fmt.Errorf("backend returned status: %d, body: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
