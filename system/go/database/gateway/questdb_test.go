package gateway

import (
	"context"
	"testing"
)

func TestQuestDBChecker(t *testing.T) {
	tests := []struct {
		name     string
		checker  *QuestDBChecker
		wantName string
		wantErr  bool
	}{
		{
			name:     "nil client returns no error",
			checker:  NewQuestDBChecker(nil),
			wantName: "questdb",
			wantErr:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.checker.Name() != tc.wantName {
				t.Errorf("expected name %q, got %q", tc.wantName, tc.checker.Name())
			}

			err := tc.checker.Check(context.Background())
			if tc.wantErr && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
