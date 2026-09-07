# 每日上游同步

每天北京时间（`Asia/Shanghai`，UTC+8）04:00 检查上游提交，不依赖版本号
或发布标签变化。默认跟随上游的后端、数据库迁移、支付、OAuth、安全策略和
公共接口更新；保留本项目明确要求的 UI、文案、24 个主题及同步自动化。

## 执行与验证

生产同步机只负责读取、合并、静态审核、推送和查询 GitHub 状态。包括这台
机器上的临时 worktree 在内，都禁止执行后端测试、Go/TypeScript 编译、前端
构建、包管理器脚本、代码生成、Docker 构建或开发服务器。允许
`git diff --check`、`gofmt -d`、`bash -n`、`sh -n` 等低成本静态检查。

1. `scripts/check-upstream.sh` 抓取上游，生成 `.codex-upstream-sync/report.md`。
2. 包装脚本在受资源限制的 systemd 服务中运行合并代理，写入审核记录和
   `result.json`。若前次 CI 失败，先查看其运行链接、失败注释和日志，修复原因。
3. 保存 `candidate.json` 后，推送到 `sync/upstream-candidate`。GitHub `CI`
   运行同步恢复测试、网关初始化/会话捕获回归、完整后端单元与集成测试、前端
   检查（包含 24 个主题、文案键完整性）及静态检查。
4. 只有该候选 SHA 的 CI 明确成功、且远端 main 和候选均未被替换，才将同一
   提交推送到 `origin/main` 并推进 `last-seen-head`。
5. `Build and Deploy` 在 GitHub 再次校验、构建并发布镜像，然后按仓库部署
   配置更新生产服务并检查健康状态。候选 CI 成功与生产部署成功是两个状态，
   分别以各自的 Action 结果为准。

前后端失败摘要会出现在 Action 注释和 Summary 中，完整测试日志作为 artifact
保留 14 天。不要关闭测试、忽略退出码，或因暂时无法查询状态而直接推送生产。

## 定时配置

当前生产主机时区为 `America/New_York`，Debian cron 不支持 `CRON_TZ`。
已安装的任务同时覆盖夏令时、冬令时，并用北京小时筛选实际执行时间：

```cron
0 15,16 * * * [ "$(TZ=Asia/Shanghai date +\%H)" = 04 ] && /sub2api/deploy/cron/upstream-sync
```

这条表达式仅适用于当前纽约时区主机。GitHub/API 时间通常以 UTC 显示，包装
脚本日志含主机时区偏移，不要直接把日志小时当成北京时间。

需要 Git、Node.js 20+、Codex、systemd 和 flock；`upstream` 的 push URL 必须
保持 `DISABLED`，只向自己的 `origin` 推送。并发任务通过锁排他执行。

## 状态查询与恢复

默认等待 CI 最长 45 分钟，每 60 秒查询一次。查询按候选分支和 SHA 筛选，
避免读取大量无关运行；HTTP 403/429、网络错误、5xx 和无效响应均视为状态
未知。遇到限流遵循 `Retry-After` / `X-RateLimit-Reset`，其他短暂错误采用
退避等待。错误不会变成“测试失败”或“测试通过”。

公共仓库可匿名查询。为减少共享 IP 限流，可通过运行环境提供 `GITHUB_TOKEN`
或 `GH_TOKEN`；也支持 `GITHUB_TOKEN_FILE` 指向仓库外的只读令牌文件。令牌
只需读取 Actions 状态的权限，文件限制为运行用户可读（0600），不提交令牌
或把它打印到日志。缺少令牌不会阻止公共仓库同步。401/404 会记录查询配置
问题并保留候选，修正权限或地址后续跑。

| 情况 | 本次处理 | 下次执行 |
| --- | --- | --- |
| CI 尚未出现或正在执行 | 继续查询，日志保留运行链接 | 续接同一候选 |
| 查询异常或等待到期 | 保存状态和候选；不推进同步基线 | 读取 `candidate.json`，免重复合并和推送 |
| CI 明确失败、取消或 runner 超时 | 保存运行链接；归档为 `candidate.failed.json` | 代理读取失败记录，修复后生成新候选 |
| main 或远端候选已被替换 | 拒绝用旧 CI 结果推广，归档旧候选 | 对当前 main 重新审核；保留不同步的本地改动 |
| 推送 main 后进程中断 | 保留尚未清理的候选记录 | 核对该 SHA 的 CI 和 main，补写同步状态 |

记录都在 `.codex-upstream-sync/`：`analysis.md` 是合并审核，`candidate.json`
是待完成候选，`ci-status.json` 是最后观察到的状态及运行链接，
`last-seen-head` 只表示已推广/完成审核的上游位置。日志在 `logs/` 中。
本地临时合并在失败退出时可能回到审核前的位置，远端候选和续跑记录仍保留；
这不表示生产服务被回滚。不要手工把 `last-seen-head` 改成待处理上游头。

正常续跑使用同一入口，它会优先恢复已有候选：

```sh
/sub2api/deploy/cron/upstream-sync
```

## 已知兼容点与保留要求

- Ent：新增本地后端字段后，合并 schema 与生成代码必须一起核对，特别是
  `ent/runtime/runtime.go` 的字段索引。此前的 `int` 转 `string` panic 来源
  于索引与 schema 错位；详细处理见技能 playbook，生成和验证都交给 Actions。
- 网关：保留上游单 handler 的 `rootRoute` 及认证→模型白名单→复合路由顺序。
  本地会话捕获通过 `ConversationCaptureWithHandler` 包装最终 handler，保留
  Responses、Chat Completions、Anthropic、Gemini 和 WebSocket 捕获能力。
- 文案：跟随上游翻译键的移动并更新所有引用，保留中性文案和自有品牌要求。
  例如 `admin.settings.siteNamePlaceholder` 已迁移到
  `admin.settings.site.siteNamePlaceholder`。
- 主题：以 `frontend/src/composables/useAppTheme.ts` 注册的全部 24 个主题为准。
  CSS、加载入口、切换/预览组件、存储键及多语言名称必须保留；旧文档中的
  “五个主题”不能作为更新清单。Actions 检查数量、实际文件和导入是否完整。

`custom/protected-paths.txt` 是源码合并保护清单，不是操作系统的不可变文件
标志。部署拉取 GitHub 构建的镜像，压缩后的 JS/CSS 与源码不同属于正常构建
结果，不能要求二者哈希相同。自动上游合并不得覆盖保护路径；仓库所有者明确
要求修订自有流程或 UI 时，可进行对应修改并让 Actions 验证。
