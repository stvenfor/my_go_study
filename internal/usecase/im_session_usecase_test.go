package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/pkg/rongcloud"
)

type stubRongToken struct {
	res            rongcloud.TokenResult
	err            error
	gotName        string
	gotPortraitURI string
}

func (s *stubRongToken) GetToken(_ context.Context, _, name, portraitURI string) (rongcloud.TokenResult, error) {
	s.gotName = name
	s.gotPortraitURI = portraitURI
	return s.res, s.err
}

type stubImUserLookup struct {
	user *entity.User
}

func (s stubImUserLookup) FindByUserID(context.Context, string) (*entity.User, error) {
	return s.user, nil
}
func (s stubImUserLookup) FindByUserIDs(context.Context, []string) ([]*entity.User, error) {
	if s.user == nil {
		return nil, nil
	}
	return []*entity.User{s.user}, nil
}
func (s stubImUserLookup) FindByPhone(context.Context, string) (*entity.User, error) {
	return s.user, nil
}

func TestImSessionUsecase_IssueSession_OK(t *testing.T) {
	stub := &stubRongToken{res: rongcloud.TokenResult{
		Code: 200, Token: "t1", UserID: "uuid-1",
	}}
	uc := NewImSessionUsecase(stub, nil)
	out, err := uc.IssueSession(context.Background(), "uuid-1", "Ada")
	if err != nil {
		t.Fatal(err)
	}
	if out.UserID != "uuid-1" || out.Token != "t1" {
		t.Fatalf("%+v", out)
	}
	if stub.gotName != "Ada" {
		t.Fatalf("name=%q", stub.gotName)
	}
}

func TestImSessionUsecase_IssueSession_UsesDBProfile(t *testing.T) {
	stub := &stubRongToken{res: rongcloud.TokenResult{
		Code: 200, Token: "t2", UserID: "uuid-2",
	}}
	uc := NewImSessionUsecase(stub, stubImUserLookup{user: &entity.User{
		UserID: "uuid-2", UserName: "Bob", AvatarURL: "https://cdn.example/b.png",
	}})
	out, err := uc.IssueSession(context.Background(), "uuid-2", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Token != "t2" {
		t.Fatalf("%+v", out)
	}
	if stub.gotName != "Bob" || stub.gotPortraitURI != "https://cdn.example/b.png" {
		t.Fatalf("name=%q portrait=%q", stub.gotName, stub.gotPortraitURI)
	}
}

func TestImSessionUsecase_IssueSession_EmptyUser(t *testing.T) {
	uc := NewImSessionUsecase(&stubRongToken{}, nil)
	_, err := uc.IssueSession(context.Background(), "  ", "")
	if !errors.Is(err, ErrImUserIDRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestImSessionUsecase_IssueSession_NotConfigured(t *testing.T) {
	uc := NewImSessionUsecase(&stubRongToken{err: rongcloud.ErrNotConfigured}, nil)
	_, err := uc.IssueSession(context.Background(), "u1", "")
	if !errors.Is(err, ErrImNotConfigured) {
		t.Fatalf("got %v", err)
	}
	uc = NewImSessionUsecase(nil, nil)
	_, err = uc.IssueSession(context.Background(), "u1", "")
	if !errors.Is(err, ErrImNotConfigured) {
		t.Fatalf("got %v", err)
	}
}
