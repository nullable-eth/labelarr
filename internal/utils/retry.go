package utils

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// RetryConfig holds configuration for exponential backoff retry logic
type RetryConfig struct {
	MaxRetries      int           // Maximum number of retry attempts
	InitialDelay    time.Duration // Initial delay before first retry
	MaxDelay        time.Duration // Maximum delay between retries
	Multiplier      float64       // Multiplier for exponential backoff
	JitterFactor    float64       // Random jitter factor (0-1) to prevent thundering herd
	RetryableStatus []int         // HTTP status codes that should trigger a retry
}

// DefaultRetryConfig returns sensible defaults for API clients
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:   7,
		InitialDelay: 2 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.5,
		JitterFactor: 0.3,
		RetryableStatus: []int{
			http.StatusTooManyRequests,     // 429
			http.StatusServiceUnavailable,  // 503
			http.StatusGatewayTimeout,      // 504
			http.StatusBadGateway,          // 502
			http.StatusRequestTimeout,      // 408
			http.StatusInternalServerError, // 500
		},
	}
}

// IsRetryableStatus checks if the HTTP status code should trigger a retry
func (c *RetryConfig) IsRetryableStatus(statusCode int) bool {
	for _, code := range c.RetryableStatus {
		if code == statusCode {
			return true
		}
	}
	return false
}

// CalculateDelay calculates the delay for a given attempt with jitter
func (c *RetryConfig) CalculateDelay(attempt int) time.Duration {
	delay := float64(c.InitialDelay) * math.Pow(c.Multiplier, float64(attempt))
	
	// Apply max delay cap
	if delay > float64(c.MaxDelay) {
		delay = float64(c.MaxDelay)
	}
	
	// Apply jitter to prevent thundering herd
	if c.JitterFactor > 0 {
		jitter := delay * c.JitterFactor * (rand.Float64()*2 - 1) // -jitter to +jitter
		delay += jitter
	}
	
	return time.Duration(delay)
}

// RetryableHTTPClient wraps an http.Client with retry logic
type RetryableHTTPClient struct {
	client *http.Client
	config *RetryConfig
}

// NewRetryableHTTPClient creates a new HTTP client with retry capabilities
func NewRetryableHTTPClient(client *http.Client, config *RetryConfig) *RetryableHTTPClient {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryableHTTPClient{
		client: client,
		config: config,
	}
}

// Do executes the request with exponential backoff retry logic
func (r *RetryableHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return r.DoWithContext(req.Context(), req)
}

// DoWithContext executes the request with context and exponential backoff
func (r *RetryableHTTPClient) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response
	
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check if context is cancelled
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		
		// Wait before retry (skip on first attempt)
		if attempt > 0 {
			delay := r.config.CalculateDelay(attempt - 1)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		
		// Clone the request for retry (body needs special handling if present)
		reqClone := req.Clone(ctx)
		
		resp, err := r.client.Do(reqClone)
		if err != nil {
			lastErr = err
			// Network errors are retryable
			continue
		}
		
		// Check if we should retry based on status code
		if r.config.IsRetryableStatus(resp.StatusCode) {
			lastResp = resp
			lastErr = fmt.Errorf("server returned status %d", resp.StatusCode)
			resp.Body.Close()
			continue
		}
		
		// Success or non-retryable error
		return resp, nil
	}
	
	// All retries exhausted
	if lastErr != nil {
		return lastResp, fmt.Errorf("request failed after %d retries: %w", r.config.MaxRetries, lastErr)
	}
	
	return lastResp, nil
}
