package response

import (
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

func TestFromUserStoreListMarksCurrent(t *testing.T) {
	role := int16(1)
	body := FromUserStoreList([]entity.UserStoreListItem{
		{StoreID: 1, StoreName: "A", Role: &role, RoleLabel: "销售经理", IsCurrent: false},
		{StoreID: 2, StoreName: "B", Role: &role, RoleLabel: "销售经理", IsCurrent: true},
	}, 2)
	if body.CurrentStoreID != 2 {
		t.Fatalf("current_store_id=%d", body.CurrentStoreID)
	}
	if len(body.List) != 2 || !body.List[1].IsCurrent {
		t.Fatalf("list=%+v", body.List)
	}
}
