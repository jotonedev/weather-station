package nodecoords

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestUpsertEmpty(t *testing.T) {
	repo := &Repository{client: nil, log: zap.NewNop()}
	stored, err := repo.UpsertNodeCoordinates(context.Background(), nil)
	if err != nil {
		t.Fatalf("UpsertNodeCoordinates(empty): %v", err)
	}
	if stored != 0 {
		t.Errorf("rows = %d, want 0", stored)
	}
}
