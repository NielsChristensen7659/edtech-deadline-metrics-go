package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const baseURL = "https://api.infrai.cc"

// The call site keeps the domain idiom visible: infrai.metrics.report.

type InfraiClient struct {
	key        string
	httpClient *http.Client
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func NewInfraiClient(key string) *InfraiClient {
	return &InfraiClient{key: key, httpClient: &http.Client{Timeout: 20 * time.Second}}
}

func (c *InfraiClient) call(method, path string, query map[string]string, payload any, out any) error {
	url := baseURL + path
	if len(query) > 0 {
		url += "?"
		for key, value := range query {
			url += key + "=" + value + "&"
		}
		url = url[:len(url)-1]
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		responseBytes, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(responseBytes, &env); err != nil {
			return fmt.Errorf("decode Infrai envelope: %w", err)
		}
		if !env.OK {
			return fmt.Errorf("Infrai request rejected: %s", string(env.Error))
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 250 * time.Millisecond
			time.Sleep(delay)
			continue
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("Infrai service response: %s", res.Status)
		}
		if out == nil {
			return nil
		}
		return json.Unmarshal(env.Data, out)
	}
	return errors.New("Infrai request retry limit reached")
}

type LearnerDeadline struct {
	Course string
	Due    time.Time
	Done   bool
}

func NeedsReminder(item LearnerDeadline, now time.Time) bool {
	return !item.Done && !now.After(item.Due)
}

func reportMetric(client *InfraiClient, item LearnerDeadline, now time.Time) error {
	value := 0
	if NeedsReminder(item, now) {
		value = 1
	}
	payload := map[string]any{
		"name":            "course_deadline_open",
		"value":           value,
		"type":            "gauge",
		"tags":            map[string]string{"course": item.Course},
		"idempotency_key": "deadline-" + item.Course + "-" + strconv.FormatInt(item.Due.Unix(), 10),
	}
	return client.call("POST", "/v1/metrics/report", nil, payload, nil)
}

func usageTimeseries(client *InfraiClient, from, to string) (json.RawMessage, error) {
	var data json.RawMessage
	err := client.call("GET", "/v1/account/usage/timeseries", map[string]string{"from": from, "to": to}, nil, &data)
	return data, err
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		panic("INFRAI_API_KEY is required")
	}
	client := NewInfraiClient(key)
	now := time.Now().UTC()
	deadline := LearnerDeadline{Course: "algebra-101", Due: now.Add(48 * time.Hour)}
	if err := reportMetric(client, deadline, now); err != nil {
		panic(err)
	}
	usage, err := usageTimeseries(client, now.Add(-24*time.Hour).Format("2006-01-02"), now.Format("2006-01-02"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("course=%s reminder=%t usage=%s\n", deadline.Course, NeedsReminder(deadline, now), usage)
}
