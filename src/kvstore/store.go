package kvstore

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
)

import (
	"store"
)


// read instructions
const (
	GET		= "GET"
)

// write instructions
const (
	SET		= "SET"
	DEL		= "DEL"
)


type KVStore struct {

	data map[string] store.Value

	// TODO: delete
	// temporary lock, used until
	// things are broken out into
	// goroutines
	lock sync.RWMutex

}

var _ = store.Store(&KVStore{})

func NewKVStore() *KVStore {
	r := &KVStore{
		data:make(map[string] store.Value),
	}
	return r
}

func (s *KVStore) SerializeValue(v store.Value) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := WriteValue(buf, v) ; err != nil { return nil, err }
	return buf.Bytes(), nil
}

func (s *KVStore) DeserializeValue(b []byte) (store.Value, store.ValueType, error) {
	buf := bytes.NewBuffer(b)
	val, vtype, err := ReadValue(buf)
	if err != nil { return nil, "", err }
	return val, vtype, nil
}

func (s *KVStore) Start() error {
	return nil
}

func (s *KVStore) Stop() error {
	return nil
}

func (s *KVStore) ExecuteInstruction(instruction store.Instruction) (store.Value, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	cmd := instruction.Cmd
	key := instruction.Key
	args := instruction.Args
	timestamp := instruction.Timestamp

	switch cmd {
	case GET:
		//
		if err := s.validateGet(key, args); err != nil { return nil, err }
		rval, err := s.get(key)
		if err != nil { return nil, err }
		return rval, nil
	case SET:
		if err := s.validateSet(key, args, timestamp); err != nil { return nil, err }
		return s.set(key, args[0], timestamp), nil
	case DEL:
		if err := s.validateDel(key, args, timestamp); err != nil { return nil, err }
		return s.del(key, timestamp)
	default:
		return nil, fmt.Errorf("Unrecognized write command: %v", cmd)
	}
	return nil, nil
}

// reconciles multiple values and returns instructions for correcting
// the values on inaccurate nodes
//
// Reconcile should handle value maps with one value without hitting
// a value type specific reconciliation function
//
// value type specific reconciliation functions should be able to handle
// getting unfamiliar types, but can operate under the assumption that if
// they're being called, the oldest timestamp of the given values belongs
// to a value of it's type.
func (s *KVStore) Reconcile(key string, values []store.Value) (store.Value, [][]store.Instruction, error) {
	switch len(values){
	case 0:
		return nil, nil, fmt.Errorf("At least one value must be provided")
	case 1:
		var val store.Value
		for _, v := range values { val = v }
		return val, nil, nil
	default:
		highValue := getHighValue(values)

		switch highValue.GetValueType() {
		case STRING_VALUE:
			return reconcileString(key, highValue.(*String), values)
		case TOMBSTONE_VALUE:
			return reconcileTombstone(key, highValue.(*Tombstone), values)
		default:
			return nil, [][]store.Instruction{}, fmt.Errorf("Unknown value type: %T", highValue)
		}
	}
	return nil, [][]store.Instruction{}, nil
}

func (s *KVStore) IsReadOnly(instruction store.Instruction) bool {
	switch strings.ToUpper(instruction.Cmd) {
	case GET:
		return true
	}
	return false
}

func (s *KVStore) IsWriteOnly(instruction store.Instruction) bool {
	switch strings.ToUpper(instruction.Cmd) {
	case SET, DEL:
		return true
	}
	return false
}

func (s *KVStore) ReturnsValue(cmd string) bool {
	switch strings.ToUpper(cmd) {
	case GET:
		return true
	}
	return false
}

func (s *KVStore) InterferingKeys(instruction store.Instruction) []string {
	return []string{instruction.Key}
}

// ----------- data import / export -----------


// blindly gets the contents of the given key
func (s *KVStore) GetRawKey(key string) (store.Value, error) {
	val, ok := s.data[key]
	if !ok {
		return nil, fmt.Errorf("key [%v] does not exist", key)
	}
	return val, nil
}

// blindly sets the contents of the given key
func (s *KVStore) SetRawKey(key string, val store.Value) error {
	s.data[key] = val
	return nil
}

// returns all of the keys held by the store, including keys containing
// tombstones
func (s *KVStore) GetKeys() []string {
	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	return keys
}

func (s *KVStore) KeyExists(key string) bool {
	_, ok := s.data[key]
	return ok
}
