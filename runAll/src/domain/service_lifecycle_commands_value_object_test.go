package domain

import "testing"

func TestNewServiceLifecycleCommands_RequiresStartAndStop(t *testing.T) {
	_, err := NewServiceLifecycleCommands("", "bash run.sh stop", "", "")
	if err != ErrLifecycleStartCommandRequired {
		t.Fatalf("err = %v", err)
	}
	_, err = NewServiceLifecycleCommands("bash run.sh start", "", "", "")
	if err != ErrLifecycleStopCommandRequired {
		t.Fatalf("err = %v", err)
	}
}

func TestNewServiceLifecycleCommands_DetachLaunch(t *testing.T) {
	cmds, err := NewServiceLifecycleCommands("bash run.sh managed", "bash run.sh stop", "", "detach")
	if err != nil {
		t.Fatal(err)
	}
	if !cmds.IsDetachLaunch() {
		t.Fatal("expected detach launch")
	}
}
