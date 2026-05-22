package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/moderato-app/live-pprof/internal/metrics"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ProfileCatalogEntry struct {
	Name            string `json:"name"`
	Count           int64  `json:"count"`
	Description     string `json:"description"`
	SupportsChart   bool   `json:"supportsChart"`
	SupportsRawText bool   `json:"supportsRawText"`
	SupportsImport  bool   `json:"supportsImport"`
	SupportsExport  bool   `json:"supportsExport"`
	SupportsFlame   bool   `json:"supportsFlame"`
}

var profileDescriptions = map[string]string{
	"allocs":       "A sampling of all past memory allocations",
	"block":        "Stack traces that led to blocking on synchronization primitives",
	"cmdline":      "The command line invocation of the current program",
	"goroutine":    "Stack traces of all current goroutines. Use debug=2 to export in panic-like format.",
	"heap":         "A sampling of memory allocations of live objects. You can specify gc=1 before taking the sample.",
	"mutex":        "Stack traces of holders of contended mutexes",
	"profile":      "CPU profile. You can specify the duration in seconds and inspect it with go tool pprof.",
	"symbol":       "Maps given program counters to function names.",
	"threadcreate": "Stack traces that led to the creation of new OS threads",
	"trace":        "A trace of execution. You can specify the duration in seconds and inspect it with go tool trace.",
}

var profileRow = regexp.MustCompile(`<tr><td>(\d+)</td><td><a href='([^']+)'>([^<]+)</a></td></tr>`)
var serveLine = regexp.MustCompile(`http://[^\s]+`)

func (s *Service) FetchProfileCatalog(parent context.Context, input string) ([]ProfileCatalogEntry, error) {
	baseURL, err := normalizeURL(input)
	if err != nil {
		return nil, err
	}

	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	indexURL := strings.TrimRight(baseURL, "/") + "/"
	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("bad status code: %s", resp.Status)
	}

	results := make([]ProfileCatalogEntry, 0, 12)
	for _, match := range profileRow.FindAllStringSubmatch(string(body), -1) {
		if len(match) != 4 {
			continue
		}
		count := int64(0)
		fmt.Sscanf(match[1], "%d", &count)
		name := strings.TrimSpace(strings.ToLower(match[3]))
		results = append(results, ProfileCatalogEntry{
			Name:            name,
			Count:           count,
			Description:     profileDescriptions[name],
			SupportsChart:   supportsChart(name),
			SupportsRawText: supportsRawText(name),
			SupportsImport:  supportsImport(name),
			SupportsExport:  supportsExport(name),
			SupportsFlame:   supportsFlame(name),
		})
	}
	return results, nil
}

func (s *Service) FetchProfileText(parent context.Context, input string, profile string, debug int, seconds uint64) (string, error) {
	baseURL, err := normalizeURL(input)
	if err != nil {
		return "", err
	}

	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	target, err := profileURL(baseURL, profile, debug, seconds)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("bad status code: %s", resp.Status)
	}
	return string(data), nil
}

func (s *Service) DownloadProfile(parent context.Context, input string, profile string, debug int, seconds uint64) (string, error) {
	baseURL, err := normalizeURL(input)
	if err != nil {
		return "", err
	}

	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	target, err := profileURL(baseURL, profile, debug, seconds)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("bad status code: %s", resp.Status)
	}

	defaultName := defaultDownloadedFileName(profile, debug)
	path, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           "Save " + profile,
		DefaultFilename: defaultName,
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) OpenProfileFlamegraph(parent context.Context, input string, profile string, seconds uint64) (string, error) {
	baseURL, err := normalizeURL(input)
	if err != nil {
		return "", err
	}
	if !supportsFlame(profile) {
		return "", fmt.Errorf("profile %s does not support flamegraph", profile)
	}

	target, err := profileURL(baseURL, profile, 0, seconds)
	if err != nil {
		return "", err
	}

	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	port, err := findFreePort()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, "go", "tool", "pprof", "-no_browser", fmt.Sprintf("-http=127.0.0.1:%d", port), target)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}

	lines := make(chan string, 16)
	readPipe := func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}
	go readPipe(stdout)
	go readPipe(stderr)

	timeout := time.NewTimer(20 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case <-timeout.C:
			return "", errors.New("timeout waiting for go tool pprof web UI")
		case line := <-lines:
			if strings.Contains(strings.ToLower(line), "serving web UI") || strings.Contains(strings.ToLower(line), "web ui") {
				return fmt.Sprintf("http://127.0.0.1:%d/flamegraph", port), nil
			}
			if match := serveLine.FindString(line); match != "" {
				return strings.TrimRight(match, "/") + "/flamegraph", nil
			}
		}
	}
}

func profileURL(baseURL string, profile string, debug int, seconds uint64) (string, error) {
	name := strings.TrimSpace(strings.ToLower(profile))
	switch name {
	case "allocs":
		return metrics.MetricsURL(debug > 0, metrics.MetricsTypeAllocs, baseURL, seconds)
	case "heap":
		return metrics.MetricsURL(debug > 0, metrics.MetricsTypeHeap, baseURL, seconds)
	case "goroutine":
		return metrics.MetricsURL(debug > 0, metrics.MetricsTypeGoroutine, baseURL, seconds)
	case "profile", "cpu":
		return metrics.MetricsURL(false, metrics.MetricsTypeCPU, baseURL, seconds)
	case "block":
		return metrics.MetricsURL(debug > 0, metrics.MetricsTypeBlock, baseURL, seconds)
	case "mutex":
		return metrics.MetricsURL(debug > 0, metrics.MetricsTypeMutex, baseURL, seconds)
	case "threadcreate":
		return metrics.MetricsURL(debug > 0, metrics.MetricsTypeThread, baseURL, seconds)
	case "cmdline", "symbol", "trace":
		u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/" + name)
		if err != nil {
			return "", err
		}
		q := u.Query()
		if debug > 0 {
			q.Set("debug", fmt.Sprintf("%d", debug))
		}
		if name == "trace" && seconds > 0 {
			q.Set("seconds", fmt.Sprintf("%d", seconds))
		}
		u.RawQuery = q.Encode()
		return u.String(), nil
	default:
		return "", fmt.Errorf("unsupported profile: %s", profile)
	}
}

func supportsChart(name string) bool {
	switch name {
	case "allocs", "heap", "goroutine", "profile", "block", "mutex", "threadcreate":
		return true
	default:
		return false
	}
}

func supportsRawText(name string) bool {
	switch name {
	case "allocs", "heap", "goroutine", "block", "mutex", "threadcreate", "cmdline":
		return true
	default:
		return false
	}
}

func supportsImport(name string) bool {
	switch name {
	case "allocs", "heap", "goroutine", "profile", "block", "mutex", "threadcreate":
		return true
	default:
		return false
	}
}

func supportsExport(name string) bool {
	switch name {
	case "allocs", "heap", "goroutine", "profile", "block", "mutex", "threadcreate", "cmdline", "trace":
		return true
	default:
		return false
	}
}

func supportsFlame(name string) bool {
	switch name {
	case "profile", "heap", "allocs", "block", "mutex", "threadcreate":
		return true
	default:
		return false
	}
}

func defaultDownloadedFileName(profile string, debug int) string {
	if supportsRawText(profile) && debug > 0 {
		return fmt.Sprintf("%s-debug%d-%s.txt", profile, debug, time.Now().Format("20060102-150405"))
	}
	ext := ".pb.gz"
	if profile == "cmdline" {
		ext = ".txt"
	}
	if profile == "trace" {
		ext = ".out"
	}
	return fmt.Sprintf("%s-%s%s", profile, time.Now().Format("20060102-150405"), ext)
}

func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	parts := strings.Split(ln.Addr().String(), ":")
	if len(parts) == 0 {
		return 0, errors.New("failed to parse free port")
	}
	return strconv.Atoi(parts[len(parts)-1])
}
