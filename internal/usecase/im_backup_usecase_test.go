package usecase

import (
	"context"
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

type memBackupRepo struct {
	rows map[string]*entity.WysImMessageBackup
}

func (m *memBackupRepo) InsertIgnore(_ context.Context, row *entity.WysImMessageBackup) (bool, error) {
	if m.rows == nil {
		m.rows = map[string]*entity.WysImMessageBackup{}
	}
	if _, ok := m.rows[row.MessageUID]; ok {
		return false, nil
	}
	cp := *row
	m.rows[row.MessageUID] = &cp
	return true, nil
}

func TestImBackupUsecase_Idempotent(t *testing.T) {
	uc := NewImBackupUsecase(&memBackupRepo{})
	in := ImBackupBatchInput{
		UserID: "u1",
		Events: []ImBackupEvent{{
			MessageUID: "m1", Direction: entity.ImBackupDirectionOut,
			MessageType: "text", Payload: map[string]any{"c": "hi"},
		}},
	}
	n, err := uc.Ingest(context.Background(), in)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	n, err = uc.Ingest(context.Background(), in)
	if err != nil || n != 0 {
		t.Fatalf("want 0 insert, n=%d err=%v", n, err)
	}
}
