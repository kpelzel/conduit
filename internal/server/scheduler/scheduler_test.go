// Copyright 2026. Triad National Security, LLC. All rights reserved.

package scheduler

import (
	"container/heap"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	conduitProto "github.com/lanl/conduit/api"
)

// Define a Job struct
type Jobs struct {
	JobID string
}

// TestAddJobsCheck runs a variety of tests to ensure that jobs are being added to the priority queue correctly
func TestAddJobsCheck(t *testing.T) {

	// making priority queue
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	// generating random uuids
	uid1 := uuid.New()
	uid2 := uuid.New()
	uid3 := uuid.New()
	uid4 := uuid.New()
	uid5 := uuid.New()
	uid6 := uuid.New()

	createdTime1 := time.Now()
	createdTime2 := createdTime1.Add(2 * time.Minute)
	createdTime3 := createdTime1.Add(23 * time.Minute)
	createdTime4 := createdTime1.Add(34 * time.Minute)
	createdTime5 := createdTime1.Add(45 * time.Minute)
	createdTime6 := createdTime1.Add(56 * time.Minute)

	// assigning IDs, scheduler commands, and priority values to items in queue
	pq.AddJob(uid1, conduitProto.SchedulerCommand_VALIDATION, createdTime1, 3, nil)
	pq.AddJob(uid2, conduitProto.SchedulerCommand_SETUP, createdTime2, 2, nil)
	pq.AddJob(uid3, conduitProto.SchedulerCommand_TRANSFER, createdTime3, 5, nil)

	// Test 1
	job, err := pq.PopJob()

	if err != nil {
		t.Errorf("Error removing job from top of the stack: %v", err)
	} else {
		t.Logf("Popped job with priority: %v and index: %v", job.Priority, job.Index)
	}

	// checking that priority is what we expect
	if job.Priority != 3 {
		t.Errorf("Expected 3 for top priority, but recieved: %d", job.Priority)
	} else {
		t.Logf("Expected 3 for top priority, and recieved: %d", job.Priority)
	}

	// checking that the uuid is what we expect
	if job.JobID != uid1 {
		t.Errorf("Expected %v for top uuid, but recieved: %v", uid1, job.JobID)
	} else {
		t.Logf("Expected %v for top uuid and recieved: %v", uid1, job.JobID)
	}

	// checking that the command is what we expect
	if job.SchedulerCommand != conduitProto.SchedulerCommand_VALIDATION {
		t.Errorf("Expected %v for top scheduler command, but recieved: %v", conduitProto.SchedulerCommand_VALIDATION, job.SchedulerCommand)
	} else {
		t.Logf("Expected %v for top scheduler command, and recieved: %v", conduitProto.SchedulerCommand_VALIDATION, job.SchedulerCommand)
	}

	// checking that the priority queue length is what we expect
	if pq.Len() != 2 {
		t.Errorf("Expected pq length of 2 after pop, but recieved: %d", pq.Len())
	} else {
		t.Logf("Expected pq length of 2 after pop, and recieved: %d", pq.Len())
	}

	// checking that the createdTime is what we expect
	if job.CreatedTime != createdTime1 {
		t.Errorf("Expected created time of %v, but recieved: %v", job.CreatedTime, createdTime1)
	} else {
		t.Logf("Expected created time of %v, and recieved: %v", job.CreatedTime, createdTime1)
	}

	// Test 2
	pq.AddJob(uid4, conduitProto.SchedulerCommand_TRANSFER, createdTime4, 7, nil)

	job, err = pq.PopJob()

	if err != nil {
		t.Errorf("Expected no error, recieved: %v", err)
	} else {
		t.Logf("Popped job with priority: %v and index: %v", job.Priority, job.Index)
	}

	if job.Priority != 7 {
		t.Errorf("Expect 7 for top priority, but recieved: %d", job.Priority)
	} else {
		t.Logf("Expected 7 for top priority, and recieved: %d", job.Priority)
	}

	if job.JobID != uid4 {
		t.Errorf("Expected %v for top uuid, but recieved: %v", uid4, job.JobID)
	} else {
		t.Logf("Expected %v for top uiid and recieved: %v", uid4, job.JobID)
	}

	if job.SchedulerCommand != conduitProto.SchedulerCommand_TRANSFER {
		t.Errorf("Expected %v for top scheduler command, but recieved: %v", conduitProto.SchedulerCommand_TRANSFER, job.SchedulerCommand)
	} else {
		t.Logf("Expected %v for top scheduler command, and recieved: %v", conduitProto.SchedulerCommand_TRANSFER, job.SchedulerCommand)
	}

	// Test 3
	pq.AddJob(uid5, conduitProto.SchedulerCommand_TRANSFER, createdTime5, 10, nil)
	pq.AddJob(uid6, conduitProto.SchedulerCommand_TRANSFER, createdTime6, 10, nil)

	t.Logf("UUID for uid5: %v", uid5)
	t.Logf("UUID for uid6: %v", uid6)

	job, err = pq.PopJob()

	if err != nil {
		t.Errorf("Error removing job from top of the stack: %v", err)
	} else {
		t.Logf("Popped job with priority: %v and index: %v", job.Priority, job.Index)
	}

	if job.Priority != 10 {
		t.Errorf("Expect 10 for top priority, but recieved: %d", job.Priority)
	} else {
		t.Logf("Expected 10 for top priority, and recieved: %d", job.Priority)
	}

	if job.JobID != uid5 {
		t.Errorf("Expected %v for top uuid, but recieved: %v", uid5, job.JobID)
	} else {
		t.Logf("Expected %v for top uuid and recieved: %v", uid5, job.JobID)
	}

	// checking that the createdTime is what we expect
	if job.CreatedTime != createdTime5 {
		t.Errorf("Expected created time of %v, but recieved: %v", job.CreatedTime, createdTime5)
	} else {
		t.Logf("Expected created time of %v, and recieved: %v", job.CreatedTime, createdTime5)
	}
}

// TestETCDKeytoUUID runs a variety of tests to ensure that the ETCD key is being correctly converted to UUIDs
func TestETCDKeytoUUID(t *testing.T) {

	type keyToUUID struct {
		uuid string
		err  bool
	}

	var etcdKeys = []keyToUUID{
		{"f47ac10b-58cc-0372-8567-0e02b2c3d479", false},
		{"01ee836c-e7c9-619d-929a-525400475911", false},
		{"018bd12c-58b0-7683-8a5b-8752d0e86651", false},
		{"11111111-1111-1111-1111-111111111111", false},
		// {"NONE", true},
	}

	for _, etcdKeyTests := range etcdKeys {
		uid, err := ETCDKeytoUUID(etcdKeyTests.uuid)

		// checking that error matches expected error
		if err == nil && !etcdKeyTests.err {
			t.Logf("Expected no error for uuid [%v] and found no error: %v", etcdKeyTests.uuid, err)
		} else {
			t.Errorf("Expected no error for uuid [%v] but found error: %v", etcdKeyTests.err, err)
		}

		// checking that the uuid matches expected uuid
		if err != nil && uid != uuid.MustParse(etcdKeyTests.uuid) {
			t.Errorf("Expected uuid [%v], but got %s", etcdKeyTests.uuid, uid)
		} else {
			t.Logf("Expected uuid [%v], and got %s", etcdKeyTests.uuid, uid)
		}
	}
}

// TestETCDValuestoSchedulerCommand runs a series of tests to ensure that the ETCD values are being converted to a scheduler command
func TestETCDValuestoSchedulerCommand(t *testing.T) {

	type schedCommands struct {
		input   string
		command conduitProto.SchedulerCommand
		err     bool
	}

	var etcdValues = []schedCommands{
		{"NONE", conduitProto.SchedulerCommand_NONE, false},
		{"VALIDATION", conduitProto.SchedulerCommand_VALIDATION, false},
		{"SETUP", conduitProto.SchedulerCommand_SETUP, false},
		{"TRANSFER", conduitProto.SchedulerCommand_TRANSFER, false},
		{"TEARDOWN", conduitProto.SchedulerCommand_TEARDOWN, false},
		// {"unknown", 0, true},
		// {"", 0, true},
		// {"Err", 0, true},
	}

	for _, schedCommandTests := range etcdValues {
		commands, err := ETCDValuestoSchedulerCommand(schedCommandTests.input)

		// checking that error matches expected error
		if err == nil && !schedCommandTests.err {
			t.Logf("Expected no error for input [%v] and found no error: %v", schedCommandTests.err, err)
		} else if err != nil {
			t.Logf("Expected error for input [%v] and found error: %v", schedCommandTests.err, err)
		} else {
			t.Errorf("Expected no error for input [%v] but found error: %v", schedCommandTests.err, err)
		}

		// checking that input matches expected scheduler command
		if commands.String() != schedCommandTests.input {
			t.Errorf("Expected scheduler command: %v, but got %s", schedCommandTests.command, commands)
			continue
		} else {
			t.Logf("Expected scheduler command: %v, and got %s", schedCommandTests.command, commands)
		}
	}
}

// TestSortNodes runs a series of tests to ensure that the nodes are being sorted correctly
func TestSortNodes(t *testing.T) {

	node := new(NodeInfo)

	avaliableNodes := []*NodeInfo{
		{Jobs: node.Jobs, Memory: 50, Name: node.Name, client: node.client},
		{Jobs: node.Jobs, Memory: 500, Name: node.Name, client: node.client},
		{Jobs: node.Jobs, Memory: 100, Name: node.Name, client: node.client},
	}

	expectedSort := []*NodeInfo{
		{Jobs: node.Jobs, Memory: 500, Name: node.Name, client: node.client},
		{Jobs: node.Jobs, Memory: 100, Name: node.Name, client: node.client},
		{Jobs: node.Jobs, Memory: 50, Name: node.Name, client: node.client},
	}

	// Getting nodes' avaliable memory
	sortNodes, err := sortNodes(avaliableNodes)
	if err != nil {
		t.Logf("Error when sorting nodes: %v", err)
	}

	if !reflect.DeepEqual(sortNodes, expectedSort) {
		t.Errorf("Expected [%v] but got [%v]", expectedSort, sortNodes)
	}
}
