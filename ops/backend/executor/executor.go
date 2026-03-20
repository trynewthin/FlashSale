package executor

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"flashsale/ops/backend/model"
	"flashsale/ops/backend/taskrun"
)

const (
	EventStarted  = "started"
	EventLogLine  = "log_line"
	EventFinished = "finished"
	EventFailed   = "failed"
)

type Event struct {
	Type     string
	Time     time.Time
	Source   string
	Line     string
	PID      int
	ExitCode int
	Err      error
}

type Executor struct {
	repoRoot   string
	selfBinary string
}

var internalExecMu sync.Mutex

func New(repoRoot string) *Executor {
	selfBinary, _ := os.Executable()
	return &Executor{repoRoot: repoRoot, selfBinary: selfBinary}
}

func (e *Executor) ValidateCommand(command []string) error {
	if len(command) == 0 {
		return errors.New("empty command")
	}
	switch command[0] {
	case "self":
		if strings.TrimSpace(e.selfBinary) == "" {
			return errors.New("self binary not found")
		}
		return nil
	case "internal":
		if len(command) < 2 || strings.TrimSpace(command[1]) == "" {
			return errors.New("internal task id is required")
		}
		if !taskrun.CanRunTask(command[1]) {
			return fmt.Errorf("unsupported internal task: %s", command[1])
		}
		return nil
	default:
		if _, err := exec.LookPath(command[0]); err != nil {
			return fmt.Errorf("command not found: %s", command[0])
		}
		return nil
	}
}

func (e *Executor) Run(task model.TaskDef, args []string) (<-chan Event, error) {
	if err := e.ValidateCommand(task.Command); err != nil {
		return nil, err
	}
	if len(task.Command) > 0 && task.Command[0] == "internal" {
		return e.runInternal(task, args), nil
	}

	cmd, err := e.newTaskCommand(task, args)
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	ch := make(chan Event, 256)
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		defer close(ch)
		ch <- Event{Type: EventStarted, Time: time.Now(), PID: cmd.Process.Pid}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			pipeStream(ch, stdout, "stdout")
		}()
		go func() {
			defer wg.Done()
			pipeStream(ch, stderr, "stderr")
		}()

		waitErr := cmd.Wait()
		wg.Wait()
		exitCode := -1
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		if waitErr != nil {
			ch <- Event{Type: EventFailed, Time: time.Now(), ExitCode: exitCode, Err: waitErr}
			return
		}
		ch <- Event{Type: EventFinished, Time: time.Now(), ExitCode: exitCode}
	}()

	return ch, nil
}

func (e *Executor) runInternal(task model.TaskDef, args []string) <-chan Event {
	ch := make(chan Event, 256)
	taskID := task.Command[1]

	go func() {
		defer close(ch)
		ch <- Event{Type: EventStarted, Time: time.Now(), PID: os.Getpid()}

		internalExecMu.Lock()
		defer internalExecMu.Unlock()

		stdoutReader, stdoutWriter, _ := os.Pipe()
		stderrReader, stderrWriter, _ := os.Pipe()
		originalStdout := os.Stdout
		originalStderr := os.Stderr
		os.Stdout = stdoutWriter
		os.Stderr = stderrWriter

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			pipeStream(ch, stdoutReader, "stdout")
		}()
		go func() {
			defer wg.Done()
			pipeStream(ch, stderrReader, "stderr")
		}()

		runErr := taskrun.RunTask(e.repoRoot, taskID, args)

		_ = stdoutWriter.Close()
		_ = stderrWriter.Close()
		os.Stdout = originalStdout
		os.Stderr = originalStderr
		wg.Wait()

		if runErr != nil {
			ch <- Event{Type: EventFailed, Time: time.Now(), ExitCode: 1, Err: runErr}
			return
		}
		ch <- Event{Type: EventFinished, Time: time.Now(), ExitCode: 0}
	}()

	return ch
}

func (e *Executor) newTaskCommand(task model.TaskDef, extraArgs []string) (*exec.Cmd, error) {
	if len(task.Command) == 0 {
		return nil, errors.New("task command is empty")
	}
	command := task.Command[0]
	args := append([]string{}, task.Command[1:]...)
	args = append(args, extraArgs...)
	if command == "self" {
		command = e.selfBinary
	}
	cmd := exec.Command(command, args...)
	cmd.Dir = e.repoRoot
	return cmd, nil
}

func pipeStream(ch chan<- Event, stream io.Reader, source string) {
	scanner := bufio.NewScanner(stream)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)
	for scanner.Scan() {
		ch <- Event{Type: EventLogLine, Time: time.Now(), Source: source, Line: scanner.Text()}
	}
	if err := scanner.Err(); err != nil {
		ch <- Event{Type: EventLogLine, Time: time.Now(), Source: source, Line: fmt.Sprintf("scanner error: %v", err)}
	}
}
