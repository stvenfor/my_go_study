package entity

import "testing"

func TestWithPositionOmitsTitleWhenNotMember(t *testing.T) {
	stats := UserStoreStats{StoreID: 1, StoreName: "沃德", DaysJoined: 3}.WithPosition(nil)
	if stats.Role != nil || stats.RoleLabel != "" {
		t.Fatalf("non-member got role=%v label=%q", stats.Role, stats.RoleLabel)
	}
	if stats.DaysJoined != 3 {
		t.Fatalf("days = %d", stats.DaysJoined)
	}
}

func TestWithPositionUsesMemberTitle(t *testing.T) {
	position := StoreRoleManager
	stats := ZeroUserStoreStats().WithPosition(&position)
	if stats.Role == nil || *stats.Role != StoreRoleManager || stats.RoleLabel != "销售经理" {
		t.Fatalf("role=%v label=%q", stats.Role, stats.RoleLabel)
	}
}

func TestZeroUserStoreStatsIsNotAdvisor(t *testing.T) {
	stats := ZeroUserStoreStats()
	if stats.Role != nil || stats.RoleLabel != "" {
		t.Fatalf("zero stats role=%v label=%q", stats.Role, stats.RoleLabel)
	}
}
