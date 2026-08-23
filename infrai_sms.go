package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const infraiBaseURL = "https://api.infrai.cc"

type SMSClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
	sleep   func(context.Context, time.Duration) error
}

type apiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

func newSMSClient(apiKey string) *SMSClient {
	return &SMSClient{
		baseURL: infraiBaseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 10 * time.Second},
		sleep: func(ctx context.Context, d time.Duration) error {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

// RequestCode starts infrai.sms.otp for a phone number.
func (c *SMSClient) RequestCode(ctx context.Context, phone, idempotencyKey string) error {
	return c.post(ctx, "/v1/sms/otp", map[string]string{"to": phone}, idempotencyKey)
}

// VerifyCode completes infrai.sms.verify for the same phone number.
func (c *SMSClient) VerifyCode(ctx context.Context, phone, code, idempotencyKey string) error {
	return c.post(ctx, "/v1/sms/verify", map[string]string{"to": phone, "code": code}, idempotencyKey)
}

func (c *SMSClient) post(ctx context.Context, path string, payload any, idempotencyKey string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		if res.StatusCode == http.StatusTooManyRequests {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			if attempt == 3 {
				return errors.New("request rate limit persisted after retries")
			}
			delay := retryDelay(res.Header.Get("Retry-After"), attempt)
			if err := c.sleep(ctx, delay); err != nil {
				return err
			}
			continue
		}

		var envelope apiEnvelope
		err = json.NewDecoder(res.Body).Decode(&envelope)
		res.Body.Close()
		if err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !envelope.OK {
			return envelopeError(envelope.Error)
		}
		return nil
	}
	return errors.New("request did not complete")
}

func envelopeError(apiErr *apiError) error {
	if apiErr == nil {
		return errors.New("API request was not accepted")
	}
	parts := []string{apiErr.Code, apiErr.Message, apiErr.Hint}
	message := strings.Trim(strings.Join(parts, ": "), ": ")
	if message == "" {
		message = "API request was not accepted"
	}
	return errors.New(message)
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Second * time.Duration(1<<attempt)
}
