package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ─── 场景路由 ───

func runScenario(client *http.Client, cfg runConfig, tokens []string) []requestResult {
	emitProgress := cfg.Output == outputJSON
	if cfg.isOpenModel() {
		return runOpenScenario(client, cfg, tokens, emitProgress)
	}
	return runClosedScenario(client, cfg, tokens, emitProgress)
}

// ─── 闭环场景 ───

func runClosedScenario(client *http.Client, cfg runConfig, tokens []string, emitProgress bool) []requestResult {
	results := make([]requestResult, 0, cfg.Requests)
	jobs := make(chan int, cfg.Requests)
	out := make(chan requestResult, cfg.Requests)
	collectDone := make(chan struct{})
	startedAt := time.Now()
	var collectMu sync.Mutex

	go func() {
		for r := range out {
			collectMu.Lock()
			results = append(results, r)
			collectMu.Unlock()
		}
		close(collectDone)
	}()

	progressStop := make(chan struct{})
	progressDone := make(chan struct{})
	if emitProgress {
		progressTicker := time.NewTicker(1 * time.Second)
		go func() {
			defer close(progressDone)
			defer progressTicker.Stop()
			for {
				select {
				case <-progressStop:
					return
				case <-progressTicker.C:
					collectMu.Lock()
					snapshot := append([]requestResult(nil), results...)
					collectMu.Unlock()
					if len(snapshot) == 0 {
						continue
					}
					printProgressJSON(cfg, summarize(snapshot, time.Since(startedAt)))
				}
			}
		}()
	} else {
		close(progressDone)
	}

	var wg sync.WaitGroup
	workerCount := cfg.Concurrency
	if workerCount > cfg.Requests {
		workerCount = cfg.Requests
	}
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for idx := range jobs {
				out <- executeOne(client, cfg, tokens, workerID, idx)
			}
		}(i)
	}

	for i := 0; i < cfg.Requests; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	close(out)
	<-collectDone
	close(progressStop)
	<-progressDone
	return results
}

// ─── 开环场景 ───

// errOpenModelDropped 表示开环发压时因本地并发槽满被主动丢弃。
var errOpenModelDropped = errors.New("open model request dropped by local inflight limiter")

// runOpenScenario 以固定速率投递请求，模拟开环到达率模型。
func runOpenScenario(client *http.Client, cfg runConfig, tokens []string, emitProgress bool) []requestResult {
	capHint := cfg.Concurrency * 4
	if capHint < 128 {
		capHint = 128
	}
	if capHint > 8192 {
		capHint = 8192
	}

	results := make([]requestResult, 0, capHint)
	out := make(chan requestResult, capHint)
	jobs := make(chan int, cfg.Concurrency)
	collectDone := make(chan struct{})
	startedAt := time.Now()
	var collectMu sync.Mutex
	go func() {
		for r := range out {
			collectMu.Lock()
			results = append(results, r)
			collectMu.Unlock()
		}
		close(collectDone)
	}()
	progressStop := make(chan struct{})
	progressDone := make(chan struct{})
	if emitProgress {
		progressTicker := time.NewTicker(1 * time.Second)
		go func() {
			defer close(progressDone)
			defer progressTicker.Stop()
			for {
				select {
				case <-progressStop:
					return
				case <-progressTicker.C:
					collectMu.Lock()
					snapshot := append([]requestResult(nil), results...)
					collectMu.Unlock()
					if len(snapshot) == 0 {
						continue
					}
					printProgressJSON(cfg, summarize(snapshot, time.Since(startedAt)))
				}
			}
		}()
	} else {
		close(progressDone)
	}

	workerCount := cfg.Concurrency
	if workerCount < 1 {
		workerCount = 1
	}
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for idx := range jobs {
				out <- executeOne(client, cfg, tokens, workerID, idx)
			}
		}(i)
	}

	ticker := time.NewTicker(time.Second / time.Duration(cfg.OpenRate))
	defer ticker.Stop()
	timer := time.NewTimer(cfg.OpenDuration)
	defer timer.Stop()

	sent := 0
loop:
	for {
		select {
		case <-timer.C:
			break loop
		case <-ticker.C:
			select {
			case jobs <- sent:
				sent++
			default:
				sent++
				out <- requestResult{Err: errOpenModelDropped}
			}
		}
	}
	close(jobs)
	wg.Wait()
	close(out)
	<-collectDone
	close(progressStop)
	<-progressDone
	return results
}

// ─── 请求执行 ───

func executeOne(client *http.Client, cfg runConfig, tokens []string, workerID, idx int) requestResult {
	start := time.Now()
	var (
		status        int
		code          string
		order         string
		trackAccepted bool
		trackKnown    bool
		err           error
	)

	switch cfg.Scenario {
	case scenarioPurchaseStress, scenarioPurchaseOpen:
		idem := fmt.Sprintf("perf-purchase-%d-%d", time.Now().UnixNano(), idx)
		token := tokens[idx%len(tokens)]
		status, code, order, err = doPurchase(client, cfg, token, idem)
	case scenarioIdempotency:
		idem := fmt.Sprintf("perf-idem-%s", strings.TrimSpace(cfg.IdempotencyGroup))
		// 幂等场景必须固定同一用户，否则会被"多用户多订单"误判为幂等失效。
		token := tokens[0]
		status, code, order, err = doPurchase(client, cfg, token, idem)
	case scenarioTrackStress, scenarioTrackOpen:
		idem := fmt.Sprintf("perf-track-%d-%d", time.Now().UnixNano(), idx)
		clientID := fmt.Sprintf("%s-%d-%d", strings.TrimSpace(cfg.ClientIDPrefix), workerID, idx)
		status, code, trackAccepted, err = doTrack(client, cfg, idem, clientID)
		trackKnown = true
	}
	return requestResult{
		Latency:           time.Since(start),
		HTTPStatus:        status,
		Code:              code,
		OrderNo:           order,
		TrackAccepted:     trackAccepted,
		TrackAcceptedKnow: trackKnown,
		Err:               err,
	}
}

// ─── HTTP 请求 ───

func doPurchase(client *http.Client, cfg runConfig, token, idempotencyKey string) (int, string, string, error) {
	path := fmt.Sprintf("%s/api/v1/seckill/activities/%d/purchase", strings.TrimRight(cfg.BaseURL, "/"), cfg.ActivityID)
	body := map[string]any{
		"activity_item_id": cfg.ActivityItemID,
		"quantity":         cfg.Quantity,
		"idempotency_key":  idempotencyKey,
	}
	statusCode, code, data, err := doJSON(client, http.MethodPost, path, token, body)
	if err != nil {
		return statusCode, code, "", err
	}
	orderNo := ""
	if len(data) > 0 {
		var order struct {
			OrderNo string `json:"order_no"`
		}
		if err := json.Unmarshal(data, &order); err == nil {
			orderNo = strings.TrimSpace(order.OrderNo)
		}
	}
	return statusCode, code, orderNo, nil
}

func doTrack(client *http.Client, cfg runConfig, idempotencyKey, clientID string) (int, string, bool, error) {
	path := fmt.Sprintf("%s/api/v1/seckill/activities/%d/track", strings.TrimRight(cfg.BaseURL, "/"), cfg.ActivityID)
	body := map[string]any{
		"activity_item_id": cfg.ActivityItemID,
		"event_type":       cfg.EventType,
		"client_id":        clientID,
		"idempotency_key":  idempotencyKey,
		"occurred_at_unix": time.Now().Unix(),
	}
	statusCode, code, data, err := doJSON(client, http.MethodPost, path, "", body)
	if err != nil {
		return statusCode, code, false, err
	}
	accepted := false
	if len(data) > 0 {
		var track struct {
			Accepted bool `json:"accepted"`
		}
		if err := json.Unmarshal(data, &track); err == nil {
			accepted = track.Accepted
		}
	}
	return statusCode, code, accepted, nil
}

func doJSON(client *http.Client, method, url, token string, payload any) (int, string, json.RawMessage, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, "", nil, err
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(raw))
	if err != nil {
		return 0, "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", nil, err
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return resp.StatusCode, "", nil, err
	}
	return resp.StatusCode, strings.TrimSpace(env.Code), env.Data, nil
}
