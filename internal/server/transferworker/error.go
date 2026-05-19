// Copyright 2026. Triad National Security, LLC. All rights reserved.

package transferworker

import (
	"strconv"

	"github.com/google/uuid"
	proto "github.com/lanl/conduit/api"
	"github.com/lanl/conduit/defaults"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// handleLeaseError sets the transfer status to error and cancels all other scheduler jobs for this transfer
func (tw *TransferWorker) handleTransferError(it proto.IncompleteTransfer, eventID uuid.UUID) {
	defer tw.removeJob(eventID)

	// set transfer state to error
	comparisons := []clientv3.Cmp{}
	comparisons = append(comparisons, clientv3.Compare(clientv3.Value(it.ETCDStateKey()), "!=", proto.TransferState_TRANSFER_ERROR.String()))
	comparisons = append(comparisons, clientv3.Compare(clientv3.Value(it.ETCDStateKey()), "!=", proto.TransferState_TRANSFER_ABORTED.String()))
	actions := []clientv3.Op{}
	actions = append(actions, clientv3.OpPut(it.ETCDStateKey(), proto.TransferState_TRANSFER_ERROR.String()))

	resp, err := tw.em.RetryTxn(&comparisons, &actions, defaults.MaxRetries, defaults.RetryDelay)
	if err != nil {
		tw.log.Errorf("error committing to etcd for transfer[%s]: %v", it.GetTransferID(), err)
		err := tw.em.CompleteTransfer(it)
		if err != nil {
			tw.log.Error(err)
		}
		return
	}
	if !resp.Succeeded {
		tw.log.Warnf("failed to add error state to transfer[%v], it was already set to error state", it.GetTransferID())
		return
	}

	// get the full transfer details
	tid, err := uuid.Parse(it.GetTransferID())
	if err != nil {
		tw.log.Errorf("failed to parse uuid from [%s]: %v", it.GetTransferID(), err)
	}

	t, _, err := tw.em.GetTransfer(tid)
	if err != nil {
		tw.log.Errorf("failed to get transfer[%s] from etcd: %v", t.GetTransferID(), err)
		err := tw.em.CompleteTransfer(it)
		if err != nil {
			tw.log.Error(err)
		}
		return
	}

	// check if this happened because of validation failing
	// go through each lease and see if any of them made it to the setup stage. If they didn't then it didn't pass validation
	foundPastValidation := false
	if t.GetSchedulerNodes().GetSetup() != "" {
		foundPastValidation = true
	}

	if !foundPastValidation {
		tw.log.Debugf("no leases for transfer[%v] made it to the setup stage", t.GetTransferID())

		comparisons := []clientv3.Cmp{}
		actions := []clientv3.Op{}

		// TODO: is this going to check this correctly?
		comparisons = append(comparisons, clientv3.Compare(clientv3.Value(it.ETCDSchedulerNodesKey(proto.SchedulerCommand_SETUP)), "=", ""))

		actions = append(actions, clientv3.OpPut(it.ETCDActiveKey(), strconv.FormatBool(false)))
		actions = append(actions, clientv3.OpPut(it.ETCDArchiveStateKey(), proto.ArchiveState_ARCHIVE_READY.String()))
		actions = append(actions, clientv3.OpDelete(it.ETCDLeaseListKey(), clientv3.WithPrefix()))

		resp, err := tw.em.RetryTxn(&comparisons, &actions, defaults.MaxRetries, defaults.RetryDelay)
		if err != nil || !resp.Succeeded {
			tw.log.Errorf("failed to set transfer[%s] to inactive because there was a scheduler setup job found for a lease: %v", t.GetTransferID(), err)
		} else {
			tw.log.Infof("successfully set transfer[%s] to inactive", t.GetTransferID())
			return
		}
	}

	// TODO: cancel all scheduler jobs
	// _, err = tw.sm.CancelAllSlurmJobs(t)
	// if err != nil {
	// 	tw.log.Errorf("failed to cancel all slurm jobs, continuing anyway: %v", err)
	// }

	// mark transfer as inactive
	err = tw.em.CompleteTransfer(it)
	if err != nil {
		tw.log.Error(err)
	}

}
