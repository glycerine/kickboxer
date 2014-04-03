package consensus

import (
	"time"
)

import (
	"launchpad.net/gocheck"
)

import (
	"store"
)

type baseDependencyTest struct {
	baseManagerTest
}

func (s *baseDependencyTest) newInstruction(key string) store.Instruction {
	return store.NewInstruction("SET", key, []string{}, time.Now())
}

type DependencyMapTest struct {
	baseDependencyTest
}

var _ = gocheck.Suite(&DependencyMapTest{})

// tests that a new dependencies object is created for new root nodes
func (s *DependencyMapTest) TestNewRootDependencyMap(c *gocheck.C) {
	instance := s.manager.makeInstance(s.newInstruction("a"))

	c.Assert(s.manager.depsMngr.deps.deps["a"], gocheck.IsNil)

	deps, err := s.manager.depsMngr.GetAndSetDeps(instance)
	c.Assert(err, gocheck.IsNil)
	c.Assert(deps, gocheck.NotNil)
	c.Assert(deps, gocheck.DeepEquals, []InstanceID{})

	c.Assert(s.manager.depsMngr.deps.deps["a"], gocheck.NotNil)
}

// tests that an existing dependencies object is used for a key if it exists
func (s *DependencyMapTest) TestExistingRootDependencyMap(c *gocheck.C) {
	instance := s.manager.makeInstance(s.newInstruction("a"))

	depsNode := s.manager.depsMngr.deps.get("a")
	lastWrite := NewInstanceID()
	depsNode.lastWrite = lastWrite
	c.Assert(s.manager.depsMngr.deps.deps["a"], gocheck.NotNil)

	deps, err := s.manager.depsMngr.GetAndSetDeps(instance)
	c.Assert(err, gocheck.IsNil)
	c.Assert(deps, gocheck.NotNil)
	c.Assert(deps, gocheck.DeepEquals, []InstanceID{lastWrite})

	c.Assert(s.manager.depsMngr.deps.deps["a"], gocheck.NotNil)
	c.Assert(s.manager.depsMngr.deps.get("a"), gocheck.Equals, depsNode)

}

type DependenciesTest struct {
	baseDependencyTest
}

var _ = gocheck.Suite(&DependenciesTest{})

// tests that a new dependencies object is created for new leaf nodes
func (s *DependenciesTest) TestNewDependencyMap(c *gocheck.C) {
	instance := s.manager.makeInstance(s.newInstruction("a:b"))
	keys := []string{"a", "b"}
	deps := newDependencies()

	c.Assert(len(deps.subDependencies.deps), gocheck.Equals, 0)
	c.Assert(deps.subDependencies.deps["b"], gocheck.IsNil)

	deps.GetAndSetDeps(keys, instance)

	c.Assert(len(deps.subDependencies.deps), gocheck.Equals, 1)
	c.Assert(deps.subDependencies.deps["b"], gocheck.NotNil)
}

// tests that existing dependencies object is used if it exists
func (s *DependenciesTest) TestExistingDependencyMap(c *gocheck.C) {
	instance := s.manager.makeInstance(s.newInstruction("a:b"))
	keys := []string{"a", "b"}
	deps := newDependencies()


	bdeps := deps.subDependencies.get("b")
	c.Assert(len(deps.subDependencies.deps), gocheck.Equals, 1)
	c.Assert(deps.subDependencies.deps["b"], gocheck.NotNil)

	deps.GetAndSetDeps(keys, instance)

	c.Assert(len(deps.subDependencies.deps), gocheck.Equals, 1)
	c.Assert(deps.subDependencies.deps["b"], gocheck.NotNil)
	c.Assert(deps.subDependencies.get("b"), gocheck.Equals, bdeps)
}

// tests the last reads array is updated if the instance is a read
func (s *DependenciesTest) TestLastKeyReadIsUpdated(c *gocheck.C) {
	instance := s.manager.makeInstance(s.newInstruction("a"))
	instance.ReadOnly = true
	keys := []string{"a"}
	deps := newDependencies()

	c.Assert(deps.lastWrite.IsZero(), gocheck.Equals, true)
	c.Assert(deps.lastReads, gocheck.DeepEquals, []InstanceID{})

	deps.GetAndSetDeps(keys, instance)

	expectedReads := []InstanceID{instance.InstanceID}
	c.Assert(deps.lastWrite.IsZero(), gocheck.Equals, true)
	c.Assert(deps.lastReads, gocheck.DeepEquals, expectedReads)
}

// tests the last write is updated if the instance is a write
func (s *DependenciesTest) TestLastKeyWriteIsUpdated(c *gocheck.C) {
	instance := s.manager.makeInstance(s.newInstruction("a"))
	keys := []string{"a"}
	deps := newDependencies()

	c.Assert(deps.lastWrite.IsZero(), gocheck.Equals, true)
	c.Assert(deps.lastReads, gocheck.DeepEquals, []InstanceID{})

	deps.GetAndSetDeps(keys, instance)

	c.Assert(deps.lastWrite, gocheck.Equals, instance.InstanceID)
	c.Assert(deps.lastReads, gocheck.DeepEquals, []InstanceID{})
}

// tests the deps reported by a single deps node for a read
func (s *DependenciesTest) TestLocalReadDeps(c *gocheck.C) {
	depsNode := newDependencies()
	depsNode.lastWrite = NewInstanceID()
	depsNode.lastReads = []InstanceID{NewInstanceID(), NewInstanceID()}

	instance := s.manager.makeInstance(s.newInstruction("a"))
	instance.ReadOnly = true

	deps := depsNode.getLocalDeps(instance)

	c.Assert(deps, gocheck.DeepEquals, []InstanceID{depsNode.lastWrite})
}

// tests the deps reported by a single deps node for a write
func (s *DependenciesTest) TestLocalWriteDeps(c *gocheck.C) {
	depsNode := newDependencies()
	depsNode.lastWrite = NewInstanceID()
	depsNode.lastReads = []InstanceID{NewInstanceID(), NewInstanceID()}

	expected := NewInstanceIDSet(depsNode.lastReads)
	expected.Add(depsNode.lastWrite)

	instance := s.manager.makeInstance(s.newInstruction("a"))

	deps := depsNode.getLocalDeps(instance)

	actual := NewInstanceIDSet(deps)

	c.Assert(actual, gocheck.DeepEquals, expected)
}

func (s *DependenciesTest) TestIntegration(c *gocheck.C) {
	// the key "a:b" is the key being used for tests
	addInstance := func(key string, readOnly bool) InstanceID {
		instance := s.manager.makeInstance(s.newInstruction(key))
		instance.ReadOnly = readOnly
		_, err := s.manager.depsMngr.GetAndSetDeps(instance)
		c.Assert(err, gocheck.IsNil)
		return instance.InstanceID
	}

	aWrite := addInstance("a", false)
	aRead := addInstance("a", true)
	abWrite := addInstance("a:b", false)
	abRead := addInstance("a:b", true)
	abcWrite := addInstance("a:b:c", false)
	abcRead := addInstance("a:b:c", true)
	abcdWrite := addInstance("a:b:c:d", false)
	abcdRead := addInstance("a:b:c:d", true)

	// add sibling deps, these should never be returned
	addInstance("a:b1", false)
	addInstance("a:b1", true)

	// check read deps
	readInstance := s.manager.makeInstance(s.newInstruction("a:b"))
	readInstance.ReadOnly = true
	expected := NewInstanceIDSet([]InstanceID{aWrite, abWrite, abcWrite, abcdWrite})
	deps, err := s.manager.depsMngr.GetAndSetDeps(readInstance)
	c.Assert(err, gocheck.IsNil)
	actual := NewInstanceIDSet(deps)
	c.Check(actual, gocheck.DeepEquals, expected)

	// check write deps
	writeInstance := s.manager.makeInstance(s.newInstruction("a:b"))
	expected.Add(aRead, abRead, abcRead, abcdRead, readInstance.InstanceID)
	deps, err = s.manager.depsMngr.GetAndSetDeps(writeInstance)
	c.Assert(err, gocheck.IsNil)
	actual = NewInstanceIDSet(deps)
	c.Check(actual, gocheck.DeepEquals, expected)

	// check children have been cleared
	expected = NewInstanceIDSet([]InstanceID{aWrite, aRead, writeInstance.InstanceID})
	writeInstance = s.manager.makeInstance(s.newInstruction("a:b"))
	deps, err = s.manager.depsMngr.GetAndSetDeps(writeInstance)
	c.Assert(err, gocheck.IsNil)
	actual = NewInstanceIDSet(deps)
	c.Check(actual, gocheck.DeepEquals, expected)

}
