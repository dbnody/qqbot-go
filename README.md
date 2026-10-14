# QQ Bot Go

这是一个基于 Go 的 OneBot V11 QQ 机器人。QQ 登录和消息协议由 NapCat 提供，本项目通过反向 WebSocket 接收事件并调用 OneBot API。

本仓库是独立的 Go 项目，不包含 QQ 登录数据、真实 Token、AI 密钥、数据库、NapCat 安装包或原 Python 项目的运行文件。

## 功能

### 协议与基础设施

- OneBot V11 反向 WebSocket
- Token 鉴权、事件接收和 API 请求响应关联
- 断线清理、请求超时和失败状态处理
- SQLite 数据库及并发写入保护
- /health 健康检查接口

### 已迁移的命令和插件

- /ping、/help
- /天气
- /签到
- /修仙、/修炼、/突破
- /今日运势、/一言
- /塔罗
- /点歌：QQ 音乐、网易云音乐搜索和音乐卡片回退
- /群统计、/发言排行、/词云
- /cp：群内每日 CP 配对及故事
- /投胎：投胎模拟器
- /meme：Meme 模板列表和图片生成
- 21 点、轮盘赌、猜谜、成语接龙、你画我猜、猜歌
- B 站链接解析和视频消息
- 群欢迎、退群通知和戳一戳响应
- 违禁词检测、删除消息、禁言、解禁和踢人
- AI 群聊/私聊、图片输入、记忆隔离、限流和记忆清理
- /锐评、/锐评头像
- 撤回消息监控及管理员转发

逐插件迁移状态和差异见 [MIGRATION_STATUS.md](MIGRATION_STATUS.md)。

## 快速开始

要求 Go 1.24 或更高版本。

1. 复制配置模板：

`powershell
Copy-Item .env.example .env
`

2. 编辑 .env，至少设置 OneBot Token：

`dotenv
ONEBOT_V11_ACCESS_TOKEN=填写与 NapCat 相同的 Token
`

3. 在 NapCat 的 OneBot V11 网络配置中添加反向 WebSocket：

`text
ws://127.0.0.1:8080/onebot/v11/ws
`

消息格式建议选择 array。

4. 检查、测试并启动：

`powershell
go test ./...
go vet ./...
go run ./cmd/qqbot
`

默认数据库为 data/qqbot.db。如需使用旧 Python 项目的 SQLite 数据库，可通过 QQBOT_DATABASE_PATH 指定路径。不要让两个 Bot 实例同时写入同一个数据库。

如果需要迁移 Python CP 历史记录，可以设置：

`dotenv
QQBOT_CP_HISTORY_PATH=C:/path/to/cp_history.json
`

启动时会以事务方式导入，重复导入不会覆盖已有 Go 记录。

## 主要配置

| 配置项 | 作用 |
| --- | --- |
| HOST、PORT | Go 服务监听地址和端口 |
| ONEBOT_V11_ACCESS_TOKEN | OneBot 反向 WebSocket Token |
| COMMAND_START | 命令前缀，默认 / |
| QQBOT_DATABASE_PATH | SQLite 数据库路径 |
| QQBOT_ADMIN_IDS | 撤回监控通知的管理员 QQ 号 |
| QQBOT_FORBIDDEN_WORDS | 逗号分隔的违禁词 |
| AI_API_KEY、AI_API_BASE_URL、AI_MODEL | AI 服务配置 |
| QQBOT_FONT_PATH | PNG 图片生成使用的中文字体路径 |

完整模板见 [.env.example](.env.example)。

## 未经真实环境验证的部分

当前代码测试使用模拟 OneBot WebSocket、模拟 HTTP 服务和临时 SQLite 数据库。以下内容还需要在目标 QQ/NapCat 环境中逐项验证：

- NapCat 真实连接、消息收发和断线重连
- 图片、视频、音乐卡片和 Base64 图片消息的实际发送效果
- AI 服务、视觉模型、B 站、音乐平台和天气接口的真实网络响应
- 群主/管理员权限下的禁言、解禁、踢人和撤回
- 不同 NapCat 版本的消息段格式差异
- Windows 系统字体、词云和 Meme 图片的实际渲染效果

本地曾使用当前 NapCat 配置启动 Go 服务并等待真实反向 WebSocket，但观察期间未建立连接；使用前需要在 NapCat 中确认 OneBot 配置已加载并完成重连。

已知行为差异：

- 21 点、轮盘赌等游戏状态保存在内存中，进程重启后会丢失
- 一言轮换状态保存在内存中，进程重启后会重置
- Tarot 牌面使用 Go 生成的抽象 PNG，不是 Python 版本的原始插画
- 部分外部服务的响应格式和限流策略可能发生变化
- 目前不能据此宣称与 Python/NoneBot 运行时完全等价

## 发布前检查

`powershell
go fmt ./...
go test ./...
go vet ./...
go build -o qqbot.exe ./cmd/qqbot
`

GitHub Actions 会在 Windows Runner 上执行格式化、测试、静态检查和构建。

请确认提交前没有加入 .env、数据库、日志、缓存、QQ 数据或 NapCat 安装目录。仓库当前未包含 GitHub 远端地址，也不会自动推送代码。

## 许可证

当前仓库未附带许可证。发布到 GitHub 前，请根据你的使用和分发需求补充合适的开源许可证。
