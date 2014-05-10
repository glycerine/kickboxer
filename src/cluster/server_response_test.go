package cluster

import (
	"testing"
)

import (
	"node"
	"testing_helpers"
)

func TestServerDiscoverPeersResponse(t *testing.T) {
	c := makeRing(5, 3)
	server := &PeerServer{cluster:c}

	n := NewRemoteNodeInfo(
		node.NewNodeId(),
		"DC1",
		c.partitioner.GetToken("asdfghjkl"),
		"New Node",
		"127.0.0.5:9999",
		c,
	)
	msg := &DiscoverPeersRequest{NodeId:n.GetId()}
	response, err := server.executeRequest(n, msg)

	if err != nil {
		t.Fatalf("Unexpected error executing request: %v", err)
	}

	peerResponse, ok := response.(*DiscoverPeerResponse)
	if !ok {
		t.Fatalf("Unexpected response type: %T", response)
	}

	testing_helpers.AssertEqual(t, "num peers", len(c.getPeerData()), len(peerResponse.Peers))
}

func TestStreamRequestResonse(t *testing.T) {

}


