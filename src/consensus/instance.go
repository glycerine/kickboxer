package consensus

import (
	"encoding/binary"
	"time"
)

import (
	"code.google.com/p/go-uuid/uuid"
)

import (
	"node"
	"serializer"
	"store"
	"bufio"
)

type InstanceStatus byte

const (
	_ = InstanceStatus(iota)
	INSTANCE_PREACCEPTED
	INSTANCE_ACCEPTED
	INSTANCE_COMMITTED
	INSTANCE_EXECUTED
)

type InstanceID string

func NewInstanceID() InstanceID {
	return InstanceID(uuid.NewUUID())
}

func (i InstanceID) UUID() uuid.UUID {
	return uuid.UUID(i)
}

func (i InstanceID) String() string {
	return i.UUID().String()
}

type InstanceIDSet map[InstanceID]bool

func NewInstanceIDSet(ids []InstanceID) InstanceIDSet {
	s := make(InstanceIDSet, len(ids))
	for _, id := range ids {
		s[id] = true
	}
	return s
}

func (i InstanceIDSet) Equal(o InstanceIDSet) bool {
	if len(i) != len(o) {
		return false
	}
	for key := range i {
		if !o[key] {
			return false
		}
	}
	return true
}

func (i InstanceIDSet) Union(o InstanceIDSet) InstanceIDSet {
	u := make(InstanceIDSet, (len(i)*3)/2)
	for k := range i {
		u[k] = true
	}
	for k := range o {
		u[k] = true
	}
	return u
}

func (i InstanceIDSet) Add(ids ...InstanceID) {
	for _, id := range ids {
		i[id] = true
	}
}

// returns all of the keys in i, that aren't in o
func (i InstanceIDSet) Subtract(o InstanceIDSet) InstanceIDSet {
	s := NewInstanceIDSet([]InstanceID{})
	for key := range i {
		if !o.Contains(key) {
			s.Add(key)
		}
	}
	return s
}

func (i InstanceIDSet) Contains(id InstanceID) bool {
	_, exists := i[id]
	return exists
}

func (i InstanceIDSet) List() []InstanceID {
	l := make([]InstanceID, 0, len(i))
	for k := range i {
		l = append(l, k)
	}
	return l
}

func (i InstanceIDSet) String() string {
	s := "{"
	n := 0
	for k := range i {
		if n > 0 {
			s += ", "
		}
		s += k.String()
		n++
	}
	s += "}"
	return s
}

type InstanceMap map[InstanceID]*Instance

func NewInstanceMap() InstanceMap {
	return make(InstanceMap)
}

func (i InstanceMap) Add(instance *Instance) {
	i[instance.InstanceID] = instance
}

func (i InstanceMap) Remove(instance *Instance) {
	delete(i, instance.InstanceID)
}

func (i InstanceMap) RemoveID(id InstanceID) {
	delete(i, id)
}

func (i InstanceMap) ContainsID(id InstanceID) bool {
	_, exists := i[id]
	return exists
}

func (i InstanceMap) Contains(instance *Instance) bool {
	return i.ContainsID(instance.InstanceID)
}

func (i InstanceMap) InstanceIDs() []InstanceID {
	arr := make([]InstanceID, 0, len(i))
	for key := range i {
		arr = append(arr, key)
	}
	return arr
}

// a serializable set of instructions
type Instance struct {
	// the uuid of this instance
	InstanceID InstanceID

	// the node id of the instance leader
	LeaderID node.NodeId

	// the Instructions(s) to be executed
	Commands []*store.Instruction

	// a list of other instance ids that
	// execution of this instance depends on
	Dependencies []InstanceID

	// the sequence number of this instance (like an array index)
	Sequence uint64

	// the current status of this instance
	Status InstanceStatus

	// the highest seen message number for this instance
	MaxBallot uint32

	// indicates that the paxos protocol
	// for this instance failed, and this
	// instance should be ignored
	Noop bool

	// indicates that the dependencies from the leader
	// matched the replica's local dependencies. This
	// is used when there are only 3 replicas and another
	// replica takes over leadership for a command.
	// Even if the old command leader is unreachable,
	// the new leader will know that at least 2 replicas
	// had identical dependency graphs at the time of proposal,
	// it may still be useful in other situations
	// * not message serialized *
	DependencyMatch bool

	// indicates the time that we can stop waiting
	// for a commit on this command, and force one
	// * not message serialized *
	commitTimeout time.Time

	// indicates the time that we can stop waiting for the
	// the command to be executed by ExecuteQuery
	// * not message serialized *
	executeTimeout time.Time
}

// merges sequence and dependencies onto this instance, and returns
// true/false to indicate if there were any changes
func (i *Instance) mergeAttributes(seq uint64, deps []InstanceID) bool {
	changes := false
	if seq > i.Sequence {
		changes = true
		i.Sequence = seq
	}
	iSet := NewInstanceIDSet(i.Dependencies)
	oSet := NewInstanceIDSet(deps)
	if !iSet.Equal(oSet) {
		changes = true
		union := iSet.Union(oSet)
		i.Dependencies = make([]InstanceID, 0, len(union))
		for id := range union {
			i.Dependencies = append(i.Dependencies, id)
		}
	}
	return changes
}

func instructionSerialize(instruction *store.Instruction, buf *bufio.Writer) error {
	if err := serializer.WriteFieldString(buf, instruction.Cmd); err != nil { return err }
	if err := serializer.WriteFieldString(buf, instruction.Key); err != nil { return err }
	numArgs := uint32(len(instruction.Args))
	if err := binary.Write(buf, binary.LittleEndian, &numArgs); err != nil { return err }
	for _, arg := range instruction.Args {
		if err := serializer.WriteFieldString(buf, arg); err != nil { return err }
	}
	if err := serializer.WriteTime(buf, instruction.Timestamp); err != nil { return err }
	return nil
}

func instructionDeserialize(buf *bufio.Reader) (*store.Instruction, error) {
	instruction := &store.Instruction{}
	if val, err := serializer.ReadFieldString(buf); err != nil { return nil, err } else {
		instruction.Cmd = val
	}
	if val, err := serializer.ReadFieldString(buf); err != nil { return nil, err } else {
		instruction.Key = val
	}

	var numArgs uint32
	if err := binary.Read(buf, binary.LittleEndian, &numArgs); err != nil { return nil, err }
	instruction.Args = make([]string, numArgs)
	for i := range instruction.Args {
		if val, err := serializer.ReadFieldString(buf); err != nil { return nil, err } else {
			instruction.Args[i] = val
		}
	}
	if val, err := serializer.ReadTime(buf); err != nil { return nil, err } else {
		instruction.Timestamp = val
	}
	return instruction, nil
}

func instanceLimitedSerialize(instance *Instance, buf *bufio.Writer) error {
	if err := serializer.WriteFieldString(buf, string(instance.InstanceID)); err != nil { return err }
	if err := serializer.WriteFieldString(buf, string(instance.LeaderID)); err != nil { return err }
	numInstructions := uint32(len(instance.Commands))
	if err := binary.Write(buf, binary.LittleEndian, &numInstructions); err != nil { return err }
	for _, inst := range instance.Commands {
		if err := instructionSerialize(inst, buf); err != nil { return err }
	}
	numDeps := uint32(len(instance.Dependencies))
	if err := binary.Write(buf, binary.LittleEndian, &numDeps); err != nil { return err }
	for _, dep := range instance.Dependencies {
		if err := serializer.WriteFieldString(buf, string(dep)); err != nil { return err }
	}

	if err := binary.Write(buf, binary.LittleEndian, &instance.Sequence); err != nil { return err }
	if err := binary.Write(buf, binary.LittleEndian, &instance.Status); err != nil { return err }
	if err := binary.Write(buf, binary.LittleEndian, &instance.MaxBallot); err != nil { return err }

	var noop byte
	if instance.Noop { noop = 0xff }
	if err := binary.Write(buf, binary.LittleEndian, &noop); err != nil { return err }

	var match byte
	if instance.DependencyMatch { match = 0xff }
	if err := binary.Write(buf, binary.LittleEndian, &match); err != nil { return err }

	return nil
}

func instanceLimitedDeserialize(buf *bufio.Reader) (*Instance, error) {
	instance := &Instance{}
	if val, err := serializer.ReadFieldString(buf); err != nil { return nil, err } else {
		instance.InstanceID = InstanceID(val)
	}
	if val, err := serializer.ReadFieldString(buf); err != nil { return nil, err } else {
		instance.LeaderID = node.NodeId(val)
	}

	var numInstructions uint32
	if err := binary.Read(buf, binary.LittleEndian, &numInstructions); err != nil { return nil, err }
	instance.Commands = make([]*store.Instruction, numInstructions)
	for i := range instance.Commands {
		instr, err := instructionDeserialize(buf)
		if err != nil { return nil, err }
		instance.Commands[i] = instr
	}

	var numDeps uint32
	if err := binary.Read(buf, binary.LittleEndian, &numDeps); err != nil { return nil, err }
	instance.Dependencies = make([]InstanceID, numDeps)
	for i := range instance.Dependencies {
		if dep, err := serializer.ReadFieldString(buf); err != nil { return nil, err } else {
			instance.Dependencies[i] = InstanceID(dep)
		}
	}

	if err := binary.Read(buf, binary.LittleEndian, &instance.Sequence); err != nil { return nil, err }
	if err := binary.Read(buf, binary.LittleEndian, &instance.Status); err != nil { return nil, err }
	if err := binary.Read(buf, binary.LittleEndian, &instance.MaxBallot); err != nil { return nil, err }

	var noop byte
	if err := binary.Read(buf, binary.LittleEndian, &noop); err != nil { return nil, err }
	instance.Noop = noop != 0x0

	var match byte
	if err := binary.Read(buf, binary.LittleEndian, &match); err != nil { return nil, err }
	instance.DependencyMatch = match != 0x0

	return instance, nil
}

func instanceSerialize(instance *Instance, buf *bufio.Writer) error {
	if err := instanceLimitedSerialize(instance, buf); err != nil { return err }
	if err := serializer.WriteTime(buf, instance.commitTimeout); err != nil { return err }
	if err := serializer.WriteTime(buf, instance.executeTimeout); err != nil { return err }

	return nil
}

func instanceDeserialize(buf *bufio.Reader) (*Instance, error) {
	var instance *Instance
	if val, err := instanceLimitedDeserialize(buf); err != nil { return nil, err } else {
		instance = val
	}
	if val, err := serializer.ReadTime(buf); err != nil { return nil, err } else {
		instance.commitTimeout = val
	}
	if val, err := serializer.ReadTime(buf); err != nil { return nil, err } else {
		instance.executeTimeout = val
	}

	return instance, nil
}
