package svc

import (
	"context"
	"time"

	"flashsale/apps/seckill/rpc/internal/model"

	"go.uber.org/zap"
)

// EnqueueOrderLink 异步写入订单关联，返回 false 表示队列不可用或已满。
func (s *ServiceContext) EnqueueOrderLink(link *model.OrderLink) bool {
	if s == nil || link == nil {
		return false
	}
	if s.orderLinkTasks == nil {
		if s.Perf != nil {
			s.Perf.MarkOrderLinkDropped()
		}
		return false
	}
	cp := *link
	select {
	case s.orderLinkTasks <- &cp:
		if s.Perf != nil {
			s.Perf.MarkOrderLinkEnqueued()
		}
		return true
	default:
		if s.Perf != nil {
			s.Perf.MarkOrderLinkDropped()
		}
		return false
	}
}

// EnqueueTrafficEvent 异步写入埋点事件，返回 false 表示队列不可用或已满。
func (s *ServiceContext) EnqueueTrafficEvent(event *model.TrafficEvent) bool {
	if s == nil || event == nil || s.trafficTasks == nil {
		return false
	}
	cp := *event
	select {
	case s.trafficTasks <- &cp:
		if s.Perf != nil {
			s.Perf.MarkTrackEventEnqueued()
		}
		return true
	default:
		if s.Perf != nil {
			s.Perf.MarkTrackEventDropped()
		}
		return false
	}
}

// OrderLinkWriteTimeout 返回订单关联落库超时配置。
func (s *ServiceContext) OrderLinkWriteTimeout() time.Duration {
	if s == nil || s.orderLinkWriteTimeout <= 0 {
		return 600 * time.Millisecond
	}
	return s.orderLinkWriteTimeout
}

func (s *ServiceContext) startOrderLinkWorkers(workerCount int) {
	if s == nil || s.orderLinkTasks == nil || workerCount <= 0 {
		return
	}
	for i := 0; i < workerCount; i++ {
		s.orderLinkWG.Add(1)
		go func() {
			defer s.orderLinkWG.Done()
			for {
				select {
				case <-s.orderLinkCtx.Done():
					return
				case link := <-s.orderLinkTasks:
					if link == nil || s.SeckillRepo == nil {
						continue
					}
					writeCtx, cancel := context.WithTimeout(context.Background(), s.OrderLinkWriteTimeout())
					err := s.SeckillRepo.CreateOrderLink(writeCtx, link)
					cancel()
					if err != nil && s.Logger != nil {
						s.Logger.Warn("async create seckill order link failed", zap.Error(err), zap.Int64("order_id", link.OrderID), zap.String("order_no", link.OrderNo))
					}
				}
			}
		}()
	}
}

func (s *ServiceContext) startTrafficWorkers(workerCount int) {
	if s == nil || s.trafficTasks == nil || workerCount <= 0 {
		return
	}
	for i := 0; i < workerCount; i++ {
		s.trafficWG.Add(1)
		go func() {
			defer s.trafficWG.Done()
			for {
				select {
				case <-s.trafficCtx.Done():
					return
				case event := <-s.trafficTasks:
					if event == nil {
						continue
					}
					if err := s.RecordTrafficEvent(event); err != nil && s.Logger != nil {
						s.Logger.Warn("async record traffic event failed",
							zap.Error(err),
							zap.Int64("activity_id", event.ActivityID),
							zap.Int64("activity_item_id", event.ActivityItemID),
							zap.String("event_type", event.EventType),
						)
					}
				}
			}
		}()
	}
}
