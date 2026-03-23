package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"flashsale/ops/backend/executor"
	"flashsale/ops/backend/model"
	"flashsale/ops/backend/store"
	"flashsale/ops/backend/stream"

	"github.com/google/uuid"
)

type jobRecord struct {
	model.JobDetail
	log            bytes.Buffer
	archivedBytes  int64
	archivedByHour map[string]string
	archivedHours  []string
	perfSamples    []model.PerfProgress
	perfReport     *model.PerfReport
	seq            int64
	mu             sync.Mutex
}

func (r *jobRecord) appendLog(line string) {
	_, _ = r.log.WriteString(line)
	if !strings.HasSuffix(line, "\n") {
		_, _ = r.log.WriteString("\n")
	}
}

func (r *jobRecord) appendLogWithTrim(line string, maxBytes int) []byte {
	r.appendLog(line)
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

func (r *jobRecord) nextSeqLocked() int64 {
	r.seq++
	return r.seq
}

type Scheduler struct {
	repoRoot    string
	tasks       map[string]model.TaskDef
	executor    *executor.Executor
	sampleStore *store.SampleStore
	logRootDir  string
	maxJobs     int
	maxLogSize  int

	mu      sync.RWMutex
	jobs    map[string]*jobRecord
	jobList []string
	subs    map[string]map[string]chan stream.Envelope
}

const (
	defaultMaxJobs    = 200
	defaultMaxLogSize = 512 * 1024
	jobMetaFile       = "meta.json"
)

func New(repoRoot string, tasks map[string]model.TaskDef) *Scheduler {
	logRoot := filepath.Join(repoRoot, "log", "ops-jobs")
	_ = os.MkdirAll(logRoot, 0o755)
	s := &Scheduler{
		repoRoot:    repoRoot,
		tasks:       tasks,
		executor:    executor.New(repoRoot),
		sampleStore: store.NewSampleStore(repoRoot),
		logRootDir:  logRoot,
		maxJobs:     defaultMaxJobs,
		maxLogSize:  defaultMaxLogSize,
		jobs:        make(map[string]*jobRecord),
		jobList:     make([]string, 0, 64),
		subs:        make(map[string]map[string]chan stream.Envelope),
	}
	s.loadPersistedJobs()
	return s
}

func (s *Scheduler) Tasks() []model.TaskDef {
	ids := make([]string, 0, len(s.tasks))
	for id := range s.tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]model.TaskDef, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.tasks[id])
	}
	return out
}

func (s *Scheduler) StartJob(taskID string, extraArgs []string) (model.JobDetail, error) {
	task, ok := s.tasks[taskID]
	if !ok {
		return model.JobDetail{}, fmt.Errorf("unknown task: %s", taskID)
	}
	if err := s.executor.ValidateCommand(task.Command); err != nil {
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

	s.mu.Lock()
	s.jobs[id] = rec
	s.jobList = append(s.jobList, id)
	s.trimJobsLocked()
	s.mu.Unlock()

	s.appendJobLog(id, fmt.Sprintf("[%s] created task=%s", now.Format(time.RFC3339), task.ID), "")
	evCh, err := s.executor.Run(task, rec.Args)
	if err != nil {
		s.failJob(id, -1, err)
		return model.JobDetail{}, err
	}
	go s.consumeExecution(id, evCh)
	return rec.JobDetail, nil
}

func (s *Scheduler) ListJobs(limit int) []model.JobSummary {
	if limit <= 0 {
		limit = 100
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := len(s.jobList)
	if limit > count {
		limit = count
	}
	out := make([]model.JobSummary, 0, limit)
	for i := count - 1; i >= 0 && len(out) < limit; i-- {
		id := s.jobList[i]
		if rec, ok := s.jobs[id]; ok {
			out = append(out, rec.JobSummary)
		}
	}
	return out
}

func (s *Scheduler) GetJob(id string) (model.JobDetail, bool) {
	s.mu.RLock()
	rec, ok := s.jobs[id]
	s.mu.RUnlock()
	if !ok {
		return model.JobDetail{}, false
	}
	return rec.JobDetail, true
}

func (s *Scheduler) GetJobLog(id string) (string, bool) {
	s.mu.RLock()
	rec, ok := s.jobs[id]
	s.mu.RUnlock()
	if !ok {
		return "", false
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return rec.logText(), true
}

func (s *Scheduler) Subscribe(id string) (<-chan stream.Envelope, func(), bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[id]; !ok {
		return nil, nil, false
	}
	if _, ok := s.subs[id]; !ok {
		s.subs[id] = make(map[string]chan stream.Envelope)
	}
	subID := uuid.NewString()
	ch := make(chan stream.Envelope, 256)
	s.subs[id][subID] = ch
	cancel := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if subs, ok := s.subs[id]; ok {
			delete(subs, subID)
			if len(subs) == 0 {
				delete(s.subs, id)
			}
		}
	}
	return ch, cancel, true
}

func (s *Scheduler) SnapshotEnvelope(id string) (stream.Envelope, bool) {
	logText, ok := s.GetJobLog(id)
	if !ok {
		return stream.Envelope{}, false
	}
	return stream.Snapshot(id, logText), true
}

func (s *Scheduler) PingEnvelope(id string) stream.Envelope {
	return stream.Ping(id)
}

func (s *Scheduler) TopFailingJobs(limit int) []model.JobSummary {
	if limit <= 0 {
		limit = 5
	}
	items := s.ListJobs(200)
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

func (s *Scheduler) consumeExecution(jobID string, evCh <-chan executor.Event) {
	for event := range evCh {
		switch event.Type {
		case executor.EventStarted:
			s.onStarted(jobID, event)
		case executor.EventLogLine:
			s.onLogLine(jobID, event)
		case executor.EventFinished:
			s.onFinished(jobID, event)
		case executor.EventFailed:
			s.failJob(jobID, event.ExitCode, event.Err)
		}
	}
}

func (s *Scheduler) onStarted(jobID string, event executor.Event) {
	start := event.Time
	s.updateJob(jobID, func(rec *jobRecord) {
		rec.Status = model.JobRunning
		rec.StartedAt = start
	})
	s.appendJobLog(jobID, fmt.Sprintf("[%s] started", start.Format(time.RFC3339)), "")
	s.appendJobLog(jobID, fmt.Sprintf("[%s] pid=%d", time.Now().Format(time.RFC3339), event.PID), "")
	if job, ok := s.GetJob(jobID); ok {
		s.publish(jobID, stream.State(jobID, s.nextSeq(jobID), time.Now(), job.Status, job.ExitCode, job.StartedAt, job.FinishedAt))
	}
}

func (s *Scheduler) onLogLine(jobID string, event executor.Event) {
	line := event.Line
	if event.Source == "stdout" && len(event.Line) > 0 && event.Line[0] == '{' {
		progress, report := model.ParseSeckillloadLine(event.Line)
		if progress != nil {
			*progress = s.enrichPerfProgress(*progress)
			line = enrichPerfRawLine(event.Line, *progress)
			enrichPerfReportSummary(report, *progress)
			s.updateJob(jobID, func(rec *jobRecord) {
				rec.mu.Lock()
				defer rec.mu.Unlock()
				rec.perfSamples = append(rec.perfSamples, *progress)
				if report != nil {
					rec.perfReport = report
				}
			})
			s.publish(jobID, stream.PerfProgress(jobID, s.nextSeq(jobID), *progress))
		}
	}
	formatted := fmt.Sprintf("[%s][%s] %s", event.Time.Format(time.RFC3339), event.Source, line)
	s.appendJobLog(jobID, formatted, event.Source)
}

func (s *Scheduler) onFinished(jobID string, event executor.Event) {
	end := event.Time
	s.updateJob(jobID, func(rec *jobRecord) {
		s.finalizePerfReport(rec)
		rec.Status = model.JobSuccess
		rec.ExitCode = event.ExitCode
		rec.FinishedAt = end
	})
	s.appendJobLog(jobID, fmt.Sprintf("[%s] finished success exit=%d", end.Format(time.RFC3339), event.ExitCode), "")
	if rec := s.getJobRecord(jobID); rec != nil {
		s.persistJobMeta(rec)
		s.publish(jobID, stream.Done(jobID, s.nextSeq(jobID), end, rec.Status, rec.ExitCode, rec.PerfReport))
		s.closeJobSubs(jobID)
	}
}

func (s *Scheduler) failJob(jobID string, exitCode int, err error) {
	end := time.Now()
	s.updateJob(jobID, func(rec *jobRecord) {
		s.finalizePerfReport(rec)
		rec.Status = model.JobFailed
		rec.ExitCode = exitCode
		rec.FinishedAt = end
	})
	s.appendJobLog(jobID, fmt.Sprintf("[%s] finished failed exit=%d err=%v", end.Format(time.RFC3339), exitCode, err), "")
	if rec := s.getJobRecord(jobID); rec != nil {
		s.persistJobMeta(rec)
		if err != nil {
			s.publish(jobID, stream.Error(jobID, s.nextSeq(jobID), end, err.Error()))
		}
		s.publish(jobID, stream.Done(jobID, s.nextSeq(jobID), end, rec.Status, rec.ExitCode, rec.PerfReport))
		s.closeJobSubs(jobID)
	}
}

func (s *Scheduler) getJobRecord(jobID string) *jobRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jobs[jobID]
}

func (s *Scheduler) updateJob(jobID string, fn func(*jobRecord)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.jobs[jobID]; ok {
		fn(rec)
	}
}

func (s *Scheduler) nextSeq(jobID string) int64 {
	s.mu.RLock()
	rec := s.jobs[jobID]
	s.mu.RUnlock()
	if rec == nil {
		return 0
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return rec.nextSeqLocked()
}

func (s *Scheduler) appendJobLog(jobID string, line, source string) {
	var rec *jobRecord
	var overflow []byte
	now := time.Now()
	s.mu.Lock()
	if item, ok := s.jobs[jobID]; ok {
		rec = item
		rec.mu.Lock()
		overflow = rec.appendLogWithTrim(line, s.maxLogSize)
		rec.mu.Unlock()
	}
	s.mu.Unlock()

	if len(overflow) > 0 && rec != nil {
		hourKey, path, err := s.writeArchivedChunk(jobID, now, overflow)
		if err == nil {
			rec.mu.Lock()
			rec.addArchive(hourKey, path, int64(len(overflow)))
			rec.mu.Unlock()
		}
	}

	if source != "" {
		s.publish(jobID, stream.LogLine(jobID, s.nextSeq(jobID), now, source, line))
	}
}

func (s *Scheduler) publish(jobID string, event stream.Envelope) {
	s.mu.RLock()
	subs := s.subs[jobID]
	outputs := make([]chan stream.Envelope, 0, len(subs))
	for _, ch := range subs {
		outputs = append(outputs, ch)
	}
	s.mu.RUnlock()
	for _, ch := range outputs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *Scheduler) closeJobSubs(jobID string) {
	s.mu.Lock()
	subs := s.subs[jobID]
	delete(s.subs, jobID)
	s.mu.Unlock()
	for _, ch := range subs {
		close(ch)
	}
}

func (s *Scheduler) finalizePerfReport(rec *jobRecord) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.perfSamples) == 0 {
		return
	}
	rec.perfSamples = s.enrichPerfProgressSeries(rec.perfSamples)
	if rec.perfReport == nil {
		rec.perfReport = &model.PerfReport{}
	}
	rec.perfReport.Samples = make([]model.PerfProgress, len(rec.perfSamples))
	copy(rec.perfReport.Samples, rec.perfSamples)
	rec.PerfReport = rec.perfReport
}

func (s *Scheduler) trimJobsLocked() {
	if s.maxJobs <= 0 {
		return
	}
	for len(s.jobList) > s.maxJobs {
		oldestID := s.jobList[0]
		rec, ok := s.jobs[oldestID]
		if !ok {
			s.jobList = s.jobList[1:]
			continue
		}
		if rec.Status == model.JobQueued || rec.Status == model.JobRunning {
			break
		}
		s.jobList = s.jobList[1:]
		delete(s.jobs, oldestID)
		delete(s.subs, oldestID)
	}
}

func (s *Scheduler) writeArchivedChunk(jobID string, ts time.Time, chunk []byte) (hourKey string, path string, err error) {
	hourKey = ts.Format("2006010215")
	dir := filepath.Join(s.logRootDir, jobID)
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

func (s *Scheduler) persistJobMeta(rec *jobRecord) {
	dir := filepath.Join(s.logRootDir, rec.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	data, err := json.Marshal(rec.JobDetail)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, jobMetaFile), data, 0o644)
	rec.mu.Lock()
	remaining := rec.log.Bytes()
	rec.mu.Unlock()
	if len(remaining) > 0 {
		_ = os.WriteFile(filepath.Join(dir, "final.log"), remaining, 0o644)
	}
}

func (s *Scheduler) loadPersistedJobs() {
	entries, err := os.ReadDir(s.logRootDir)
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
		metaPath := filepath.Join(s.logRootDir, entry.Name(), jobMetaFile)
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

	sort.Slice(loaded, func(i, j int) bool {
		return loaded[i].detail.CreatedAt.Before(loaded[j].detail.CreatedAt)
	})

	restored := 0
	for _, item := range loaded {
		id := item.detail.ID
		if _, exists := s.jobs[id]; exists {
			continue
		}
		rec := &jobRecord{JobDetail: item.detail}
		rec.perfReport = item.detail.PerfReport
		logDir := filepath.Join(s.logRootDir, id)
		logFiles, _ := filepath.Glob(filepath.Join(logDir, "*.log"))
		sort.Strings(logFiles)
		for _, lf := range logFiles {
			base := filepath.Base(lf)
			if base == "final.log" {
				data, err := os.ReadFile(lf)
				if err == nil {
					rec.log.Write(data)
				}
				continue
			}
			info, err := os.Stat(lf)
			if err != nil {
				continue
			}
			rec.addArchive(strings.TrimSuffix(base, ".log"), lf, info.Size())
		}
		s.jobs[id] = rec
		s.jobList = append(s.jobList, id)
		restored++
	}

	if restored > 0 {
		log.Printf("[ops-scheduler] restored %d persisted jobs from %s", restored, s.logRootDir)
	}
}
