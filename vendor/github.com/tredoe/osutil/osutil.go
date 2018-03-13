// Copyright 2014 Jonas mg
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package osutil

import (
	"errors"
	"os"
	"os/exec"
)

// Exec executes a command setting both standard input, output and error.
func Exec(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return err
	}
	return nil
}

// ExecSudo executes a command under "sudo".
func ExecSudo(cmd string, args ...string) error {
	return Exec("sudo", append([]string{cmd}, args...)...)
}

// Sudo executes command "sudo".
// If some command needs to use "sudo", then could be used this function at
// the beginning so there is not to wait until that it been requested later.
func Sudo() error {
	return Exec("sudo", "/bin/true")
}

var ErrNoRoot = errors.New("MUST have administrator privileges")

// MustbeRoot returns an error message if the user is not root.
func MustbeRoot() error {
	if os.Getuid() != 0 {
		return ErrNoRoot
	}
	return nil
}
