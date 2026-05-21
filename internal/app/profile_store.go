package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/moderato-app/pprof/moderato"
	pprofProfile "github.com/moderato-app/pprof/profile"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ProfileMeta struct {
	Metric     string `json:"metric"`
	Source     string `json:"source"`
	FileName   string `json:"fileName"`
	Imported   bool   `json:"imported"`
	Exportable bool   `json:"exportable"`
	Timestamp  int64  `json:"timestamp"`
}

type profileBlob struct {
	Metric    string
	Data      []byte
	Source    string
	FileName  string
	Imported  bool
	Timestamp int64
}

type profileStore struct {
	mu    sync.RWMutex
	items map[string]profileBlob
}

func newProfileStore() *profileStore {
	return &profileStore{
		items: map[string]profileBlob{},
	}
}

func (s *profileStore) set(metric string, blob profileBlob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[metric] = blob
}

func (s *profileStore) get(metric string) (profileBlob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	blob, ok := s.items[metric]
	return blob, ok
}

func (s *profileStore) meta(metric string) ProfileMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	blob, ok := s.items[metric]
	if !ok {
		return ProfileMeta{
			Metric:     metric,
			Exportable: false,
		}
	}
	return ProfileMeta{
		Metric:     metric,
		Source:     blob.Source,
		FileName:   blob.FileName,
		Imported:   blob.Imported,
		Exportable: len(blob.Data) > 0,
		Timestamp:  blob.Timestamp,
	}
}

func openProfileDialog(ctx context.Context, metric string) (string, error) {
	path, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "Import " + strings.ToUpper(metric) + " pprof file",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "pprof profiles",
				Pattern:     "*.pb.gz;*.pb;*.pprof;*.prof;*.profile;*",
			},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	return path, nil
}

func openMultipleProfileDialog(ctx context.Context, metric string) ([]string, error) {
	paths, err := runtime.OpenMultipleFilesDialog(ctx, runtime.OpenDialogOptions{
		Title: "Import " + strings.ToUpper(metric) + " pprof files",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "pprof profiles",
				Pattern:     "*.pb.gz;*.pb;*.pprof;*.prof;*.profile;*",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func saveProfileDialog(ctx context.Context, metric string, fileName string) (string, error) {
	defaultName := fileName
	if strings.TrimSpace(defaultName) == "" {
		defaultName = fmt.Sprintf("%s-%s.pb.gz", metric, time.Now().Format("20060102-150405"))
	}

	path, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           "Export " + strings.ToUpper(metric) + " pprof file",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "pprof profiles",
				Pattern:     "*.pb.gz",
			},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	return path, nil
}

func readProfileFile(path string) ([]byte, string, int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", 0, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", 0, err
	}
	return data, filepath.Base(path), info.ModTime().UnixNano(), nil
}

func writeProfileFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func profileSnapshotFromData(metric string, source string, data []byte, timestamp int64) (*MetricsSnapshot, error) {
	mtr, err := moderato.GetMetricsFromData(data)
	if err != nil {
		return nil, err
	}

	var stacks []GoroutineStack
	if metric == "goroutine" {
		if parsed, err := parseGoroutineStacks(data); err == nil {
			stacks = parsed
		}
	}

	resp := metricsToSnapshot(metric, source, mtr, timestamp, stacks)
	return resp, nil
}

func metricsToSnapshot(metric string, source string, mtr *moderato.Metrics, timestamp int64, stacks []GoroutineStack) *MetricsSnapshot {
	if timestamp == 0 {
		timestamp = time.Now().UnixNano()
	}

	items := make([]MetricPoint, 0, len(mtr.Items)+1)
	items = append(items, MetricPoint{
		Function: "total",
		Line:     "",
		Flat:     mtr.Total,
		Cum:      mtr.Total,
	})
	for _, item := range mtr.Items {
		items = append(items, MetricPoint{
			Function: item.Func,
			Line:     item.Line,
			Flat:     item.Flat,
			Cum:      item.Cum,
		})
	}

	return &MetricsSnapshot{
		Type:      metric,
		URL:       source,
		Timestamp: timestamp,
		Total:     mtr.Total,
		Items:     items,
		Stacks:    stacks,
	}
}

func parseGoroutineStacks(data []byte) ([]GoroutineStack, error) {
	prof, err := pprofProfile.ParseData(data)
	if err != nil {
		return nil, err
	}

	stackMap := make(map[string]*GoroutineStack)
	for _, sample := range prof.Sample {
		var frames []StackFrame
		var keyParts []string
		// Locations are in innermost-first order; reverse for display (outermost first)
		for i := len(sample.Location) - 1; i >= 0; i-- {
			loc := sample.Location[i]
			for _, line := range loc.Line {
				frame := StackFrame{
					Func: line.Function.Name,
					File: line.Function.Filename,
					Line: int(line.Line),
				}
				frames = append(frames, frame)
				keyParts = append(keyParts, fmt.Sprintf("%s@%s:%d", frame.Func, frame.File, frame.Line))
			}
		}

		count := int64(1)
		if len(sample.Value) > 0 {
			count = sample.Value[0]
		}

		key := strings.Join(keyParts, "|")
		if existing, ok := stackMap[key]; ok {
			existing.Count += count
		} else {
			framesCopy := make([]StackFrame, len(frames))
			copy(framesCopy, frames)
			stackMap[key] = &GoroutineStack{
				Count:  count,
				Frames: framesCopy,
			}
		}
	}

	var stacks []GoroutineStack
	for _, stack := range stackMap {
		stacks = append(stacks, *stack)
	}

	slices.SortFunc(stacks, func(a, b GoroutineStack) int {
		if a.Count > b.Count {
			return -1
		}
		if a.Count < b.Count {
			return 1
		}
		return 0
	})

	return stacks, nil
}

func readAllProfile(reader io.Reader) ([]byte, error) {
	return io.ReadAll(reader)
}
