package stream

import (
	"time"

	"flashsale/ops/backend/model"
)

func Snapshot(jobID string, logText string) Envelope {
	return Envelope{
		Type:  EventSnapshot,
		JobID: jobID,
		TS:    time.Now().UnixMilli(),
		Seq:   0,
		Payload: SnapshotPayload{
			Log: logText,
		},
	}
}

func LogLine(jobID string, seq int64, ts time.Time, source, line string) Envelope {
	return Envelope{
		Type:  EventLogLine,
		JobID: jobID,
		TS:    ts.UnixMilli(),
		Seq:   seq,
		Payload: LogLinePayload{
			Source: source,
			Line:   line,
		},
	}
}

func State(jobID string, seq int64, ts time.Time, status model.JobStatus, exitCode int, startedAt, finishedAt time.Time) Envelope {
	payload := JobStatePayload{
		Status:   status,
		ExitCode: exitCode,
	}
	if !startedAt.IsZero() {
		payload.StartedAt = &startedAt
	}
	if !finishedAt.IsZero() {
		payload.FinishedAt = &finishedAt
	}
	return Envelope{
		Type:    EventJobState,
		JobID:   jobID,
		TS:      ts.UnixMilli(),
		Seq:     seq,
		Payload: payload,
	}
}

func PerfProgress(jobID string, seq int64, sample model.PerfProgress) Envelope {
	return Envelope{
		Type:    EventPerfProgress,
		JobID:   jobID,
		TS:      sample.Timestamp,
		Seq:     seq,
		Payload: sample,
	}
}

func Done(jobID string, seq int64, ts time.Time, status model.JobStatus, exitCode int, report *model.PerfReport) Envelope {
	payload := DonePayload{
		Status:     status,
		ExitCode:   exitCode,
		PerfReport: report,
	}
	if !ts.IsZero() {
		payload.FinishedAt = &ts
	}
	return Envelope{
		Type:    EventDone,
		JobID:   jobID,
		TS:      ts.UnixMilli(),
		Seq:     seq,
		Payload: payload,
	}
}

func Error(jobID string, seq int64, ts time.Time, message string) Envelope {
	return Envelope{
		Type:  EventError,
		JobID: jobID,
		TS:    ts.UnixMilli(),
		Seq:   seq,
		Payload: ErrorPayload{
			Message: message,
		},
	}
}

func Ping(jobID string) Envelope {
	now := time.Now()
	return Envelope{
		Type:  EventPing,
		JobID: jobID,
		TS:    now.UnixMilli(),
		Seq:   0,
		Payload: PingPayload{
			TS: now.UnixMilli(),
		},
	}
}
