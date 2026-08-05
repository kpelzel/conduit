// Copyright 2026. Triad National Security, LLC. All rights reserved.

package grpcserver

import (
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

func (s *ConduitServer) updateUserStreams(user string, nm *proto.NotifyMessage) {
	s.usMutex.RLock()

	userStreams := make(
		[]*userStream,
		0,
		len(s.userStreams[user]),
	)

	for _, stream := range s.userStreams[user] {
		userStreams = append(userStreams, stream)
	}

	s.usMutex.RUnlock()

	for _, stream := range userStreams {
		select {
		case stream.ch <- nm:
			// Block until the active stream accepts it.
		case <-stream.done:
			// The stream disconnected before accepting it.
		}
	}
}
