// Copyright (c) 2014 Kelsey Hightower. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package user

// Current returns the current user.
// Returns UnknownUserIdError if the user cannot be found.
func Current() (*User, error) {
	return current()
}

// Lookup looks up a user by username.
// Returns UnknownUserError if the user cannot be found.
func Lookup(username string) (*User, error) {
	return lookup(username)
}

// LookupId looks up a user by userid.
// Returns UnknownUserIdError if the user cannot be found.
func LookupId(uid string) (*User, error) {
	return lookupId(uid)
}
