# my_go_study — 本机局域网后端

本仓库在「家庭/信任 Wi‑Fi 上，用开发者 Mac 充当 Flutter 可连的 BFF」这一上下文中的用语。与公网生产托管无关。

## Language

**局域网后端主机 (LAN Backend Host)**：
在同 Wi‑Fi 下向真机提供 Go BFF 的那台 Mac（按需启停，不是公网服务器）。
_Avoid_: 生产服务器、云主机、公网部署

**LAN 配置档 (LAN Profile)**：
专为局域网真机联调准备的运行姿态，与纯本机 `dev`、以及 `prod` 区分开。
_Avoid_: 直接把 prod 当作家用联调、长期混用未区分的 dev 当「家里服务器」

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
