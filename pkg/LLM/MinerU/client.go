package mineru

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token: token,
		HTTP: &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) CreateTask(ctx context.Context, req CreateTaskRequest) (*CreateTaskResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL + "/api/v4/extract/task"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
    httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out CreateTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Code != 0 {
		return &out, fmt.Errorf("mineru create failed: code=%d msg=%s trace_id=%s", out.Code, out.Msg, out.TraceID)
	}
	return &out, nil
}

func (c *Client) GetTask(ctx context.Context, taskID string) (*GetTaskResponse, error) {
	url := c.BaseURL + "/api/v4/extract/task/" + taskID
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
			return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	httpReq.Header.Set("Accept", "*/*")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
			return nil, err
	}
	defer resp.Body.Close()

	var out GetTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
	}
	if out.Code != 0 {
			return &out, fmt.Errorf("mineru get failed: code=%d msg=%s trace_id=%s", out.Code, out.Msg, out.TraceID)
	}
	return &out, nil
}