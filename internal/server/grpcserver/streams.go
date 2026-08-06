// Copyright 2026. Triad National Security, LLC. All rights reserved.

package grpcserver

import (
	"sync"

	"github.com/google/uuid"
	proto "github.com/lanl/conduit/api"
)

type userStream struct {
	ch   chan *proto.NotifyMessage
	done <-chan struct{}
}

func (s *ConduitServer) updateTransferStreams(transferID uuid.UUID) {
	s.asMutex.RLock()
	defer s.asMutex.RUnlock()

	for _, streamChan := range s.activeStreams[transferID] {
		select {
		case streamChan <- true:
		default:
			// An update is already pending.
		}
	}
}

func (s *ConduitServer) updateUserStreams(user string, message *proto.NotifyMessage, workerDone <-chan struct{}) bool {
	s.usMutex.RLock()

	streamMap := s.userStreams[user]
	streams := make([]*userStream, 0, len(streamMap))

	for _, stream := range streamMap {
		streams = append(streams, stream)
	}

	s.usMutex.RUnlock()

	for _, stream := range streams {
		select {
		case stream.ch <- message:
			// Delivered to this stream.

		case <-stream.done:
			// This particular stream disconnected.
			// Continue delivering to the user's other streams.

		case <-workerDone:
			// The user no longer has any active streams.
			return false
		}
	}

	return true
}

func (s *ConduitServer) runUserNotificationWorker(user string, worker *userNotificationWorker) {
	for {
		batch, ok := worker.nextBatch()
		if !ok {
			return
		}

		for _, message := range batch {
			if !s.updateUserStreams(user, message, worker.done) {
				return
			}
		}
	}
}

func (s *ConduitServer) enqueueUserNotifications(user string, messages []*proto.NotifyMessage) {
	if len(messages) == 0 {
		return
	}

	s.usMutex.RLock()
	worker := s.userStreamsWorkers[user]
	s.usMutex.RUnlock()

	// No notification stream is currently registered for this user.
	if worker == nil {
		return
	}

	// A false result means the user's last stream disconnected
	// while this batch was being enqueued.
	worker.enqueue(messages)
}

type userNotificationWorker struct {
	mu      sync.Mutex
	cond    *sync.Cond
	queue   [][]*proto.NotifyMessage
	stopped bool
	done    chan struct{}
}

func newUserNotificationWorker() *userNotificationWorker {
	worker := &userNotificationWorker{
		done: make(chan struct{}),
	}

	worker.cond = sync.NewCond(&worker.mu)

	return worker
}

// enqueue adds one etcd event batch to the worker's FIFO queue.
func (w *userNotificationWorker) enqueue(messages []*proto.NotifyMessage) bool {
	if len(messages) == 0 {
		return true
	}

	// Copy the slice so the worker owns the batch container.
	// The NotifyMessage values themselves are treated as immutable.
	batch := append([]*proto.NotifyMessage(nil), messages...)

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return false
	}

	w.queue = append(w.queue, batch)
	w.cond.Signal()

	return true
}

func (w *userNotificationWorker) nextBatch() ([]*proto.NotifyMessage, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for len(w.queue) == 0 && !w.stopped {
		w.cond.Wait()
	}

	if w.stopped {
		return nil, false
	}

	batch := w.queue[0]

	// Avoid retaining references to delivered messages.
	w.queue[0] = nil
	w.queue = w.queue[1:]

	if len(w.queue) == 0 {
		w.queue = nil
	}

	return batch, true
}

func (w *userNotificationWorker) stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}

	w.stopped = true
	w.queue = nil

	close(w.done)
	w.cond.Broadcast()
}
