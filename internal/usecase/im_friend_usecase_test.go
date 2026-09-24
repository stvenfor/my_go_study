package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

type memFriendRepo struct {
	reqs    map[string]*entity.WysImFriendRequest
	friends map[string]bool
}

func newMemFriendRepo() *memFriendRepo {
	return &memFriendRepo{reqs: map[string]*entity.WysImFriendRequest{}, friends: map[string]bool{}}
}

func friendKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "|" + b
}

func (m *memFriendRepo) CreateRequest(_ context.Context, req *entity.WysImFriendRequest) error {
	cp := *req
	m.reqs[req.ID] = &cp
	return nil
}
func (m *memFriendRepo) GetRequest(_ context.Context, id string) (*entity.WysImFriendRequest, error) {
	r, ok := m.reqs[id]
	if !ok {
		return nil, repository.ErrImFriendRequestNotFound
	}
	cp := *r
	return &cp, nil
}
func (m *memFriendRepo) UpdateRequestStatus(_ context.Context, id, status string) error {
	r, ok := m.reqs[id]
	if !ok {
		return repository.ErrImFriendRequestNotFound
	}
	r.Status = status
	r.UpdatedAt = time.Now().UTC()
	return nil
}
func (m *memFriendRepo) FindPending(_ context.Context, from, to string) (*entity.WysImFriendRequest, error) {
	for _, r := range m.reqs {
		if r.FromUserID == from && r.ToUserID == to && r.Status == entity.ImFriendRequestPending {
			cp := *r
			return &cp, nil
		}
	}
	return nil, nil
}
func (m *memFriendRepo) ListPendingTo(_ context.Context, to string) ([]*entity.WysImFriendRequest, error) {
	var out []*entity.WysImFriendRequest
	for _, r := range m.reqs {
		if r.ToUserID == to && r.Status == entity.ImFriendRequestPending {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (m *memFriendRepo) AreFriends(_ context.Context, a, b string) (bool, error) {
	return m.friends[friendKey(a, b)], nil
}
func (m *memFriendRepo) UpsertFriendship(_ context.Context, a, b string) error {
	m.friends[friendKey(a, b)] = true
	return nil
}
func (m *memFriendRepo) ListFriendIDs(_ context.Context, userID string) ([]string, error) {
	var out []string
	for k := range m.friends {
		parts := strings.Split(k, "|")
		if parts[0] == userID {
			out = append(out, parts[1])
		} else if parts[1] == userID {
			out = append(out, parts[0])
		}
	}
	return out, nil
}

type memUsers map[string]*entity.User

func (m memUsers) FindByUserID(_ context.Context, id string) (*entity.User, error) {
	u, ok := m[id]
	if !ok {
		return nil, errors.New("missing")
	}
	return u, nil
}
func (m memUsers) FindByUserIDs(_ context.Context, ids []string) ([]*entity.User, error) {
	var out []*entity.User
	for _, id := range ids {
		if u, ok := m[id]; ok {
			out = append(out, u)
		}
	}
	return out, nil
}
func (m memUsers) FindByPhone(_ context.Context, phone string) (*entity.User, error) {
	for _, u := range m {
		if digitsOnly(u.Phone) == phone {
			return u, nil
		}
	}
	return nil, errors.New("missing")
}

func TestImFriendUsecase_RequestAcceptAdmission(t *testing.T) {
	users := memUsers{
		"a": {UserID: "a", UserName: "A", Phone: "13800000001"},
		"b": {UserID: "b", UserName: "B", Phone: "13800000002"},
	}
	repo := newMemFriendRepo()
	uc := NewImFriendUsecase(repo, users)

	found, err := uc.Search(context.Background(), "a", "13800000002")
	if err != nil || len(found) != 1 || found[0].UserID != "b" {
		t.Fatalf("search %+v err=%v", found, err)
	}

	req, err := uc.Request(context.Background(), "a", "b")
	if err != nil {
		t.Fatal(err)
	}
	ok, _ := uc.CanPrivateChat(context.Background(), "a", "b")
	if ok {
		t.Fatal("should not chat before accept")
	}
	_, err = uc.Respond(context.Background(), "b", req.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	ok, err = uc.CanPrivateChat(context.Background(), "a", "b")
	if err != nil || !ok {
		t.Fatalf("want friends, ok=%v err=%v", ok, err)
	}
}

func TestImFriendUsecase_RejectSelf(t *testing.T) {
	uc := NewImFriendUsecase(newMemFriendRepo(), memUsers{"a": {UserID: "a"}})
	_, err := uc.Request(context.Background(), "a", "a")
	if !errors.Is(err, ErrImFriendSelf) {
		t.Fatalf("got %v", err)
	}
}

func TestImFriendUsecase_ListIncoming(t *testing.T) {
	users := memUsers{
		"a": {UserID: "a", UserName: "Alice", Phone: "13400000000"},
		"b": {UserID: "b", UserName: "Bob", Phone: "13400000001"},
	}
	repo := newMemFriendRepo()
	uc := NewImFriendUsecase(repo, users)
	if _, err := uc.Request(context.Background(), "a", "b"); err != nil {
		t.Fatal(err)
	}
	items, err := uc.ListIncoming(context.Background(), "b")
	if err != nil || len(items) != 1 {
		t.Fatalf("incoming %+v err=%v", items, err)
	}
	if items[0].FromUserID != "a" || items[0].DisplayName != "Alice" {
		t.Fatalf("item %+v", items[0])
	}
	empty, err := uc.ListIncoming(context.Background(), "a")
	if err != nil || len(empty) != 0 {
		t.Fatalf("sender should see empty inbox, got %+v err=%v", empty, err)
	}
}
