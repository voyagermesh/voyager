package testserverclient

import (
	"fmt"
	"testing"
)

func TestTCPService(t *testing.T) {
	resp, err := NewTestTCPClient("35.184.243.145:4545").DoWithRetry(5)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(*resp)
}
