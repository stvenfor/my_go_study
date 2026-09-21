// address_usecase.go 用户收货地址簿。
package usecase

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

const maxAddressesPerUser = 20

var (
	ErrAddressNotFound = repository.ErrAddressNotFound
	ErrAddressInvalid  = repository.ErrAddressInvalid
	ErrAddressLimit    = repository.ErrAddressLimit
)

// AddressUsecase 地址簿。
type AddressUsecase struct {
	repo repository.AddressRepository
}

// NewAddressUsecase 创建。
func NewAddressUsecase(repo repository.AddressRepository) *AddressUsecase {
	return &AddressUsecase{repo: repo}
}

// AddressInput 新建/更新入参。
type AddressInput struct {
	ReceiverName  string
	ReceiverPhone string
	Province      string
	City          string
	District      string
	DetailAddress string
	PostalCode    string
	IsDefault     bool
	Label         string
}

// List 当前用户未删地址。
func (u *AddressUsecase) List(ctx context.Context, userID string) ([]entity.AddressView, error) {
	rows, err := u.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]entity.AddressView, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ToAddressView())
	}
	return out, nil
}

// Get 单条。
func (u *AddressUsecase) Get(ctx context.Context, userID string, addressID int64) (*entity.AddressView, error) {
	a, err := u.repo.GetByID(ctx, userID, addressID)
	if err != nil {
		return nil, err
	}
	v := a.ToAddressView()
	return &v, nil
}

// Create 新建。首条自动默认。
func (u *AddressUsecase) Create(ctx context.Context, userID string, in AddressInput) (*entity.AddressView, error) {
	name, phone, detail, err := normalizeAddress(in)
	if err != nil {
		return nil, err
	}
	n, err := u.repo.CountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if n >= maxAddressesPerUser {
		return nil, ErrAddressLimit
	}
	isDefault := in.IsDefault || n == 0
	a := &entity.WysUserAddress{
		UserID:        userID,
		ReceiverName:  name,
		ReceiverPhone: phone,
		Province:      strings.TrimSpace(in.Province),
		City:          strings.TrimSpace(in.City),
		District:      strings.TrimSpace(in.District),
		DetailAddress: detail,
		PostalCode:    strings.TrimSpace(in.PostalCode),
		IsDefault:     isDefault,
		Label:         strings.TrimSpace(in.Label),
	}
	if err := u.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	v := a.ToAddressView()
	return &v, nil
}

// Update 更新。
func (u *AddressUsecase) Update(ctx context.Context, userID string, addressID int64, in AddressInput) (*entity.AddressView, error) {
	existing, err := u.repo.GetByID(ctx, userID, addressID)
	if err != nil {
		return nil, err
	}
	name, phone, detail, err := normalizeAddress(in)
	if err != nil {
		return nil, err
	}
	existing.ReceiverName = name
	existing.ReceiverPhone = phone
	existing.Province = strings.TrimSpace(in.Province)
	existing.City = strings.TrimSpace(in.City)
	existing.District = strings.TrimSpace(in.District)
	existing.DetailAddress = detail
	existing.PostalCode = strings.TrimSpace(in.PostalCode)
	existing.Label = strings.TrimSpace(in.Label)
	existing.IsDefault = in.IsDefault
	if err := u.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	v := existing.ToAddressView()
	return &v, nil
}

// SetDefault 设默认。
func (u *AddressUsecase) SetDefault(ctx context.Context, userID string, addressID int64) (*entity.AddressView, error) {
	if err := u.repo.SetDefault(ctx, userID, addressID); err != nil {
		return nil, err
	}
	return u.Get(ctx, userID, addressID)
}

// Delete 软删；若删的是默认则晋升最新一条。
func (u *AddressUsecase) Delete(ctx context.Context, userID string, addressID int64) error {
	a, err := u.repo.GetByID(ctx, userID, addressID)
	if err != nil {
		return err
	}
	wasDefault := a.IsDefault
	if err := u.repo.SoftDelete(ctx, userID, addressID); err != nil {
		return err
	}
	if wasDefault {
		return u.repo.PromoteNewestDefault(ctx, userID)
	}
	return nil
}

func normalizeAddress(in AddressInput) (name, phone, detail string, err error) {
	name = strings.TrimSpace(in.ReceiverName)
	phone = strings.TrimSpace(in.ReceiverPhone)
	detail = strings.TrimSpace(in.DetailAddress)
	if name == "" || detail == "" || utf8.RuneCountInString(phone) < 6 {
		return "", "", "", ErrAddressInvalid
	}
	return name, phone, detail, nil
}
