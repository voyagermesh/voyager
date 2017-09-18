package framework

import "github.com/appscode/go/crypto/rand"

func (r *rootInvocation) UniqueName() string {
	return rand.WithUniqSuffix("e2e-test")
}
