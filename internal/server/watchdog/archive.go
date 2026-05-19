// Copyright 2026. Triad National Security, LLC. All rights reserved.

package watchdog

import (
	"fmt"

	"github.com/google/uuid"
	proto "github.com/lanl/conduit/api"
	"github.com/lanl/conduit/internal/etcd"
)

func (w *Watchdog) archiveTransfer(it proto.IncompleteTransfer, eventID uuid.UUID) {
	defer w.removeJob(eventID)

	// get uuid from string
	tid, err := uuid.Parse(it.GetTransferID())
	if err != nil {
		w.log.Error(fmt.Errorf("archive transfer failed to parse transferid[%v]: %v", it.GetTransferID(), err))
		return
	}

	// get full transfer from etcd
	t, _, err := w.em.GetTransfer(tid)
	if err != nil {
		w.log.Error(fmt.Errorf("archive transfer failed to get transfer from etcd: %v", err))
		return
	}

	// set archive state to submit
	succeeded, _, err := w.em.SafelySetTransferArchiveState(it, proto.ArchiveState_ARCHIVE_READY, proto.ArchiveState_ARCHIVE_SUBMIT)
	if err != nil {
		tErr := fmt.Errorf("error committing new transfer archive state to etcd for transfer[%s]: %v", it.GetTransferID(), err)
		w.log.Error(tErr)
		return
	} else if !succeeded {
		w.log.Warnf("failed to set transfer[%s] archive state to %v. Another worker probably took care of it", it.GetTransferID(), proto.ArchiveState_ARCHIVE_SUBMIT.String())
		return
	}
	w.log.Infof("successfully set transfer[%s] state to %v", it.GetTransferID(), proto.ArchiveState_ARCHIVE_SUBMIT.String())

	// send transfer to rqlite
	t.ArchiveState = proto.ArchiveState_ARCHIVE_COMPLETE
	err = w.rm.AddTransfer(t)
	if err != nil {
		w.log.Error(fmt.Errorf("failed to add transfer[%s] to rqlite: %v", it.GetTransferID(), err))
		return
	}

	// delete transfer from etcd
	w.log.Infof("successfully archived transfer %v", it.GetTransferID())
	tDeleted, err := w.em.DeleteTransfer(tid)
	if err != nil {
		w.log.Errorf("failed to delete transfer[%v] from etcd: %v", it.GetTransferID(), err)
	}

	lDeleted, err := w.em.DeleteTransferLease(it)
	if err != nil {
		w.log.Errorf("failed to delete transfer[%v] lease from etcd: %v", it.GetTransferID(), err)
	}

	// compact etcd
	err = w.compactETCD()
	if err != nil {
		if err.Error() == etcd.ErrRevCompacted {
			w.log.Warnf("failed to compact etcd: %v", err)
		} else {
			w.log.Errorf("failed to compact etcd: %v", err)
		}
	}

	w.log.Infof("successfully deleted transfer[%s] from etcd: %v %v", it.GetTransferID(), tDeleted, lDeleted)
}
