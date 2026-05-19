// Copyright 2026. Triad National Security, LLC. All rights reserved.

package util

import (
	"fmt"
	"testing"

	"github.com/lanl/conduit/defaults"
)

type certTestCase struct {
	userCert     string
	userKey      string
	userKeypair  string
	userKeyExist bool
	resultCert   string
	resultKey    string
	resultErr    bool
}

var (
	defaultKeypairPath = fmt.Sprintf("~/%s", defaults.DefaultBundleName)
	testCases          = []certTestCase{
		{
			userCert:     "",
			userKey:      "",
			userKeypair:  defaultKeypairPath,
			userKeyExist: true,
			resultCert:   defaultKeypairPath,
			resultKey:    defaultKeypairPath,
			resultErr:    false,
		},
		{
			userCert:     "",
			userKey:      "",
			userKeypair:  defaultKeypairPath,
			userKeyExist: false,
			resultCert:   "",
			resultKey:    "",
			resultErr:    false,
		},
		{
			userCert:     "path/to/user/cert",
			userKey:      "path/to/user/key",
			userKeypair:  defaultKeypairPath,
			userKeyExist: false,
			resultCert:   "path/to/user/cert",
			resultKey:    "path/to/user/key",
			resultErr:    false,
		},
		{
			userCert:     "path/to/user/cert",
			userKey:      "path/to/user/key",
			userKeypair:  "nondefault/path",
			userKeyExist: true,
			resultCert:   "nondefault/path",
			resultKey:    "nondefault/path",
			resultErr:    false,
		},
		{
			userCert:     "path/to/user/cert",
			userKey:      "path/to/user/key",
			userKeypair:  "nondefault/path",
			userKeyExist: false,
			resultCert:   "nondefault/path",
			resultKey:    "nondefault/path",
			resultErr:    false,
		},
		{
			userCert:     "path/to/user/cert",
			userKey:      "",
			userKeypair:  "nondefault/path",
			userKeyExist: true,
			resultCert:   "nondefault/path",
			resultKey:    "nondefault/path",
			resultErr:    false,
		},
		{
			userCert:     "",
			userKey:      "path/to/user/key",
			userKeypair:  "nondefault/path",
			userKeyExist: true,
			resultCert:   "nondefault/path",
			resultKey:    "nondefault/path",
			resultErr:    false,
		},
		{
			userCert:     "path/to/user/cert",
			userKey:      "",
			userKeypair:  defaultKeypairPath,
			userKeyExist: true,
			resultCert:   "path/to/user/cert",
			resultKey:    "path/to/user/cert",
			resultErr:    false,
		},
		{
			userCert:     "",
			userKey:      "path/to/user/key",
			userKeypair:  defaultKeypairPath,
			userKeyExist: true,
			resultCert:   "path/to/user/key",
			resultKey:    "path/to/user/key",
			resultErr:    false,
		},
	}
)

func init() {
	TEST_MODE = true
}

func TestGetUserCertAndKey(t *testing.T) {
	for i, c := range testCases {
		TEST_UserCertBundleExists = c.userKeyExist

		cert, key, err := GetUserCertAndKey(c.userCert, c.userKey, c.userKeypair, defaultKeypairPath)
		if cert != c.resultCert {
			t.Errorf("%v: got cert: [%v], expected: [%v]", i, cert, c.resultCert)
		}
		if key != c.resultKey {
			t.Errorf("%v: got key: [%v], expected: [%v]", i, key, c.resultKey)
		}
		if c.resultErr && err == nil {
			t.Errorf("%v: expected and error but didn't get one", i)
		}
		if !c.resultErr && err != nil {
			t.Errorf("%v: did not expect and error but got one: %v", i, err)
		}
	}
}
