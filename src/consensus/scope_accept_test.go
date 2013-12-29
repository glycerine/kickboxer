package consensus

import (
	"testing"
)

/** acceptInstance **/

// tests that an instance is marked as accepted,
// added to the inProgress set, and persisted
// if it doesn't have a higher status
func TestAcceptInstanceSuccess(t *testing.T) {

}

// tests that an instance is not marked as accepted,
// or added to the inProgress set if it already has
// a higher status
func TestAcceptInstanceHigherStatusFailure(t *testing.T) {

}

/** leader **/

func TestSendAcceptSuccess(t *testing.T) {

}

func TestSendAcceptQuorumFailure(t *testing.T) {

}

func TestSendAcceptBallotFailure(t *testing.T) {
	// TODO: figure out what to do in this situation
	// the only way this would happen if is the command
	// was taken over by another replica, in which case,
	// should we just wait for the other leader to
	// execute it?
	t.Skip("figure out the expected behavior")
}

/** replica **/

// test that instances are marked as accepted when
// an accept request is received, and there are no
// problems with the request
func TestHandleAcceptSuccessCase(t *testing.T) {

}

// tests that accept messages fail if an higher
// ballot number has been seen for this message
func TestHandleAcceptOldBallotFailure(t *testing.T) {

}

// tests that handle accept adds any missing instances
// in the missing instances message
func TestHandleAcceptMissingInstanceBehavior(t *testing.T) {

}

// tests that accepts are handled properly if
// the commit if for an instance the node has
// not been previously seen by this replica
func TestHandleAcceptUnknownInstance(t *testing.T) {

}
