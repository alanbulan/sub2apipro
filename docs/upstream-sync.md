# Pro 的官方上游自动更新

唯一上游为 `https://github.com/Wei-Shaw/sub2api.git` 的 `main`。Pro 的发布仓库
仍为 `alanbulan/sub2apipro`；旧 `ranxi2001/production` 不再进入自动合并链路。
首次迁移见 [迁移对照](canonical-upstream-migration.md)，所有者确认前不改变生产来源或部署。

## 上线时切换来源

以下属于首次迁移批准后的切换步骤，不能在迁移审阅前运行：

```sh
git remote set-url upstream https://github.com/Wei-Shaw/sub2api.git
git remote set-url --push upstream DISABLED
git config remote.upstream.fetch '+refs/heads/main:refs/remotes/upstream/main'
```

将已审阅的原版 SHA 写入 `.codex-upstream-sync/last-seen-head`。旧候选和失败状态
先按原值归档，不能把旧 `production` 的 SHA 当成新的官方基线。正式同步 checkout
必须是干净的 `main`，且与 `origin/main` 一致；不得重置其他开发工作区。

## 后续自动更新

入口仍为 `deploy/cron/upstream-sync`，保留现有每日定时安排与互斥锁：

1. 检查分支、干净工作区、官方远端和禁用的 upstream 推送地址。
2. 有待处理候选时优先续跑；否则抓取官方 `main`，生成逐提交审核报告。
3. 在受内存、交换空间、进程数和时间限制的 systemd 服务中运行合并代理。
4. 代理按保护清单接入新版，并只进行低开销源码/格式检查。禁止本机测试、
   安装依赖、代码生成、构建或启动开发服务。
5. 推送 `sync/upstream-candidate`，等待 `backend-ci.yml` 的精确 SHA 检查。
   这个 CI 包含官方当前检查、Pro 回归、临时数据库迁移、前端构建和安全扫描。
6. 仅全部成功且 main/候选未改变时推广 main；之后由 `Build and Deploy`
   构建镜像并执行健康检查。只有推广成功后才推进已同步的上游位置。

`candidate.json` 保存前后 SHA 和上游 SHA；`ci-status.json` 保存运行 URL、状态；
`analysis.md` 保存审核依据。GitHub API 错误或超时属于未知状态，遵守
`Retry-After` / `X-RateLimit-Reset` 后重试，不推进基线。明确失败、取消或超时
归档为 `candidate.failed.json`，修复原因后重新提交；不能把它当成检查通过。

公共仓库可匿名读取状态；也支持仓库外只读 `GITHUB_TOKEN` / `GH_TOKEN` /
`GITHUB_TOKEN_FILE`，文件应为 0600。日志和报告不得输出令牌。

## 保留范围与数据库

保护清单覆盖 24 主题、白标、记住登录、飞书、邮箱回填、员工 Key、批量充值、
独立会话记录、用户账号池和智商检测。共享接口继续合入官方修改，同时保留
这些功能的接入点。CI 可以调整已有行，但必须保留当前官方及 Pro 检查契约；
已删除旧分支的历史任务矩阵和“只能追加 CI 行”的规定。

`custom/legacy-migrations/manifest.json` 记录迁移前 461 个 SQL 的来源、原始哈希
及程序使用的归一化 checksum。退役 SQL 放在不参与运行的归档目录；已存在的
旧表和 `schema_migrations` 记录保留，不能为裁剪功能而删除员工数据。
旧 250 号迁移仅末尾空行不同，程序先 TrimSpace 后计算 checksum，实际一致；
不需要放宽 checksum 规则。

`900_pro_canonical_policy_bridge.sql` 为有单组关联记录的旧 Key 兼容官方字段。
遇到多组或冲突分组、仍启用的旧垫付策略会停止迁移并保留现场，不能猜测策略。
新安装不创建退役的质量运维、垫付或多分组 Key 表。

## 部署与恢复

沿用精确 SHA 镜像、已固定 SSH 主机、公钥校验、磁盘/ inode 预检、排空请求与
健康检查。部署只更新应用，不在服务器构建、不操作数据卷。失败时恢复旧镜像，
保留数据库；加法迁移和历史 SQL 的可重放性必须先在 Actions 验证。
现有部署策略不自动制作数据库或配置备份，首次迁移审阅会明确这个边界。

脚本遇到用户未提交工作时立即退出，不自动清理。中断状态、旧候选与当前 main
不一致时，先保留现场并定位来源，不能直接 `reset --hard` 用户工作。
