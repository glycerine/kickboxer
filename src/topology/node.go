package topology

import (
	"node"
	"partitioner"
)

type NodeStatus string

const (
	NODE_INITIALIZING 	= NodeStatus("")
	NODE_UP 			= NodeStatus("UP")
	NODE_DOWN 			= NodeStatus("DOWN")
)

type TopologyNode interface {
	node.Node

	Name() string
	GetToken() partitioner.Token
	GetDatacenterId() DatacenterID
	GetStatus() NodeStatus
}
