package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// compile-time proofs of WebhookNotificationService interface implementation
var _ NotificationService = (*WebhookNotificationService)(nil)

type WebhookNotificationService struct {
	BaseUrl                  string
	SendNotificationEndpoint string
	ApiKey                   string
	HttpClient               *http.Client
	MaxRetries               int
	InitialRetryDelay        time.Duration
	MaxRetryDelay            time.Duration
	RateLimiter              *rate.Limiter
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
	MaxRetries             int
	InitialRetryDelayMs    int
	MaxRetryDelayMs        int
	RateLimitPerSecond     int
	RateLimitBurst         int
}

func (s *WebhookNotificationService) SendNotification(ctx context.Context, input Input) (AcknowledgeResponse, error) {
	url := fmt.Sprintf("%s/%s", s.BaseUrl, s.SendNotificationEndpoint)

	jsonData, err := json.Marshal(input)
	if err != nil {
		return AcknowledgeResponse{}, err
	}

	var lastErr error
	for attempt := 0; attempt <= s.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := s.calculateBackoff(attempt)
			log.Printf("retrying webhook request (attempt %d/%d) after %v", attempt, s.MaxRetries, delay)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return AcknowledgeResponse{}, ctx.Err()
			}
		}

		if err := s.RateLimiter.Wait(ctx); err != nil {
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
			lastErr = err
			if s.isRetriable(err, 0) {
				continue
			}
			return AcknowledgeResponse{}, err
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var response AcknowledgeResponse
			if err := json.Unmarshal(bodyBytes, &response); err != nil {
				return AcknowledgeResponse{}, fmt.Errorf("failed to decode response: %w", err)
			}
			return response, nil
		}

		lastErr = fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(bodyBytes))
		log.Printf("webhook error: %v", lastErr)

		if !s.isRetriable(nil, resp.StatusCode) {
			return AcknowledgeResponse{}, lastErr
		}
	}

	return AcknowledgeResponse{}, fmt.Errorf("webhook failed after %d retries: %w", s.MaxRetries, lastErr)
}

func (s *WebhookNotificationService) calculateBackoff(attempt int) time.Duration {
	delay := float64(s.InitialRetryDelay) * math.Pow(2, float64(attempt-1))
	if delay > float64(s.MaxRetryDelay) {
		delay = float64(s.MaxRetryDelay)
	}
	return time.Duration(delay)
}

func (s *WebhookNotificationService) isRetriable(err error, statusCode int) bool {
	if err != nil {
		return true
	}

	return statusCode == 408 || statusCode == 429 || statusCode >= 500
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

	maxRetries := config.MaxRetries

	if maxRetries < 0 {
		maxRetries = 0
	}

	initialRetryDelay := time.Duration(config.InitialRetryDelayMs) * time.Millisecond
	maxRetryDelay := time.Duration(config.MaxRetryDelayMs) * time.Millisecond

	rateLimitPerSecond := config.RateLimitPerSecond
	rateLimitBurst := config.RateLimitBurst

	limiter := rate.NewLimiter(rate.Limit(rateLimitPerSecond), rateLimitBurst)

	return WebhookNotificationService{
		BaseUrl:                  config.BaseUrl,
		SendNotificationEndpoint: config.Endpoint,
		ApiKey:                   config.ApiKey,
		HttpClient:               httpClient,
		MaxRetries:               maxRetries,
		InitialRetryDelay:        initialRetryDelay,
		MaxRetryDelay:            maxRetryDelay,
		RateLimiter:              limiter,
	}
}
