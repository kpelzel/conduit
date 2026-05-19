// Copyright 2026. Triad National Security, LLC. All rights reserved.

package scheduler

import (
	"container/heap"
	"fmt"
	"time"

	"github.com/google/uuid" // Action stuff
	proto "github.com/lanl/conduit/api"
	"github.com/lanl/conduit/internal/etcd"
)

type Job struct {
	JobID            uuid.UUID // value of the item (111-111-111)
	Priority         uint32    // priority of item in queue (= 1)
	SchedulerCommand proto.SchedulerCommand
	Index            int // index of the item in the heap
	CreatedTime      time.Time
	StopChannel      chan bool
}

// A priority queue implements heap.Interface and holds the Jobs
type PriorityQueue []*Job

func (pq PriorityQueue) Len() int {
	return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest priority
	// which is why we use the greater than symbol
	// priority queue is sorted by these values:
	// 1. validation jobs
	// 2. higher priority value
	// 3. created datetime
	switch {
	case pq[i].SchedulerCommand == proto.SchedulerCommand_VALIDATION && pq[j].SchedulerCommand != proto.SchedulerCommand_VALIDATION:
		return true
	case pq[i].SchedulerCommand != proto.SchedulerCommand_VALIDATION && pq[j].SchedulerCommand == proto.SchedulerCommand_VALIDATION:
		return false
	case pq[i].Priority > pq[j].Priority:
		return true
	case pq[i].Priority < pq[j].Priority:
		return false
	case pq[i].CreatedTime.Before(pq[j].CreatedTime):
		return true
	default:
		return false
	}
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Job)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // to avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) AddJob(jobID uuid.UUID, schedulerCommand proto.SchedulerCommand, createdTime time.Time, priority uint32, em *etcd.ETCDManager) {

	// create Stop Channel here
	// make this channel a buffer channel to avoid blocking
	// give it a size of 1 (double check)
	updateExpiryStopChan := make(chan bool, 1)

	// Insert a new item and then modify its priority
	item := &Job{
		JobID:            jobID,
		Priority:         priority,
		SchedulerCommand: schedulerCommand,
		CreatedTime:      createdTime,
		StopChannel:      updateExpiryStopChan,
	}
	heap.Push(pq, item)

	it := proto.IncompleteTransfer(&proto.TransferDetails{TransferID: jobID.String()})

	// start updating the expiry for the transfer
	if em != nil {
		go em.UpdateExpiryConstantly(it, item.StopChannel)
	}
}

func (pq *PriorityQueue) PopJob() (*Job, error) {
	if pq.Len() == 0 {
		return nil, fmt.Errorf("priority queue is empty")
	}
	// Look at priority queue and remove the top value to send to the runners
	top := heap.Pop(pq).(*Job)

	// stop updating the expiry for the transfer
	top.StopChannel <- true

	return top, nil
}

// requires a lock
func (pq *PriorityQueue) RemoveJob(id uuid.UUID) (newQueue *PriorityQueue) {
	queue := *pq

	for i := 0; i < len(queue); i++ {
		if queue[i].JobID == id {
			// stop updating the expiry for the transfer
			queue[i].StopChannel <- true

			queue = append(queue[:i], queue[i+1:]...)
		}
	}

	heap.Init(&queue)
	return &queue
}
