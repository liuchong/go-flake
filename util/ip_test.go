package util

import (
	"testing"
)

func TestGetIP(t *testing.T) {
	ip, err := GetIP()
	if err != nil {
		t.Errorf("Test GetIP failed. Err: %s", err)
	}
	t.Logf("Got IP: %+v\n", ip)
	t.Logf("Got IP number: %d\n", IP4toInt(ip))
}
