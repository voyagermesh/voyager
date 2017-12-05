// Copyright (c) 2014 Kelsey Hightower. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package user

import (
	"fmt"
	"strconv"
)

// User represents a unix user account found in
// the "/etc/passwd" password file.
type User struct {
	Username string
	Uid      string
	Gid      string
	Name     string
	HomeDir  string
}

// UnknownUserIdError is returned by LookupId when a user cannot be found
// in the "/etc/passwd" password file.
type UnknownUserIdError int

func (e UnknownUserIdError) Error() string {
	return fmt.Sprintf("user: unknown userid %s", strconv.Itoa(int(e)))
}

// UnknownUserError is returned by Lookup when a user cannot be found
// in the "/etc/passwd" password file.
type UnknownUserError string

func (e UnknownUserError) Error() string {
	return fmt.Sprintf("user: unknown user %s", string(e))
}
