// runner 负责任务白名单执行、状态维护与日志发布。
package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"flashsale/cmd/fs/internal/logdir"
	"flashsale/cmd/fs/internal/ops/model"

	"github.com/google/uuid"
)

type jobRecord struct {
	model.JobDetail
	log            bytes.Buffer
	archivedBytes  int64
	archivedByHour map[string]string
	archivedHours  []string
	mu             sync.Mutex
}

func (r *jobRecord) appendLog(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = r.log.WriteString(line)
	if !strings.HasSuffix(line, "\n") {
		_, _ = r.log.WriteString("\n")
	}
}

func (r *jobRecord) appendLogWithTrim(line string, maxBytes int) []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = r.log.WriteString(line)
	if !strings.HasSuffix(line, "\n") {
		_, _ = r.log.WriteString("\n")
	}
	if maxBytes <= 0 || r.log.Len() <= maxBytes {
		return nil
	}
	overflowSize := r.log.Len() - maxBytes
	overflow := append([]byte(nil), r.log.Bytes()[:overflowSize]...)
	remain := append([]byte(nil), r.log.Bytes()[overflowSize:]...)
	r.log.Reset()
	_, _ = r.log.Write(remain)
	return overflow
}

func (r *jobRecord) addArchive(hour, path string, bytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.archivedBytes += bytes
	if r.archivedByHour == nil {
		r.archivedByHour = make(map[string]string)
	}
	if _, exists := r.archivedByHour[hour]; !exists {
		r.archivedByHour[hour] = path
		r.archivedHours = append(r.archivedHours, hour)
		sort.Strings(r.archivedHours)
	}
}

func (r *jobRecord) logText() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.archivedBytes <= 0 {
		return r.log.String()
	}
	latest := ""
	if n := len(r.archivedHours); n > 0 {
		latest = r.archivedByHour[r.archivedHours[n-1]]
	}
	prefix := fmt.Sprintf("[log-archive] earlier logs archived by hour: bytes=%d", r.archivedBytes)
	if latest != "" {
		prefix += ", latest_file=" + latest
	}
	return prefix + "\n" + r.log.String()
}

// Runner 负责执行白名单任务。
type Runner struct {
	RepoRoot   string
	tasks      map[string]model.TaskDef
	selfBinary string
	logRootDir string
	maxJobs    int
	maxLogSize int

	mu      sync.RWMutex
	jobs    map[string]*jobRecord
	jobList []string
	subs    map[string]map[string]chan string
}

const (
	defaultMaxJobs    = 200
	defaultMaxLogSize = 512 * 1024
)

// NewRunner 创建任务执行器。
func NewRunner(repoRoot string, tasks map[string]model.TaskDef) *Runner {
	selfBinary, _ := os.Executable()
	logRoot := logdir.OpsJobsDir(repoRoot)
	_ = os.MkdirAll(logRoot, 0o755)
	r := &Runner{
		RepoRoot:   repoRoot,
		tasks:      tasks,
		selfBinary: selfBinary,
		logRootDir: logRoot,
		maxJobs:    defaultMaxJobs,
		maxLogSize: defaultMaxLogSize,
		jobs:       make(map[string]*jobRecord),
		jobList:    make([]string, 0, 64),
		subs:       make(map[string]map[string]chan string),
	}
	r.loadPersistedJobs()
	return r
}

// Tasks 返回排序后的任务列表。
func (r *Runner) Tasks() []model.TaskDef {
	return SortedTasks(r.tasks)
}

// StartJob 创建并异步执行任务。
func (r *Runner) StartJob(taskID string, extraArgs []string) (model.JobDetail, error) {
	task, ok := r.tasks[taskID]
	if !ok {
		return model.JobDetail{}, fmt.Errorf("unknown task: %s", taskID)
	}
	if err := r.validateCommand(task.Command); err != nil {
		return model.JobDetail{}, err
	}

	id := uuid.NewString()
	now := time.Now()
	args := append([]string{}, task.DefaultArgs...)
	args = append(args, extraArgs...)
	rec := &jobRecord{
		JobDetail: model.JobDetail{
			JobSummary: model.JobSummary{
				ID:        id,
				TaskID:    task.ID,
				Status:    model.JobQueued,
				CreatedAt: now,
				ExitCode:  -1,
			},
			TaskName: task.Name,
			Args:     args,
		},
	}

	r.mu.Lock()
	r.jobs[id] = rec
	r.jobList = append(r.jobList, id)
	r.trimJobsLocked()
	r.mu.Unlock()
	r.appendJobLog(id, fmt.Sprintf("[%s] created task=%s", now.Format(time.RFC3339), task.ID))

	go r.execute(rec, task)
	return rec.JobDetail, nil
}

// ListJobs 返回任务列表（按创建时间倒序）。
func (r *Runner) ListJobs(limit int) []model.JobSummary {
	if limit <= 0 {
		limit = 100
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := len(r.jobList)
	if limit > count {
		limit = count
	}
	out := make([]model.JobSummary, 0, limit)
	for i := count - 1; i >= 0 && len(out) < limit; i-- {
		id := r.jobList[i]
		if rec, ok := r.jobs[id]; ok {
			out = append(out, rec.JobSummary)
		}
	}
	return out
}

// GetJob 获取任务详情。
func (r *Runner) GetJob(id string) (model.JobDetail, bool) {
	r.mu.RLock()
	rec, ok := r.jobs[id]
	r.mu.RUnlock()
	if !ok {
		return model.JobDetail{}, false
	}
	return rec.JobDetail, true
}

// GetJobLog 获取任务日志文本。
func (r *Runner) GetJobLog(id string) (string, bool) {
	r.mu.RLock()
	rec, ok := r.jobs[id]
	r.mu.RUnlock()
	if !ok {
		return "", false
	}
	return rec.logText(), true
}

// SubscribeJobLog 订阅任务日志增量。
func (r *Runner) SubscribeJobLog(id string) (<-chan string, func(), bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.jobs[id]; !ok {
		return nil, nil, false
	}
	if _, ok := r.subs[id]; !ok {
		r.subs[id] = make(map[string]chan string)
	}
	subID := uuid.NewString()
	ch := make(chan string, 256)
	r.subs[id][subID] = ch
	cancel := func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if subs, ok := r.subs[id]; ok {
			delete(subs, subID)
			if len(subs) == 0 {
				delete(r.subs, id)
			}
		}
	}
	return ch, cancel, true
}

func (r *Runner) validateCommand(command []string) error {
	if len(command) == 0 {
		return errors.New("empty command")
	}
	if command[0] == "self" {
		if strings.TrimSpace(r.selfBinary) == "" {
			return errors.New("self binary not found")
		}
		return nil
	}
	if _, err := exec.LookPath(command[0]); err != nil {
		return fmt.Errorf("command not found: %s", command[0])
	}
	return nil
}

func (r *Runner) execute(rec *jobRecord, task model.TaskDef) {
	start := time.Now()
	r.updateJob(rec.ID, func(j *jobRecord) {
		j.Status = model.JobRunning
		j.StartedAt = start
	})
	r.appendJobLog(rec.ID, fmt.Sprintf("[%s] started", start.Format(time.RFC3339)))

	cmd, err := r.newTaskCommand(task, rec.Args)
	if err != nil {
		r.failJob(rec.ID, -1, err)
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.failJob(rec.ID, -1, err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		r.failJob(rec.ID, -1, err)
		return
	}

	if err := cmd.Start(); err != nil {
		r.failJob(rec.ID, -1, err)
		return
	}
	r.appendJobLog(rec.ID, fmt.Sprintf("[%s] pid=%d", time.Now().Format(time.RFC3339), cmd.Process.Pid))

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		r.pipeToLog(rec.ID, stdout, "stdout")
	}()
	go func() {
		defer wg.Done()
		r.pipeToLog(rec.ID, stderr, "stderr")
	}()

	waitErr := cmd.Wait()
	wg.Wait()
	exitCode := cmd.ProcessState.ExitCode()
	if waitErr != nil {
		r.failJob(rec.ID, exitCode, waitErr)
		return
	}

	end := time.Now()
	r.updateJob(rec.ID, func(j *jobRecord) {
		j.Status = model.JobSuccess
		j.ExitCode = exitCode
		j.FinishedAt = end
	})
	r.appendJobLog(rec.ID, fmt.Sprintf("[%s] finished success exit=%d", end.Format(time.RFC3339), exitCode))
	r.persistJobMeta(rec)
	r.closeJobSubs(rec.ID)
}

func (r *Runner) pipeToLog(jobID string, stream io.Reader, source string) {
	scanner := bufio.NewScanner(stream)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		r.appendJobLog(jobID, fmt.Sprintf("[%s][%s] %s", time.Now().Format(time.RFC3339), source, line))
	}
	if err := scanner.Err(); err != nil {
		r.appendJobLog(jobID, fmt.Sprintf("[%s][%s] scanner error: %v", time.Now().Format(time.RFC3339), source, err))
	}
}

func (r *Runner) failJob(jobID string, exitCode int, err error) {
	end := time.Now()
	r.updateJob(jobID, func(j *jobRecord) {
		j.Status = model.JobFailed
		j.ExitCode = exitCode
		j.FinishedAt = end
	})
	r.appendJobLog(jobID, fmt.Sprintf("[%s] finished failed exit=%d err=%v", end.Format(time.RFC3339), exitCode, err))

	r.mu.RLock()
	rec := r.jobs[jobID]
	r.mu.RUnlock()
	if rec != nil {
		r.persistJobMeta(rec)
	}
	r.closeJobSubs(jobID)
}

// closeJobSubs 关闭 job 的所有 SSE subscriber channel，立即通知 SSE handler。
func (r *Runner) closeJobSubs(jobID string) {
	r.mu.Lock()
	subs := r.subs[jobID]
	delete(r.subs, jobID)
	r.mu.Unlock()
	for _, ch := range subs {
		close(ch)
	}
}

func (r *Runner) updateJob(jobID string, fn func(*jobRecord)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, ok := r.jobs[jobID]; ok {
		fn(rec)
	}
}

func (r *Runner) appendJobLog(jobID string, line string) {
	var outputs []chan string
	var rec *jobRecord
	var overflow []byte
	now := time.Now()
	r.mu.Lock()
	if item, ok := r.jobs[jobID]; ok {
		rec = item
		overflow = rec.appendLogWithTrim(line, r.maxLogSize)
		if subs, ok := r.subs[jobID]; ok {
			outputs = make([]chan string, 0, len(subs))
			for _, ch := range subs {
				outputs = append(outputs, ch)
			}
		}
	}
	r.mu.Unlock()

	if len(overflow) > 0 && rec != nil {
		hourKey, path, err := r.writeArchivedChunk(jobID, now, overflow)
		if err == nil {
			rec.addArchive(hourKey, path, int64(len(overflow)))
		}
	}

	for _, ch := range outputs {
		select {
		case ch <- line:
		default:
		}
	}
}

func (r *Runner) trimJobsLocked() {
	if r.maxJobs <= 0 {
		return
	}
	for len(r.jobList) > r.maxJobs {
		oldestID := r.jobList[0]
		rec, ok := r.jobs[oldestID]
		if !ok {
			r.jobList = r.jobList[1:]
			continue
		}
		if rec.Status == model.JobQueued || rec.Status == model.JobRunning {
			break
		}
		r.jobList = r.jobList[1:]
		delete(r.jobs, oldestID)
		if subs, ok := r.subs[oldestID]; ok {
			_ = subs
			delete(r.subs, oldestID)
		}
	}
}

func (r *Runner) writeArchivedChunk(jobID string, ts time.Time, chunk []byte) (hourKey string, path string, err error) {
	hourKey = ts.Format("2006010215")
	dir := filepath.Join(r.logRootDir, jobID)
	if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
		return "", "", mkErr
	}
	path = filepath.Join(dir, hourKey+".log")
	file, openErr := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if openErr != nil {
		return "", "", openErr
	}
	defer file.Close()
	_, writeErr := file.Write(chunk)
	if writeErr != nil {
		return "", "", writeErr
	}
	return hourKey, path, nil
}

func (r *Runner) newTaskCommand(task model.TaskDef, extraArgs []string) (*exec.Cmd, error) {
	if len(task.Command) == 0 {
		return nil, errors.New("task command is empty")
	}
	command := task.Command[0]
	args := append([]string{}, task.Command[1:]...)
	args = append(args, extraArgs...)

	if command == "self" {
		command = r.selfBinary
	}

	cmd := exec.Command(command, args...)
	cmd.Dir = r.RepoRoot
	return cmd, nil
}

// TopFailingJobs 返回最近失败任务，便于首页展示。
func (r *Runner) TopFailingJobs(limit int) []model.JobSummary {
	if limit <= 0 {
		limit = 5
	}
	items := r.ListJobs(200)
	fails := make([]model.JobSummary, 0, limit)
	for _, item := range items {
		if item.Status == model.JobFailed {
			fails = append(fails, item)
		}
		if len(fails) >= limit {
			break
		}
	}
	sort.Slice(fails, func(i, j int) bool {
		return fails[i].CreatedAt.After(fails[j].CreatedAt)
	})
	return fails
}

// ─── Job 持久化 ───

const jobMetaFile = "meta.json"

// persistJobMeta 将已完成的 job 元数据和剩余日志写入磁盘。
func (r *Runner) persistJobMeta(rec *jobRecord) {
	dir := filepath.Join(r.logRootDir, rec.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	data, err := json.Marshal(rec.JobDetail)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, jobMetaFile), data, 0o644)

	// flush 内存中剩余的日志到 final.log
	rec.mu.Lock()
	remaining := rec.log.Bytes()
	rec.mu.Unlock()
	if len(remaining) > 0 {
		finalPath := filepath.Join(dir, "final.log")
		_ = os.WriteFile(finalPath, remaining, 0o644)
	}
}

// loadPersistedJobs 在启动时扫描磁盘，恢复已完成的历史 job。
func (r *Runner) loadPersistedJobs() {
	entries, err := os.ReadDir(r.logRootDir)
	if err != nil {
		return
	}

	type loadedJob struct {
		detail model.JobDetail
	}

	var loaded []loadedJob
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(r.logRootDir, entry.Name(), jobMetaFile)
		raw, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var detail model.JobDetail
		if err := json.Unmarshal(raw, &detail); err != nil {
			continue
		}
		loaded = append(loaded, loadedJob{detail: detail})
	}

	// 按 CreatedAt 排序
	sort.Slice(loaded, func(i, j int) bool {
		return loaded[i].detail.CreatedAt.Before(loaded[j].detail.CreatedAt)
	})

	restored := 0
	for _, item := range loaded {
		id := item.detail.ID
		if _, exists := r.jobs[id]; exists {
			continue
		}
		// 加载日志：hourly 归档 + final.log（内存 buffer）
		rec := &jobRecord{JobDetail: item.detail}
		logDir := filepath.Join(r.logRootDir, id)
		logFiles, _ := filepath.Glob(filepath.Join(logDir, "*.log"))
		sort.Strings(logFiles)
		for _, lf := range logFiles {
			base := filepath.Base(lf)
			if base == "final.log" {
				// final.log → 加载到内存 buffer
				data, err := os.ReadFile(lf)
				if err == nil {
					rec.log.Write(data)
				}
				continue
			}
			// hourly 归档 → addArchive
			info, err := os.Stat(lf)
			if err != nil {
				continue
			}
			hour := strings.TrimSuffix(base, ".log")
			rec.addArchive(hour, lf, info.Size())
		}
		r.jobs[id] = rec
		r.jobList = append(r.jobList, id)
		restored++
	}

	if restored > 0 {
		log.Printf("[ops-runner] restored %d persisted jobs from %s", restored, r.logRootDir)
	}
}
