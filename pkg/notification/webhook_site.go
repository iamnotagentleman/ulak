package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// compile-time proofs of WebhookNotificationService interface implementation
var _ NotificationService = (*WebhookNotificationService)(nil)

type WebhookNotificationService struct {
	BaseUrl                  string
	SendNotificationEndpoint string
	ApiKey                   string
	HttpClient               *http.Client
}

type WebhookConfig struct {
	BaseUrl                string
	Endpoint               string
	ApiKey                 string
	TimeoutSeconds         int
	MaxIdleConns           int
	MaxIdleConnsPerHost    int
	IdleConnTimeoutSeconds int
	DisableCompression     bool
	DisableKeepAlives      bool
}

func (s *WebhookNotificationService) SendNotification(ctx context.Context, input Input) (AcknowledgeResponse, error) {
	url := fmt.Sprintf("%s/%s", s.BaseUrl, s.SendNotificationEndpoint)

	jsonData, err := json.Marshal(input)

	if err != nil {
		return AcknowledgeResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return AcknowledgeResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-ins-auth-key", s.ApiKey)

	resp, err := s.HttpClient.Do(req)

	if err != nil {
		return AcknowledgeResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// TODO log this data
		return AcknowledgeResponse{}, fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	var response AcknowledgeResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return AcknowledgeResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return response, nil
}

func NewWebhookNotificationService(config WebhookConfig) WebhookNotificationService {
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     time.Duration(config.IdleConnTimeoutSeconds) * time.Second,
		DisableCompression:  config.DisableCompression,
		DisableKeepAlives:   config.DisableKeepAlives,
	}

	timeout := time.Duration(config.TimeoutSeconds) * time.Second

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return WebhookNotificationService{
		BaseUrl:                  config.BaseUrl,
		SendNotificationEndpoint: config.Endpoint,
		ApiKey:                   config.ApiKey,
		HttpClient:               httpClient,
	}
}
