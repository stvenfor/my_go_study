package postgres

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:embed sql/im_schema.sql
var imSchemaSQL string

// EnsureImSchema 幂等建 IM 相关表。
func EnsureImSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db nil")
	}
	if err := db.Exec(imSchemaSQL).Error; err != nil {
		return fmt.Errorf("im schema: %w", err)
	}
	return db.AutoMigrate(
		&entity.WysImFriendRequest{},
		&entity.WysImFriendship{},
		&entity.WysImFreeGroup{},
		&entity.WysImFreeGroupMember{},
		&entity.WysImStoreGroup{},
		&entity.WysImMessageBackup{},
	)
}

// ImFriendRepository Postgres。
type ImFriendRepository struct{ db *gorm.DB }

func NewImFriendRepository(db *gorm.DB) *ImFriendRepository {
	return &ImFriendRepository{db: db}
}

func (r *ImFriendRepository) CreateRequest(ctx context.Context, req *entity.WysImFriendRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *ImFriendRepository) GetRequest(ctx context.Context, id string) (*entity.WysImFriendRequest, error) {
	var row entity.WysImFriendRequest
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, repository.ErrImFriendRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ImFriendRepository) UpdateRequestStatus(ctx context.Context, id, status string) error {
	res := r.db.WithContext(ctx).Model(&entity.WysImFriendRequest{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repository.ErrImFriendRequestNotFound
	}
	return nil
}

func (r *ImFriendRepository) FindPending(ctx context.Context, fromUserID, toUserID string) (*entity.WysImFriendRequest, error) {
	var row entity.WysImFriendRequest
	err := r.db.WithContext(ctx).
		Where("from_user_id = ? AND to_user_id = ? AND status = ?", fromUserID, toUserID, entity.ImFriendRequestPending).
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ImFriendRepository) ListPendingTo(ctx context.Context, toUserID string) ([]*entity.WysImFriendRequest, error) {
	var rows []entity.WysImFriendRequest
	err := r.db.WithContext(ctx).
		Where("to_user_id = ? AND status = ?", toUserID, entity.ImFriendRequestPending).
		Order("created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*entity.WysImFriendRequest, 0, len(rows))
	for i := range rows {
		out = append(out, &rows[i])
	}
	return out, nil
}

func (r *ImFriendRepository) AreFriends(ctx context.Context, userA, userB string) (bool, error) {
	a, b := orderedPair(userA, userB)
	var n int64
	err := r.db.WithContext(ctx).Model(&entity.WysImFriendship{}).
		Where("user_a = ? AND user_b = ?", a, b).Count(&n).Error
	return n > 0, err
}

func (r *ImFriendRepository) UpsertFriendship(ctx context.Context, userA, userB string) error {
	a, b := orderedPair(userA, userB)
	row := entity.WysImFriendship{UserA: a, UserB: b, CreatedAt: time.Now().UTC()}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func (r *ImFriendRepository) ListFriendIDs(ctx context.Context, userID string) ([]string, error) {
	var rows []entity.WysImFriendship
	if err := r.db.WithContext(ctx).
		Where("user_a = ? OR user_b = ?", userID, userID).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.UserA == userID {
			out = append(out, row.UserB)
		} else {
			out = append(out, row.UserA)
		}
	}
	return out, nil
}

func orderedPair(a, b string) (string, string) {
	if a > b {
		return b, a
	}
	return a, b
}

// ImGroupRepository Postgres。
type ImGroupRepository struct{ db *gorm.DB }

func NewImGroupRepository(db *gorm.DB) *ImGroupRepository {
	return &ImGroupRepository{db: db}
}

func (r *ImGroupRepository) CreateFreeGroup(ctx context.Context, g *entity.WysImFreeGroup, memberIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(g).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, id := range memberIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			m := entity.WysImFreeGroupMember{GroupID: g.GroupID, UserID: id, JoinedAt: now}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&m).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ImGroupRepository) GetFreeGroup(ctx context.Context, groupID string) (*entity.WysImFreeGroup, error) {
	var row entity.WysImFreeGroup
	err := r.db.WithContext(ctx).Where("group_id = ?", groupID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, repository.ErrImFreeGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ImGroupRepository) AddFreeMembers(ctx context.Context, groupID string, memberIDs []string) error {
	now := time.Now().UTC()
	for _, id := range memberIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		m := entity.WysImFreeGroupMember{GroupID: groupID, UserID: id, JoinedAt: now}
		if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *ImGroupRepository) RemoveFreeMembers(ctx context.Context, groupID string, memberIDs []string) error {
	return r.db.WithContext(ctx).
		Where("group_id = ? AND user_id IN ?", groupID, memberIDs).
		Delete(&entity.WysImFreeGroupMember{}).Error
}

func (r *ImGroupRepository) IsFreeMember(ctx context.Context, groupID, userID string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&entity.WysImFreeGroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).Count(&n).Error
	return n > 0, err
}

func (r *ImGroupRepository) DismissFreeGroup(ctx context.Context, groupID string) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&entity.WysImFreeGroup{}).
		Where("group_id = ?", groupID).
		Updates(map[string]any{"dismissed_at": now, "updated_at": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repository.ErrImFreeGroupNotFound
	}
	return nil
}

func (r *ImGroupRepository) GetStoreGroup(ctx context.Context, storeID string) (*entity.WysImStoreGroup, error) {
	var row entity.WysImStoreGroup
	err := r.db.WithContext(ctx).Where("store_id = ?", storeID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, repository.ErrImFreeGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ImGroupRepository) UpsertStoreGroup(ctx context.Context, g *entity.WysImStoreGroup) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "store_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"group_id", "updated_at"}),
	}).Create(g).Error
}

// ImBackupRepository Postgres。
type ImBackupRepository struct{ db *gorm.DB }

func NewImBackupRepository(db *gorm.DB) *ImBackupRepository {
	return &ImBackupRepository{db: db}
}

func (r *ImBackupRepository) InsertIgnore(ctx context.Context, row *entity.WysImMessageBackup) (bool, error) {
	res := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(row)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ImUserLookupRepository 本地 users 查找。
type ImUserLookupRepository struct{ db *gorm.DB }

func NewImUserLookupRepository(db *gorm.DB) *ImUserLookupRepository {
	return &ImUserLookupRepository{db: db}
}

func (r *ImUserLookupRepository) FindByUserID(ctx context.Context, userID string) (*entity.User, error) {
	var row entity.User
	err := r.db.WithContext(ctx).Where("user_id = ? AND deleted_at IS NULL", userID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ImUserLookupRepository) FindByUserIDs(ctx context.Context, userIDs []string) ([]*entity.User, error) {
	ids := make([]string, 0, len(userIDs))
	seen := map[string]struct{}{}
	for _, id := range userIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []entity.User
	err := r.db.WithContext(ctx).Where("user_id IN ? AND deleted_at IS NULL", ids).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*entity.User, 0, len(rows))
	for i := range rows {
		out = append(out, &rows[i])
	}
	return out, nil
}

func (r *ImUserLookupRepository) FindByPhone(ctx context.Context, phoneDigits string) (*entity.User, error) {
	phoneDigits = strings.TrimSpace(phoneDigits)
	if phoneDigits == "" {
		return nil, gorm.ErrRecordNotFound
	}
	// ponytail: Postgres regexp_replace；升级路径=E.164 规范化列 + 唯一索引
	var row entity.User
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND regexp_replace(coalesce(phone,''), '[^0-9]', '', 'g') = ?", phoneDigits).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}
