package store

import (
	"testing"
	"testing_helpers"
	"time"
)

// table of instructions, and whether they're
// a write (true) or read (false)
var isWrite = []struct {
	cmd string
	result bool
}{
	{"GET", false},
	{"SET", true},
	{"DEL", true},
}

// compile time assertion that Value is implemented
func valueInterfaceCheck(v Value) {}

func TestIsWriteCmd(t *testing.T) {
	r := &Redis{}
	for _, c := range isWrite {
		if result := r.IsWriteCommand(c.cmd); result != c.result {
			if result {
				t.Errorf("%v erroneously identified as a write", c.cmd)
			} else {
				t.Errorf("%v not identified as a write", c.cmd)
			}
		}
	}
}

func TestIsReadCmd(t *testing.T) {
	r := &Redis{}
	for _, c := range isWrite {
		if result := r.IsReadCommand(c.cmd); result != !c.result {
			if result {
				t.Errorf("%v erroneously identified as a read", c.cmd)
			} else {
				t.Errorf("%v not identified as a read", c.cmd)
			}
		}
	}
}

/***************** value tests *****************/

// tests the single value
func TestSingleValue(t *testing.T) {
	store := &Redis{}
	src := newSingleValue("blake", time.Now())
	valueInterfaceCheck(src)

	b, err := store.SerializeValue(src)
	if err != nil {
		t.Fatalf("Unexpected serialization error: %v", err)
	}

	val, vtype, err := store.DeserializeValue(b)
	if err != nil {
		t.Fatalf("Unexpected deserialization error: %v", err)
	}
	if vtype != SINGLE_VALUE {
		t.Fatalf("Unexpected value type enum: %v", vtype)
	}
	dst, ok := val.(*singleValue)
	if !ok {
		t.Fatalf("Unexpected value type: %T", val)
	}

	testing_helpers.AssertEqual(t, "data", src.data, dst.data)
	testing_helpers.AssertEqual(t, "time", src.time, dst.time)
}

/***************** query tests *****************/

func setupRedis() *Redis {
	r := NewRedis()
	return r
}

// tests basic function of set
func TestSet(t *testing.T) {

}

// if set is called with a timestamp which is lower than
// the existing value, it should be ignored
func TestSetConflictingTimestamp(t *testing.T) {

}

// tests validation of SET insructions
func TestSetValidation(t *testing.T) {

}

func TestGet(t *testing.T) {
	r := setupRedis()
	expected := newSingleValue("b", time.Now())
	r.data["a"] = expected

	val, err := r.ExecuteRead("GET", "a", []string{})
	if err != nil {
		t.Fatalf("Unexpected error on read: %v", err)
	}
	actual, ok := val.(*singleValue)
	if !ok {
		t.Fatalf("Unexpected value type: %T", val)
	}

	testing_helpers.AssertEqual(t, "data", expected.data, actual.data)
	testing_helpers.AssertEqual(t, "time", expected.time, actual.time)
}

// tests validation of GET insructions
func TestGetValidation(t *testing.T) {
	r := setupRedis()

	// too many args
	val, err := r.ExecuteRead("GET", "a", []string{"b"})
	if val != nil {
		t.Errorf("Unexpected non-nil value")
	}
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}
