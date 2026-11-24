# 飞书员工同步、API Key 发放、充值与用量查询接口

站点：`https://api.beaconchip-token.com`。接口统一前缀：`/api/v1`。

用途：外部系统查询员工、指定一个或多个本站用户，创建或获取各自的企业专用 Key，然后自行处理后续交付。**这两个新接口均不发送邮件，也不创建邮件任务，不需要配置 SMTP。**

本文也包含系统**已经上线的用户余额充值、飞书人员同步和用户用量查询接口**，分别见第 6、7、8 节，无需增加后端接口。全部示例统一使用同一个管理员登录 Token。

| 用途 | 方法与路径 | 章节 |
| --- | --- | --- |
| 管理员登录 | `POST /api/v1/auth/login` | 1 |
| 查询员工及本站用户 ID | `GET /api/v1/admin/users/feishu-key-delivery/employees` | 2 |
| 创建或复用专用 Key | `POST /api/v1/admin/users/feishu-key-delivery/issue` | 3 |
| 单人／批量增加余额 | `POST /api/v1/admin/users/batch-recharge` | 6 |
| 启动飞书人员同步 | `POST /api/v1/admin/users/feishu-sync` | 7 |
| 查询同步进度 | `GET /api/v1/admin/users/feishu-sync` | 7 |
| 查询指定用户的请求数、Token 与消费金额 | `GET /api/v1/admin/usage/stats?user_id=25` | 8 |
| 批量查询用户消费金额 | `POST /api/v1/admin/dashboard/users-usage` | 8 |

每人只有一把企业发放专用 Key，与用户管理页「一键发送 Key」共用。首次调用创建，以后返回同一把；员工自己创建的其他 Key 不受影响。首次赠额只记一次，重试或换入口调用不重复赠额。

## 最短调用流程

1. 用管理员登录邮箱和密码登录，取得 `access_token`。
2. 有新员工尚未同步时，按第 7 节启动同步，并用 GET 等待同一任务完成。
3. 查员工列表，取得 `synced:true` 且 `user_id>0` 的要发放人员。
4. 如果员工没有可用余额，且未配置首次赠额或有效订阅，先按第 6 节给用户账户充值。
5. 调用创建接口，传 `{"user_id":25}` 或 `{"user_ids":[25,26]}`，读取返回的 `items[].key`。
6. 后续按第 8 节用本站 `user_id` 查询该员工的用量与消费金额。

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

需要通过接口同步尚未建立本站账号的员工时，使用第 7 节的启动与进度查询接口。同步成功后重新查询本节列表，才能取得新创建的本站用户 ID。

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
import time
import urllib.parse
import urllib.request
from datetime import date

BASE = "https://api.beaconchip-token.com/api/v1"

def request(path, token=None, body=None, idempotency_key=None, method=None):
    headers = {"User-Agent": "Sub2API-KeyProvision/1.0"}
    if token:
        headers["Authorization"] = "Bearer " + token
    if idempotency_key:
        headers["Idempotency-Key"] = idempotency_key
    data = None
    if body is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(body).encode()
    req = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
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

def sync_employees(token, timeout=960):
    # POST 无请求体；服务端异步执行。之后只用 GET 轮询，不反复 POST。
    job = request("/admin/users/feishu-sync", token, method="POST")
    job_id = job["id"]
    stop_at = time.monotonic() + timeout
    while True:
        if job is None or job["id"] != job_id:
            raise RuntimeError("当前同步任务已过期或被替换，请人工核对")
        if job["status"] == "succeeded":
            return job
        if job["status"] in ("partial", "failed", "approval_required"):
            raise RuntimeError("同步未完整成功：" + json.dumps(job, ensure_ascii=False))
        if job["status"] not in ("preparing", "reading", "syncing"):
            raise RuntimeError("未知同步状态：" + str(job["status"]))
        if time.monotonic() >= stop_at:
            raise TimeoutError("等待超时；先 GET 查询原任务，不要直接重新启动")
        time.sleep(3)
        job = request("/admin/users/feishu-sync", token)

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

def user_usage(token, user_id, period="today", start_date=None, end_date=None,
               api_key_id=None):
    # 必须传正数 user_id，避免退化为全站汇总；日期范围必须成对传入。
    if isinstance(user_id, bool) or not isinstance(user_id, int) or user_id <= 0:
        raise ValueError("user_id 必须是本站用户正整数 ID")
    if (start_date is None) != (end_date is None):
        raise ValueError("start_date 与 end_date 必须同时填写")
    if start_date is None and period not in ("today", "week", "month"):
        raise ValueError("period 仅支持 today、week、month")
    query = {"user_id": user_id, "timezone": "Asia/Shanghai", "nocache": 1}
    if start_date is not None:
        first, last = date.fromisoformat(start_date), date.fromisoformat(end_date)
        if first.isoformat() != start_date or last.isoformat() != end_date or first > last:
            raise ValueError("请传入开始不晚于结束的 YYYY-MM-DD 日期")
        query.update(start_date=start_date, end_date=end_date)
    else:
        query["period"] = period
    if api_key_id is not None:
        if isinstance(api_key_id, bool) or not isinstance(api_key_id, int) or api_key_id <= 0:
            raise ValueError("api_key_id 必须是 Key 的正整数 ID")
        query["api_key_id"] = api_key_id
    return request("/admin/usage/stats?" + urllib.parse.urlencode(query), token)

def batch_user_costs(token, user_ids):
    # 返回以字符串用户 ID 为键的字典；统计窗口固定，见第 8.2 节。
    return request("/admin/dashboard/users-usage", token,
                   {"user_ids": list(user_ids)})["stats"]

# token = login(admin_email, admin_password)
# 有新员工时先同步；返回成功后再读取名单。
# sync_result = sync_employees(token)
# employees = unissued_employees(token)
# 需要充值时：每人增加 100；下一笔业务需换新 request_id。
# recharge_users(token, [25, 26], 100, "employee-credit-20261124-001", "员工额度充值")
# result = issue_keys(token, [25, 26])
# for item in result["items"]:
#     if item["status"] in ("created", "reused"):
#         your_delivery_function(item["user_id"], item["key"])
#     else:
#         your_failure_handler(item["user_id"], item["reason"])
# stats = user_usage(token, 25, start_date="2026-11-01", end_date="2026-11-24")
# print(stats["total_requests"], stats["total_tokens"], stats["total_actual_cost"])
# costs = batch_user_costs(token, [25, 26])
# print(costs["25"]["today_actual_cost"])
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

## 7. 同步飞书人员（现有接口）

使用系统设置中已保存并启用的飞书应用；请求中不用再传应用密钥、企业 ID 或员工邮箱。同步按当前应用与企业的飞书身份绑定识别人，不按姓名或邮箱合并账号。

### 7.1 启动同步

`POST /api/v1/admin/users/feishu-sync`

只需管理员 `Authorization` 请求头，**无请求体**，不需要 `Idempotency-Key`。返回 HTTP 200 表示后台任务已启动或复用了正在执行的任务，**不代表同步已完成**。

```bash
curl --fail-with-body -sS --max-time 125 -X POST \
  'https://api.beaconchip-token.com/api/v1/admin/users/feishu-sync' \
  -H "Authorization: Bearer $TOKEN"
```

返回任务对象：初始 `status` 是 `preparing`，`id` 用于识别本次任务。已有任务正在执行时返回原任务，不另开一份；任务结束后再 POST 才会新建任务。

服务端先为当前应用配置通讯录只读权限、申请全员通讯录范围并提交发布，再读取部门和员工的全部分页。实际可读取范围受飞书审批结果限制；企业管理员仍需在官方页面完成必要的授权或审核。

### 7.2 查询进度与结果

`GET /api/v1/admin/users/feishu-sync`

```bash
curl --fail-with-body -sS --max-time 125 \
  'https://api.beaconchip-token.com/api/v1/admin/users/feishu-sync' \
  -H "Authorization: Bearer $TOKEN"
```

建议每 2–5 秒 GET 一次。该接口返回**全站当前／最近一份任务**，没有 `job_id` 查询参数；应核对返回的 `id` 与启动时一致。所有管理员看到同一份进度，关闭页面不会取消任务。每次进度保存的有效期为 1 小时；没有可用任务记录时返回 `{"code":0,"message":"success","data":null}`。

完成示例（演示数据）：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "启动接口返回的任务ID",
    "status": "succeeded",
    "total": 20,
    "processed": 20,
    "created": 3,
    "updated": 16,
    "skipped": 1,
    "failed": 0,
    "emails_filled": 3,
    "emails_missing": 0,
    "emails_conflict": 0,
    "emails_protected": 0,
    "emails_unchanged": 16,
    "started_at": 1795492800,
    "deadline": 1795493700
  }
}
```

| `status` | 含义与后续动作 |
| --- | --- |
| `preparing` | 配置权限和提交应用发布；继续 GET |
| `reading` | 读取飞书通讯录；继续 GET |
| `syncing` | 创建或更新本站用户；继续 GET |
| `succeeded` | 遍历完成且没有单人写入失败；重新查询第 2 节员工列表，取得新的 `user_id` |
| `partial` | 遍历完成但有单人失败，`failed>0`；已成功人员保留，不要按全员成功处理 |
| `approval_required` | 权限／审批不足；按 `error_code` 处理，完成飞书审批后重新 POST |
| `failed` | 整体任务失败；检查 `error_code`，已完成的单人同步不会整体回滚 |

后台执行时限为 **15 分钟**。`started_at` 与 `deadline` 是 Unix 秒时间戳。`total` 是通讯录读取并按应用内 `open_id` 去重后的记录数；读取阶段可能仍为 0。`processed` 是已处理数量；`created`、`updated`、`skipped`、`failed` 分别为新建、已有账号处理成功、跳过和单人失败数量，`updated` 不表示每人的资料都发生了变化。

| 可选 `error_code` | 处理 |
| --- | --- |
| `permission_required` | 当前实现识别到通讯录字段、权限或审批不足；到飞书处理权限与发布审核 |
| `expired` | 超过任务执行时限；先核对已同步的名单，再决定是否重新启动 |
| `config_changed` | 同步期间飞书配置已变化；核对当前应用和企业后重新启动 |
| `upstream_error` | 飞书接口返回错误；核对应用发布状态、权限和可访问范围 |
| `service_error` | 本站服务或任务存储异常；查看服务日志后重试 |

`error_code` 没有值时可能省略；`partial` 的单人失败不会另附逐人错误列表。客户端等待超时或网络断开时，先 GET 查询原任务，不要直接连续 POST。

### 7.3 同步会修改什么

- 在职、已激活且未冻结的员工会创建普通用户，用户名取飞书姓名；已有身份绑定复用原本站用户。离职、未加入等状态跳过；本站已禁用用户也跳过。站点开启“仅管理后台”模式时，会阻止创建新普通用户。
- 首次创建沿用飞书注册渠道的默认余额、并发和订阅等配置；重复同步不会重新发放注册赠额。同步不创建企业专用 Key、不执行第 6 节批量充值、不发送邮件。
- 邮箱优先取飞书 **`enterprise_email`（企业邮箱）**，不可用时取 **`email`（联系邮箱）**。只回填飞书注册账号的占位邮箱，不覆盖已有真实邮箱，也不会因为邮箱相同而把两个账号合并。
- 邮箱计数：`emails_filled` 已回填；`emails_missing` 无可用邮箱；`emails_conflict` 邮箱已被占用；`emails_protected` 保留现有邮箱；`emails_unchanged` 无需修改。即使 `succeeded`，也应核对邮箱缺失、冲突等计数；这不等于每人都有可发送的真实邮箱。
- 当前流程不会因某人从通讯录消失而自动删除或禁用其本站账号。员工离职停用需由现有账户管理流程处理。

同步任务不返回新用户的 ID 列表。**完成后再 GET 第 2 节员工接口**，以 `synced:true`、`user_id>0` 为准；需要找待发放人员时加 `key_issued=false`，取完所有分页再批量发放。

## 8. 查询用户用量与消费金额（现有真实统计接口）

### 8.1 指定用户：请求数、Token 与消费金额

`GET /api/v1/admin/usage/stats`

使用同一个管理员登录 Token。本文查询员工时必须明确传入**正整数本站 `user_id`**；不传或传非正数会失去用户过滤，可能变成全站汇总。接口不会仅凭“统计结果为 0”证明该用户存在，应先从第 2 节取得 ID。

| 查询参数 | 本文用法 | 说明 |
| --- | --- | --- |
| `user_id` | 必填 | 本站用户 ID；如 `25` |
| `period` | 可选 | `today`（默认）、`week`、`month`，含义见下文 |
| `start_date`、`end_date` | 可选，成对填写 | `YYYY-MM-DD`，开始日期不晚于结束日期；同时填写时覆盖 `period` |
| `timezone` | 建议填写 | 如 `Asia/Shanghai`；未填或无效时回退服务器时区 |
| `api_key_id` | 可选 | Key 的数据库数字 ID；与 `user_id` 同时过滤，可只查发放响应中的 `key_id`，不要传完整 Key |
| `group_id` | 可选 | 只查指定分组；不传则包含该用户各分组 |
| `nocache` | 可选 | `1` 绕过统计缓存直接查库；默认统计缓存 TTL 为 30 秒 |

`today` 是所选时区当天 00:00 至现在；`week` 是从当前时间往前 7 天至现在；`month` 是从当前时间往前一个月至现在，**不是自然月，也不等同固定 30 天**。不要使用未实现的 `period=all`，未知值会回退为今天。

日期范围包含 `end_date` 整天：如 `2026-11-01` 至 `2026-11-24`，查询边界为所选时区的 11 月 1 日 00:00（含）至 11 月 25 日 00:00（不含）。两个日期必须一起传；只填一个会被现有实现忽略，转而按 `period` 查询。

```bash
# 查询员工今天的全部 Key 用量。
curl --fail-with-body -sS --max-time 125 --get \
  'https://api.beaconchip-token.com/api/v1/admin/usage/stats' \
  -H "Authorization: Bearer $TOKEN" \
  --data-urlencode 'user_id=25' \
  --data-urlencode 'period=today' \
  --data-urlencode 'timezone=Asia/Shanghai' \
  --data-urlencode 'nocache=1'

# 查询指定日期范围；如仅查企业专用 Key，再加 --data-urlencode 'api_key_id=101'。
curl --fail-with-body -sS --max-time 125 --get \
  'https://api.beaconchip-token.com/api/v1/admin/usage/stats' \
  -H "Authorization: Bearer $TOKEN" \
  --data-urlencode 'user_id=25' \
  --data-urlencode 'start_date=2026-11-01' \
  --data-urlencode 'end_date=2026-11-24' \
  --data-urlencode 'timezone=Asia/Shanghai' \
  --data-urlencode 'nocache=1'
```

成功响应示例（演示数据，省略可选的接口维度明细）：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_requests": 12,
    "total_input_tokens": 10000,
    "total_output_tokens": 2000,
    "total_cache_tokens": 3000,
    "total_cache_creation_tokens": 1000,
    "total_cache_read_tokens": 2000,
    "total_tokens": 15000,
    "total_cost": 1.50,
    "total_actual_cost": 1.20,
    "total_account_cost": 0.90,
    "average_duration_ms": 850
  }
}
```

| 返回字段 | 含义 |
| --- | --- |
| `total_requests` | 匹配条件的已入库使用日志条数；此接口未排除零费用日志，不能当作成功请求数 |
| `total_input_tokens`、`total_output_tokens` | 输入／输出 Token 合计 |
| `total_cache_creation_tokens`、`total_cache_read_tokens` | 缓存创建／读取 Token 合计 |
| `total_cache_tokens` | 上面两个缓存字段之和 |
| `total_tokens` | 输入 + 输出 + 缓存 Token；已包含缓存，不能再重复加缓存量 |
| `total_actual_cost` | 用户计费口径的消费金额，包含实际分组倍率等计费结果；金额单位沿用本站美元计费口径 |
| `total_cost` | 日志中的标准费用合计；不是用户余额，也不等同实际消费金额 |
| `total_account_cost` | 来源账号统计费用；核对员工消费时使用 `total_actual_cost` |
| `average_duration_ms` | 平均请求耗时，毫秒 |

统计按已入库日志计算，正在执行或尚未完成异步入库的请求可能尚未出现，`nocache=1` 也不会等待其完成。没有匹配日志时返回零值。只传 `user_id` 会累计该用户全部 Key；增加 `api_key_id` 才只查某一把。

消费金额和余额是两个查询：余额仍用第 6.3 节的 `GET /api/v1/admin/users/25` 读取 `data.balance`。`total_actual_cost` 可包括订阅计费记录，因此不能直接用“充值总额减此值”推算当前现金余额。

**不要使用 `GET /api/v1/admin/users/:id/usage` 做本次对接。**当前代码中的这个旧接口仍是固定返回零值的占位实现；本节使用的 `/admin/usage/stats` 会查询真实使用日志。

### 8.2 批量查询用户消费金额

`POST /api/v1/admin/dashboard/users-usage`

这是只读查询，使用同一个管理员 Token，不需要 `Idempotency-Key`，不会扣费。请求体只需本站用户 ID 数组：

```json
{ "user_ids": [25, 26] }
```

```bash
curl --fail-with-body -sS --max-time 125 \
  'https://api.beaconchip-token.com/api/v1/admin/dashboard/users-usage' \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"user_ids":[25,26]}'
```

成功响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "stats": {
      "25": { "user_id": 25, "today_actual_cost": 1.20, "total_actual_cost": 12.50 },
      "26": { "user_id": 26, "today_actual_cost": 0, "total_actual_cost": 0 }
    }
  }
}
```

- `data.stats` 是以**字符串用户 ID**为键的对象，不是数组；Python 读 `result["stats"]["25"]`。可选 `by_platform` 是按平台拆分的同口径金额。
- `today_actual_cost` 为**服务器时区**当天的用户消费金额；`total_actual_cost` 在这个批量接口中是**最近 30 天至查询时刻**的用户消费金额，**不是历史累计**。
- 本接口仅汇总 `actual_cost>0` 的使用日志，不返回请求数或 Token。要按自定义日期、时区或 Key 查询，逐用户调用 8.1。
- 请求不支持 `period`、起止日期或 `timezone`，不要向请求体追加这些参数并期待其生效。结果缓存 TTL 为 30 秒；本接口没有实现 `nocache` 参数。
- 正数 ID 自动去重；没有用量时返回零值，非正数会被忽略。不存在的正数 ID 也可能返回零值，不能据此确认员工存在；始终先取本站员工列表。

### 8.3 本文核对范围

本次按当前已上线版本的路由、处理器、服务与数据库查询代码逐项核对，包括同步任务状态、邮箱字段、统计时间边界、缓存及费用字段。新增内容是现有接口的说明与调用示例，没有修改后端行为，不需要重新部署。

核对代码依据：[同步任务与状态](https://github.com/alanbulan/sub2apipro/blob/612cee25c704b43487cedb291c5a35b77cff00a7/backend/internal/service/feishu_directory.go)、[用量参数解析](https://github.com/alanbulan/sub2apipro/blob/612cee25c704b43487cedb291c5a35b77cff00a7/backend/internal/handler/admin/usage_handler.go#L221)、[用量与批量消费 SQL](https://github.com/alanbulan/sub2apipro/blob/612cee25c704b43487cedb291c5a35b77cff00a7/backend/internal/repository/usage_log_repo_stats.go)、[旧用量接口的占位实现](https://github.com/alanbulan/sub2apipro/blob/612cee25c704b43487cedb291c5a35b77cff00a7/backend/internal/service/admin_user.go#L703)。链接固定到本次核对的版本。

按项目要求未在本地运行测试或构建；未使用真实管理员凭证执行线上同步、充值、发放或认证后的用量查询。文中的 JSON 均为结构示例，不是线上实测数据。

## 错误与原有登录策略

| 情况 | 处理 |
| --- | --- |
| HTTP 400 | 检查参数、ID、默认/指定分组与服务端发放配置 |
| HTTP 401 | Token 缺失、失效或会话已撤销；重新登录或刷新 |
| HTTP 403 | 需要管理员登录 JWT，或现有安全策略要求二次验证 |
| 同步 HTTP 404 / `OAUTH_DISABLED` | 系统设置中的飞书登录未启用；先配置并启用 |
| 同步 HTTP 400 / `FEISHU_CONFIG_INVALID`、`FEISHU_REDIRECT_INVALID` | 系统设置中的当前飞书应用或回调配置不完整／无效 |
| 同步 HTTP 409 / `FEISHU_SETUP_BUSY` | 其他飞书配置操作占用锁；稍后先 GET 查询任务，再决定是否重试 POST |
| 用量查询 HTTP 400 | 检查数字 ID、成对的 `YYYY-MM-DD` 日期等；不要传姓名、邮箱或完整 Key 代替 ID |
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

如果登录返回 `requires_2fa:true`，将 `temp_token` 和 `totp_code` 提交到 `POST /api/v1/auth/login/2fa`，取得正式 `access_token`。如果发放或启动同步返回 `STEP_UP_REQUIRED`，用同一 Token 调用 `POST /api/v1/user/totp/step-up`，请求体 `{"code":"六位验证码"}`，成功后重试原操作。未启用这些策略时不增加额外步骤。

开启登录验证码时，`/auth/login` 仍需要原有的 `turnstile_token`，或腾讯验证码的 `tencent_captcha_ticket` 与 `tencent_captcha_randstr`。也可使用已经在本站完成登录的管理员 Token；不会为此接口绕过验证码。启用会话 IP/UA 绑定时保持同一出口和 User-Agent。

Token 刷新沿用 `POST /api/v1/auth/refresh`，请求体 `{"refresh_token":"登录返回的刷新Token"}`，随后使用新 `access_token`。具体会话策略沿用系统设置。

员工列表和 Key 发放接口返回 `Cache-Control: no-store`。完整 Key 只出现在明确的发放响应中，员工列表和任务记录不包含完整 Key；外部系统自行保管并交付这些凭证。
