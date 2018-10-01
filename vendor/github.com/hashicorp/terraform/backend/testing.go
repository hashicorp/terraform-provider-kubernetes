package backend

import (
	"reflect"
	"sort"
	"testing"

	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// TestBackendConfig validates and configures the backend with the
// given configuration.
func TestBackendConfig(t *testing.T, b Backend, c map[string]interface{}) Backend {
	t.Helper()

	// Get the proper config structure
	rc, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	conf := terraform.NewResourceConfig(rc)

	// Validate
	warns, errs := b.Validate(conf)
	if len(warns) > 0 {
		t.Fatalf("warnings: %s", warns)
	}
	if len(errs) > 0 {
		t.Fatalf("errors: %s", errs)
	}

	// Configure
	if err := b.Configure(conf); err != nil {
		t.Fatalf("err: %s", err)
	}

	return b
}

// TestBackend will test the functionality of a Backend. The backend is
// assumed to already be configured. This will test state functionality.
// If the backend reports it doesn't support multi-state by returning the
// error ErrNamedStatesNotSupported, then it will not test that.
func TestBackendStates(t *testing.T, b Backend) {
	t.Helper()

	noDefault := false
	if _, err := b.State(DefaultStateName); err != nil {
		if err == ErrDefaultStateNotSupported {
			noDefault = true
		} else {
			t.Fatalf("error: %v", err)
		}
	}

	states, err := b.States()
	if err != nil {
		if err == ErrNamedStatesNotSupported {
			t.Logf("TestBackend: named states not supported in %T, skipping", b)
			return
		}
		t.Fatalf("error: %v", err)
	}

	// Test it starts with only the default
	if !noDefault && (len(states) != 1 || states[0] != DefaultStateName) {
		t.Fatalf("should have default to start: %#v", states)
	}

	// Create a couple states
	foo, err := b.State("foo")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := foo.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	if v := foo.State(); v.HasResources() {
		t.Fatalf("should be empty: %s", v)
	}

	bar, err := b.State("bar")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := bar.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	if v := bar.State(); v.HasResources() {
		t.Fatalf("should be empty: %s", v)
	}

	// Verify they are distinct states that can be read back from storage
	{
		// start with a fresh state, and record the lineage being
		// written to "bar"
		barState := terraform.NewState()

		// creating the named state may have created a lineage, so use that if it exists.
		if s := bar.State(); s != nil && s.Lineage != "" {
			barState.Lineage = s.Lineage
		}
		barLineage := barState.Lineage

		// the foo lineage should be distinct from bar, and unchanged after
		// modifying bar
		fooState := terraform.NewState()
		// creating the named state may have created a lineage, so use that if it exists.
		if s := foo.State(); s != nil && s.Lineage != "" {
			fooState.Lineage = s.Lineage
		}
		fooLineage := fooState.Lineage

		// write a known state to foo
		if err := foo.WriteState(fooState); err != nil {
			t.Fatal("error writing foo state:", err)
		}
		if err := foo.PersistState(); err != nil {
			t.Fatal("error persisting foo state:", err)
		}

		// write a distinct known state to bar
		if err := bar.WriteState(barState); err != nil {
			t.Fatalf("bad: %s", err)
		}
		if err := bar.PersistState(); err != nil {
			t.Fatalf("bad: %s", err)
		}

		// verify that foo is unchanged with the existing state manager
		if err := foo.RefreshState(); err != nil {
			t.Fatal("error refreshing foo:", err)
		}
		fooState = foo.State()
		switch {
		case fooState == nil:
			t.Fatal("nil state read from foo")
		case fooState.Lineage == barLineage:
			t.Fatalf("bar lineage read from foo: %#v", fooState)
		case fooState.Lineage != fooLineage:
			t.Fatal("foo lineage alterred")
		}

		// fetch foo again from the backend
		foo, err = b.State("foo")
		if err != nil {
			t.Fatal("error re-fetching state:", err)
		}
		if err := foo.RefreshState(); err != nil {
			t.Fatal("error refreshing foo:", err)
		}
		fooState = foo.State()
		switch {
		case fooState == nil:
			t.Fatal("nil state read from foo")
		case fooState.Lineage != fooLineage:
			t.Fatal("incorrect state returned from backend")
		}

		// fetch the bar  again from the backend
		bar, err = b.State("bar")
		if err != nil {
			t.Fatal("error re-fetching state:", err)
		}
		if err := bar.RefreshState(); err != nil {
			t.Fatal("error refreshing bar:", err)
		}
		barState = bar.State()
		switch {
		case barState == nil:
			t.Fatal("nil state read from bar")
		case barState.Lineage != barLineage:
			t.Fatal("incorrect state returned from backend")
		}
	}

	// Verify we can now list them
	{
		// we determined that named stated are supported earlier
		states, err := b.States()
		if err != nil {
			t.Fatal(err)
		}

		sort.Strings(states)
		expected := []string{"bar", "default", "foo"}
		if noDefault {
			expected = []string{"bar", "foo"}
		}
		if !reflect.DeepEqual(states, expected) {
			t.Fatalf("bad: %#v", states)
		}
	}

	// Delete some states
	if err := b.DeleteState("foo"); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the default state can't be deleted
	if err := b.DeleteState(DefaultStateName); err == nil {
		t.Fatal("expected error")
	}

	// Create and delete the foo state again.
	// Make sure that there are no leftover artifacts from a deleted state
	// preventing re-creation.
	foo, err = b.State("foo")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := foo.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	if v := foo.State(); v.HasResources() {
		t.Fatalf("should be empty: %s", v)
	}
	// and delete it again
	if err := b.DeleteState("foo"); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify deletion
	{
		states, err := b.States()
		if err == ErrNamedStatesNotSupported {
			t.Logf("TestBackend: named states not supported in %T, skipping", b)
			return
		}

		sort.Strings(states)
		expected := []string{"bar", "default"}
		if noDefault {
			expected = []string{"bar"}
		}
		if !reflect.DeepEqual(states, expected) {
			t.Fatalf("bad: %#v", states)
		}
	}
}

// TestBackendStateLocks will test the locking functionality of the remote
// state backend.
func TestBackendStateLocks(t *testing.T, b1, b2 Backend) {
	t.Helper()
	testLocks(t, b1, b2, false)
}

// TestBackendStateForceUnlock verifies that the lock error is the expected
// type, and the lock can be unlocked using the ID reported in the error.
// Remote state backends that support -force-unlock should call this in at
// least one of the acceptance tests.
func TestBackendStateForceUnlock(t *testing.T, b1, b2 Backend) {
	t.Helper()
	testLocks(t, b1, b2, true)
}

func testLocks(t *testing.T, b1, b2 Backend, testForceUnlock bool) {
	t.Helper()

	// Get the default state for each
	b1StateMgr, err := b1.State(DefaultStateName)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := b1StateMgr.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Fast exit if this doesn't support locking at all
	if _, ok := b1StateMgr.(state.Locker); !ok {
		t.Logf("TestBackend: backend %T doesn't support state locking, not testing", b1)
		return
	}

	t.Logf("TestBackend: testing state locking for %T", b1)

	b2StateMgr, err := b2.State(DefaultStateName)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := b2StateMgr.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Reassign so its obvious whats happening
	lockerA := b1StateMgr.(state.Locker)
	lockerB := b2StateMgr.(state.Locker)

	infoA := state.NewLockInfo()
	infoA.Operation = "test"
	infoA.Who = "clientA"

	infoB := state.NewLockInfo()
	infoB.Operation = "test"
	infoB.Who = "clientB"

	lockIDA, err := lockerA.Lock(infoA)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	// Make sure we can still get the state.State from another instance even
	// when locked.  This should only happen when a state is loaded via the
	// backend, and as a remote state.
	_, err = b2.State(DefaultStateName)
	if err != nil {
		t.Errorf("failed to read locked state from another backend instance: %s", err)
	}

	// If the lock ID is blank, assume locking is disabled
	if lockIDA == "" {
		t.Logf("TestBackend: %T: empty string returned for lock, assuming disabled", b1)
		return
	}

	_, err = lockerB.Lock(infoB)
	if err == nil {
		lockerA.Unlock(lockIDA)
		t.Fatal("client B obtained lock while held by client A")
	}

	if err := lockerA.Unlock(lockIDA); err != nil {
		t.Fatal("error unlocking client A", err)
	}

	lockIDB, err := lockerB.Lock(infoB)
	if err != nil {
		t.Fatal("unable to obtain lock from client B")
	}

	if lockIDB == lockIDA {
		t.Errorf("duplicate lock IDs: %q", lockIDB)
	}

	if err = lockerB.Unlock(lockIDB); err != nil {
		t.Fatal("error unlocking client B:", err)
	}

	// test the equivalent of -force-unlock, by using the id from the error
	// output.
	if !testForceUnlock {
		return
	}

	// get a new ID
	infoA.ID, err = uuid.GenerateUUID()
	if err != nil {
		panic(err)
	}

	lockIDA, err = lockerA.Lock(infoA)
	if err != nil {
		t.Fatal("unable to get re lock A:", err)
	}
	unlock := func() {
		err := lockerA.Unlock(lockIDA)
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err = lockerB.Lock(infoB)
	if err == nil {
		unlock()
		t.Fatal("client B obtained lock while held by client A")
	}

	infoErr, ok := err.(*state.LockError)
	if !ok {
		unlock()
		t.Fatalf("expected type *state.LockError, got : %#v", err)
	}

	// try to unlock with the second unlocker, using the ID from the error
	if err := lockerB.Unlock(infoErr.Info.ID); err != nil {
		unlock()
		t.Fatalf("could not unlock with the reported ID %q: %s", infoErr.Info.ID, err)
	}
}
