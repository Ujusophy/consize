package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Series struct {
	Metric map[string]string
	Points []Point
}

type Point struct {
	Timestamp time.Time
	Value     float64
}

type Client interface {
	QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]Series, error)
}

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient(baseURL string, client *http.Client) (*HTTPClient, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("prometheus base url is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &HTTPClient{baseURL: strings.TrimRight(baseURL, "/"), client: client}, nil
}

func (p *HTTPClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]Series, error) {
	u, err := url.Parse(p.baseURL + "/api/v1/query_range")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("query", query)
	q.Set("start", strconv.FormatInt(start.Unix(), 10))
	q.Set("end", strconv.FormatInt(end.Unix(), 10))
	q.Set("step", fmt.Sprintf("%ds", int(step.Seconds())))
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus query_range %q: %w", query, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("prometheus query_range %q: status %s: %s", query, resp.Status, body)
	}
	var envelope struct {
		Status string `json:"status"`
		Data   struct {
			Result json.RawMessage `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode prometheus response: %w", err)
	}
	if envelope.Status != "success" {
		return nil, fmt.Errorf("prometheus returned status %q", envelope.Status)
	}
	var raw []struct {
		Metric map[string]string `json:"metric"`
		Values [][2]any          `json:"values"`
	}
	if err := json.Unmarshal(envelope.Data.Result, &raw); err != nil {
		return nil, fmt.Errorf("parse prometheus result: %w", err)
	}
	out := make([]Series, 0, len(raw))
	for _, r := range raw {
		s := Series{Metric: r.Metric}
		for _, v := range r.Values {
			ts, ok := v[0].(float64)
			if !ok {
				continue
			}
			val, err := strconv.ParseFloat(fmt.Sprintf("%v", v[1]), 64)
			if err != nil {
				continue
			}
			s.Points = append(s.Points, Point{Timestamp: time.Unix(int64(ts), 0).UTC(), Value: val})
		}
		out = append(out, s)
	}
	return out, nil
}
