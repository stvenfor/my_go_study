package postgres

import (
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

func TestUserProfileUpdatesIgnoresStatus(t *testing.T) {
	name := "新名字"
	status := int16(1)
	updates := userProfileUpdates(entity.UpdateProfileInput{
		DisplayName: &name,
		Status:      &status,
	}, time.Unix(10, 0).UTC())
	if updates["user_name"] != "新名字" {
		t.Fatalf("user_name=%v", updates["user_name"])
	}
	if _, ok := updates["status"]; ok {
		t.Fatal("status must not be written")
	}
	for _, banned := range []string{"deleted_at", "email", "user_id", "password_hash", "failed_login_count", "locked_until"} {
		if _, ok := updates[banned]; ok {
			t.Fatalf("%s must not be written", banned)
		}
	}
}

func TestUserProfileUpdatesPrefersUserName(t *testing.T) {
	userName := "canonical"
	display := "alias"
	updates := userProfileUpdates(entity.UpdateProfileInput{
		UserName:    &userName,
		DisplayName: &display,
	}, time.Unix(10, 0).UTC())
	if updates["user_name"] != "canonical" {
		t.Fatalf("user_name=%v", updates["user_name"])
	}
}
