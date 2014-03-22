package consensus

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

import (
	"store"
)

// sorts the strongly connected subgraph components
type iidSorter struct {
	depMap map[InstanceID]*Instance
	iids []InstanceID
}

func (i *iidSorter) Len() int {
	return len(i.iids)
}

// returns true if the item at index x is less than
// the item and index y
func (i *iidSorter) Less(x, y int) bool {
	i0 := i.depMap[i.iids[x]]
	i1 := i.depMap[i.iids[y]]

	// first check the sequence#
	if i0s, i1s := i0.getSeq(), i1.getSeq(); i0s != i1s {
		return i0s < i1s
	} else {
		// then the embedded timestamp
		t0 := i0.InstanceID.Time()
		t1 := i1.InstanceID.Time()
		if t0 != t1 {
			return t0 < t1
		} else {
			// finally the lexicographic comparison
			return bytes.Compare(i0.InstanceID.Bytes(), i1.InstanceID.Bytes()) == -1
		}
	}
	return false
}

// switches the position of nodes at indices i & j
func (i *iidSorter) Swap(x, y int) {
	i.iids[x], i.iids[y] = i.iids[y], i.iids[x]
}

// topologically sorts instance dependencies, grouped by strongly
// connected components
func (m *Manager) getExecutionOrder(instance *Instance) ([]InstanceID, error) {
	start := time.Now()
	defer m.statsTiming("execute.dependencies.order.time", start)

	// build a directed graph
	targetDeps := instance.getDependencies()
	targetDepSet := NewInstanceIDSet(targetDeps)
	depMap := make(map[InstanceID]*Instance, len(targetDeps) * m.inProgress.Len())
	depMap = m.instances.GetMap(depMap, targetDeps)
	depGraph := make(map[InstanceID][]InstanceID, m.inProgress.Len() + m.committed.Len() + 1)
	var addInstance func(*Instance) error
	addInstance = func(inst *Instance) error {
		deps := inst.getDependencies()
		depMap = m.instances.GetMap(depMap, deps)

		// if the instance is already executed, and it's not a dependency
		// of the target execution instance, only add it to the dep graph
		// if it's connected to an uncommitted instance, since that will
		// make it part of a strongly connected subgraph of at least one
		// unexecuted instance, and will therefore affect the execution
		// ordering
		if inst.getStatus() == INSTANCE_EXECUTED {
			if !targetDepSet.Contains(inst.InstanceID) {
				connected := false
				for _, dep := range deps {
					_, exists := depGraph[dep]
					notExecuted := depMap[dep].getStatus() < INSTANCE_EXECUTED
					targetDep := targetDepSet.Contains(dep)
					connected = connected || exists || notExecuted || targetDep
				}
				if !connected {
					return nil
				}
			}
		}
		depGraph[inst.InstanceID] = deps
		for _, iid := range deps {
			// don't add instances that are already in the graph,
			// strongly connected instances would create an infinite loop
			if _, exists := depGraph[iid]; exists {
				continue
			}

			inst := depMap[iid]
			if inst == nil {
				return fmt.Errorf("getExecutionOrder: Unknown instance id: %v", iid)
			}
			if err := addInstance(inst); err != nil {
				return err
			}
		}
		return nil
	}

	// add ALL instances to the dependency graph, there may
	// be instances higher upstream that change the ordering
	// of instances down here.
	// TODO: figure out why that is
	prepStart := time.Now()
	if err := addInstance(instance); err != nil {
		return nil, err
	}
	m.statsTiming("execute.dependencies.order.sort.prep.time", prepStart)

	// sort with tarjan's algorithm
	sortStart := time.Now()
	tSorted := TarjanSort(depGraph)
	m.statsTiming("execute.dependencies.order.sort.tarjan.time", sortStart)
	subSortStart := time.Now()
	exOrder := make([]InstanceID, 0, len(depGraph))
	for _, iids := range tSorted {
		sorter := &iidSorter{depMap: depMap, iids:iids}
		sort.Sort(sorter)
		exOrder = append(exOrder, sorter.iids...)
	}
	m.statsTiming("execute.dependencies.order.sort.sub_graph.time", subSortStart)
	m.statsTiming("execute.dependencies.order.sort.time", sortStart)

	for _, iid := range exOrder {
		if m.instances.Get(iid) == nil {
			return nil, fmt.Errorf("getExecutionOrder: Unknown instance id: %v", iid)
		}
	}

	return exOrder, nil
}

func (m *Manager) getUncommittedInstances(iids []InstanceID) []*Instance {
	instances := make([]*Instance, 0)
	for _, iid := range iids {
		instance := m.instances.Get(iid)
		if instance.getStatus() < INSTANCE_COMMITTED {
			instances = append(instances, instance)
		}
	}

	return instances
}

// executes an instance against the store
func (m *Manager) applyInstance(instance *Instance) (store.Value, error) {
	start := time.Now()
	defer m.statsTiming("execute.instance.apply.time", start)
	m.statsInc("execute.instance.apply.count", 1)

	// lock both
	synchronizedApply := func() (store.Value, error) {
		if status := instance.getStatus(); status == INSTANCE_EXECUTED {
			return nil, nil
		} else if status != INSTANCE_COMMITTED {
			return nil, fmt.Errorf("instance not committed")
		}

		instance.lock.Lock()
		defer instance.lock.Unlock()
		if status := instance.Status; status == INSTANCE_EXECUTED {
			return nil, nil
		} else if status != INSTANCE_COMMITTED {
			return nil, fmt.Errorf("instance not committed")
		}
		var val store.Value
		var err error
		if !instance.Noop {
			for _, instruction := range instance.Commands {
				val, err = m.cluster.ApplyQuery(
					instruction.Cmd,
					instruction.Key,
					instruction.Args,
					instruction.Timestamp,
				)
				if err != nil {
					return nil, err
				}
			}
		} else {
			m.statsInc("execute.instance.noop.count", 1)
		}

		// update manager bookkeeping
		instance.Status = INSTANCE_EXECUTED
		func() {
			m.executedLock.Lock()
			defer m.executedLock.Unlock()

			m.executed = append(m.executed, instance.InstanceID)
			m.committed.Remove(instance)
		}()
		if err := m.Persist(); err != nil {
			return nil, err
		}
		m.statsInc("execute.instance.success.count", 1)

		return val, err
	}

	val, err := synchronizedApply()
	if err != nil {
		return nil, err
	}
	// wake up any goroutines waiting on this instance
	instance.broadcastExecuteEvent()

	logger.Debug("Execute: success: %v on %v", instance.InstanceID, m.GetLocalID())
	return val, nil
}
// executes the dependencies up to the given instance
func (m *Manager) executeDependencyChain(iids []InstanceID, target *Instance) (store.Value, error) {
	var val store.Value
	var err error

	// don't execute instances 'out from under' client requests. Use the
	// execution grace period first, check the leader id, if it'm not this
	// node, go ahead and execute it if it is, wait for the execution timeout
	imap := make(map[InstanceID]*Instance)
	imap = m.instances.GetMap(imap, iids)
	for _, iid := range iids {
		val = nil
		err = nil
		instance := imap[iid]
		switch instance.getStatus() {
		case INSTANCE_COMMITTED:
			//
			if instance.InstanceID == target.InstanceID {
				// execute
				val, err = m.applyInstance(instance)
				if err != nil { return nil, err }
				m.statsInc("execute.local.success.count", 1)
			} else if instance.LeaderID != m.GetLocalID() {
				// execute
				val, err = m.applyInstance(instance)
				if err != nil { return nil, err }
				m.statsInc("execute.remote.success.count", 1)
			} else {
				// wait for the execution grace period to end
				if time.Now().After(instance.executeTimeout) {
					val, err = m.applyInstance(instance)
					if err != nil { return nil, err }
					m.statsInc("execute.local.timeout.count", 1)
				} else {

					select {
					case <- instance.getExecuteEvent().getChan():
						// instance was executed by another goroutine
						m.statsInc("execute.local.wait.event.count", 1)
					case <- instance.getExecuteTimeoutEvent():
						// execution timed out
						val, err = m.applyInstance(instance)
						if err != nil { return nil, err }
						m.statsInc("execute.local.timeout.count", 1)
						m.statsInc("execute.local.timeout.wait.count", 1)
					}
				}
			}
		case INSTANCE_EXECUTED:
			continue
		default:
			return nil, fmt.Errorf("Uncommitted dependencies should be handled before calling executeDependencyChain")
		}

		// only execute up to the target instance
		if instance.InstanceID == target.InstanceID {
			break
		}
	}

	return val, nil
}

var managerExecuteInstance = func(m *Manager, instance *Instance) (store.Value, error) {
	start := time.Now()
	defer m.statsTiming("execute.phase.time", start)
	m.statsInc("execute.phase.count", 1)

	logger.Debug("Execute phase started")
	// get dependency instance ids, sorted in execution order
	exOrder, err := m.getExecutionOrder(instance)
	if err != nil {
		return nil, err
	}

	// prepare uncommitted instances
	var uncommitted []*Instance
	uncommitted = m.getUncommittedInstances(exOrder)

	for len(uncommitted) > 0 {
		logger.Info("Execute, %v uncommitted on %v: %+v", len(uncommitted), m.GetLocalID(), uncommitted)
		wg := sync.WaitGroup{}
		wg.Add(len(uncommitted))
		errors := make(chan error, len(uncommitted))
		prepare := func(inst *Instance) {
			var err error
			var success bool
			var ballotErr bool
			var commitEvent <- chan bool
			for i:=0; i<BALLOT_FAILURE_RETRIES; i++ {
				prepareStart := time.Now()
				defer m.statsTiming("execute.phase.prepare.time", prepareStart)
				m.statsInc("execute.phase.prepare.count", 1)

				ballotErr = false
				if err = m.preparePhase(inst); err != nil {
					if _, ok := err.(BallotError); ok {
						// refresh the local instance pointer and
						// assign the commit event so a goroutine is not
						// spun up for each attempt
						inst = m.instances.Get(inst.InstanceID)
						if commitEvent == nil {
							commitEvent = inst.getCommitEvent().getChan()
						}

						logger.Info("Prepare failed with BallotError, waiting to try again")
						ballotErr = true

						// wait on broadcast event or timeout
						waitTime := BALLOT_FAILURE_WAIT_TIME * uint64(i + 1)
						waitTime += uint64(rand.Uint32()) % (waitTime / 2)
						logger.Info("Prepare failed with BallotError, waiting for %v ms to try again", waitTime)
						timeoutEvent := getTimeoutEvent(time.Duration(waitTime) * time.Millisecond)
						select {
						case <- commitEvent:
							// another goroutine committed
							// the instance
							success = true
							break
						case <-timeoutEvent:
							// continue with the prepare
						}

					} else {
						m.statsInc("execute.phase.prepare.error", 1)
						logger.Warning("Execute prepare failed with error: %v", err)
						errors <- err
						break
					}
				} else {
					success = true
					break
				}
			}
			if !success && ballotErr {
				errors <- fmt.Errorf("Prepare failed to commit instance in %v tries", BALLOT_FAILURE_RETRIES)
			}
			wg.Done()
		}
		for _, inst := range uncommitted {
			go prepare(inst)
		}
		wg.Wait()

		// catch any errors, if any
		var err error
		select {
		case err = <- errors:
			return nil, err
		default:
			// everything's ok, continue
		}

		logger.Debug("Recalculating dependency chain for %v on %v", instance.InstanceID, m.GetLocalID())
		// if the prepare phase came across instances
		// that no other replica was aware of, it would
		// have run a preaccept phase for it, changing the
		// dependency chain, so the exOrder and uncommitted
		// list need to be updated before continuing
		exOrder, err = m.getExecutionOrder(instance)
		if err != nil {
			return nil, err
		}
		uncommitted = m.getUncommittedInstances(exOrder)
	}

	logger.Debug("Executing dependency chain")
	val, err := m.executeDependencyChain(exOrder, instance)
	if err != nil {
		logger.Error("Execute: executeDependencyChain: %v", err)
	}
	logger.Debug("Execution phase completed")
	return val, err
}

// applies an instance to the store
// first it will resolve all dependencies, then wait for them to commit/reject, or force them
// to do one or the other. Then it will execute it's committed dependencies, then execute itself
func (m *Manager) executeInstance(instance *Instance) (store.Value, error) {
	return managerExecuteInstance(m, instance)
}
