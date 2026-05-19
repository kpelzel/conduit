// Copyright 2026. Triad National Security, LLC. All rights reserved.

package fta

import (
	"time"

	proto "github.com/lanl/conduit/api"
	"github.com/lanl/conduit/defaults"
	"github.com/lanl/conduit/internal/etcd"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func updateTransferExpiry(t proto.IncompleteTransfer, em *etcd.ETCDManager) chan bool {
	expiryTicker := time.NewTicker(viper.GetDuration(defaults.ConfigExpiryIntervalKey))
	abortTicker := time.NewTicker(viper.GetDuration(defaults.ConfigExpiryIntervalKey))
	quit := make(chan bool)
	logrus.Debugf("starting UpdateExpiry for transfer[%s]", t.GetTransferID())
	go func() {
		// update expiry immediatly, then do it every internal
		succeeded, err, newExpiry := updateTransferExpiryOnce(t, em)
		if err != nil {
			logrus.Errorf("error committing updated expiry to etcd for transfer[%s]: %v", t.GetTransferID(), err)
		} else if !succeeded {
			logrus.Fatalf("failed to update expiry in etcd for transfer[%s]: error for transfer was not none", t.GetTransferID())
		} else {
			logrus.Debugf("successfully updated expiry for transfer[%s]: %s", t.GetTransferID(), newExpiry)
		}

		for {
			select {
			case <-expiryTicker.C:
				logrus.Debugf("updating expiry for transfer[%s]", t.GetTransferID())
				succeeded, err, newExpiry := updateTransferExpiryOnce(t, em)
				if err != nil {
					logrus.Errorf("error committing updated expiry to etcd for transfer[%s]: %v", t.GetTransferID(), err)
				} else if !succeeded {
					logrus.Fatalf("failed to update expiry in etcd for transfer[%s]: error for transfer was not none", t.GetTransferID())
				} else {
					logrus.Debugf("successfully updated expiry for transfer[%s]: %s", t.GetTransferID(), newExpiry)
				}

			case <-quit:
				expiryTicker.Stop()
				logrus.Warnf("UpdateExpiry was stopped for transfer[%s]", t.GetTransferID())
				return
			}
		}
	}()
	go func() {
		for {
			select {
			// check if the user aborted the transfer
			case <-abortTicker.C:
				logrus.Debugf("checking for transfer[%s] abort", t.GetTransferID())
				resp, err := em.Get(t.ETCDErrorKey())
				if err != nil {
					logrus.Errorf("error getting lease error state for transfer[%s]: %v", t.GetTransferID(), err)
				} else if len(resp.Kvs) < 1 {
					logrus.Errorf("etcd didn't return any error state for this entry: %s", t.ETCDErrorKey())
				} else {
					if val, ok := proto.Error_value[string(resp.Kvs[0].Value)]; ok {
						es := proto.Error(val)
						if es == proto.Error_ERROR_ABORTED {
							logrus.Fatal("The Transfer was aborted, stopping...")
						}
					} else {
						logrus.Errorf("could not cast %s to a error state type", string(resp.Kvs[0].Value))
					}
				}

			case <-quit:
				abortTicker.Stop()
				logrus.Warnf("abort check was stopped for transfer[%s]", t.GetTransferID())
				return
			}
		}
	}()
	return quit
}

func updateTransferExpiryOnce(t proto.IncompleteTransfer, em *etcd.ETCDManager) (succeed bool, err error, newExpiry string) {
	expiryAdvance := viper.GetDuration(defaults.ConfigExpiryAdvanceKey)

	txn, cancel := em.Txn()

	txn.If(clientv3.Compare(clientv3.Value(t.ETCDErrorKey()), "=", proto.Error_ERROR_NONE.String()))

	newExpiry = timestamppb.New(time.Now().Add(expiryAdvance)).AsTime().Format(time.RFC3339)
	txn.Then(clientv3.OpPut(t.ETCDExpiryKey(), newExpiry))
	resp, err := txn.Commit()
	cancel()

	return resp.Succeeded, err, newExpiry
}
