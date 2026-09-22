# my_go_study — 本机局域网后端与本地认证

本仓库在「家庭/信任 Wi‑Fi 上，用开发者 Mac 充当 Flutter 可连的 BFF」以及「本地 Auth/Postgres 可迁回 Supabase」上下文中的用语。

## Language

**局域网后端主机 (LAN Backend Host)**：
在同 Wi‑Fi 下向真机提供 Go BFF 的那台 Mac（按需启停，不是公网服务器）。
_Avoid_: 生产服务器、云主机、公网部署

**LAN 配置档 (LAN Profile)**：
局域网真机联调姿态；现语义为本地全栈（`auth.provider=local`）+ 局域网暴露。
_Avoid_: LAN + Supabase Cloud 混用、直接把 prod 当作家用联调

**本地认证后端 (Local Auth Provider)**：
`auth.provider=local`：本机 Postgres `auth_users` 签发 UUID JWT，业务表在本机；`supabase` 为可切回的 Cloud 路径。
_Avoid_: 把自托管 GoTrue 说成「只要 Postgres」

**固定局域网地址 (Pinned LAN Address)**：
为该 Mac 稳定下来的 IPv4（DHCP 预留或静态），供 HTTP 与 Realtime 主机名共用。
_Avoid_: 每次手抄漂移 IP、把家庭 IP 提交进 Git

**本机密钥与主机文件 (Local Host Env)**：
仅存在于开发者机器、不入库的环境文件；存放真实 LAN IP、测试白名单等。
_Avoid_: 把真实 IP/白名单写进可提交的 yaml

**联调白名单账号 (Lab Whitelist Account)**：
允许在单设备策略下多端同时在线的指定测试账号；普通账号仍单设备。
_Avoid_: 局域网模式下全局关闭单设备限制

**明文局域网传输 (Cleartext LAN Transport)**：
在信任局域网内使用 `http`/`ws`；HTTPS/`wss` 为后续可选加固，不是当前前提。
_Avoid_: 一上来就要求内网证书/反代才能联调

## 门店与身份

**门店 (Store)**：
多名用户可以同属的一家组织。一名用户也可以同属多家门店。
_Avoid_: 个人统计名片、把 `wys_user_store_stats` 的一行当成店

**门店职务 (Store Position)**：
某人在某家门店的岗位标签（销售顾问、销售经理、总经理）。不是访问权限。
_Avoid_: 角色、role、权限角色

**权限角色 (Access Role)**：
决定能做什么的角色。不是门店职务。
_Avoid_: 职务、岗位、统计卡上的 role

**平台角色 (Platform Role)**：
不挂任何门店的权限角色。
_Avoid_: 店内角色、超级管理员职务

**店内角色 (Store Role)**：
必须挂在某一家门店上的权限角色。离开那家店就不生效。
_Avoid_: 平台角色、门店职务

**角色分配 (Role Assignment)**：
把一个权限角色授给一个用户。平台分配不属于任何门店；店内分配只在指定的那一家门店成立。同一套角色定义各店共用。
_Avoid_: 把店写在角色定义上、每店复制一套角色

**当前店 (Current Store)**：
这个用户此刻正在操作的那一家门店。
_Avoid_: 他所属的全部门店、统计卡上的 store_id

**有效权限 (Effective Permissions)**：
平台角色能做的事始终算数，不依赖当前店，也不要求是成员。店内角色能做的事，只有在当前店、且他是该店成员时才算数。账号能不能登录不由角色决定。
_Avoid_: 只看职务、平台管理员必须先成为成员才能管店

**门店成员 (Store Member)**：
一个用户属于某一家门店的身份，上面有且只有一个门店职务。不是该店成员，就不能持有这家店的店内角色。
_Avoid_: 统计卡、权限角色、一人兼多个职务

**门店统计 (Store Stats)**：
某个用户在某家门店的展示数字，每人每店至多一行。不是成员名单，也不是职务来源。
_Avoid_: 一店一行、从统计卡读职务、把统计卡上的人当成成员

**权限 (Permission)**：
可以授给权限角色的一项能力。作用范围包含在这项能力的含义里。多角色只做并集，没有「拒绝」。
_Avoid_: 菜单、职务、账号状态

**内置角色 (Built-in Role)**：
随迁移发布的权限角色。运行时不能新建角色，也不能改它上面的权限。
_Avoid_: 店内自建角色、运行时角色目录

## 社区动态

**动态 (Post)**：
社区里一条用户发布的内容单元；可带正文、可选媒体、可选关联话题。
_Avoid_: 帖子 feed item、朋友圈、说说

**媒体类型 (Media Type)**：
一条动态上媒体的形态：无媒体、仅图片、仅视频；图与视频互斥。
_Avoid_: 附件类型、混排 media、图+视频同帖

**话题 (Topic)**：
可被动态关联的社区话题实体；展示名带 `#`，可检索、可统计热度。
_Avoid_: 正文里的任意 hashtag 字符串、Realtime topic、频道

**关联话题 (Topic Association)**：
一条动态与至多一个话题实体之间的绑定；未关联时可为空。
_Avoid_: 多话题挂载、把正文 `#` 解析结果当成关联本身

**问大家 (Ask Everyone)**：
一种特殊发帖模式：动态带 `is_ask_everyone`；本阶段无真实持仓域，发帖后向种子用户列表投递 `sys.notify` 邀请来回答（评论即回答）。
_Avoid_: 真实持仓匹配、门店职务 position、普通话题

**社区公约弹窗 (Convention Dialog)**：
发布页上的知悉弹窗；纯客户端控制，每天最多展示一次，展示记录只存在客户端。
_Avoid_: 服务端 ack、发帖前置校验、法律签署
