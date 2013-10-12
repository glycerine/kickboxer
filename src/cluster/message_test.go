package cluster

import (
	"bytes"
	"fmt"
	"testing"
	"time"
	"code.google.com/p/go-uuid/uuid"
)


func messageInterfaceCheck(_ Message) {}


func TestConnectionRequest(t *testing.T) {
	buf := &bytes.Buffer{}
	src := &ConnectionRequest{PeerData{
		NodeId:NewNodeId(),
		Addr:"127.0.0.1:9999",
		Name:"Test Node",
		Token:Token([]byte{0,1,2,3,4,5,6,7,0,1,2,3,4,5,6,7}),
	}}

	// interface check
	messageInterfaceCheck(src)

	// write, then read message
	if err := WriteMessage(buf, src); err != nil {
		t.Fatalf("unexpected Serialize error: %v", err)
	}
	msg, mtype, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("unexpected Deserialize error: %v", err)
	}
	if mtype != CONNECTION_REQUEST {
		t.Fatalf("unexpected message type enum: %v", mtype)
	}
	dst, ok := msg.(*ConnectionRequest)
	if !ok {
		t.Fatalf("unexpected message type %T", msg)
	}

	// check values
	equalityCheck(t, "Type", CONNECTION_REQUEST, dst.GetType())
	equalityCheck(t, "NodeId", src.NodeId, dst.NodeId)
	equalityCheck(t, "Addr", src.Addr, dst.Addr)
	equalityCheck(t, "Name", src.Name, dst.Name)
	sliceEqualityCheck(t, "Token", src.Token, dst.Token)

}


func TestConnectionAcceptedResponse(t *testing.T) {
	buf := &bytes.Buffer{}
	src := &ConnectionAcceptedResponse{
		NodeId:NewNodeId(),
		Name:"Test Node",
		Token:Token([]byte{0,1,2,3,4,5,6,7,0,1,2,3,4,5,6,7}),
	}

	// interface check
	messageInterfaceCheck(src)

	// write, then read message
	if err := WriteMessage(buf, src); err != nil {
		t.Fatalf("unexpected Serialize error: %v", err)
	}
	msg, mtype, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("unexpected Deserialize error: %v", err)
	}
	if mtype != CONNECTION_ACCEPTED_RESPONSE {
		t.Fatalf("unexpected message type enum: %v", mtype)
	}
	dst, ok := msg.(*ConnectionAcceptedResponse)
	if !ok {
		t.Fatalf("unexpected message type %T", msg)
	}

	// check value
	equalityCheck(t, "Type", CONNECTION_ACCEPTED_RESPONSE, dst.GetType())
	equalityCheck(t, "NodeId", src.NodeId, dst.NodeId)
	equalityCheck(t, "Name", src.Name, dst.Name)
	sliceEqualityCheck(t, "Token", src.Token, dst.Token)
}

func TestConnectionRefusedResponse(t *testing.T) {
	buf := &bytes.Buffer{}
	src := &ConnectionRefusedResponse{Reason:"you suck"}

	// interface check
	messageInterfaceCheck(src)

	// write, then read message
	if err := WriteMessage(buf, src); err != nil {
		t.Fatalf("unexpected Serialize error: %v", err)
	}
	msg, mtype, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("unexpected Deserialize error: %v", err)
	}
	if mtype != CONNECTION_REFUSED_RESPONSE {
		t.Fatalf("unexpected message type enum: %v", mtype)
	}
	dst, ok := msg.(*ConnectionRefusedResponse)
	if !ok {
		t.Fatalf("unexpected message type %T", msg)
	}

	// check value
	equalityCheck(t, "Type", CONNECTION_REFUSED_RESPONSE, dst.GetType())
	equalityCheck(t, "Reason", src.Reason, dst.Reason)
}

func TestDiscoverPeersRequest(t *testing.T) {
	buf := &bytes.Buffer{}
	src := &DiscoverPeersRequest{
		NodeId:NewNodeId(),
	}

	// interface check
	messageInterfaceCheck(src)

	// write, then read message
	if err := WriteMessage(buf, src); err != nil {
		t.Fatalf("unexpected Serialize error: %v", err)
	}
	msg, mtype, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("unexpected Deserialize error: %v", err)
	}
	if mtype != DISCOVER_PEERS_REQUEST {
		t.Fatalf("unexpected message type enum: %v", mtype)
	}
	dst, ok := msg.(*DiscoverPeersRequest)
	if !ok {
		t.Fatalf("unexpected message type %T", msg)
	}

	equalityCheck(t, "Type", DISCOVER_PEERS_REQUEST, dst.GetType())
	equalityCheck(t, "NodeId", src.NodeId, dst.NodeId)
}


func TestDiscoverPeersResponse(t *testing.T) {
	buf := &bytes.Buffer{}
	src := &DiscoverPeerResponse{
		Peers: []*PeerData{
			&PeerData{
				NodeId:NewNodeId(),
				Addr:"127.0.0.1:9998",
				Name:"Test Node1",
				Token:Token([]byte{0,1,2,3,4,5,6,7,0,1,2,3,4,5,6,7}),
			},
			&PeerData{
				NodeId:NewNodeId(),
				Addr:"127.0.0.1:9999",
				Name:"Test Node2",
				Token:Token([]byte{1,2,3,4,5,6,7,0,1,2,3,4,5,6,7,0}),
			},
		},
	}

	// interface check
	messageInterfaceCheck(src)

	// write, then read message
	if err := WriteMessage(buf, src); err != nil {
		t.Fatalf("unexpected Serialize error: %v", err)
	}
	msg, mtype, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("unexpected Deserialize error: %v", err)
	}
	if mtype != DISCOVER_PEERS_RESPONSE {
		t.Fatalf("unexpected message type enum: %v", mtype)
	}
	dst, ok := msg.(*DiscoverPeerResponse)
	if !ok {
		t.Fatalf("unexpected message type %T", msg)
	}

	equalityCheck(t, "Type", DISCOVER_PEERS_RESPONSE, dst.GetType())
	for i:=0; i<2; i++ {
		s := src.Peers[i]
		d := dst.Peers[i]

		equalityCheck(t, fmt.Sprintf("NodeId:%v", i), s.NodeId, d.NodeId)
		equalityCheck(t, fmt.Sprintf("Addr:%v", i), s.Addr, d.Addr)
		equalityCheck(t, fmt.Sprintf("Name:%v", i), s.Name, d.Name)
		sliceEqualityCheck(t, fmt.Sprintf("Token:%v", i), s.Token, d.Token)
	}

}

func TestReadRequest(t *testing.T) {
	buf := &bytes.Buffer{}
	src := &ReadRequest{
		Cmd:"GET",
		Key:"A",
		Args:[]string{"B", "C"},
	}

	// interface check
	messageInterfaceCheck(src)

	// write, then read message
	if err := WriteMessage(buf, src); err != nil {
		t.Fatalf("unexpected Serialize error: %v", err)
	}
	msg, mtype, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("unexpected Deserialize error: %v", err)
	}
	if mtype != READ_REQUEST {
		t.Fatalf("unexpected message type enum: %v", mtype)
	}
	dst, ok := msg.(*ReadRequest)
	if !ok {
		t.Fatalf("unexpected message type %T", msg)
	}

	equalityCheck(t, "Type", READ_REQUEST, dst.GetType())
	equalityCheck(t, "Cmd", src.Cmd, dst.Cmd)
	equalityCheck(t, "Key", src.Key, dst.Key)
	equalityCheck(t, "Arg len", len(src.Args), len(dst.Args))
	equalityCheck(t, "Arg[0]", src.Args[0], dst.Args[0])
	equalityCheck(t, "Arg[1]", src.Args[1], dst.Args[1])
}

func TestWriteRequest(t *testing.T) {
	buf := &bytes.Buffer{}
	src := &WriteRequest{
		ReadRequest:ReadRequest{
			Cmd:"GET",
			Key:"A",
			Args:[]string{"B", "C"},
		},
		Timestamp:time.Now(),
	}

	// interface check
	messageInterfaceCheck(src)

	// write, then read message
	if err := WriteMessage(buf, src); err != nil {
		t.Fatalf("unexpected Serialize error: %v", err)
	}
	msg, mtype, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("unexpected Deserialize error: %v", err)
	}
	if mtype != WRITE_REQUEST {
		t.Fatalf("unexpected message type enum: %v", mtype)
	}
	dst, ok := msg.(*WriteRequest)
	if !ok {
		t.Fatalf("unexpected message type %T", msg)
	}

	equalityCheck(t, "Type", WRITE_REQUEST, dst.GetType())
	equalityCheck(t, "Cmd", src.Cmd, dst.Cmd)
	equalityCheck(t, "Key", src.Key, dst.Key)
	equalityCheck(t, "Arg len", len(src.Args), len(dst.Args))
	equalityCheck(t, "Arg[0]", src.Args[0], dst.Args[0])
	equalityCheck(t, "Arg[1]", src.Args[1], dst.Args[1])
	equalityCheck(t, "Timestamp", src.Timestamp, dst.Timestamp)
}

func TestQueryResponse(t *testing.T) {
	buf := &bytes.Buffer{}
	src := &QueryResponse{
		Data:[][]byte{
			[]byte(uuid.NewRandom()),
			[]byte(uuid.NewRandom()),
			[]byte(uuid.NewRandom()),
		},
	}

	// interface check
	messageInterfaceCheck(src)

	// write, then read message
	if err := WriteMessage(buf, src); err != nil {
		t.Fatalf("unexpected Serialize error: %v", err)
	}
	msg, mtype, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("unexpected Deserialize error: %v", err)
	}
	if mtype != QUERY_RESPONSE {
		t.Fatalf("unexpected message type enum: %v", mtype)
	}
	dst, ok := msg.(*QueryResponse)
	if !ok {
		t.Fatalf("unexpected message type %T", msg)
	}

	equalityCheck(t, "Type", QUERY_RESPONSE, dst.GetType())
	equalityCheck(t, "Data len", len(src.Data), len(dst.Data))
	sliceEqualityCheck(t, "Data[0]", src.Data[0], dst.Data[0])
	sliceEqualityCheck(t, "Data[1]", src.Data[1], dst.Data[1])
	sliceEqualityCheck(t, "Data[2]", src.Data[2], dst.Data[2])
}