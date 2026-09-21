package usecase_test

import (
	"context"
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

type memAddressRepo struct {
	rows   []entity.WysUserAddress
	nextID int64
}

func (m *memAddressRepo) ListByUser(_ context.Context, userID string) ([]entity.WysUserAddress, error) {
	var out []entity.WysUserAddress
	for _, r := range m.rows {
		if r.UserID == userID && r.DeletedAt == nil {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memAddressRepo) GetByID(_ context.Context, userID string, addressID int64) (*entity.WysUserAddress, error) {
	for i := range m.rows {
		r := &m.rows[i]
		if r.AddressID == addressID && r.UserID == userID && r.DeletedAt == nil {
			cp := *r
			return &cp, nil
		}
	}
	return nil, repository.ErrAddressNotFound
}

func (m *memAddressRepo) CountByUser(_ context.Context, userID string) (int64, error) {
	var n int64
	for _, r := range m.rows {
		if r.UserID == userID && r.DeletedAt == nil {
			n++
		}
	}
	return n, nil
}

func (m *memAddressRepo) Create(_ context.Context, a *entity.WysUserAddress) error {
	if a.IsDefault {
		for i := range m.rows {
			if m.rows[i].UserID == a.UserID && m.rows[i].DeletedAt == nil {
				m.rows[i].IsDefault = false
			}
		}
	}
	m.nextID++
	a.AddressID = m.nextID
	m.rows = append(m.rows, *a)
	return nil
}

func (m *memAddressRepo) Update(_ context.Context, a *entity.WysUserAddress) error {
	for i := range m.rows {
		if m.rows[i].AddressID == a.AddressID && m.rows[i].UserID == a.UserID && m.rows[i].DeletedAt == nil {
			if a.IsDefault {
				for j := range m.rows {
					if m.rows[j].UserID == a.UserID && m.rows[j].DeletedAt == nil {
						m.rows[j].IsDefault = false
					}
				}
			}
			m.rows[i] = *a
			return nil
		}
	}
	return repository.ErrAddressNotFound
}

func (m *memAddressRepo) SoftDelete(_ context.Context, userID string, addressID int64) error {
	for i := range m.rows {
		if m.rows[i].AddressID == addressID && m.rows[i].UserID == userID && m.rows[i].DeletedAt == nil {
			now := m.rows[i].CreatedAt
			m.rows[i].DeletedAt = &now
			m.rows[i].IsDefault = false
			return nil
		}
	}
	return repository.ErrAddressNotFound
}

func (m *memAddressRepo) ClearDefault(_ context.Context, userID string) error {
	for i := range m.rows {
		if m.rows[i].UserID == userID && m.rows[i].DeletedAt == nil {
			m.rows[i].IsDefault = false
		}
	}
	return nil
}

func (m *memAddressRepo) SetDefault(_ context.Context, userID string, addressID int64) error {
	found := false
	for i := range m.rows {
		if m.rows[i].UserID == userID && m.rows[i].DeletedAt == nil {
			m.rows[i].IsDefault = m.rows[i].AddressID == addressID
			if m.rows[i].IsDefault {
				found = true
			}
		}
	}
	if !found {
		return repository.ErrAddressNotFound
	}
	return nil
}

func (m *memAddressRepo) PromoteNewestDefault(ctx context.Context, userID string) error {
	var best *entity.WysUserAddress
	for i := range m.rows {
		r := &m.rows[i]
		if r.UserID == userID && r.DeletedAt == nil {
			if best == nil || r.AddressID > best.AddressID {
				best = r
			}
		}
	}
	if best == nil {
		return nil
	}
	return m.SetDefault(ctx, userID, best.AddressID)
}

func TestAddressCreateFirstIsDefault(t *testing.T) {
	uc := usecase.NewAddressUsecase(&memAddressRepo{})
	v, err := uc.Create(context.Background(), "u1", usecase.AddressInput{
		ReceiverName: "甲", ReceiverPhone: "13800000001", DetailAddress: "路1号",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !v.IsDefault {
		t.Fatal("first should be default")
	}
}

func TestAddressRejectCrossUser(t *testing.T) {
	repo := &memAddressRepo{}
	uc := usecase.NewAddressUsecase(repo)
	v, err := uc.Create(context.Background(), "u1", usecase.AddressInput{
		ReceiverName: "甲", ReceiverPhone: "13800000001", DetailAddress: "路1号",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Get(context.Background(), "u2", v.AddressID); err != usecase.ErrAddressNotFound {
		t.Fatalf("got %v", err)
	}
}

func TestAddressDeletePromotesDefault(t *testing.T) {
	uc := usecase.NewAddressUsecase(&memAddressRepo{})
	a, _ := uc.Create(context.Background(), "u1", usecase.AddressInput{
		ReceiverName: "甲", ReceiverPhone: "13800000001", DetailAddress: "路1号",
	})
	b, _ := uc.Create(context.Background(), "u1", usecase.AddressInput{
		ReceiverName: "乙", ReceiverPhone: "13800000002", DetailAddress: "路2号", IsDefault: true,
	})
	if err := uc.Delete(context.Background(), "u1", b.AddressID); err != nil {
		t.Fatal(err)
	}
	rest, err := uc.Get(context.Background(), "u1", a.AddressID)
	if err != nil {
		t.Fatal(err)
	}
	if !rest.IsDefault {
		t.Fatal("remaining should become default")
	}
}

func TestAddressInvalidPhone(t *testing.T) {
	uc := usecase.NewAddressUsecase(&memAddressRepo{})
	_, err := uc.Create(context.Background(), "u1", usecase.AddressInput{
		ReceiverName: "甲", ReceiverPhone: "123", DetailAddress: "路1号",
	})
	if err != usecase.ErrAddressInvalid {
		t.Fatalf("got %v", err)
	}
}
