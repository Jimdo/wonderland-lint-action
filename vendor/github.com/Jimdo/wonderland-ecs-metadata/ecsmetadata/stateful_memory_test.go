package ecsmetadata

import "testing"

func TestStatefulInMemory_interfaceFulfilled(t *testing.T) {
	var _ StatefulMetadata = &StatefulInMemory{}
}
