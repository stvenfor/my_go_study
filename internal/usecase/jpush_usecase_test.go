package usecase

import (
	"context"
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/pkg/jpush"
)

type memPushDeviceRepo struct {
	rows []entity.WysPushDevice
}

func (m *memPushDeviceRepo) Upsert(ctx context.Context, device entity.WysPushDevice) (*entity.WysPushDevice, error) {
	for i, r := range m.rows {
		if r.UserID == device.UserID && r.DeviceID == device.DeviceID {
			device.ID = r.ID
			m.rows[i] = device
			out := device
			return &out, nil
		}
	}
	device.ID = int64(len(m.rows) + 1)
	m.rows = append(m.rows, device)
	out := device
	return &out, nil
}

func (m *memPushDeviceRepo) ListByUser(ctx context.Context, userID string) ([]entity.WysPushDevice, error) {
	var out []entity.WysPushDevice
	for _, r := range m.rows {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memPushDeviceRepo) ListByAlias(ctx context.Context, alias string) ([]entity.WysPushDevice, error) {
	var out []entity.WysPushDevice
	for _, r := range m.rows {
		if r.Alias == alias {
			out = append(out, r)
		}
	}
	return out, nil
}

func TestJPushUsecaseRegisterDefaultsAlias(t *testing.T) {
	repo := &memPushDeviceRepo{}
	uc := NewJPushUsecase(repo, jpush.NewClient(jpush.Config{}))
	got, err := uc.RegisterDevice(context.Background(), RegisterDeviceInput{
		UserID:         "u1",
		DeviceID:       "d1",
		Platform:       "ios",
		RegistrationID: "rid1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Alias != "u1" {
		t.Fatalf("alias=%q", got.Alias)
	}
}

func TestJPushUsecaseSendSkippedWhenNotConfigured(t *testing.T) {
	uc := NewJPushUsecase(&memPushDeviceRepo{}, jpush.NewClient(jpush.Config{}))
	res, err := uc.Send(context.Background(), SendInput{
		UserID: "u1",
		Title:  "t",
		Body:   "b",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped {
		t.Fatalf("expected skipped, got %+v", res)
	}
}
