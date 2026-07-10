package redis_test

import (
	"context"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	redisrepo "github.com/stvenfor/my_go_study/internal/repository/redis"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

func TestSessionRepositoryListActiveUserIDs(t *testing.T) {
	client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("redis not available:", err)
	}
	defer client.Close()

	repo := redisrepo.NewSessionRepository(client)
	userID := "list-session-user-" + time.Now().Format("150405")
	defer func() {
		_ = repo.Delete(ctx, userID)
	}()

	if err := repo.Save(ctx, userID, repository.DeviceSession{
		SessionID: "s1",
		DeviceID:  "d1",
		Platform:  "ios",
		CreatedAt: time.Now().Unix(),
	}, time.Hour); err != nil {
		t.Fatalf("save: %v", err)
	}

	ids, err := repo.ListActiveUserIDs(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, id := range ids {
		if id == userID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("user %s not in %v", userID, ids)
	}
}

func TestSessionRepositorySaveNoExpiration(t *testing.T) {
	client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("redis not available:", err)
	}
	defer client.Close()

	repo := redisrepo.NewSessionRepository(client)
	userID := "no-expire-session-" + time.Now().Format("150405")
	defer func() {
		_ = repo.Delete(ctx, userID)
	}()

	if err := repo.Save(ctx, userID, repository.DeviceSession{
		SessionID: "s1",
		DeviceID:  "d1",
		Platform:  "ios",
		CreatedAt: time.Now().Unix(),
	}, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	ttl, err := client.TTL(ctx, "auth:session:"+userID).Result()
	if err != nil {
		t.Fatalf("ttl: %v", err)
	}
	if ttl != -1 {
		t.Fatalf("expected no expiration (TTL=-1), got %v", ttl)
	}
}
