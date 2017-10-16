package logging

import (
	"context"
	"log"
	"testing"
)

func TestLocalLogger(t *testing.T) {
	infoLog, alertLog, cleanup, err := NewLogger(context.Background(), "a project ID", "local")
	if err != nil {
		t.Fatalf("creating local logger: %v", err)
	}
	defer cleanup()
	if got, want := infoLog.Flags(), log.LstdFlags|log.Lshortfile; got != want {
		t.Errorf("flags for infoLog differ, got=%d, want=%d", got, want)
	}
	if got, want := alertLog.Flags(), log.LstdFlags|log.Lshortfile; got != want {
		t.Errorf("flags for infoLog differ, got=%d, want=%d", got, want)
	}
}

func TestWrongType(t *testing.T) {
	if _, _, _, err := NewLogger(context.Background(), "a project ID", "wrong param"); err == nil {
		t.Fatal("wanted an error creating a logger but got nil")
	}
}
