package usecase

import (
	"context"
	"fmt"
	"testing"

	domainrepo "github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
	"github.com/stvenfor/my_go_study/pkg/config"
)

type mockUserRepo struct {
	users map[string]*entity.User
	next  uint
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*entity.User), next: 1}
}

func (m *mockUserRepo) Create(_ context.Context, user *entity.User) error {
	user.ID = m.next
	m.next++
	m.users[user.Username] = user
	return nil
}

func (m *mockUserRepo) FindByID(_ context.Context, id uint) (*entity.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			copy := *u
			return &copy, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) FindByUsername(_ context.Context, username string) (*entity.User, error) {
	if u, ok := m.users[username]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, nil
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			copy := *u
			return &copy, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) List(_ context.Context, offset, limit int) ([]entity.User, int64, error) {
	all := make([]entity.User, 0, len(m.users))
	for _, u := range m.users {
		all = append(all, *u)
	}
	total := int64(len(all))
	if offset >= len(all) {
		return []entity.User{}, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

var _ domainrepo.UserRepository = (*mockUserRepo)(nil)

func TestUserUsecase_RegisterAndLogin(t *testing.T) {
	repo := newMockUserRepo()
	jwtMgr := jwtmanager.NewManager(config.JWTConfig{Secret: "test-secret", ExpireHours: 1})
	uc := NewUserUsecase(repo, jwtMgr)

	user, err := uc.Register(context.Background(), RegisterInput{
		Username: "alice",
		Password: "123456",
		Email:    "alice@test.com",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected user id")
	}

	auth, err := uc.Login(context.Background(), LoginInput{Username: "alice", Password: "123456"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if auth.Token == "" {
		t.Fatal("expected token")
	}

	profile, err := uc.GetProfile(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get profile failed: %v", err)
	}
	if profile.Username != "alice" {
		t.Fatalf("unexpected username: %s", profile.Username)
	}
}

func TestUserUsecase_LoginInvalidPassword(t *testing.T) {
	repo := newMockUserRepo()
	jwtMgr := jwtmanager.NewManager(config.JWTConfig{Secret: "test-secret", ExpireHours: 1})
	uc := NewUserUsecase(repo, jwtMgr)

	_, err := uc.Register(context.Background(), RegisterInput{
		Username: "bob",
		Password: "123456",
		Email:    "bob@test.com",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, err = uc.Login(context.Background(), LoginInput{Username: "bob", Password: "wrong"})
	if err == nil {
		t.Fatal("expected login error")
	}
}

func TestUserUsecase_ListUsers(t *testing.T) {
	repo := newMockUserRepo()
	jwtMgr := jwtmanager.NewManager(config.JWTConfig{Secret: "test-secret", ExpireHours: 1})
	uc := NewUserUsecase(repo, jwtMgr)

	for i := 1; i <= 5; i++ {
		_, err := uc.Register(context.Background(), RegisterInput{
			Username: fmt.Sprintf("user%d", i),
			Password: "123456",
			Email:    fmt.Sprintf("user%d@test.com", i),
		})
		if err != nil {
			t.Fatalf("register failed: %v", err)
		}
	}

	users, total, err := uc.ListUsers(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if total != 5 || len(users) != 2 {
		t.Fatalf("unexpected list result total=%d len=%d", total, len(users))
	}
}
