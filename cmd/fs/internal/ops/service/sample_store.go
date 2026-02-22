package service

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

// ─── 采样数据模型 ───

// MonitorSample 持久化的监测采样点。
type MonitorSample struct {
	Timestamp int64  `json:"ts"`
	Label     string `json:"label"`
	// 基础设施健康
	PortRate          float64 `json:"portRate"`
	HttpRate          float64 `json:"httpRate"`
	ReplicaRate       float64 `json:"replicaRate"`
	RunningContainers int     `json:"runningContainers"`
	TotalContainers   int     `json:"totalContainers"`
	RunningReplicas   int     `json:"runningReplicas"`
	TotalReplicas     int     `json:"totalReplicas"`
	// Prometheus 服务端视角
	PromQps          *float64 `json:"promQps,omitempty"`
	PromP99LatencyMs *float64 `json:"promP99LatencyMs,omitempty"`
	PromErrorRate    *float64 `json:"promErrorRate,omitempty"`
	// 异步购买队列
	PurchaseTaskQueueDepth *float64 `json:"purchaseTaskQueueDepth,omitempty"`
	PurchaseTaskQueueCap   *float64 `json:"purchaseTaskQueueCap,omitempty"`
	PurchaseTaskDropped    *float64 `json:"purchaseTaskDropped,omitempty"`
}

// ─── SampleStore ───

// SampleStore 管理监测采样数据的持久化存储。
// 数据按 小时 存为 JSONL 文件，按 天 组织目录。
// 路径: {repoRoot}/log/monitor/YYYY-MM-DD/HH.jsonl
type SampleStore struct {
	baseDir    string // log/monitor
	mu         sync.Mutex
	curFile    *os.File
	curPath    string
	retainDays int
}

const sampleStoreSubDir = "log/monitor"

// NewSampleStore 创建采样存储实例。
func NewSampleStore(repoRoot string) *SampleStore {
	return &SampleStore{
		baseDir:    filepath.Join(repoRoot, sampleStoreSubDir),
		retainDays: 7,
	}
}

// Append 追加一条采样记录。
func (s *SampleStore) Append(sample *MonitorSample) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.UnixMilli(sample.Timestamp)
	targetPath := s.filePath(now)

	// 需要轮转文件?
	if s.curPath != targetPath {
		if s.curFile != nil {
			_ = s.curFile.Close()
			s.curFile = nil
		}
		dir := filepath.Dir(targetPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
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
	data = append(data, '\n')
	_, err = s.curFile.Write(data)
	return err
}

// Query 查询指定时间范围内的采样数据。
// stepMs 指定降采样步长（毫秒），0 = 原始粒度。
func (s *SampleStore) Query(startMs, endMs int64, stepMs int64) ([]MonitorSample, error) {
	startTime := time.UnixMilli(startMs)
	endTime := time.UnixMilli(endMs)

	// 收集需要读取的文件列表
	var files []string
	cur := startTime.Truncate(time.Hour)
	for !cur.After(endTime) {
		fp := s.filePath(cur)
		if _, err := os.Stat(fp); err == nil {
			files = append(files, fp)
		}
		cur = cur.Add(time.Hour)
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

	// 按时间排序
	sort.Slice(allSamples, func(i, j int) bool {
		return allSamples[i].Timestamp < allSamples[j].Timestamp
	})

	// 降采样
	if stepMs > 0 && len(allSamples) > 0 {
		allSamples = downsample(allSamples, stepMs)
	}

	return allSamples, nil
}

// Cleanup 清理过期数据（保留 retainDays 天）。
func (s *SampleStore) Cleanup() {
	cutoff := time.Now().AddDate(0, 0, -s.retainDays).Format("2006-01-02")
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 目录名格式: YYYY-MM-DD
		if len(name) == 10 && name < cutoff {
			dirPath := filepath.Join(s.baseDir, name)
			if err := os.RemoveAll(dirPath); err != nil {
				log.Printf("[sample_store] cleanup %s failed: %v", dirPath, err)
			} else {
				log.Printf("[sample_store] cleaned up expired directory: %s", name)
			}
		}
	}
}

// Close 关闭当前文件。
func (s *SampleStore) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.curFile != nil {
		_ = s.curFile.Close()
		s.curFile = nil
	}
}

// ListAvailableDays 返回有数据的日期列表。
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

// ─── 内部函数 ───

func (s *SampleStore) filePath(t time.Time) string {
	dayDir := t.Format("2006-01-02")
	hourFile := fmt.Sprintf("%02d.jsonl", t.Hour())
	return filepath.Join(s.baseDir, dayDir, hourFile)
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
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			continue
		}
		if s.Timestamp >= startMs && s.Timestamp <= endMs {
			samples = append(samples, s)
		}
	}
	return samples, nil
}

// downsample 按固定步长降采样，每个桶取最后一个点。
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
