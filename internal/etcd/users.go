// Copyright 2026. Triad National Security, LLC. All rights reserved.

package etcd

import (
	"context"
	"fmt"

	proto "github.com/lanl/conduit/api"
	"go.etcd.io/etcd/api/v3/authpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func (em *ETCDManager) AddRoot() {
	em.cmutex.Lock()
	defer em.cmutex.Unlock()
	rootExists, err := em.DoesUserExist("root")
	if err != nil {
		em.log.Fatalf("failed to check for etcd root user: %v", err)
	}
	if !rootExists {
		err := em.AddUser("root")
		if err != nil {
			em.log.Fatalf("failed to add user root: %v", err)
		}
	}

	_, err = em.client.UserGrantRole(context.TODO(), "root", "root")
	if err != nil {
		em.log.Fatalf("failed to add root user to root role: %v", err)
	}

	// at the time I wrote this, client.authStatus was not implemented on etcd 3.5.0-beta.4
	// authStatus, err := em.client.AuthStatus(context.TODO())
	// if err != nil {
	// 	em.log.Errorf("failed to get etcd auth status: %v", err)
	// }
	// if authStatus.Enabled {
	// 	em.log.Info("auth is already enabled on etcd")
	// } else {
	// 	_, err := em.client.AuthEnable(context.TODO())
	// 	if err != nil {
	// 		em.log.Fatalf("failed to enable auth on etcd: %v", err)
	// 	}
	// }

	_, err = em.client.AuthEnable(context.TODO())
	if err != nil {
		em.log.Fatalf("failed to enable auth on etcd: %v", err)
	}
}

// DoesUserExist will check if a user already exists in etcd
//
// REQUIRES A LOCK BEFOREHAND
func (em *ETCDManager) DoesUserExist(username string) (bool, error) {
	// check if user is already in the list of etcd users
	userExists := false
	userList, err := em.client.UserList(context.TODO())
	if err != nil {
		return false, fmt.Errorf("failed to get user list: %v", err)
	}
	for _, user := range userList.Users {
		if user == username {
			userExists = true
			break
		}
	}

	return userExists, nil
}

// DoesRoleExist will check if a role already exists in etcd
//
// REQUIRES A LOCK BEFOREHAND
func (em *ETCDManager) DoesRoleExist(roleName string) (bool, error) {
	// check if role is already in the list of etcd roles
	roleExists := false
	roleList, err := em.client.RoleList(context.TODO())
	if err != nil {
		return false, fmt.Errorf("failed to get role list: %v", err)
	}
	for _, role := range roleList.Roles {
		if role == roleName {
			roleExists = true
			break
		}
	}

	return roleExists, nil
}

// AddUser will add a user into etcd
//
// REQUIRES A LOCK BEFOREHAND
func (em *ETCDManager) AddUser(username string) error {
	userExists, err := em.DoesUserExist(username)
	if err != nil {
		return err
	}

	// if it isn't, then add it
	if userExists {
		em.log.Warnf("the etcd instance already has user: %v", username)
		return nil
		// return fmt.Errorf("the etcd instance already has user: %v", username)
	} else {
		em.log.Debugf("sending request to add user: %v", username)
		noPass := &clientv3.UserAddOptions{
			NoPassword: true,
		}
		_, err := em.client.UserAddWithOptions(context.TODO(), username, "", noPass)
		if err != nil {
			return fmt.Errorf("failed to add user: %v to etcd: %v", username, err)
		} else {
			em.log.Debugf("successfully added user: %v to etcd", username)
		}
	}

	return nil
}

// AddRole will add a role into etcd
//
// REQUIRES A LOCK BEFOREHAND
func (em *ETCDManager) AddRole(roleName string) error {
	roleExists, err := em.DoesRoleExist(roleName)
	if err != nil {
		return err
	}

	// if it isn't, then add it
	if roleExists {
		em.log.Warnf("the etcd instance already has role: %v", roleName)
		return nil
		// return fmt.Errorf("the etcd instance already has user: %v", username)
	} else {
		em.log.Debugf("sending request to add role: %v", roleName)
		_, err = em.client.RoleAdd(context.TODO(), roleName)
		if err != nil {
			return fmt.Errorf("failed to add role: %v to etcd: %v", roleName, err)
		} else {
			em.log.Debugf("successfully added role: %v to etcd", roleName)
		}
	}

	return nil
}

// AddTransferUser will add a user and role for a transfer
func (em *ETCDManager) AddTransferUser(transferID string) error {
	em.cmutex.Lock()
	defer em.cmutex.Unlock()
	// add transferID as etcd user
	err := em.AddUser(transferID)
	if err != nil {
		return fmt.Errorf("failed to add transfer user %v to etcd: %v", transferID, err)
	}

	// add role for this user
	err = em.AddRole(transferID)
	if err != nil {
		return fmt.Errorf("failed to add role %v to etcd: %v", transferID, err)
	} else {
		em.log.Debugf("successfully added role: %v to etcd", transferID)
	}

	// add user to the role
	_, err = em.client.UserGrantRole(context.TODO(), transferID, transferID)
	if err != nil {
		return fmt.Errorf("failed to add user %v to role %v: %v", transferID, transferID, err)
	} else {
		em.log.Debugf("successfully added user: %v to role: %v", transferID, transferID)
	}

	// give role permission to read/write to "transfers/<transferID>"
	_, err = em.client.RoleGrantPermission(context.TODO(), transferID, proto.TransferPrefix+transferID, clientv3.GetPrefixRangeEnd(proto.TransferPrefix+transferID), clientv3.PermissionType(authpb.READWRITE))
	if err != nil {
		return fmt.Errorf("failed to grant etcd permission to user %v: %v", transferID, err)
	} else {
		em.log.Debugf("successfully added permissions: %v to role: %v at: %v", authpb.READWRITE.String(), transferID, proto.TransferPrefix+transferID)
	}

	return nil
}
