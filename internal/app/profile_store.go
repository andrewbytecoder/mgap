package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/moderato-app/live-pprof/internal/metrics"
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
	profileInfo, _ := metrics.ReadProfileInfo(data)

	var stacks []GoroutineStack
	var rawText string
	if metric == "goroutine" {
		parsed, stackErr := parseGoroutineStacks(data)
		if stackErr != nil {
			rawText = fmt.Sprintf("[parse stacks error] %v\n", stackErr)
			log.Printf("[goroutine stacks] parse error: %v", stackErr)
		} else {
			stacks = parsed
		}
		text, textErr := generateGoroutineText(data)
		if textErr != nil {
			rawText += fmt.Sprintf("[generate text error] %v\n\n--- raw data (first 2KB) ---\n%s", textErr, truncateBytes(data, 2048))
			log.Printf("[goroutine text] generate error: %v", textErr)
		} else {
			if rawText != "" {
				rawText += "\n" + text
			} else {
				rawText = text
			}
		}
		if rawText == "" {
			rawText = fmt.Sprintf("[debug] dataLen=%d sampleCount=%d stacks=%d", len(data), len(stacks), len(stacks))
		}
	}

	return metricsToSnapshot(metric, source, mtr, timestamp, stacks, rawText, profileInfo), nil
}

func metricsToSnapshot(metric string, source string, mtr *moderato.Metrics, timestamp int64, stacks []GoroutineStack, rawText string, profileInfo *metrics.ProfileInfo) *MetricsSnapshot {
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
		RawText:   rawText,
		DefaultSampleType: func() string {
			if profileInfo == nil {
				return ""
			}
			return profileInfo.DefaultSampleType
		}(),
		DefaultSampleUnit: func() string {
			if profileInfo == nil {
				return ""
			}
			return profileInfo.DefaultSampleUnit
		}(),
		DurationNanos: func() int64 {
			if profileInfo == nil {
				return 0
			}
			return profileInfo.DurationNanos
		}(),
		Period: func() int64 {
			if profileInfo == nil {
				return 0
			}
			return profileInfo.Period
		}(),
		PeriodType: func() string {
			if profileInfo == nil {
				return ""
			}
			return profileInfo.PeriodType
		}(),
		PeriodUnit: func() string {
			if profileInfo == nil {
				return ""
			}
			return profileInfo.PeriodUnit
		}(),
	}
}

func parseGoroutineStacks(data []byte) ([]GoroutineStack, error) {
	prof, err := pprofProfile.ParseData(data)
	if err != nil {
		return nil, err
	}

	if len(prof.Sample) == 0 {
		return nil, fmt.Errorf("profile contains no samples")
	}

	stackMap := make(map[string]*GoroutineStack)
	for _, sample := range prof.Sample {
		var frames []StackFrame
		var keyParts []string
		// Locations are in innermost-first order; reverse for display (outermost first)
		for i := len(sample.Location) - 1; i >= 0; i-- {
			loc := sample.Location[i]
			if loc == nil {
				continue
			}
			// Handle both symbolised (Line) and raw-address-only locations.
			if len(loc.Line) > 0 {
				for _, line := range loc.Line {
					funcName := "??"
					fileName := ""
					fileLine := int64(0)
					if line.Function != nil {
						funcName = line.Function.Name
						fileName = line.Function.Filename
					}
					fileLine = line.Line
					frame := StackFrame{
						Func: funcName,
						File: fileName,
						Line: int(fileLine),
					}
					frames = append(frames, frame)
					keyParts = append(keyParts, fmt.Sprintf("%s@%s:%d", frame.Func, frame.File, frame.Line))
				}
			} else {
				// No line info available – use raw address as frame identifier.
				label := fmt.Sprintf("0x%x", loc.Address)
				frame := StackFrame{
					Func: label,
					File: "",
					Line: 0,
				}
				frames = append(frames, frame)
				keyParts = append(keyParts, label)
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

// generateGoroutineText produces a human-readable text representation of a
// goroutine profile, similar to what /debug/pprof/goroutine?debug=2 shows.
func generateGoroutineText(data []byte) (string, error) {
	prof, err := pprofProfile.ParseData(data)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("goroutine profile: total %d\n", len(prof.Sample)))

	// Group samples by stack and count.
	stackCounts := make(map[string]struct {
		count  int64
		frames []string
	})
	var orderedKeys []string

	for _, sample := range prof.Sample {
		var frameLines []string
		var keyParts []string

		for i := len(sample.Location) - 1; i >= 0; i-- {
			loc := sample.Location[i]
			if loc == nil {
				continue
			}
			addrStr := fmt.Sprintf("0x%x", loc.Address)

			if len(loc.Line) > 0 && loc.Line[0].Function != nil {
				fn := loc.Line[0].Function
				frameLines = append(frameLines, fmt.Sprintf("#\t0x%x\t%s+0x%x\t\t\t%s:%d",
					loc.Address, fn.Name, 0, fn.Filename, loc.Line[0].Line))
			} else {
				frameLines = append(frameLines, fmt.Sprintf("#\t0x%x", loc.Address))
			}
			keyParts = append(keyParts, addrStr)
		}

		count := int64(1)
		if len(sample.Value) > 0 {
			count = sample.Value[0]
		}

		key := strings.Join(keyParts, " ")
		if _, exists := stackCounts[key]; !exists {
			orderedKeys = append(orderedKeys, key)
		}
		entry := stackCounts[key]
		entry.count += count
		entry.frames = frameLines
		stackCounts[key] = entry
	}

	for _, key := range orderedKeys {
		entry := stackCounts[key]
		b.WriteString(fmt.Sprintf("%d @ %s\n", entry.count, key))
		for _, frameLine := range entry.frames {
			b.WriteString(frameLine + "\n")
		}
	}

	return b.String(), nil
}

func readAllProfile(reader io.Reader) ([]byte, error) {
	return io.ReadAll(reader)
}

func truncateBytes(data []byte, max int) string {
	if len(data) <= max {
		return string(data)
	}
	return string(data[:max]) + "..."
}
