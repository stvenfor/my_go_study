package schemaboot

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func writeMigrations(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("-- noop"), 0o600); err != nil {
			t.Fatalf("写入 %s 失败: %v", name, err)
		}
	}
	return dir
}

func TestLoadMigrationsSortsAndReportsMissingDown(t *testing.T) {
	dir := writeMigrations(t,
		"20260102000000_b.up.sql",
		"20260101000000_a.up.sql",
		"20260101000000_a.down.sql",
		"README.md",
	)

	ups, missingDown, err := LoadMigrations(dir)
	if err != nil {
		t.Fatalf("LoadMigrations 失败: %v", err)
	}
	if len(ups) != 2 || ups[0].Version != 20260101000000 || ups[1].Version != 20260102000000 {
		t.Fatalf("未按版本号升序: %+v", ups)
	}
	if len(missingDown) != 1 || missingDown[0] != "20260102000000_b.up.sql" {
		t.Fatalf("缺 down 的文件没识别出来: %v", missingDown)
	}
}

func TestLoadMigrationsRejectsDuplicateVersion(t *testing.T) {
	// 本仓库真撞过：同版本号的第二个文件在库恰好停在该版本时被静默跳过。
	dir := writeMigrations(t, "20260101000000_a.up.sql", "20260101000000_b.up.sql")

	_, _, err := LoadMigrations(dir)
	if err == nil || !strings.Contains(err.Error(), "重复版本号") {
		t.Fatalf("重复版本号必须硬失败，得到: %v", err)
	}
}

func TestLoadMigrationsRejectsDirWithoutUps(t *testing.T) {
	dir := writeMigrations(t, "notes.md")

	if _, _, err := LoadMigrations(dir); err == nil {
		t.Fatal("没有 .up.sql 时必须报错，否则版本校验会形同关闭")
	}
	if _, _, err := LoadMigrations(filepath.Join(dir, "nope")); err == nil {
		t.Fatal("目录不存在时必须报错")
	}
}

// 仓库自身的迁移链必须能被解析（防再次引入重复版本号 / 非法文件名）。
func TestRepoMigrationsLoadClean(t *testing.T) {
	ups, _, err := LoadMigrations(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatalf("migrations/ 解析失败: %v", err)
	}
	if ups[len(ups)-1].Version < 20260924110000 {
		t.Fatalf("迁移链疑似被截断，最高版本只有 %d", ups[len(ups)-1].Version)
	}
}

func TestNamesAfter(t *testing.T) {
	all := []Migration{{Version: 1, Name: "a"}, {Version: 2, Name: "b"}, {Version: 3, Name: "c"}}
	got := namesAfter(all, 1)
	if strings.Join(got, ",") != "b,c" {
		t.Fatalf("namesAfter=%v", got)
	}
}

// Boot 是「谁能建表」的唯一裁决点：这两条用例就是验收口径①（生产零 DDL）的最小自证。
func TestBootRunsNoDDLWhenAutoMigrateOff(t *testing.T) {
	ran := 0
	count := func() error { ran++; return nil }

	// db 传 nil：VersionCheck=false 时 Boot 不得碰库，否则这里会 panic。
	if err := Boot(nil, Options{
		AppEnv:      "prod",
		AutoMigrate: false,
		Required:    []Step{{Name: "core", Fn: count}},
		BestEffort:  []Step{{Name: "im", Fn: count}},
	}, zap.NewNop()); err != nil {
		t.Fatalf("Boot: %v", err)
	}
	if ran != 0 {
		t.Fatalf("auto_migrate=false 仍执行了 %d 条 DDL", ran)
	}
}

func TestBootRunsStepsAndFailsFastOnRequired(t *testing.T) {
	var order []string
	step := func(name string) Step {
		return Step{Name: name, Fn: func() error { order = append(order, name); return nil }}
	}

	err := Boot(nil, Options{
		AutoMigrate: true,
		Required: []Step{
			step("a"),
			{Name: "boom", Fn: func() error { return errors.New("db down") }},
			step("never"),
		},
		BestEffort: []Step{step("im")},
	}, zap.NewNop())
	if err == nil || !strings.Contains(err.Error(), "启动建表失败（boom）") {
		t.Fatalf("Required 步骤失败必须拒绝启动，得到: %v", err)
	}
	if strings.Join(order, ",") != "a" {
		t.Fatalf("失败后不该继续跑后续步骤: %v", order)
	}
}

func TestBootBestEffortFailureKeepsGoing(t *testing.T) {
	// 域内兜底建表历史上只 Warn；改成硬失败会让 dev 环境因单域问题整体起不来。
	var ran int
	err := Boot(nil, Options{
		AutoMigrate: true,
		BestEffort: []Step{
			{Name: "bad", Fn: func() error { return errors.New("nope") }},
			{Name: "good", Fn: func() error { ran++; return nil }},
		},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("BestEffort 失败不该阻断启动: %v", err)
	}
	if ran != 1 {
		t.Fatalf("BestEffort 失败后应继续执行后续步骤: %d", ran)
	}
}
