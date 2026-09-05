package main

import "testing"

func TestCommentBindingBlocksStartVM(t *testing.T) {
	setupCloudTestDB(t)

	waitRow, _, err := ensureCommentContainerBinding("t1", "taskGate", "cWait", ccbExecutionWaitPrevious, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := updateCommentContainerBindingStatus(waitRow.ID, ccbStatusWaitingPrevious); err != nil {
		t.Fatal(err)
	}

	indepRow, _, err := ensureCommentContainerBinding("t1", "taskGate", "cIndep", ccbExecutionIndependent, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := updateCommentContainerBindingStatus(indepRow.ID, ccbStatusStarting); err != nil {
		t.Fatal(err)
	}

	blocked, reason := commentBindingBlocksStartVM("t1", "taskGate", "cWait")
	if !blocked {
		t.Fatalf("waiting_previous must block start-vm, reason=%q", reason)
	}
	if reason == "" {
		t.Fatal("blocked reason must be non-empty")
	}

	if blocked, _ := commentBindingBlocksStartVM("t1", "taskGate", "cIndep"); blocked {
		t.Fatal("independent starting comment must not be blocked")
	}
	if blocked, _ := commentBindingBlocksStartVM("t1", "taskGate", "missing"); blocked {
		t.Fatal("missing binding must not block")
	}
	if blocked, _ := commentBindingBlocksStartVM("t1", "taskGate", ""); blocked {
		t.Fatal("empty comment_id must not block")
	}
}
