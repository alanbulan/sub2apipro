# 飞书企业登录与扫码配置

更新于 2026-10-20。实现已合入 main；后续修复的部署和 CI 结果以交付记录为准。
本机不运行测试、构建、代码生成或安装依赖。没有真实企业凭证时，CI 验证官方协议契约，
最终企业授权与发布审批须在管理员扫码时由飞书平台执行，不能宣称已完成真实租户联调。

## 管理员操作

1. 使用公网 HTTPS 域名打开站点。回调优先使用已配置前端地址，否则从当前浏览器地址自动生成。
2. 系统设置的登录设置内，打开“飞书企业登录”，点击“扫码配置飞书登录”。
3. 使用本企业飞书扫码，在飞书官方页面确认创建应用和授权。
4. 保持设置页打开。后端自动获取 App ID / App Secret，通过企业 API 识别 tenant_key，
   添加 OAuth 回调、启用网页应用、设置本企业全员可见并提交发布。
5. 页面显示“已提交”后配置已保存，登录入口启用。若企业要求审批，需在飞书开放平台
   完成管理员审批。发布 API 成功不等于审批通过；企业政策不能由本站绕过。

无需手工输入 App ID、App Secret 或 tenant_key。飞书配置独立保存，不依赖底部保存按钮。
飞书开关可关闭入口；关闭后尚未完成的飞书认证失效，已有本站会话按原策略管理。
重新扫码会创建新的应用并替换配置；open_id 按应用隔离，原应用身份不会自动迁移。
旧账号仍可通过原有站内方式登录并重新绑定。仅企业成员登录限制作用于飞书入口，
其他登录方式及公开注册开关沿用原配置；已验证的本企业员工自动开户独立于公开注册设置。

## 官方接口与权限

依据飞书开放平台 Markdown 文档及官方 Go SDK 的 One-Click App Registration：

| 阶段 | 官方接口 |
| --- | --- |
| 二维码建应用 | POST `https://accounts.feishu.cn/oauth/v1/app/registration`，action=begin/poll |
| 获取企业应用 token | POST `https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal` |
| 识别企业 | GET `/open-apis/tenant/v2/tenant/query` |
| 添加回调和可见范围 | PATCH `/open-apis/application/v7/applications/:app_id/config` |
| 网页应用能力 | PATCH `/open-apis/application/v7/applications/:app_id/ability` |
| 提交发布 | POST `/open-apis/application/v7/applications/:app_id/publish` |
| 网页登录授权 | GET `https://accounts.feishu.cn/open-apis/authen/v1/authorize` |
| 授权码换用户 token | POST `https://accounts.feishu.cn/oauth/v3/token`，form-urlencoded，confidential client |
| 用户身份 | GET `https://open.feishu.cn/open-apis/authen/v1/user_info` |

扫码模板 `archetype=PersonalAgent`、`auth_method=client_secret`，addons 禁用默认大权限预设，
应用身份申请 `application:application:patch`（配置自己的应用）及 `tenant:tenant:readonly`（识别企业）；
用户身份申请 `contact:user.email:readonly` 与 `contact:user.employee:readonly`，读取当前员工的
联系方式邮箱及飞书企业邮箱。不申请文档、聊天、手机号或 offline_access 权限。应用能力/权限是否允许创建及
审批要求由官方和企业决定；平台返回错误时设置页提供重试，原有效配置保持不变。

网页能力配置显式传 `pc_new_page_open_mode=browser`。官方文档虽标为可选，真实扫码新建应用
在缺省时返回 HTTP 400 / `210011`；补齐后同一应用返回 HTTP 200 / `code=0`。
OpenAPI 的 `error` 为诊断对象，OAuth 的 `error` 为字符串，必须分开解析；保留数字业务码，
不把业务拒绝误报为 `invalid_response`，也不向浏览器暴露原始响应、凭证或诊断链接。

二维码地址仅接受 HTTPS 的 `open.feishu.cn` 和 `accounts.feishu.cn`，不允许用户信息、其他域名或额外端口。
实际注册响应使用 `https://open.feishu.cn/page/launcher?user_code=…` 及 `expires_in`；兼容旧 SDK 的
`expire_in`，但本站二维码状态最多保留 10 分钟。轮询的 `authorization_pending` / `slow_down`
可能与 HTTP 400、非零诊断 `code` 同时返回，按 OAuth 状态处理；其他错误仍终止流程。
专用 Actions 工作流调用真实官方接口验证申请和未授权轮询，不打印设备码、二维码地址或凭证，
不扫码、不创建企业应用，也不代替真实企业授权验证。

回调为站点前端地址加 `/api/v1/auth/oauth/feishu/callback`，前端落地页为
`/auth/feishu/callback`。生产使用 HTTPS，开发仅允许 loopback HTTP。
授权和换码的 redirect_uri 完全一致。授权码为短期、一次性凭证。
当前 v3 confidential client 通过 client_secret 换码；官方授权文档对 PKCE 指向历史 v2
接口，因此本实现不混用 v3 与未经确认的 PKCE 参数。

tenant_key 来自企业查询 API，不能把企业 display_id 当成 tenant_key；只接受 tenant_tag=0 的企业/团队版，拒绝个人版。
用户 user_info 每次必须成功且包含 open_id、tenant_key，离职、冻结、缺字段、其他企业均拒绝。

邮箱自动补齐：优先 `enterprise_email`（企业须启用飞书邮箱），其次 `email`。
已有应用可在系统设置点击“升级当前应用的邮箱读取权限”：只追加两项用户权限并提交发布，
不新建应用、不替换凭证、不改变 `open_id` 绑定。企业审批仍须在飞书官方页面完成。
登录开始时通过 `/application/v6/scopes` 查询已授权权限，只把已批准的邮箱权限加入 OAuth
scope，避免待审权限导致已有登录报 20027；查询失败则继续基础登录。

官方依据：

- [官方 SDK 一键注册](https://github.com/larksuite/oapi-sdk-go#one-click-app-registration)
- [授权码](https://open.feishu.cn/document/authentication-management/access-token/obtain-oauth-code)
- [v3 用户 token](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/authentication-management/access-token/get-user-access-token-v3)
- [用户信息](https://open.feishu.cn/document/server-docs/authentication-management/login-state-management/get)
- [企业信息](https://open.feishu.cn/document/server-docs/tenant-v2/query)
- [应用配置](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/application-v7/application-v7/application-config/patch)
- [应用能力](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/application-v7/application-v7/application-ability/patch)
- [应用发布](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/application-v7/application-v7/application-publish/create)

## 身份、安全与现有行为

- provider_type=`feishu`，provider_key=`feishu:<app_id>:<tenant_key>`，subject=`open_id`。
  union_id、姓名、飞书管理员填写的邮箱均不作为本站账号自动合并依据。
- 本企业新员工首次登录自动创建普通用户，采用飞书姓名及有效头像；无需填写邮箱、密码、
  邀请码或确认资料，不受公开注册开关及邮件域名策略限制，仍遵守登录验证码和后台模式。
  仅在姓名为空时生成默认显示名；后续登录不覆盖员工已修改的姓名、头像、权限或余额。
  公开注册的同 IP 终身开户数限制不作用于已验证员工，办公室共享 IP 可正常使用；
  原请求的限流、封禁和防暴力尝试检查继续生效，其他登录提供方不受此豁免影响。
- 账号、飞书身份和头像在同一事务创建，按应用/企业/open_id 加进程锁及 PostgreSQL 事务锁，
  并发首次登录只创建一个账号、发放一次新用户默认权益；失败回滚，不留下无身份的账号。
  没有有效、可用的飞书邮箱时，账号使用身份摘要加 `@feishu-connect.invalid` 满足历史字段要求。
  有邮箱时自动写入 `users.email`；已有占位邮箱在下次飞书登录时补齐。已是真实邮箱的账号保持原值。
  补齐只改邮箱、不改密码、姓名、余额、权限或默认权益；邮箱及收件箱别名被占用时继续原账号登录，
  不自动合并账号。数据库按邮箱/收件箱身份锁查重，并以占位邮箱条件更新，避免覆盖并发的验证绑定。
  官方说明 `email` 为管理员导入、未实时验证的联系方式，因此导入邮箱不建立 email 身份；
  登录记录和普通资料更新也不能补写 email 身份。不接受该资料邮箱的密码登录、密码找回或跨提供方
  邮箱自动合并，且不能据此解绑唯一飞书登录。员工可在已登录状态下验证邮箱并设置本地密码，
  完成后才建立 email 身份、启用原有邮箱认证和找回流程；飞书扫码登录全程不要求邮箱验证。
- 已有本站账号的员工应先登录原账号，在个人设置中绑定飞书；目标来自已登录用户的签名 cookie，
  不按同名或未验证的飞书邮箱自动合并账号。本站已停用或已删除账号不能通过重新扫码恢复。
- 已绑定账号也检查本站状态、后台模式和 TOTP，再签发本站 token。
- 复用共享 pending session、绑定、解绑、单次消费和登录持久化。每一步重新检查
  飞书配置指纹，禁用、换企业、换应用或换密钥后旧 pending 会话不能完成。
- 浏览器不接触 App Secret、device_code、飞书用户 token。二维码凭证存在 Redis，
  使用 JWT secret 派生的独立用途 AES-GCM 密钥加密，最长 10 分钟。
- 二维码状态绑定发起管理员；Redis 分布式锁、5 秒最短轮询、slow_down 退避、
  同管理员最新二维码限制、30 秒创建限制、配置指纹比较，防止并发覆盖及旧二维码替换。
- 配置 API 仅追加本站回调，不删除其他回调。配置完整并提交发布后才替换当前配置。
  网络失败可继续原流程；发布请求超时存在平台端已处理的可能，按平台结果处理，不能把失败伪装成功。
- Pro“记住我”、24 个主题、品牌、原有登录提供方和更新渠道保留。飞书新用户继承全局注册
  默认权益，不额外发放 provider 首次绑定赠额。
- 本次仅增加 `291_add_feishu_provider.sql`，扩展五张表的允许值；不改写历史 SQL，
  不新增 Ent 字段或改变 runtime 字段索引。

## 上游合并契约

飞书专用文件列入 `custom/protected-paths.txt`。共享文件仍接收上游认证、安全、设置和 UI 改动，
合并时保留：登录/注册入口、系统设置卡片、公开配置、profile 绑定、路由、Wire 注入、
provider 校验、schema 验证器、pending 配置重验、TOTP 分支及 CI 测试入口。
保留员工自动开户、保留域名的邮箱身份排除、仅已验证企业员工的注册 IP 额度豁免；
保留飞书联系方式邮箱不自动转换成邮箱登录身份的资料更新/登录回填/找回/合并防护及显式验证绑定入口；
共享注册防护代码仍合并上游安全修复，不可把企业豁免扩大到其他提供方。
共享 LinuxDoCallbackView 的 provider 参数继续兼容默认 LinuxDo 行为，飞书拒绝旧 fragment token 流程。

CI 应包含自动开户直接登录、并发与回滚、同名不同员工、内部地址不绑定邮箱、公开注册限制隔离、
官方 API 契约、tenant 隔离、state 校验、未验证邮箱不合并、TOTP、配置指纹变更、
扫码凭证不泄漏、轮询/重试/过期、系统设置、现有 OAuth、rememberMePro、主题与文案完整性，
并在 GitHub 上完成前端生产构建及后端 embed 构建。禁止在同步机本地执行。

后续上游迁移若重建 provider/signup_source 约束并遗漏 feishu，须追加在其后执行的新迁移，
同时保留上游新增 provider；不得改写已应用的 SQL。`TestLatestAuthProviderConstraintsPreserveFeishu`
会在 Actions 检查最后一版约束，防止无冲突合并后静默丢失飞书支持。
