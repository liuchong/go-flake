package flake

import (
	"testing"
)

func TestFlakeGen(t *testing.T) {
	g, err := NewGenerator(123, 0)
	if err != nil {
		t.Errorf("Test flake ID generator failed. Err: %s", err)
	}

	t.Logf("New flakeID: %s", g.NextID().ToString())
	t.Logf("New flakeID: %s", g.NextID().ToString())
	t.Logf("New flakeID: %s", g.NextID().ToString())

	id0 := g.NextID().ToString()
	id1 := g.NextID().ToString()
	if id0 == id1 {
		t.Errorf("Test flake ID generator failed, duplicate ID")
	}
}
