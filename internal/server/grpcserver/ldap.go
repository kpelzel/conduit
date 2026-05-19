// Copyright 2026. Triad National Security, LLC. All rights reserved.

package grpcserver

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/jcmturner/goidentity/v6"
	"gopkg.in/ldap.v2"
)

const (
	uidAttribute = "uid"
)

type LDAP struct {
	host                string
	port                int
	basedn              []string
	krb5Attributes      []string
	unameAttributes     []string
	uidNumberAttributes []string
}

func NewLDAP(host string, port int, basedn, krb5Attributes, unameAttributes, uidNumberAttributes []string) (*LDAP, error) {
	if host == "" {
		return nil, fmt.Errorf("ldap host must be specified")
	}

	if port == 0 {
		return nil, fmt.Errorf("ldap port must be specified")
	}

	if len(basedn) == 0 {
		return nil, fmt.Errorf("must specify at least one ldap basedn")
	}

	if len(krb5Attributes) == 0 && len(unameAttributes) == 0 && len(uidNumberAttributes) == 0 {
		return nil, fmt.Errorf("at least one ldap attribute must be specified")
	}

	ldap := &LDAP{
		host:                host,
		port:                port,
		basedn:              basedn,
		krb5Attributes:      krb5Attributes,
		unameAttributes:     unameAttributes,
		uidNumberAttributes: uidNumberAttributes,
	}

	return ldap, nil
}

// ldapSearch takes in a goidentity or a uid (number) and queries ldap for it.
// If id is nil then only uid is queried. If uid is nil then only id is queried.
func (l *LDAP) ldapSearch(id goidentity.Identity, uid int64, timeout time.Duration) (string, error) {
	idUsername := ""
	if id != nil {
		idUsername = id.UserName()
	}

	if id == nil && uid == 0 {
		return "", fmt.Errorf("must provide an id or uid to query ldap: %v %v", id, uid)
	}

	if l.host == "" {
		return "", fmt.Errorf("no ldap host specified")
	}

	ldap.DefaultTimeout = timeout

	addr := net.JoinHostPort(l.host, strconv.Itoa(l.port))
	lc, err := ldap.Dial("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("failed to dial ldap server: %v", err)
	}
	defer lc.Close()

	lc.SetTimeout(timeout)

	entries := []*ldap.Entry{}
	sErrors := []error{}

	for _, bdn := range l.basedn {
		searchRequest := ldap.NewSearchRequest(
			bdn, // The base dn to search
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0,
			0,
			false,
			l.ldapFilter(id, uid),  // The filter to apply
			[]string{uidAttribute}, // A list attributes to retrieve
			nil,
		)

		sr, err := lc.Search(searchRequest)
		if err != nil {
			sErrors = append(sErrors, fmt.Errorf("failed to ldap search for user (%v | %v): %v", idUsername, uid, err))
		}

		if sr != nil {
			entries = append(entries, sr.Entries...)
		}
	}

	// check if every ldap search failed
	if len(l.basedn) == len(sErrors) {
		return "", sErrors[0]
	}

	// check if we didn't find any entries
	if len(entries) == 0 {
		return "", fmt.Errorf("failed to find entry in ldap for user (%v | %v)", idUsername, uid)
	}

	// check that all entries provided the same uname
	uname := entries[0].GetAttributeValue(uidAttribute)
	for _, e := range entries {
		if e.GetAttributeValue(uidAttribute) != uname {
			return "", fmt.Errorf("found conflicting entries in ldap for user (%v | %v): %v vs %v", idUsername, uid, e.GetAttributeValue(uidAttribute), uname)
		}
	}

	// check that we actually found a uname
	if uname == "" {
		return "", fmt.Errorf("failed to find entry in ldap for user (%v | %v)", idUsername, uid)
	}

	return uname, nil
}

// ldapFilter will generate an ldap filter based off a provided Identity and/or uid
func (l *LDAP) ldapFilter(id goidentity.Identity, uid int64) string {
	filter := "(|"

	if id != nil {
		for _, a := range l.krb5Attributes {
			filter += fmt.Sprintf("(%s=%s@%s)", a, id.UserName(), id.Domain())
		}

		for _, a := range l.unameAttributes {
			filter += fmt.Sprintf("(%s=%s)", a, id.UserName())
		}
	}

	if uid != 0 {
		for _, a := range l.uidNumberAttributes {
			filter += fmt.Sprintf("(%s=%d)", a, uid)
		}
	}

	filter += ")"

	return filter
}
