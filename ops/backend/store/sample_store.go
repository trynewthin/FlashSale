package store

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type MonitorSample struct {
	Timestamp                  int64    `json:"ts"`
	Label                      string   `json:"label"`
	PortRate                   float64  `json:"portRate"`
	HttpRate                   float64  `json:"httpRate"`
	ReplicaRate                float64  `json:"replicaRate"`
	RunningContainers          int      `json:"runningContainers"`
	TotalContainers            int      `json:"totalContainers"`
	RunningReplicas            int      `json:"runningReplicas"`
	TotalReplicas              int      `json:"totalReplicas"`
	PromQps                    *float64 `json:"promQps,omitempty"`
	PromP99LatencyMs           *float64 `json:"promP99LatencyMs,omitempty"`
	PromErrorRate              *float64 `json:"promErrorRate,omitempty"`
	PurchaseKafkaPublishRate   *float64 `json:"purchaseKafkaPublishRate,omitempty"`
	PurchaseKafkaPublishFailed *float64 `json:"purchaseKafkaPublishFailed,omitempty"`
	OrderStateConsumeRate      *float64 `json:"orderStateConsumeRate,omitempty"`
}

type SampleStore struct {
	baseDir    string
	mu         sync.Mutex
	curFile    *os.File
	curPath    string
	retainDays int
}

const sampleStoreSubDir = "log/monitor"

func NewSampleStore(repoRoot string) *SampleStore {
	return &SampleStore{baseDir: filepath.Join(repoRoot, sampleStoreSubDir), retainDays: 7}
}

func (s *SampleStore) Append(sample *MonitorSample) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	targetPath := s.filePath(time.UnixMilli(sample.Timestamp))
	if s.curPath != targetPath {
		if s.curFile != nil {
			_ = s.curFile.Close()
			s.curFile = nil
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("create sample dir: %w", err)
		}
		f, err := os.OpenFile(targetPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open sample file: %w", err)
		}
		s.curFile = f
		s.curPath = targetPath
	}
	data, err := json.Marshal(sample)
	if err != nil {
		return fmt.Errorf("marshal sample: %w", err)
	}
	_, err = s.curFile.Write(append(data, '\n'))
	return err
}

func (s *SampleStore) Query(startMs, endMs int64, stepMs int64) ([]MonitorSample, error) {
	startTime := time.UnixMilli(startMs)
	endTime := time.UnixMilli(endMs)
	var files []string
	for cur := startTime.Truncate(time.Hour); !cur.After(endTime); cur = cur.Add(time.Hour) {
		fp := s.filePath(cur)
		if _, err := os.Stat(fp); err == nil {
			files = append(files, fp)
		}
	}
	if len(files) == 0 {
		return nil, nil
	}
	var allSamples []MonitorSample
	for _, fp := range files {
		samples, err := readJSONLFile(fp, startMs, endMs)
		if err != nil {
			log.Printf("[sample_store] warning: read %s failed: %v", fp, err)
			continue
		}
		allSamples = append(allSamples, samples...)
	}
	sort.Slice(allSamples, func(i, j int) bool { return allSamples[i].Timestamp < allSamples[j].Timestamp })
	if stepMs > 0 && len(allSamples) > 0 {
		allSamples = downsample(allSamples, stepMs)
	}
	return allSamples, nil
}

func (s *SampleStore) Cleanup() {
	cutoff := time.Now().AddDate(0, 0, -s.retainDays).Format("2006-01-02")
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) == 10 && entry.Name() < cutoff {
			dirPath := filepath.Join(s.baseDir, entry.Name())
			if err := os.RemoveAll(dirPath); err != nil {
				log.Printf("[sample_store] cleanup %s failed: %v", dirPath, err)
			}
		}
	}
}

func (s *SampleStore) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.curFile != nil {
		_ = s.curFile.Close()
		s.curFile = nil
	}
}

func (s *SampleStore) ListAvailableDays() []string {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil
	}
	var days []string
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) == 10 {
			days = append(days, entry.Name())
		}
	}
	sort.Strings(days)
	return days
}

func (s *SampleStore) filePath(t time.Time) string {
	return filepath.Join(s.baseDir, t.Format("2006-01-02"), fmt.Sprintf("%02d.jsonl", t.Hour()))
}

func readJSONLFile(fp string, startMs, endMs int64) ([]MonitorSample, error) {
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var samples []MonitorSample
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var s MonitorSample
		if err := json.Unmarshal([]byte(line), &s); err == nil && s.Timestamp >= startMs && s.Timestamp <= endMs {
			samples = append(samples, s)
		}
	}
	return samples, nil
}

func downsample(samples []MonitorSample, stepMs int64) []MonitorSample {
	if len(samples) == 0 {
		return samples
	}
	var result []MonitorSample
	bucketStart := samples[0].Timestamp
	var lastInBucket *MonitorSample
	for i := range samples {
		s := &samples[i]
		if s.Timestamp-bucketStart >= stepMs {
			if lastInBucket != nil {
				result = append(result, *lastInBucket)
			}
			bucketStart = s.Timestamp
		}
		lastInBucket = s
	}
	if lastInBucket != nil {
		result = append(result, *lastInBucket)
	}
	return result
}
