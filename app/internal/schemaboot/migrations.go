package schemaboot

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// schemaMigrationsTable 由 golang-migrate 自己建：version=已应用到的迁移号，dirty=上次中断标记。
const schemaMigrationsTable = "schema_migrations"

// baselineTable 用于区分「跑过 AutoMigrate 但没迁移版本」与「压根是空库」。
const baselineTable = "users"

// Migration 单个迁移文件（按 up 计）。
type Migration struct {
	Version int64
	Name    string
	File    string
}

var upRe = regexp.MustCompile(`^(\d+)_(.+)\.up\.sql$`)

// LoadMigrations 解析 migrations 目录的 up 文件，按版本号升序返回。
//
// 同时返回缺 down 的文件名（只 Warn，不拦启动：回滚能力缺失不该让服务起不来）。
// 版本号重复是硬错误：golang-migrate 只记一个版本号，重复号会让后一个文件在
// 「库恰好停在这个版本」时被永久跳过（本仓库 20260923160000 就撞过）。
func LoadMigrations(dir string) ([]Migration, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("读取 migrations 目录 %s 失败: %w", dir, err)
	}

	var ups []Migration
	downVersions := make(map[int64]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if m := upRe.FindStringSubmatch(name); m != nil {
			v, err := strconv.ParseInt(m[1], 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("迁移版本号非法 %s: %w", name, err)
			}
			ups = append(ups, Migration{Version: v, Name: m[2], File: name})
			continue
		}
		if base, ok := downOf(name); ok {
			if m := upRe.FindStringSubmatch(base + ".up.sql"); m != nil {
				if v, err := strconv.ParseInt(m[1], 10, 64); err == nil {
					downVersions[v] = base
				}
			}
		}
	}
	if len(ups) == 0 {
		return nil, nil, fmt.Errorf("migrations 目录 %s 下没有 *.up.sql（database.schema_version_check 无法校验）", dir)
	}

	sort.Slice(ups, func(i, j int) bool {
		if ups[i].Version != ups[j].Version {
			return ups[i].Version < ups[j].Version
		}
		return ups[i].File < ups[j].File
	})

	var dups []string
	for i := 1; i < len(ups); i++ {
		if ups[i].Version == ups[i-1].Version {
			dups = append(dups, fmt.Sprintf("%d（%s 与 %s）", ups[i].Version, ups[i-1].File, ups[i].File))
		}
	}
	if len(dups) > 0 {
		return nil, nil, fmt.Errorf("migrations 存在重复版本号：%s；请把后一个文件改成更大的唯一版本号（迁移号只记一个，重复会静默跳过）",
			strings.Join(dups, "、"))
	}

	var missingDown []string
	for _, u := range ups {
		if _, ok := downVersions[u.Version]; !ok {
			missingDown = append(missingDown, u.File)
		}
	}
	return ups, missingDown, nil
}

// downOf 由 down 文件名取出去掉后缀的迁移基名。
func downOf(name string) (string, bool) {
	const suffix = ".down.sql"
	if len(name) <= len(suffix) || name[len(name)-len(suffix):] != suffix {
		return "", false
	}
	return name[:len(name)-len(suffix)], true
}
