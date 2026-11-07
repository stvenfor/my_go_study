# Acceptance Record — harden-h1-repo-hygiene

- 日期：2026-11-06（验收执行日；证据文件名沿用 Brief 指定的 `2026-11-05-`）
- Slice：`plans/slices/harden-h1-repo-hygiene.md`
- worktree / 分支：`/Users/qpy/orca/workspaces/my_go_study/harden-h1`（`maoqian/penguin`）
- 复证起点 HEAD：`e288553`（本轮未 commit，HEAD 未变）
- 执行范围：coordinator 指定的四件事（`.gitignore` 收窄 `logs/`、追加 `app/worker`、机检脚本、本记录）
  + **验收后补做（人批 G-new-1）**：放行 `.gitignore` 的 `go.work` / `go.work.sum` 两行。
  > 补做依据：人已于 2026-11-06 批 G-new-1「go.work / go.work.sum 入库（放宽 .gitignore:33-34）」；
  > Brief 第 23-24 行亦授权改这两行。08:25 收到的「不要自行放行」为旧指令，已被人批取代。
  > 本次补做只删两条 ignore 规则，仍不 `git add` / `commit` / `push`（跟踪由人执行）。

## status: **Partial**

理由（三条，均不满足「全绿」，禁止自称 Migrated）：

1. **R1 类缺陷仍在（本次抓到现行）**：`thunder/logs/logs.go` 被 `thunder/{midd,req,upload}` 三处 import，
   却因 `.gitignore` 的裸 `logs/` 规则被排除在 git 跟踪之外。规则已收窄、文件已可被跟踪，但**包体本身
   仍未入库**（是 Desktop 主仓的一个未跟踪文件，不在 index / HEAD / 本 worktree）。
   把 `.go` 文件 `git add` 进仓库是人的动作，worker 不做（本轮边界：不 add / commit / push）。
   因此干净检出的 `go build all` 仍是失败的（见 §2 复现），不是「通过」。
2. **`go.work` / `go.work.sum` 已按人批 G-new-1 放行（`.gitignore` 两条规则已删），但仍未 `git add` 入库**：
   本 worktree 里这两个文件是 `??` 未跟踪（由早期脚本 `go work init` 生成），跟踪由人执行；
   且即便有 go.work，`go build all` 也未绿——本机默认 go1.27.3 下 grpc/x-net 符号缺失仍报错（§8），
   需 `GOTOOLCHAIN=local` + go1.26.4 才 exit 0。二者都未达 Migrated。
3. **原 Brief「干净 worktree → `go build all` 通过 = R1 已解除」这句验收断言已被 coordinator 取证推翻并作废**，
   本记录 §2 引用原句 + 真相 + 复现命令 + 完整报错。既然断言作废，就不存在「按原验收全绿」这一结果。

---

## §1 改了什么（全部落在白名单内，未 add / commit / push）

| 文件 | 改动 |
|------|------|
| `.gitignore` | ①「# Logs」段裸 `logs/` → `/logs/`（仅锚定仓库根产物目录）+ 3 行原因注释；② 追加 `app/worker`（只 ignore，不删不提交二进制），保留「勿写 app/api，那是 proto/gen 源码目录」提醒；③ **按人批 G-new-1 删除 `go.work` / `go.work.sum` 两条 ignore 规则**（放行，待 git add），「# Go workspace」段改写为一行说明性注释，防止被加回 |
| `scripts/check-tracked-imports.sh` | 新增：`import 覆盖` 机检（主检 + `--self-test` 回归锚点），纯 git 实现，不依赖 go.work / go 工具链 / 网络 |
| `docs/acceptance-records/2026-11-05-harden-h1-repo-hygiene.md` | 本记录 |

未触碰：任何 `.go`、`go.mod` / `go.sum`、`.env*`、`app/worker` 二进制、`git config`。
未 `git branch -D`，未 `git worktree remove --force`，未创建任何注册 worktree（复证用 `git archive HEAD | tar -x` 到 `mktemp -d` 临时目录，未清理，按口径留无害）。
未改 `git config`；未 `git add` / `commit` / `push`。

`git status --short`（本 worktree，补做后最终态）：
```
 M .gitignore
?? docs/acceptance-records/2026-11-05-harden-h1-repo-hygiene.md
?? go.work
?? go.work.sum
?? scripts/check-tracked-imports.sh
```
> `?? go.work` / `?? go.work.sum` 是「已放行但待 git add」的预期表现，不是漏处理；跟踪由人执行。

`git diff --numstat -- .gitignore` → `11	5	.gitignore`（EXIT=0）；
`git diff --cached --name-only` → 空（staged count: 0）；`git worktree list` 无新增项；`git log -1` 仍 `e288553`。

---

## §2 推翻 Brief 错断言（引用原句 + 真相 + 复现 + 完整报错）

**Brief 原句（`plans/slices/harden-h1-repo-hygiene.md` 验收第 3 条，作废）：**
> `git worktree add /tmp/h1-verify $(git rev-parse HEAD)` → 在该目录 `go build all` **通过**
> （与原 R1 预期相反，即风险已解除的正面证据）；完成后 `git worktree remove /tmp/h1-verify`

**真相（coordinator 取证 + 本机复验）：**

- 复证起点时 `.gitignore:33`=`go.work`、`:34`=`go.work.sum`（加注释后曾位移到 `:35`/`:36`）→ 二者未入库：
  `git ls-files --error-unmatch go.work` → `error: pathspec 'go.work' did not match any file(s) known to git`（EXIT=1）。
  **补做更新**：这两条 ignore 规则现已按人批 G-new-1 删除（§3⑤）；但删除 ignore 规则 ≠ 入库，
  `go.work` / `go.work.sum` 仍是未跟踪文件（待 git add）。
  任何干净检出 / Orca worktree 都拿不到 go.work，`go build all` 在里面必然 `warning: "all" matched no packages`（空跑，假绿），
  `go -C app build ./cmd/api` 报 `updates to go.mod needed`。本机若曾「通过」，只因**未跟踪的 go.work 恰好还在**。
- 用 `git archive HEAD | tar -x` 造纯跟踪树、手写 go.work 后 `go build all` → **exit 1**，完整报错：

```
$ T=$(mktemp -d /tmp/h1-archive.XXXXXX)      # = /tmp/h1-archive.ciSPyn，未清理
$ git archive HEAD | tar -x -C "$T"
$ printf 'go 1.25.0\n\nuse (\n\t./common\n\t./core\n\t./model\n\t./app\n\t./thunder\n)\n' > "$T/go.work"
$ ls -la "$T/thunder/logs"                    # No such file or directory
$ ( cd "$T" && go build all )
no required module provides package github.com/mszlu521/thunder/logs; to add it:
	cd /tmp/h1-archive.ciSPyn/thunder
	go get github.com/mszlu521/thunder/logs
EXIT=1
```

- 根因：`.gitignore`「# Logs」段的裸 `logs/` 会匹配**任意深度**任何叫 `logs` 的目录，
  把 Go 源码包 `thunder/logs/logs.go`（单文件，656 B）悄悄排除。它被以下三处 import：
  `thunder/midd/cache.go:21`、`thunder/req/param.go:17`、`thunder/upload/aliyun_oss.go:21`（均为 tracked 文件）。
- 因此「R1 已解除 / 干净检出编得过」不成立。原 3f0963e 确实让 `app/internal/rag`、`core/authz`、
  `app/internal/txdb`、`app/internal/schemaboot` 入库（`git merge-base --is-ancestor 3f0963e HEAD` → yes，
  `git ls-files` 全部命中），但它没有覆盖 `thunder/logs` 这一类「被 .gitignore 误伤」的缺陷。

---

## §3 五件事逐条证据（命令 + 真实输出）

### ① `.gitignore` 收窄 `logs/`

`git diff -- .gitignore`（节选）：
```
-# Logs
-logs/
+# Logs（运行日志产物；勿用裸 logs/ —— 它会命中任意深度名为 logs 的目录，
+#       曾把 Go 源码包 thunder/logs/logs.go 误排除在跟踪之外，见
+#       docs/acceptance-records/2026-11-05-harden-h1-repo-hygiene.md）
+/logs/
 *.log
```

**「改规则前」原始缺陷证据（真实输出，收窄前）：**
```
$ git check-ignore -v thunder/logs/logs.go
.gitignore:11:logs/	thunder/logs/logs.go
EXIT=0
```
**「改规则后」（真实输出，收窄后）：**
```
$ git check-ignore -v thunder/logs/logs.go
EXIT=1        # 无输出 = 已放行（不再被忽略）
```

**「放行后确实可被跟踪」的正面证据**（`git add --dry-run`，不碰 index）：
本 worktree 里 `thunder/logs/` 目录根本不存在（该 .go 只作为未跟踪文件存在于 Desktop 主仓），
故在本 worktree 直接跑会报 `fatal: pathspec ... did not match any files`（EXIT=128，真实输出已复核）。
为不伪造文件、不污染仓库，在 `mktemp -d` scratch 仓中放一份**真实的** `thunder/logs/logs.go`
（从主仓原样 `cp`，656 B）并套用本轮新版 `.gitignore`：
```
$ S=$(mktemp -d /tmp/h1-realtrack.XXXXXX)   # = /tmp/h1-realtrack.993pGs，未清理
$ ( cd "$S" && git init -q ); cp .gitignore "$S/.gitignore"
$ cp /Users/qpy/Desktop/github/my_go_study/thunder/logs/logs.go "$S/thunder/logs/logs.go"
$ ( cd "$S" && git check-ignore -v thunder/logs/logs.go ); echo EXIT=$?
EXIT=1        # 无输出 = 放行
$ ( cd "$S" && git add --dry-run thunder/logs/logs.go )
add 'thunder/logs/logs.go'
EXIT=0
```

### ② `.gitignore` 追加 `app/worker`

```
$ git check-ignore -v app/worker
.gitignore:40:app/worker	app/worker
EXIT=0
```
> 行号说明：初版记录在 `:42`，补做删除 go.work 两行后前移到 `:40`；规则内容未变，仍是真实输出。

- 39 MB（40,506,912 B）Mach-O 二进制在 Desktop 主仓：`-rwxr-xr-x ... app/worker`；
  **只 ignore，未删除、未提交、未搬动**。
- 点名 `app/worker` 而非 `app/api`：`app/api/` 是 `proto/` + `gen/` 源码目录（含 tracked 的
  `.gitkeep`、`analytics.pb.go` 等），写成通配会误忽略源码，故保留注释提醒。
- 说明：本 worktree 的 `git status --short` 里 `app/worker` 不出现，是因为该二进制物理上不在本
  worktree（worktree 只检出 tracked 文件），不是因为「已处理」。残留见 §7。

### ③ `scripts/check-tracked-imports.sh` 机检

主检（真实输出，2.86 s，`bash -n` 通过，macOS bash 3.2 无 bash4 语法；补做后用 `bash` 复跑同样结果，见 §3⑤）：
```
$ ./scripts/check-tracked-imports.sh
[scope] 仓库内包 = app/common/core/model/thunder；>1MB 阈值 = 1048576 bytes
  MISSING  thunder/logs
缺失 tracked 的本地 import 包 = 1 （期望 0）
>1MB tracked 文件 = 0 （期望 0）
FAIL: check-tracked-imports 未通过
EXIT=1
```
> 注：这里输出 1 并 exit 1 **不是机检故障，而是它的第一个真阳性**——它正确抓到了现存的
> `thunder/logs` 缺陷。等 §7 残留中包体入库后，主检会翻转为「= 0 / EXIT=0」。
> 当前「期望 0」是健康态目标值；现状不满足，故 status 只能是 Partial，不勾全绿。

`--self-test`（以 `thunder/logs` 为回归锚点，真实输出）：
```
$ ./scripts/check-tracked-imports.sh --self-test
[self-test] 回归锚点 = github.com/mszlu521/thunder/logs （→ thunder/logs，本次首个真阳性）
  PASS 映射 github.com/mszlu521/thunder/logs -> thunder/logs
  PASS 锚点缺陷态：thunder/logs 被判为未覆盖（真阳性）
  PASS 锚点修复态：thunder/logs 判为已覆盖
  PASS 外部依赖不纳入检查
[self-test] PASS
EXIT=0
```
设计要点（对应 Brief 验收 1、4）：遍历所有 tracked `.go` 的 import，凡落到 5 个本地 module
（app/common/core/model/`github.com/mszlu521/thunder`）的，其目录必须有 tracked 直接子级文件；
批量 `git cat-file --batch-check` 复核 `git ls-files` 中无 >1MB 文件（单次 pass，规则已写进机检）。
纯 `git ls-files` / `git cat-file` / `git check-ignore`，不依赖 go.work 内容、go 工具链、`go mod tidy/get` 或网络，
可在干净检出 / 无外网 CI 复用（本机 `proxy.golang.org` curl 000，未跑任何 tidy/get）。
`--self-test` 用同一 `covered()` 谓词构造缺陷态/修复态，故人将来把 `logs.go` 入库后锚点仍有效。

### ④ 本记录

即本文件。

### ⑤ 补做（人批 G-new-1）：放行 `go.work` / `go.work.sum`

`.gitignore` 改动（真实 diff 节选）：
```
-# Go workspace
-go.work
-go.work.sum
+# Go workspace：go.work / go.work.sum 已按人批 G-new-1 放行入库，勿再加回 ignore 规则
```

**coordinator 指定的四条复验命令（本机真实原始输出，全部按预期）：**

```
$ git check-ignore -v go.work go.work.sum
EXIT=1                                   # 无输出 = 已放行（不再被 ignore）

$ git add --dry-run go.work go.work.sum
add 'go.work'
add 'go.work.sum'
EXIT=0                                   # 不碰 index；证明放行后确实可被跟踪

$ bash scripts/check-tracked-imports.sh
[scope] 仓库内包 = app/common/core/model/thunder；>1MB 阈值 = 1048576 bytes
  MISSING  thunder/logs
缺失 tracked 的本地 import 包 = 1 （期望 0）
>1MB tracked 文件 = 0 （期望 0）
FAIL: check-tracked-imports 未通过
EXIT=1                                   # 仍 =1：thunder/logs 包体入库是人的动作，非本补做范围

$ git diff --numstat -- .gitignore
11	5	.gitignore
EXIT=0
```

补充核对：`git status --short` 现含 `?? go.work`、`?? go.work.sum`（已放行、待 `git add`）；
`git diff --cached --name-only` 仍为空（staged count: 0）；全程未 `git add` / `commit` / `push`。

---

## §4 Brief 原始 5 条验收逐条状态（仅 Migrated / Partial / Deferred）

| # | 验收项 | status | 证据 / 理由 |
|---|--------|--------|-------------|
| 1 | 机检脚本跑通并输出「缺失 tracked 的本地 import 包 = 0」；脚本本身入库，可被 CI/harness 复用 | **Partial** | 脚本已落在 `scripts/` 且跑通（§3③/§3⑤），`--self-test` PASS；但主检输出 `=1`（真阳性 thunder/logs，缺陷仍在），未达「=0」；「入库」= commit，按本轮边界由人执行，worker 未 commit |
| 2 | `git check-ignore -v app/worker` 命中；`git status --short` 未跟踪项不再包含二进制产物 | **Partial** | 命中 `.gitignore:40`（§3②）；但 39MB 二进制按人批「只 ignore 不删除」，实体仍留在主仓未处置，且不在本 worktree 内，不能声称「不再包含」已彻底解决 |
| 3 | 干净 worktree `go build all` 通过 = R1 已解除 | **Deferred** | 该断言被 coordinator 取证推翻作废（§2）；`git archive HEAD` 复现 exit 1，缺 `thunder/logs`。go.work 已按 G-new-1 放行但待 git add；且 §8 记录 go1.27.3 下即便有 go.work 仍红。待包体入库 + 工具链口径统一后重验，本轮不凑这句绿灯 |
| 4 | 大文件复核：`git ls-files` 无 >1MB；规则写进机检 | **Migrated** | 机检 `scan_big` 已实现并实测 `>1MB tracked 文件 = 0`；独立交叉验证：`git ls-tree -r -l HEAD \| awk '$4+0>1048576'` → 0；`git ls-files -s` + `git cat-file --batch-check` → 0 |
| 5 | `/teach` 产物归属结论（决策门 2） | **Migrated** | `git ls-files MISSION.md NOTES.md learning-records/ reference/ assets/ data/cs-kb/` → 11 项，全部 tracked（随 `66ece97` 入库）；决策门 2 收敛为**保持入库**（人已批，默认），本轮不改 |

> 状态词只用 `Migrated / Partial / Deferred` 三种（按 harness 侧口径）。整体 status = **Partial**。

---

## §5 决策门与明确选择（不默默二选一）

- **决策门 1（人已批，本轮遵守）**：`app/worker` 39MB 二进制 → **只 ignore，不删除**。
  本轮确认：**未删除**（`-rwxr-xr-x 40506912 ... app/worker` 仍在 Desktop 主仓），**未提交**。
  只写进 `.gitignore`，不写「已处理」。
- **决策门 2（人已批，本轮遵守）**：`/teach` 产物（MISSION.md / NOTES.md / learning-records/ /
  reference/ / assets/ / data/cs-kb/）→ **保持在库**。实测 tracked 11 项，未移出。
- **决策门 3（人已批，本轮遵守）**：`app/internal/rag/` 5 文件零测试 → **本 Program 不拆 Slice**，仅留痕。
  实测 `git ls-files 'app/internal/rag/*'` 命中 5 个 `.go`，未改。
- **决策门 `go.work` / `go.work.sum`（人批 G-new-1，本轮已放行、待 git add）** —— 明确选择如下：
  - **选择：`go.work` 与 `go.work.sum` 一并放行入库**（不二选一、不静默漏掉后者）。
  - **理由**：本机与 CI 都可能无外网（`proxy.golang.org` curl 000、缺网络会去拉校验记录），
    缺 workspace 校验和会让首次构建额外去拉 `go.work.sum` 相关校验；workspace 需要 go.work 才能
    解析多模块，`go build all` 在干净检出里才会非空跑。两者成对才闭环。
  - **本轮实际动作（补做）**：已按人批 G-new-1 删除 `.gitignore` 里 `go.work` / `go.work.sum` 两条
    ignore 规则（原 `:33`/`:34`，注释后 `:35`/`:36`），「# Go workspace」段改为一行说明性注释。
    `git check-ignore -v go.work go.work.sum` → 无输出 / EXIT=1；`git add --dry-run go.work go.work.sum`
    → `add 'go.work'` / `add 'go.work.sum'`（§3⑤）。**未 `git add go.work` / `go.work.sum`，
    未 `git add` / `commit` / `push`**——跟踪由人执行。
  - **口径更正**：此前 08:25「不要自行放行 .gitignore 里的 go.work 行」是旧指令，已被人批 G-new-1 取代；
    初版记录「两行保持原样未动 / gate 未执行」的表述已作废，以本节为准。

**`go.work` / `go.work.sum` 的「改前 / 改后」证据对**（用 `git add --dry-run`，不碰 index）：

改前（真实，放行前，命中原始行号；本轮加注释后曾位移到 `:35`/`:36`）：
```
$ git check-ignore -v go.work go.work.sum
.gitignore:33:go.work	go.work
.gitignore:34:go.work.sum	go.work.sum
EXIT=0
$ git add --dry-run go.work
The following paths are ignored by one of your .gitignore files:
go.work
hint: Use -f if you really want to add them.
EXIT=1
```
改后（**真实 worktree，已按 G-new-1 删除两条规则**，非 scratch 模拟）：
```
$ git check-ignore -v go.work go.work.sum
EXIT=1        # 无输出 = 已放行
$ git add --dry-run go.work go.work.sum
add 'go.work'
add 'go.work.sum'
EXIT=0
```
> 初版记录曾用 scratch 仓 `/tmp/h1-track.nkjFaq` 模拟「人放行后」的目标态并标注
> 「真实 worktree 仍命中 :35/:36、gate 未执行」。该标注随 G-new-1 补做已失效；
> 现在真实 worktree 就是改后态，以上为真实输出。

---

## §6 跑过的命令（汇总，均为本机真实执行）

- `./scripts/check-tracked-imports.sh` 与 `--self-test`、`bash -n`、计时；补做后 `bash scripts/check-tracked-imports.sh`
- `git check-ignore -v`（thunder/logs/logs.go、go.work/go.work.sum、app/worker，改前/改后）
- `git add --dry-run`（thunder/logs/logs.go、go.work、go.work.sum；本 worktree + scratch）
- `git diff --numstat -- .gitignore`（补做复验）
- `git ls-files` / `git ls-files -s` / `git ls-tree -r -l HEAD` / `git cat-file --batch-check`
- `git archive HEAD | tar -x -C <mktemp -d>` + 手写 go.work + `go build all`（§2 复现）
- `git status --short`、`git diff --cached --name-only`、`git diff -- .gitignore`、`git log -1`、
  `git worktree list`、`git merge-base --is-ancestor 3f0963e HEAD`
- 未跑：`go mod tidy` / `go get` / `git add` / `git commit` / `git push` / `rm -rf` /
  `git branch -D` / `git worktree remove --force` / `git worktree add`（复证改用 archive，避免悬挂 worktree）
- §8 的 go1.27.3 / `GOTOOLCHAIN=local` 证据为 **coordinator 提供的新证据**，worker 未自行复跑、未改 go.mod/go.sum。

---

## §7 残留清单（Deferred / 待人处置，均已在 Brief 或本记录点名）

| 残留 | 状态 | 说明 |
|------|------|------|
| 原 Brief 验收 3「干净 worktree go build all 通过」 | **Deferred** | 断言作废；待包体入库 + go.work 入库（待 git add）+ 工具链口径（§8）后重验 |
| `thunder/logs/logs.go` 包体未入库 | **Deferred** | 656 B，现为 Desktop 主仓未跟踪文件；人 `git add` 后主检才会从 1 翻 0、干净检出才可能编过 |
| `go.work` / `go.work.sum` **入库（git add）** | **Deferred** | **放行已完成**（人批 G-new-1，`.gitignore` 两行已删，§3⑤ 四条命令复验通过），但文件仍是 `??` 未跟踪；**跟踪由人执行**，worker 未 `git add` |
| `app/worker` 39MB 二进制 | **Deferred** | 按人批只 ignore 不删除；实体未删、未提交，待人处置 |
| `app/internal/rag` 5 文件零测试 | **Deferred** | 人已批本 Program 不拆 Slice，仅留痕 |
| `/teach` 产物「移出仓库 vs 保持入库」 | **Deferred** | 决策门 2 收敛为保持入库（人已批），本轮不动 |
| 本轮改动未 commit / push | **Deferred** | 按 coordinator 统一口径：worker 只出脚本与文档，提交由人执行；改动停在 worktree 工作区 |

## §8 未验证项（明确留空，不猜）

- 未验证「`thunder/logs/logs.go` 真正入库后 `go build all` 一定通过」——本机无外网，且包体入库属人的动作，未做。
- 未验证 CI 实际调用 `scripts/check-tracked-imports.sh`（只做了本机 `bash -n` + 主检 + 自测）。
- 未验证 `go.work.sum` 在无外网 CI 首次构建的真实收益（§5 是 coordinator 给的理由与本机 curl 000 的旁证，非实测）。
- 未对 `app/worker` 二进制做任何内容/来源核验（只核对存在、大小、被 ignore）。
- `git archive` 复证树只手写 go.work，未补齐 `go.work.sum`/校验和（复现目标是「干净检出必然缺 thunder/logs」，该目标已达成）。
- **【coordinator 验收新证据，worker 未自行复跑、不修】** 本机默认 **go1.27.3** 下，即使有 go.work，
  `go build all` 仍红：`grpc v1.71.0` 的 `internal/transport/handler_server.go:271 undefined: http2.TrailerPrefix`
  （`x/net v0.54.0` 在 go1.27 下编译掉了该符号）。
  改用 `GOTOOLCHAIN=local` + `/usr/local/go`（1.26.4）时 `go build all` → **exit 0**。
  即：工具链版本是独立于本 Slice 的构建风险；按口径**不改 `go.mod` / `go.sum`**，留待人/CI 统一 toolchain 口径。
  （此条为 coordinator 提供的证据，非本机本轮实测；本轮未改任何 go.mod/go.sum。）
