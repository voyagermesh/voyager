// Copyright (c) 2014 Kelsey Hightower. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package user

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
)

var userDatabase = "/etc/passwd"

var InvalidUserDatabaseError = errors.New("invalid user database")

func current() (*User, error) {
	return lookupPasswd(syscall.Getuid(), "", false)
}

func lookup(username string) (*User, error) {
	return lookupPasswd(-1, username, true)
}

func lookupId(uid string) (*User, error) {
	i, err := strconv.Atoi(uid)
	if err != nil {
		return nil, err
	}
	return lookupPasswd(i, "", false)
}

func lookupPasswd(uid int, username string, lookupByName bool) (*User, error) {
	fields := []string{}
	f, err := os.Open(userDatabase)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fs := strings.SplitN(scanner.Text(), ":", 7)
		if strings.HasPrefix(fs[0], "#") {
			continue
		}
		if len(fs) < 7 {
			return nil, InvalidUserDatabaseError
		}
		if lookupByName {
			if username == fs[0] {
				fields = fs
			}
		} else {
			if strconv.Itoa(uid) == fs[2] {
				fields = fs
			}
		}
	}
	if len(fields) < 7 {
		if lookupByName {
			return nil, UnknownUserError(username)
		} else {
			return nil, UnknownUserIdError(uid)
		}
	}
	u := &User{
		Username: fields[0],
		Uid:      fields[2],
		Gid:      fields[3],
		Name:     fields[4],
		HomeDir:  fields[5],
	}
	if i := strings.Index(u.Name, ","); i >= 0 {
		u.Name = u.Name[:i]
	}
	return u, nil
}
