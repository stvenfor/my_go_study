package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stvenfor/my_go_study/pkg/rongcloud"
)

type stubRongToken struct {
	res rongcloud.TokenResult
	err error
}

func (s stubRongToken) GetToken(context.Context, string, string) (rongcloud.TokenResult, error) {
	return s.res, s.err
}

func TestImSessionUsecase_IssueSession_OK(t *testing.T) {
	uc := NewImSessionUsecase(stubRongToken{res: rongcloud.TokenResult{
		Code: 200, Token: "t1", UserID: "uuid-1",
	}})
	out, err := uc.IssueSession(context.Background(), "uuid-1", "Ada")
	if err != nil {
		t.Fatal(err)
	}
	if out.UserID != "uuid-1" || out.Token != "t1" {
		t.Fatalf("%+v", out)
	}
}

func TestImSessionUsecase_IssueSession_EmptyUser(t *testing.T) {
	uc := NewImSessionUsecase(stubRongToken{})
	_, err := uc.IssueSession(context.Background(), "  ", "")
	if !errors.Is(err, ErrImUserIDRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestImSessionUsecase_IssueSession_NotConfigured(t *testing.T) {
	uc := NewImSessionUsecase(stubRongToken{err: rongcloud.ErrNotConfigured})
	_, err := uc.IssueSession(context.Background(), "u1", "")
	if !errors.Is(err, ErrImNotConfigured) {
		t.Fatalf("got %v", err)
	}
	uc = NewImSessionUsecase(nil)
	_, err = uc.IssueSession(context.Background(), "u1", "")
	if !errors.Is(err, ErrImNotConfigured) {
		t.Fatalf("got %v", err)
	}
}
