package storage

import (
	"os"
	"testing"
	"time"
)

func TestSaveServiceAndRemovePID(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Bootstrap(); err != nil {
		t.Fatalf("Bootstrap() returned error: %v", err)
	}

	uid := "wf-test"
	entry := ActiveEntry{
		WorkflowName: uid,
		WorkflowUID:  uid,
		ServiceName:  "api",
		PID:          os.Getpid(),
		Detached:     true,
		StartedAt:    time.Now().Format(time.RFC3339),
	}

	if err := SaveService(uid, entry); err != nil {
		t.Fatalf("SaveService() returned error: %v", err)
	}

	all, err := GetAllActive()
	if err != nil {
		t.Fatalf("GetAllActive() returned error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 active entry, got %d", len(all))
	}
	if all[0].ServiceName != "api" {
		t.Fatalf("expected service api, got %q", all[0].ServiceName)
	}

	if err := RemovePID(uid, "api"); err != nil {
		t.Fatalf("RemovePID() returned error: %v", err)
	}

	allAfter, err := GetAllActive()
	if err != nil {
		t.Fatalf("GetAllActive() returned error after remove: %v", err)
	}
	if len(allAfter) != 0 {
		t.Fatalf("expected 0 active entries after remove, got %d", len(allAfter))
	}
}
