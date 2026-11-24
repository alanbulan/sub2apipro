# 飞书员工 API Key 发放与余额充值接口

站点：`https://api.beaconchip-token.com`。接口统一前缀：`/api/v1`。

用途：外部系统查询员工、指定一个或多个本站用户，创建或获取各自的企业专用 Key，然后自行处理后续交付。**这两个新接口均不发送邮件，也不创建邮件任务，不需要配置 SMTP。**

本文也补充了系统**已经上线的用户余额充值接口**，见第 6 节，无需增加后端接口。充值和发放 Key 使用同一个管理员登录 Token。

每人只有一把企业发放专用 Key，与用户管理页「一键发送 Key」共用。首次调用创建，以后返回同一把；员工自己创建的其他 Key 不受影响。首次赠额只记一次，重试或换入口调用不重复赠额。

## 最短调用流程

1. 用管理员登录邮箱和密码登录，取得 `access_token`。
2. 查员工列表，取得要发放的 `user_id`。
3. 如果员工没有可用余额，且未配置首次赠额或有效订阅，先按第 6 节给用户账户充值。
4. 调用创建接口，传 `{"user_id":25}` 或 `{"user_ids":[25,26]}`，读取返回的 `items[].key`。

分组、首次赠额、Key 上限和有效期默认使用用户管理页「一键发送 Key」旁齿轮中的共享配置。也可以在请求里传一个 `group_id`，覆盖首次创建时的分组。

尚未保存共享配置时，传 `group_id` 即可使用默认参数：不赠额、不限 Key 总额度、长期有效、并发继承用户限制。不需要为纯 API 调用填写邮件标题或 API 地址；账户仍需已有可用余额或有效订阅。

可用分组 ID 可用同一管理员 Token 调用 `GET /api/v1/admin/groups/all` 查询，取返回分组的 `id`；不要把来源账号 ID 当作分组 ID。

## 1. 账号密码登录

`POST /api/v1/auth/login`

```json
{
  "email": "admin@example.com",
  "password": "管理员登录密码"
}
```

这里的账号是本站管理员的**登录邮箱**，不是用户显示名称或飞书应用密钥。

成功返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "登录得到的 JWT",
    "refresh_token": "刷新 Token",
    "expires_in": 3600,
    "token_type": "Bearer",
    "user": { "id": 1, "role": "admin" }
  }
}
```

有效期以实际 `expires_in` 为准。之后的员工查询和 Key 发放只需带：

```http
Authorization: Bearer <access_token>
Content-Type: application/json
```

使用现有管理员登录认证；普通员工 Token 和 `x-api-key` 机器凭证不能调用这两个新接口。启用了验证码、登录 2FA 或敏感操作二次验证时，继续遵循本站原有登录流程，见文末。

## 2. 查询员工及是否已发放

`GET /api/v1/admin/users/feishu-key-delivery/employees`

| 查询参数 | 必填 | 说明 |
| --- | --- | --- |
| `key_issued` | 否 | `false` 查尚未发放专用 Key 的员工；`true` 查已发放；省略查全部 |
| `search` | 否 | 按姓名、企业邮箱或本站用户 ID 模糊匹配，最多 200 字节 |
| `page` | 否 | 从 1 开始，默认 1 |
| `page_size` | 否 | 1–100，默认 50 |

例：

```http
GET /api/v1/admin/users/feishu-key-delivery/employees?key_issued=false&page=1&page_size=100
Authorization: Bearer <access_token>
```

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "user_id": 25,
        "name": "陈小明",
        "enterprise_email": "xiaoming@example.com",
        "synced": true,
        "key_issued": false,
        "created_at": "2026-11-24T00:00:00Z"
      },
      {
        "user_id": 0,
        "name": "新同事",
        "enterprise_email": "new@example.com",
        "synced": false,
        "key_issued": false
      }
    ],
    "total": 2,
    "page": 1,
    "page_size": 100,
    "pages": 1
  }
}
```

- 从**当前飞书应用、当前企业的实时通讯录**读取，只列出在职且已激活的员工；已对应到本站禁用、删除或管理员账号的记录不列入发放对象。
- `user_id` 是本站用户 ID，创建接口使用它，不传飞书 `open_id`、邮箱或姓名。
- `synced:false`、`user_id:0` 表示飞书已有此人，但尚无当前应用对应的本站用户。先点「同步飞书用户」或让员工首次通过飞书登录，再重新查询。此 GET 不自动创建用户。
- `key_issued:false` 表示尚无企业专用 Key 的发放记录，便于发现新待发放人员；不代表飞书的人事入职日期。`created_at` 是本站用户创建时间。
- `key_issued:true` 表示曾发放；即使原 Key 后来被禁用或删除，该记录仍保留。列表不返回完整 Key。
- 查询时按 `pages` 取完所有分页。若要批量发放全部待发放人员，应**先收集完名单，再调用创建接口**，避免边发放边翻页导致 `key_issued=false` 的结果变化、跳过人员。

需要通过接口启动已有的用户同步时：`POST /api/v1/admin/users/feishu-sync`，无请求体；用 `GET` 同一路径查询进度，等待 `status` 为 `succeeded` 后重新查询员工。同步沿用现有的飞书权限和发布流程，企业管理员审核仍需在飞书官方页面完成；失败时查看 `error_code`，不要把失败当作名单已完整同步。

## 3. 创建或获取 Key，不发送邮件

`POST /api/v1/admin/users/feishu-key-delivery/issue`

单人：

```json
{ "user_id": 25 }
```

批量：

```json
{ "user_ids": [25, 26] }
```

临时指定首次创建的分组：

```json
{ "user_ids": [25, 26], "group_id": 3 }
```

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `user_id` | 二选一 | 单个本站用户 ID，正整数 |
| `user_ids` | 二选一 | 1–100 个本站用户 ID，重复 ID 自动去重 |
| `group_id` | 否 | 首次创建时指定一个分组；省略则沿用共享发放配置中的分组 |

`user_id` 与 `user_ids` 不能同时填写。余额、Key 额度等沿用服务端配置，请勿在请求里加入 `initial_balance`、`send_email` 等额外字段；未知字段会被拒绝。

批量结果逐人返回，成功和失败可并存：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "user_id": 25,
        "name": "陈小明",
        "status": "created",
        "key": "sk-示例完整Key",
        "key_id": 101,
        "group_ids": [3],
        "balance": 100,
        "initial_grant": 100,
        "quota": 500
      },
      {
        "user_id": 26,
        "name": "李同学",
        "status": "reused",
        "key": "sk-另一把示例Key",
        "key_id": 102,
        "group_ids": [3],
        "balance": 80,
        "initial_grant": 0,
        "quota": 500
      },
      {
        "user_id": 999,
        "name": "",
        "status": "failed",
        "reason": "identity_mismatch",
        "balance": 0,
        "initial_grant": 0,
        "quota": 0
      }
    ],
    "created": 1,
    "reused": 1,
    "failed": 1
  }
}
```

上例假设请求了 `[25,26,999]`。返回顺序遵循请求顺序，重复用户只返回一次。

| 字段 | 含义 |
| --- | --- |
| `status` | `created` 首次创建；`reused` 返回已有专用 Key；`failed` 此用户未成功返回 Key |
| `key` | 该用户的完整 Key；只在该条成功时出现 |
| `group_ids` | 这把 Key 实际绑定的分组 |
| `balance` | 该用户当前账户余额，包含本次首次赠额 |
| `initial_grant` | 本次新增的首次赠额；复用时为 0 |
| `quota` | Key 总消费上限；0 表示不限额，不等于账户余额 |
| `expires_at` | 有过期时间时返回；省略表示长期有效 |

HTTP 200 和 `code:0` 表示批次已处理，**仍须检查 `failed` 和每条 `status`**。单个失败不会回滚其他用户的成功；响应超时后可用相同参数重试，已提交的用户复用原 Key，不重复赠额。数据库未提交的单个用户会同时回滚 Key、分组授权和赠额。

复用保留原 Key 的实际分组、限额、使用量和有效期。如果显式传了 `group_id`，而已有专用 Key 不恰好绑定这个单一分组，返回 `group_mismatch`，由管理员明确调整；接口不会暗中换 Key 或重置额度。

当前企业身份、启用状态、分组权限、有效订阅和余额条件都会校验。纯 Key 发放不要求员工邮箱有效；身份以当前飞书应用和企业下的绑定关系判断。

## 4. cURL 示例

以下示例使用 `jq` 提取 JSON；`ADMIN_EMAIL`、`ADMIN_PASSWORD` 和可选 `GROUP_ID` 由调用方提供。

```bash
BASE='https://api.beaconchip-token.com'

LOGIN=$(jq -n --arg email "$ADMIN_EMAIL" --arg password "$ADMIN_PASSWORD" \
  '{email:$email,password:$password}' | \
  curl --fail-with-body -sS "$BASE/api/v1/auth/login" \
    -H 'Content-Type: application/json' --data-binary @-)
TOKEN=$(printf '%s' "$LOGIN" | jq -er '.data.access_token')

# 查未发放员工。先确认 synced=true，并取 user_id。
curl --fail-with-body -sS --max-time 125 \
  "$BASE/api/v1/admin/users/feishu-key-delivery/employees?key_issued=false&page_size=100" \
  -H "Authorization: Bearer $TOKEN"

# 单人：已配置默认分组时，仅需 user_id。
curl --fail-with-body -sS --max-time 125 \
  "$BASE/api/v1/admin/users/feishu-key-delivery/issue" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"user_id":25}' | jq -er '.data.items[0].key'

# 批量：返回各自的 Key，后续交付由调用方处理。
curl --fail-with-body -sS --max-time 125 \
  "$BASE/api/v1/admin/users/feishu-key-delivery/issue" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"user_ids":[25,26],"group_id":3}'
```

## 5. Python 封装示例（标准库）

函数直接返回结构化结果，业务系统自行使用 `items[].key`；不会发送邮件。

```python
import json
import urllib.parse
import urllib.request

BASE = "https://api.beaconchip-token.com/api/v1"

def request(path, token=None, body=None, idempotency_key=None):
    headers = {"User-Agent": "Sub2API-KeyProvision/1.0"}
    if token:
        headers["Authorization"] = "Bearer " + token
    if idempotency_key:
        headers["Idempotency-Key"] = idempotency_key
    data = None
    if body is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(body).encode()
    req = urllib.request.Request(BASE + path, data=data, headers=headers)
    with urllib.request.urlopen(req, timeout=125) as response:
        payload = json.load(response)
    if payload["code"] != 0:
        raise RuntimeError(payload.get("message", "API request failed"))
    return payload["data"]

def login(email, password):
    result = request("/auth/login", body={"email": email, "password": password})
    if result.get("requires_2fa"):
        raise RuntimeError("请先完成本站登录 2FA，再使用 access_token")
    return result["access_token"]

def unissued_employees(token):
    # 先读取全部分页；此函数不会创建用户或 Key。
    rows, page = [], 1
    while True:
        query = urllib.parse.urlencode({
            "key_issued": "false", "page": page, "page_size": 100
        })
        result = request("/admin/users/feishu-key-delivery/employees?" + query, token)
        rows.extend(result["items"])
        if page >= result["pages"]:
            return rows
        page += 1

def issue_keys(token, user_ids, group_id=None):
    body = {"user_ids": list(user_ids)}  # 每次最多 100 个
    if group_id is not None:
        body["group_id"] = group_id
    return request("/admin/users/feishu-key-delivery/issue", token, body)

def recharge_users(token, user_ids, amount, request_id, notes=""):
    # 单人同样传 [user_id]；amount 是每人的增加额，不是多人合计金额。
    # 调用前保存 request_id 和请求体；同一笔业务重试必须复用它们。
    return request("/admin/users/batch-recharge", token, {
        "user_ids": list(user_ids), "amount": amount, "notes": notes
    }, idempotency_key=request_id)

# token = login(admin_email, admin_password)
# employees = unissued_employees(token)
# 需要充值时：每人增加 100；下一笔业务需换新 request_id。
# recharge_users(token, [25, 26], 100, "employee-credit-20261124-001", "员工额度充值")
# result = issue_keys(token, [25, 26])
# for item in result["items"]:
#     if item["status"] in ("created", "reused"):
#         your_delivery_function(item["user_id"], item["key"])
#     else:
#         your_failure_handler(item["user_id"], item["reason"])
```

## 6. 用户余额充值（现有接口，可直接调用）

充值增加的是指定用户的**账户余额**，不是管理员余额，也不是 Key 自身的消费上限。普通余额分组会按本站计费规则扣这个用户的余额；订阅分组仍按订阅规则计费。充值不会创建 Key、更改 Key 额度、绑定分组或发送邮件。

沿用第 1 节的管理员登录 JWT，增加请求头 `Idempotency-Key`，用来识别同一笔充值业务。**单人和批量都推荐使用下面的批量接口，单人传一个用户 ID 即可**；它同时提交余额和充值记录，并提供独立于 HTTP 幂等缓存期限的防重记录。

### 6.1 推荐：统一的单人／批量充值接口

`POST /api/v1/admin/users/batch-recharge`

```http
Authorization: Bearer <access_token>
Content-Type: application/json
Idempotency-Key: employee-credit-20261124-001
```

```json
{
  "user_ids": [25, 26],
  "amount": 100,
  "notes": "员工额度充值"
}
```

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `user_ids` | 是 | 1–500 个本站用户 ID，正整数；重复 ID 自动去重。单人填 `[25]` |
| `amount` | 是 | **每个用户增加的余额**，单位与本站账户余额一致；范围 `0.00000001` 至 `1000000000`，服务端按 8 位小数舍入 |
| `notes` | 否 | 充值备注，最多 1000 字节，进入管理员余额调整记录 |

例如原余额分别是 20 和 5，给 `[25,26]` 充值 `amount:100` 后，各自余额变成 120 和 105，总计增加 200。**不会把每人的余额直接覆盖为 100。**

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "affected": 2,
    "amount_per_user": 100,
    "total_amount": 200
  }
}
```

`affected` 是该业务批次去重后的用户数。重放同一笔已成功的请求仍返回该批次结果，不表示又增加了一次余额。

批量充值是**整批事务**：如果其中一个用户不存在、已删除或写入失败，整批余额和充值记录一同回滚；成功时每人都有一条余额调整记录。它与第 3 节 Key 发放的逐人成功／失败处理方式不同。不同用户需要不同金额时，拆成不同业务请求，分别使用唯一的幂等键。如启用了管理员充值返佣，沿用系统现有设置。

沿用前面取得的 `$TOKEN`，调用示例：

```bash
# 这是同一笔充值的固定编号。第一次调用前生成并保存，重试时原样复用。
RECHARGE_ID='employee-credit-20261124-001'

curl --fail-with-body -sS --max-time 70 \
  "$BASE/api/v1/admin/users/batch-recharge" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -H "Idempotency-Key: $RECHARGE_ID" \
  -d '{"user_ids":[25,26],"amount":100,"notes":"员工额度充值"}'
```

同一笔业务重试，保持**管理员账号、接口、幂等键和完整请求体不变**；新一笔充值生成新的唯一幂等键，例如 UUID，并保存到调用方的业务记录。不要在每次网络重试时生成新键，否则会被视为新一笔充值。

同一键配不同金额、名单或备注会返回 `IDEMPOTENCY_KEY_CONFLICT`。遇到网络超时或请求仍在处理中时，不要换键再次充值；先查询余额和充值记录，使用原键与原请求体重试。批量接口的成功防重记录独立于 HTTP 幂等缓存期限，保留期间可避免过期重试再次入账。

### 6.2 原有单人余额调整接口（可选）

`POST /api/v1/admin/users/25/balance`

同样携带管理员 Token、`Content-Type: application/json` 和一个唯一的 `Idempotency-Key`。

```json
{
  "balance": 100,
  "operation": "add",
  "notes": "员工额度充值"
}
```

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `balance` | 是 | 大于 0 的数值；`operation:add` 时表示**增加多少余额** |
| `operation` | 是 | 充值使用 `add`；`set` 是覆盖余额，`subtract` 是减余额 |
| `notes` | 否 | 管理员余额调整备注 |

成功时 `data` 是更新后的用户对象，可从 `data.id` 和 `data.balance` 读取用户 ID、调整后的余额。该接口的请求幂等记录遵循系统配置的保存期限；需要长期重试防重时，优先用 6.1 的统一接口。

### 6.3 查询余额与充值记录

使用同一个管理员 Token：

| 用途 | 请求 | 主要返回字段 |
| --- | --- | --- |
| 查询用户当前余额 | `GET /api/v1/admin/users/25` | `data.id`、`data.balance` |
| 查询用户的管理员余额调整记录 | `GET /api/v1/admin/users/25/balance-history?type=admin_balance&page=1&page_size=20` | `data.items`、`data.total`、`data.page`、`data.pages` |

余额调整记录中的 `value` 是该条调整金额，`notes` 是充值备注。按 `pages` 分页读取；查看是否完成充值时，把调用方保存的业务编号、请求体和备注一并用于对账。`Idempotency-Key` 不会作为独立字段出现在用户余额记录中，可以将业务编号写入 `notes` 便于查询。

## 错误与原有登录策略

| 情况 | 处理 |
| --- | --- |
| HTTP 400 | 检查参数、ID、默认/指定分组与服务端发放配置 |
| HTTP 401 | Token 缺失、失效或会话已撤销；重新登录或刷新 |
| HTTP 403 | 需要管理员登录 JWT，或现有安全策略要求二次验证 |
| 充值 HTTP 400，提示 `Idempotency-Key is required` 或 `IDEMPOTENCY_KEY_REQUIRED` | 补上 `Idempotency-Key`，检查正数金额和用户 ID；批量接口每次最多 500 个 ID |
| 充值 HTTP 409 / `IDEMPOTENCY_KEY_CONFLICT` | 同一个幂等键对应了不同请求；核对业务记录，不要换键盲目重试 |
| 充值 HTTP 409 / `IDEMPOTENCY_IN_PROGRESS`、`IDEMPOTENCY_RETRY_BACKOFF` | 原请求处理中或处于退避期，按 `Retry-After` 等待后复用原键与请求体重试 |
| 充值 HTTP 503 / `IDEMPOTENCY_STORE_UNAVAILABLE` | 幂等存储暂不可用；保留原业务编号和请求体，恢复后再重试 |
| HTTP 423 / `ADMIN_COMPLIANCE_ACK_REQUIRED` | 该管理员需先在现有管理页面完成一次站点确认，再重试；沿用全站管理接口策略 |
| HTTP 429 | 触发站点现有限流，退避后重试 |
| 批次内 `identity_mismatch` | 此 ID 没有当前应用/企业的唯一飞书绑定；从员工接口重新取 ID |
| `inactive_employee` / `inactive_user` | 飞书员工或本站用户不可用 |
| `group_unavailable` / `group_not_allowed` | 分组停用、权限或订阅不足 |
| `balance_insufficient` | 余额不足；按站内规则充值或设置首次赠额 |
| `key_unavailable` | 原专用 Key 被删除、禁用、过期或耗尽；管理员先处理原 Key |
| `key_rate_limited` / `key_ip_restricted` | 原 Key 的窗口额度或 IP 限制阻止直接可用 |
| `group_mismatch` | 显式请求的分组与原专用 Key 不一致 |
| `config_changed` | 飞书应用已切换；重新查询当前企业员工 |
| `service_error` | 单人事务未完成或读取失败，可重试原参数 |

如果登录返回 `requires_2fa:true`，将 `temp_token` 和 `totp_code` 提交到 `POST /api/v1/auth/login/2fa`，取得正式 `access_token`。如果发放返回 `STEP_UP_REQUIRED`，用同一 Token 调用 `POST /api/v1/user/totp/step-up`，请求体 `{"code":"六位验证码"}`，成功后重试发放。未启用这些策略时不增加额外步骤。

开启登录验证码时，`/auth/login` 仍需要原有的 `turnstile_token`，或腾讯验证码的 `tencent_captcha_ticket` 与 `tencent_captcha_randstr`。也可使用已经在本站完成登录的管理员 Token；不会为此接口绕过验证码。启用会话 IP/UA 绑定时保持同一出口和 User-Agent。

Token 刷新沿用 `POST /api/v1/auth/refresh`，请求体 `{"refresh_token":"登录返回的刷新Token"}`，随后使用新 `access_token`。具体会话策略沿用系统设置。

员工列表和 Key 发放接口返回 `Cache-Control: no-store`。完整 Key 只出现在明确的发放响应中，员工列表和任务记录不包含完整 Key；外部系统自行保管并交付这些凭证。
