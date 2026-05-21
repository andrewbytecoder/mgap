package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/moderato-app/live-pprof/api"
	"github.com/moderato-app/live-pprof/internal/metrics"
)

type MetricInfo struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Unit        string `json:"unit"`
}

type MetricPoint struct {
	Function string `json:"function"`
	Line     string `json:"line"`
	Flat     int64  `json:"flat"`
	Cum      int64  `json:"cum"`
}

type MetricsSnapshot struct {
	Type      string        `json:"type"`
	URL       string        `json:"url"`
	Timestamp int64         `json:"timestamp"`
	Total     int64         `json:"total"`
	Items     []MetricPoint `json:"items"`
}

type EndpointResult struct {
	Endpoint   string `json:"endpoint"`
	StatusCode int32  `json:"statusCode"`
	StatusText string `json:"statusText"`
	Body       string `json:"body"`
	Error      string `json:"error"`
}

type Service struct {
	metrics     *metrics.MetricsServer
	mockMetrics *metrics.MockMetricsServer
	profiles    *profileStore
}

func NewService() *Service {
	return &Service{
		metrics:     metrics.NewMetricsServer(),
		mockMetrics: metrics.NewMockMetricsServer(),
		profiles:    newProfileStore(),
	}
}

func (s *Service) InitialURL() string {
	return ""
}

func (s *Service) AvailableMetrics() []MetricInfo {
	return []MetricInfo{
		{
			Key:         "cpu",
			Label:       "CPU",
			Description: "cpu samples: /debug/pprof/profile?seconds=N",
			Unit:        "time",
		},
		{
			Key:         "heap",
			Label:       "Heap",
			Description: "inuse_space: /debug/pprof/heap",
			Unit:        "bytes",
		},
		{
			Key:         "allocs",
			Label:       "Allocs",
			Description: "alloc_space: /debug/pprof/allocs",
			Unit:        "bytes",
		},
		{
			Key:         "goroutine",
			Label:       "Goroutine",
			Description: "goroutine: /debug/pprof/goroutine",
			Unit:        "count",
		},
	}
}

func (s *Service) NormalizeURL(input string) (string, error) {
	return normalizeURL(input)
}

func (s *Service) DetectURL(parent context.Context, input string) ([]EndpointResult, error) {
	baseURL, err := normalizeURL(input)
	if err != nil {
		return nil, err
	}

	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	endpoints := []string{baseURL}
	for _, mt := range []metrics.MetricsType{
		metrics.MetricsTypeHeap,
		metrics.MetricsTypeCPU,
		metrics.MetricsTypeAllocs,
		metrics.MetricsTypeGoroutine,
	} {
		u, metricErr := metrics.MetricsURL(true, mt, baseURL, 1)
		if metricErr != nil {
			return nil, metricErr
		}
		endpoints = append(endpoints, u)
	}

	results := make([]EndpointResult, 0, len(endpoints))
	for _, endpoint := range endpoints {
		httpResult, detectErr := detectHTTP(timeoutCtx, endpoint)
		item := EndpointResult{
			Endpoint: endpoint,
		}
		if httpResult != nil {
			item.StatusCode = httpResult.StatusCode
			item.StatusText = httpResult.Status
			item.Body = httpResult.Body
		}
		if detectErr != nil {
			item.Error = detectErr.Error()
		}
		results = append(results, item)
	}
	return results, nil
}

func (s *Service) FetchMetrics(parent context.Context, input string, metric string, profileSeconds uint64, useMock bool) (*MetricsSnapshot, error) {
	baseURL, err := normalizeURL(input)
	if err != nil {
		return nil, err
	}

	metricType, err := parseMetric(metric)
	if err != nil {
		return nil, err
	}

	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	timeoutSeconds := 5
	if metricType == metrics.MetricsTypeCPU {
		timeoutSeconds += int(max(profileSeconds, 1))
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	profileURL, err := metrics.MetricsURL(false, metricType, baseURL, profileSeconds)
	if err == nil && !useMock {
		capturedAt := time.Now().UnixNano()
		if data, fetchErr := metrics.FetchRawProfile(timeoutCtx, profileURL); fetchErr == nil {
			s.profiles.set(metric, profileBlob{
				Metric:    metric,
				Data:      data,
				Source:    profileURL,
				FileName:  suggestedProfileFileName(metric, capturedAt),
				Imported:  false,
				Timestamp: capturedAt,
			})
		}
	}

	req := &api.GoMetricsRequest{
		Url:            baseURL,
		ProfileSeconds: profileSeconds,
	}

	var resp *api.GoMetricsResponse
	if useMock {
		resp, err = s.fetchMock(timeoutCtx, req, metricType)
	} else {
		resp, err = s.fetchLive(timeoutCtx, req, metricType)
	}
	if err != nil {
		return nil, err
	}

	items := make([]MetricPoint, 0, len(resp.Items))
	for _, item := range resp.Items {
		if item == nil {
			continue
		}
		items = append(items, MetricPoint{
			Function: item.Func,
			Line:     item.Line,
			Flat:     item.Flat,
			Cum:      item.Cum,
		})
	}

	timestamp := resp.Date
	if timestamp == 0 {
		timestamp = time.Now().UnixNano()
	}

	return &MetricsSnapshot{
		Type:      metric,
		URL:       baseURL,
		Timestamp: timestamp,
		Total:     resp.Total,
		Items:     items,
	}, nil
}

func (s *Service) ImportProfile(parent context.Context, metric string) (*MetricsSnapshot, error) {
	if _, err := parseMetric(metric); err != nil {
		return nil, err
	}

	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	path, err := openProfileDialog(ctx, metric)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}

	data, fileName, fileTimestamp, err := readProfileFile(path)
	if err != nil {
		return nil, err
	}

	snapshot, err := profileSnapshotFromData(metric, path, data, fileTimestamp)
	if err != nil {
		return nil, err
	}

	s.profiles.set(metric, profileBlob{
		Metric:    metric,
		Data:      data,
		Source:    path,
		FileName:  fileName,
		Imported:  true,
		Timestamp: fileTimestamp,
	})

	return snapshot, nil
}

func (s *Service) ExportProfile(parent context.Context, metric string) (string, error) {
	if _, err := parseMetric(metric); err != nil {
		return "", err
	}

	blob, ok := s.profiles.get(metric)
	if !ok || len(blob.Data) == 0 {
		return "", errors.New("no profile data available to export yet")
	}

	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	path, err := saveProfileDialog(ctx, metric, blob.FileName)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}

	if err := writeProfileFile(path, blob.Data); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) ProfileMeta(metric string) (ProfileMeta, error) {
	if _, err := parseMetric(metric); err != nil {
		return ProfileMeta{}, err
	}
	return s.profiles.meta(metric), nil
}

func (s *Service) fetchLive(ctx context.Context, req *api.GoMetricsRequest, metricType metrics.MetricsType) (*api.GoMetricsResponse, error) {
	switch metricType {
	case metrics.MetricsTypeCPU:
		return s.metrics.CPUMetrics(ctx, req)
	case metrics.MetricsTypeHeap:
		return s.metrics.HeapMetrics(ctx, req)
	case metrics.MetricsTypeAllocs:
		return s.metrics.AllocsMetrics(ctx, req)
	case metrics.MetricsTypeGoroutine:
		return s.metrics.GoroutineMetrics(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported metric type: %s", metricType)
	}
}

func (s *Service) fetchMock(ctx context.Context, req *api.GoMetricsRequest, metricType metrics.MetricsType) (*api.GoMetricsResponse, error) {
	switch metricType {
	case metrics.MetricsTypeCPU:
		return s.mockMetrics.CPUMetrics(ctx, req)
	case metrics.MetricsTypeHeap:
		return s.mockMetrics.HeapMetrics(ctx, req)
	case metrics.MetricsTypeAllocs:
		return s.mockMetrics.AllocsMetrics(ctx, req)
	case metrics.MetricsTypeGoroutine:
		return s.mockMetrics.GoroutineMetrics(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported metric type: %s", metricType)
	}
}

func parseMetric(metric string) (metrics.MetricsType, error) {
	switch strings.ToLower(strings.TrimSpace(metric)) {
	case "cpu":
		return metrics.MetricsTypeCPU, nil
	case "heap":
		return metrics.MetricsTypeHeap, nil
	case "allocs":
		return metrics.MetricsTypeAllocs, nil
	case "goroutine":
		return metrics.MetricsTypeGoroutine, nil
	default:
		return "", fmt.Errorf("unsupported metric: %s", metric)
	}
}

func normalizeURL(input string) (string, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return "", errors.New("please input the URL of pprof endpoint")
	}

	if port, err := strconv.Atoi(value); err == nil {
		if port < 1 || port > 65535 {
			return "", fmt.Errorf("port %d is out of range: [1, 65535]", port)
		}
		return fmt.Sprintf("http://localhost:%d/debug/pprof", port), nil
	}

	if !strings.Contains(value, "://") {
		value = "http://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid url: %s", input)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid URL")
	}

	if !strings.Contains(parsed.Path, "/debug/pprof") {
		path := strings.TrimSuffix(parsed.Path, "/")
		if path == "" {
			parsed.Path = "/debug/pprof"
		} else {
			parsed.Path = path + "/debug/pprof"
		}
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func max(a uint64, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

type httpResult struct {
	StatusCode int32
	Status     string
	Body       string
}

func detectHTTP(ctx context.Context, endpoint string) (*httpResult, error) {
	reqMethod := http.MethodGet
	if strings.Contains(endpoint, "/profile?seconds=") {
		reqMethod = http.MethodHead
	}

	req, err := http.NewRequestWithContext(ctx, reqMethod, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &httpResult{
		StatusCode: int32(resp.StatusCode),
		Status:     resp.Status,
		Body:       string(data),
	}, nil
}

func suggestedProfileFileName(metric string, timestamp int64) string {
	if timestamp == 0 {
		timestamp = time.Now().UnixNano()
	}
	return fmt.Sprintf("%s-%s.pb.gz", metric, time.Unix(0, timestamp).Format("20060102-150405"))
}
