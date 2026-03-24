# 外部同步 API 对接指南

本文档说明如何通过 REST API 将外部系统（HR 系统、OA 系统、LDAP 等）的组织结构和人员数据同步到 Mattermost 组织通讯录插件。

---

## 目录

1. [准备工作](#1-准备工作)
2. [认证方式](#2-认证方式)
3. [核心概念](#3-核心概念)
4. [API 端点总览](#4-api-端点总览)
5. [全量同步](#5-全量同步)
6. [增量同步](#6-增量同步)
7. [显式删除（增量模式）](#7-显式删除增量模式)
8. [分步同步](#8-分步同步)
9. [外部查询接口](#9-外部查询接口)
10. [用户映射管理](#10-用户映射管理)
11. [查询同步日志](#11-查询同步日志)
12. [用户匹配策略](#12-用户匹配策略)
13. [完整对接示例](#13-完整对接示例)
14. [错误处理](#14-错误处理)
15. [最佳实践](#15-最佳实践)

---

## 1. 准备工作

### 1.1 配置 Sync API Token

在 Mattermost 系统控制台 → **插件** → **组织通讯录** 中，填写 **外部同步 API Token**：

- 建议使用 32 位以上的随机字符串
- Token 存储在服务端配置中，不会暴露给前端
- 可随时更换 Token（更换后旧 Token 立即失效）

```bash
# 生成随机 Token 示例（Linux/macOS）
openssl rand -hex 32
# 输出示例：a3f8b2c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1
```

### 1.2 确定 API Base URL

所有同步 API 的 Base URL 为：

```
https://<your-mattermost-host>/plugins/com.example.org-directory/api/v1/sync
```

---

## 2. 认证方式

同步 API 支持两种鉴权模式：

### 2.1 配置了 Sync API Token（推荐）

当插件配置了 **外部同步 API Token** 时，所有 `/api/v1/sync/*` 请求都必须使用 Bearer Token 鉴权，与 Mattermost 用户会话独立。

```http
Authorization: Bearer <your-sync-api-token>
Content-Type: application/json
```

### 2.2 未配置 Sync API Token（回退模式）

当插件未配置 **外部同步 API Token** 时，权限按 Mattermost 登录用户分级：

- **普通已登录 Mattermost 用户**：只能访问通讯录查询类 GET 接口
  - `GET /api/v1/sync/nodes`
  - `GET /api/v1/sync/nodes/{external_id}`
  - `GET /api/v1/sync/nodes/{external_id}/children`
  - `GET /api/v1/sync/nodes/{external_id}/members`
  - `GET /api/v1/sync/users/{external_user_id}/nodes`
- **系统管理员**：可访问全部 `/api/v1/sync/*` 接口，包括写接口、日志接口、映射接口

> 注意：`GET /api/v1/sync/logs`、`GET /api/v1/sync/logs/{id}`、`GET /api/v1/sync/user-mappings/{source}` 在未配置 Token 时仍仅限系统管理员访问。

**错误响应：**

| HTTP 状态码 | 原因 |
|------------|------|
| `401 Unauthorized` | 未提供有效 Sync Token，且请求也不是合法 Mattermost 登录用户 |
| `403 Forbidden` | 已登录用户缺少访问当前 sync 路由的权限 |

---

## 3. 核心概念

### 3.1 source 字段

每条数据都有 `source` 字段标识数据来源。不同来源的数据**完全隔离**，互不影响。

```
source = "hr_system"   # HR 系统同步的数据
source = "oa_system"   # OA 系统同步的数据
source = "ldap"        # LDAP 同步的数据
source = "local"       # 管理员在 Mattermost 中手动创建的数据（受保护，不受外部同步影响）
```

### 3.2 external_id 字段

`external_id` 是外部系统中记录的唯一标识符。插件通过 `(source, external_id)` 组合来识别和更新记录，避免重复创建。

### 3.3 同步类型

| sync_type | 说明 |
|-----------|------|
| `full` | **全量同步**：本次请求包含该来源的**全部**数据，同步完成后自动软删除该来源中不在本次请求中的旧记录 |
| `incremental` | **增量同步**：本次请求只包含**新增或变更**的数据，不删除任何旧记录 |

### 3.4 节点拓扑排序

发送节点数据时，**父节点必须在子节点之前出现**（按层级顺序）。插件内部会自动进行拓扑排序，但建议外部系统也按顺序发送，以便于排查问题。

---

## 4. API 端点总览

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/sync` | 同时同步节点和成员（推荐） |
| `POST` | `/api/v1/sync/nodes` | 仅同步节点 |
| `GET` | `/api/v1/sync/nodes?source=xxx` | 查询指定来源下节点，可按 `depth` / `max_depth` / `parent_external_id` 过滤 |
| `GET` | `/api/v1/sync/nodes/{external_id}?source=xxx` | 查询单个节点详情及路径 |
| `GET` | `/api/v1/sync/nodes/{external_id}/children?source=xxx` | 查询节点的直接子节点 |
| `GET` | `/api/v1/sync/nodes/{external_id}/members?source=xxx` | 查询节点上的用户，可递归子树 |
| `POST` | `/api/v1/sync/members` | 仅同步成员关系 |
| `GET` | `/api/v1/sync/users/{external_user_id}/nodes?source=xxx` | 按外部用户 ID 反查所属节点 |
| `POST` | `/api/v1/sync/user-mappings` | 批量写入用户映射 |
| `GET` | `/api/v1/sync/user-mappings/{source}` | 查询指定来源的用户映射 |
| `GET` | `/api/v1/sync/logs` | 查询同步日志（需 Bearer Token） |
| `GET` | `/api/v1/sync/logs/{id}` | 查询单条同步日志详情 |

> **管理员也可通过以下路由查询**（使用 Mattermost 用户会话，仅系统管理员可访问）：
> - `GET /api/v1/admin/sync/logs`
> - `GET /api/v1/admin/sync/logs/{id}`
> - `GET /api/v1/admin/sync/user-mappings/{source}`

---

## 5. 全量同步

全量同步适用于定期批量覆盖——插件会自动软删除该 `source` 下本次未包含的旧节点和成员关系。

### 5.1 请求格式

**`POST /api/v1/sync`**

```json
{
  "source": "hr_system",
  "sync_type": "full",
  "nodes": [
    {
      "external_id": "ORG-001",
      "name": "某某机构",
      "parent_external_id": "",
      "sort_order": 0,
      "description": "总机构"
    },
    {
      "external_id": "DEPT-010",
      "name": "xx管理部",
      "parent_external_id": "ORG-001",
      "sort_order": 1
    },
    {
      "external_id": "DEPT-010-01",
      "name": "xxx省",
      "parent_external_id": "DEPT-010",
      "sort_order": 0
    }
  ],
  "members": [
    {
      "node_external_id": "DEPT-010-01",
      "external_user_id": "HR-EMP-001",
      "external_username": "zhangsan",
      "external_email": "zhangsan@company.com",
      "role": "member",
      "position": "科长",
      "external_id": "EMP-001-N3",
      "sort_order": 0
    },
    {
      "node_external_id": "DEPT-010-01",
      "external_user_id": "HR-EMP-002",
      "external_username": "lisi",
      "external_email": "lisi@company.com",
      "role": "member",
      "position": "科员",
      "external_id": "EMP-002-N3",
      "sort_order": 1
    }
  ]
}
```

### 节点字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `external_id` | string | ✅ | 外部系统中该节点的唯一 ID |
| `name` | string | ✅（upsert）| 节点名称（最长 256 字符）；`action=delete` 时可省略 |
| `parent_external_id` | string | | 父节点的 `external_id`，空字符串表示根节点 |
| `sort_order` | int | | 同级节点排序序号，默认 0 |
| `description` | string | | 节点描述 |
| `icon` | string | | 节点图标 |
| `metadata` | string | | 扩展元数据（JSON 字符串），默认 `{}` |
| `action` | string | | `""` / `"upsert"`（默认，新建或更新）；`"delete"`（删除该节点）|

### 成员字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `node_external_id` | string | ✅ | 目标节点的 `external_id` |
| `external_user_id` | string | ✅ | 外部系统中该用户的唯一 ID |
| `external_username` | string | | 外部系统用户名（用于自动匹配 MM 用户）|
| `external_email` | string | | 外部系统邮箱（用于自动匹配 MM 用户）|
| `role` | string | | `member`（默认）/ `manager` / `admin` |
| `position` | string | | 职位名称 |
| `external_id` | string | | 该成员关系在外部系统中的唯一 ID；若外部系统存在独立的“人员-节点关系主键”，强烈建议传入。未传时，插件会使用 `source + node_external_id + external_user_id` 作为自然键进行幂等 upsert，并在全量同步时按该自然键清理旧记录 |
| `sort_order` | int | | 节点内用户排序序号 |
| `action` | string | | `""` / `"upsert"`（默认，新建或更新）；`"delete"`（移除该成员）|

### 5.2 响应格式

```json
{
  "sync_log_id": "xk8m2n3p4q5r6s7t8u9v0w",
  "status": "success",
  "total_nodes": 3,
  "created_nodes": 3,
  "updated_nodes": 0,
  "deleted_nodes": 0,
  "total_members": 2,
  "created_members": 2,
  "updated_members": 0,
  "deleted_members": 0,
  "skipped_users": 0,
  "skipped_details": [],
  "errors": []
}
```

### 响应字段说明

| 字段 | 说明 |
|------|------|
| `sync_log_id` | 本次同步的日志 ID，可用于后续查询详情 |
| `status` | `success` / `partial_success` / `failed` |
| `skipped_users` | 未能匹配到 Mattermost 用户的成员数量 |
| `skipped_details` | 每个跳过用户的详细信息（`external_user_id`、`reason`）|
| `errors` | 节点/成员处理中的错误信息列表 |

---

## 6. 增量同步

增量同步只传递新增或变更的数据，不会删除任何已有记录。

> 成员同步的幂等规则：
> - 传入 `external_id` 时，优先按 `source + external_id` 识别同一条成员关系；若系统中已存在同一 `node_external_id + external_user_id` 的关系，则会补写/更新该关系的 `external_id`
> - 未传入 `external_id` 时，按 `source + node_external_id + external_user_id` 识别同一条成员关系
> - `action=delete` 时，按 `node_external_id + external_user_id` 删除成员关系，不要求必须提供 `external_id`

```json
{
  "source": "hr_system",
  "sync_type": "incremental",
  "nodes": [
    {
      "external_id": "DEPT-030",
      "name": "新成立部门",
      "parent_external_id": "ORG-001"
    }
  ],
  "members": [
    {
      "node_external_id": "DEPT-030",
      "external_user_id": "HR-EMP-010",
      "external_username": "wangwu",
      "external_email": "wangwu@company.com",
      "role": "admin",
      "position": "部门负责人"
    }
  ]
}
```

**典型使用场景：**
- 人员入职/离职触发的实时推送
- 部门新增/更名的即时同步
- 每小时/每 15 分钟的变更增量推送

---

## 7. 显式删除（增量模式）

当外部系统需要**精确删除**某个节点或成员时，无需执行全量同步——在增量请求中将对应条目的 `action` 设为 `"delete"` 即可。

### 7.1 删除成员（人员离职）

```json
{
  "source": "hr_system",
  "sync_type": "incremental",
  "members": [
    {
      "action": "delete",
      "node_external_id": "DEPT-010-01",
      "external_user_id": "HR-EMP-001",
      "external_email": "zhangsan@company.com"
    }
  ]
}
```

> 删除成员时，用户匹配逻辑与 upsert 相同（映射表 → 邮箱 → 用户名）。如无法匹配到 Mattermost 用户，该条目会被记入 `skipped_users`。

### 7.2 删除节点（部门解散）

```json
{
  "source": "hr_system",
  "sync_type": "incremental",
  "nodes": [
    {
      "action": "delete",
      "external_id": "DEPT-010-01"
    }
  ]
}
```

> 节点删除遵循插件配置中的**全量同步删除策略**（`cascade_delete` 或 `move_to_parent`），行为与管理员手动删除一致。

### 7.3 同一请求中混合增删

`action` 字段可以在同一请求的不同条目中混用：

```json
{
  "source": "hr_system",
  "sync_type": "incremental",
  "nodes": [
    { "external_id": "DEPT-020", "name": "新部门", "parent_external_id": "ORG-001" }
  ],
  "members": [
    {
      "action": "delete",
      "node_external_id": "DEPT-010-01",
      "external_user_id": "HR-EMP-001",
      "external_email": "zhangsan@company.com"
    },
    {
      "node_external_id": "DEPT-020",
      "external_user_id": "HR-EMP-005",
      "external_email": "newstaff@company.com",
      "role": "member",
      "position": "科员"
    }
  ]
}
```

---

## 8. 分步同步

对于数据量较大的场景，可以将节点和成员分步同步。

### 8.1 仅同步节点

**`POST /api/v1/sync/nodes`**

```json
{
  "source": "hr_system",
  "sync_type": "full",
  "nodes": [
    { "external_id": "ORG-001", "name": "某某机构" },
    { "external_id": "DEPT-010", "name": "xx管理部", "parent_external_id": "ORG-001" }
  ]
}
```

### 8.2 仅同步成员

**`POST /api/v1/sync/members`**

```json
{
  "source": "hr_system",
  "sync_type": "incremental",
  "members": [
    {
      "node_external_id": "DEPT-010",
      "external_user_id": "HR-EMP-001",
      "external_email": "zhangsan@company.com",
      "role": "member"
    }
  ]
}
```

> **建议顺序：** 先全量同步节点，再全量同步成员；或全部通过 `/api/v1/sync` 一次性提交。

---

## 9. 外部查询接口

除写入型同步接口外，插件也提供了一组只读查询接口，方便外部系统回读组织树、节点成员，以及按外部用户 ID 反查所属组织。

### 9.1 查询某来源下全部节点

**`GET /api/v1/sync/nodes?source=hr_system`**

可选查询参数：

| 参数 | 说明 |
|------|------|
| `source` | 必填，来源系统标识 |
| `depth` | 可选，非负整数；不带 `parent_external_id` 时表示返回“全局等于该深度”的节点；带 `parent_external_id` 时表示返回“该父节点下相对深度等于该值”的节点 |
| `max_depth` | 可选，非负整数；不带 `parent_external_id` 时表示返回“全局深度小于等于该值”的节点；带 `parent_external_id` 时表示返回“该父节点下相对深度小于等于该值”的节点 |
| `parent_external_id` | 可选，指定父节点的外部 ID；与 `depth`/`max_depth` 组合使用时，会按该父节点为基准过滤其后代节点，不包含父节点自身 |

```json
{
  "source": "hr_system",
  "nodes": [
    {
      "id": "node_root_01",
      "name": "某某机构",
      "parent_id": "",
      "parent_external_id": "",
      "path": "/node_root_01",
      "depth": 0,
      "sort_order": 0,
      "description": "总机构",
      "icon": "",
      "metadata": "{}",
      "source": "hr_system",
      "external_id": "ORG-001",
      "create_at": 1700000000000,
      "update_at": 1700000000000,
      "delete_at": 0,
      "creator_id": ""
    },
    {
      "id": "node_dept_01",
      "name": "xx管理部",
      "parent_id": "node_root_01",
      "parent_external_id": "ORG-001",
      "path": "/node_root_01/node_dept_01",
      "depth": 1,
      "sort_order": 1,
      "description": "",
      "icon": "",
      "metadata": "{}",
      "source": "hr_system",
      "external_id": "DEPT-010",
      "create_at": 1700000000100,
      "update_at": 1700000000200,
      "delete_at": 0,
      "creator_id": ""
    }
  ]
}
```

### 9.2 查询单个节点详情

**`GET /api/v1/sync/nodes/DEPT-010?source=hr_system`**

```json
{
  "node": {
    "id": "node_dept_01",
    "name": "xx管理部",
    "parent_id": "node_root_01",
    "parent_external_id": "ORG-001",
    "path": "/node_root_01/node_dept_01",
    "depth": 1,
    "sort_order": 1,
    "description": "",
    "icon": "",
    "metadata": "{}",
    "source": "hr_system",
    "external_id": "DEPT-010",
    "create_at": 1700000000100,
    "update_at": 1700000000200,
    "delete_at": 0,
    "creator_id": ""
  },
  "path": [
    {
      "id": "node_root_01",
      "name": "某某机构",
      "parent_external_id": "",
      "source": "hr_system",
      "external_id": "ORG-001"
    },
    {
      "id": "node_dept_01",
      "name": "xx管理部",
      "parent_external_id": "ORG-001",
      "source": "hr_system",
      "external_id": "DEPT-010"
    }
  ]
}
```

### 9.3 查询节点直接子节点

**`GET /api/v1/sync/nodes/ORG-001/children?source=hr_system`**

返回结构与 `GET /api/v1/sync/nodes?source=...` 中的 `nodes` 数组一致，适合按层懒加载外部系统自己的树组件。

### 9.4 查询节点上的用户

**`GET /api/v1/sync/nodes/ORG-001/members?source=hr_system&recursive=true&page=0&per_page=50`**

| 参数 | 说明 |
|------|------|
| `source` | 必填，来源系统标识 |
| `recursive` | 可选，`true` 表示包含整个子树的用户 |
| `page` | 可选，页码，从 0 开始 |
| `per_page` | 可选，每页数量，默认 50 |

```json
{
  "node": {
    "id": "node_root_01",
    "name": "某某机构",
    "parent_external_id": "",
    "source": "hr_system",
    "external_id": "ORG-001"
  },
  "recursive": true,
  "total": 2,
  "members": [
    {
      "id": "member_01",
      "node_id": "node_root_01",
      "user_id": "mm_user_001",
      "role": "admin",
      "position": "负责人",
      "sort_order": 0,
      "source": "hr_system",
      "external_id": "EMP-001-ROOT",
      "username": "zhangsan",
      "first_name": "三",
      "last_name": "张",
      "nickname": "张三",
      "email": "zhangsan@company.com",
      "mm_position": "部长",
      "status": "online",
      "node_name": "某某机构",
      "node_source": "hr_system",
      "node_external_id": "ORG-001"
    },
    {
      "id": "member_02",
      "node_id": "node_dept_01",
      "user_id": "mm_user_002",
      "role": "member",
      "position": "科员",
      "sort_order": 1,
      "source": "hr_system",
      "external_id": "EMP-002-DEPT",
      "username": "lisi",
      "first_name": "四",
      "last_name": "李",
      "nickname": "李四",
      "email": "lisi@company.com",
      "mm_position": "员工",
      "status": "offline",
      "node_name": "xx管理部",
      "node_source": "hr_system",
      "node_external_id": "DEPT-010"
    }
  ]
}
```

### 9.5 按外部用户 ID 反查所属节点

**`GET /api/v1/sync/users/HR-EMP-001/nodes?source=hr_system`**

该接口先通过 `source + external_user_id` 查询映射关系，再返回该 Mattermost 用户在该来源下所属的全部节点。

```json
{
  "source": "hr_system",
  "external_user_id": "HR-EMP-001",
  "mm_user_id": "mm_user_001",
  "total": 2,
  "nodes": [
    {
      "id": "node_root_01",
      "name": "某某机构",
      "parent_external_id": "",
      "source": "hr_system",
      "external_id": "ORG-001"
    },
    {
      "id": "node_dept_01",
      "name": "xx管理部",
      "parent_external_id": "ORG-001",
      "source": "hr_system",
      "external_id": "DEPT-010"
    }
  ]
}
```

> 如果目标用户尚未建立映射关系，接口会返回 `404`。因此对于需要做用户反查的外部系统，建议先同步或预置用户映射。

---

## 10. 用户映射管理

当外部系统的用户 ID 与 Mattermost 的邮箱/用户名均无法自动对应时，可预先注册用户映射表。

### 10.1 批量写入映射

**`POST /api/v1/sync/user-mappings`**

```json
{
  "source": "hr_system",
  "mappings": [
    {
      "external_user_id": "HR-EMP-001",
      "mm_user_id": "abc123def456ghi789jkl",
      "external_username": "zhangsan",
      "external_email": "zhangsan@company.com"
    },
    {
      "external_user_id": "HR-EMP-002",
      "mm_user_id": "xyz987wvu654tsr321qpo",
      "external_username": "lisi",
      "external_email": "lisi@company.com"
    }
  ]
}
```

响应：

```json
{
  "created": 1,
  "updated": 1
}
```

### 10.2 查询映射（Bearer Token 鉴权）

**`GET /api/v1/sync/user-mappings/{source}?page=0&per_page=50`**

```json
[
  {
    "id": "...",
    "source": "hr_system",
    "external_user_id": "HR-EMP-001",
    "mm_user_id": "abc123def456ghi789jkl",
    "external_username": "zhangsan",
    "external_email": "zhangsan@company.com",
    "create_at": 1700000000000,
    "update_at": 1700000000000
  }
]
```

| 参数 | 说明 |
|------|------|
| `{source}` | 来源标识，如 `hr_system` |
| `page` | 页码，从 0 开始 |
| `per_page` | 每页数量，默认 50 |

---

## 11. 查询同步日志

### 11.1 查询日志列表

**`GET /api/v1/sync/logs?source=hr_system&page=0&per_page=20`**

```json
[
  {
    "id": "xk8m2n3p4q5r6s7t8u9v0w",
    "source": "hr_system",
    "sync_type": "full",
    "status": "partial_success",
    "total_nodes": 50,
    "created_nodes": 48,
    "updated_nodes": 2,
    "deleted_nodes": 0,
    "total_members": 120,
    "created_members": 115,
    "updated_members": 3,
    "deleted_members": 0,
    "skipped_users": 2,
    "error_message": "",
    "details": "{}",
    "started_at": 1700000000000,
    "finished_at": 1700000003500,
    "triggered_by": ""
  }
]
```

### 11.2 查询单条日志详情

**`GET /api/v1/sync/logs/{id}`**

### 同步状态说明

| status | 含义 |
|--------|------|
| `running` | 同步正在进行中 |
| `success` | 全部数据同步成功 |
| `partial_success` | 部分成功（有跳过的用户或轻微错误）|
| `failed` | 同步失败（无任何节点/成员成功写入）|

---

## 12. 用户匹配策略

同步成员时，插件按以下顺序将外部用户 ID 匹配到 Mattermost 用户：

```
1. 查询用户映射表 (external_user_id + source → mm_user_id)
         ↓ 未命中
2. 按邮箱精确匹配 (GetUserByEmail)
         ↓ 未命中
3. 按用户名精确匹配 (GetUserByUsername)
         ↓ 未命中
4. 记录到 skipped_users，跳过该成员，不中断整体同步
```

匹配成功后（步骤 2 或 3），会自动将结果写入用户映射表，下次同步直接命中步骤 1，无需重复查询。

### 策略配置

在插件配置中可选择匹配策略：

| 策略值 | 说明 |
|--------|------|
| `mapping_email_username`（推荐）| 映射表 → 邮箱 → 用户名，覆盖最广 |
| `mapping_only` | 仅查映射表，需预先注册所有映射关系 |
| `email_only` | 仅按邮箱匹配，适合邮箱与 MM 账号一一对应的场景 |

### 建议：预先注册映射

如果外部系统用户名/邮箱与 Mattermost 不完全对应，建议在首次全量同步前，先通过 `/api/v1/sync/user-mappings` 批量注册映射关系。

---

## 13. 完整对接示例

以下是一个 Python 脚本示例，展示如何从外部 HR 系统进行全量同步：

```python
import requests
import json

MATTERMOST_URL = "https://mattermost.company.com"
PLUGIN_ID = "com.example.org-directory"
SYNC_TOKEN = "your-32-char-random-token-here"
SOURCE = "hr_system"

BASE_URL = f"{MATTERMOST_URL}/plugins/{PLUGIN_ID}/api/v1/sync"
HEADERS = {
    "Authorization": f"Bearer {SYNC_TOKEN}",
    "Content-Type": "application/json",
}


def fetch_org_from_hr():
    """从 HR 系统获取组织数据（示例）"""
    # 实际对接时替换为真实的 HR API 调用
    return {
        "nodes": [
            {"external_id": "ORG-001", "name": "某某机构"},
            {"external_id": "DEPT-010", "name": "xx管理部", "parent_external_id": "ORG-001"},
            {"external_id": "DEPT-010-01", "name": "xxx省", "parent_external_id": "DEPT-010"},
        ],
        "members": [
            {
                "node_external_id": "DEPT-010-01",
                "external_user_id": "HR-EMP-001",
                "external_username": "zhangsan",
                "external_email": "zhangsan@company.com",
                "role": "member",
                "position": "科长",
                "external_id": "EMP-001-NODE-3",
            },
        ],
    }


def full_sync():
    hr_data = fetch_org_from_hr()
    payload = {
        "source": SOURCE,
        "sync_type": "full",
        "nodes": hr_data["nodes"],
        "members": hr_data["members"],
    }

    resp = requests.post(BASE_URL, headers=HEADERS, json=payload, timeout=60)
    resp.raise_for_status()
    result = resp.json()

    print(f"同步完成: {result['status']}")
    print(f"  节点: 新建 {result['created_nodes']}, 更新 {result['updated_nodes']}, 删除 {result['deleted_nodes']}")
    print(f"  成员: 新建 {result['created_members']}, 更新 {result['updated_members']}, 跳过 {result['skipped_users']}")

    if result["skipped_users"] > 0:
        print("未匹配用户:")
        for u in result.get("skipped_details", []):
            print(f"  - {u['external_user_id']} ({u.get('external_email', '')}): {u['reason']}")

    return result


if __name__ == "__main__":
    full_sync()
```

### cURL 示例

```bash
MATTERMOST_URL="https://mattermost.company.com"
SYNC_TOKEN="your-sync-token"
PLUGIN_ID="com.example.org-directory"

# 全量同步节点
curl -s -X POST \
  "${MATTERMOST_URL}/plugins/${PLUGIN_ID}/api/v1/sync/nodes" \
  -H "Authorization: Bearer ${SYNC_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "source": "hr_system",
    "sync_type": "full",
    "nodes": [
      {"external_id": "ORG-001", "name": "总公司"},
      {"external_id": "DEPT-001", "name": "研发部", "parent_external_id": "ORG-001"}
    ]
  }' | python3 -m json.tool

# 查询最新同步日志
curl -s \
  "${MATTERMOST_URL}/plugins/${PLUGIN_ID}/api/v1/sync/logs?source=hr_system&per_page=5" \
  -H "Authorization: Bearer ${SYNC_TOKEN}" | python3 -m json.tool

# 按来源查询全部节点
curl -s \
  "${MATTERMOST_URL}/plugins/${PLUGIN_ID}/api/v1/sync/nodes?source=hr_system" \
  -H "Authorization: Bearer ${SYNC_TOKEN}" | python3 -m json.tool

# 仅查询根节点（depth=0）
curl -s \
  "${MATTERMOST_URL}/plugins/${PLUGIN_ID}/api/v1/sync/nodes?source=hr_system&depth=0" \
  -H "Authorization: Bearer ${SYNC_TOKEN}" | python3 -m json.tool

# 查询 0~1 层全部节点
curl -s \
  "${MATTERMOST_URL}/plugins/${PLUGIN_ID}/api/v1/sync/nodes?source=hr_system&max_depth=1" \
  -H "Authorization: Bearer ${SYNC_TOKEN}" | python3 -m json.tool

# 查询某父节点下第一层子节点（相对深度 0）
curl -s \
  "${MATTERMOST_URL}/plugins/${PLUGIN_ID}/api/v1/sync/nodes?source=hr_system&parent_external_id=ORG-001&depth=0" \
  -H "Authorization: Bearer ${SYNC_TOKEN}" | python3 -m json.tool

# 查询某父节点下 0~1 层后代节点
curl -s \
  "${MATTERMOST_URL}/plugins/${PLUGIN_ID}/api/v1/sync/nodes?source=hr_system&parent_external_id=ORG-001&max_depth=1" \
  -H "Authorization: Bearer ${SYNC_TOKEN}" | python3 -m json.tool

# 查询某节点及其子树成员
curl -s \
  "${MATTERMOST_URL}/plugins/${PLUGIN_ID}/api/v1/sync/nodes/ORG-001/members?source=hr_system&recursive=true" \
  -H "Authorization: Bearer ${SYNC_TOKEN}" | python3 -m json.tool

# 按外部用户 ID 反查所属节点
curl -s \
  "${MATTERMOST_URL}/plugins/${PLUGIN_ID}/api/v1/sync/users/HR-EMP-001/nodes?source=hr_system" \
  -H "Authorization: Bearer ${SYNC_TOKEN}" | python3 -m json.tool
```

---

## 14. 错误处理

### HTTP 级别错误

| 状态码 | 含义 | 处理建议 |
|--------|------|---------|
| `400 Bad Request` | 请求体格式错误或必填字段缺失 | 检查 JSON 格式和必填字段 |
| `401 Unauthorized` | Token 无效或缺失 | 检查 Authorization 请求头 |
| `503 Service Unavailable` | 插件未配置 Sync API Token | 联系 Mattermost 管理员配置 Token |
| `500 Internal Server Error` | 服务端内部错误 | 查看同步日志，联系管理员 |

### 业务级别错误

响应体中的 `errors` 数组包含单条记录级别的错误：

```json
{
  "status": "partial_success",
  "errors": [
    "parent not found for node: DEPT-999",
    "failed to upsert node DEPT-100: ..."
  ]
}
```

**常见错误及处理：**

| 错误信息 | 原因 | 处理方式 |
|---------|------|---------|
| `parent not found for node: X` | 父节点 `external_id` 不存在 | 确认父节点已在本次请求中包含，且排在子节点之前 |
| `node not found for member: Y` | 成员关联的节点不存在 | 先同步节点，再同步成员；或使用 `/api/v1/sync` 一次性提交 |
| `user_not_found: no match for external_user_id=Z` | 无法匹配 Mattermost 用户 | 预先注册用户映射，或确认 MM 中已有对应邮箱/用户名的账号 |

---

## 15. 最佳实践

### 同步频率建议

| 场景 | 建议策略 |
|------|---------|
| 人员变动不频繁（< 100 人/天）| 每天凌晨全量同步一次 |
| 人员变动较频繁 | 每 15 分钟增量同步 + 每天全量对齐一次 |
| 入职/离职实时感知 | 事件触发增量同步（HR 系统 Webhook → 调用增量同步接口）|

### 数据量分批建议

单次请求节点数量超过 1000 时，建议分批处理：

```python
def sync_in_batches(nodes, batch_size=500):
    # 先全量同步所有节点（保持树结构完整性）
    for i in range(0, len(nodes), batch_size):
        batch = nodes[i:i+batch_size]
        requests.post(
            f"{BASE_URL}/nodes",
            headers=HEADERS,
            json={"source": SOURCE, "sync_type": "incremental", "nodes": batch}
        )

    # 最后一批标记为 full 触发清理
    # 或使用 incremental + 独立的清理调用
```

### Token 安全

- Token 不要硬编码在代码中，通过环境变量或密钥管理系统注入
- 定期轮换 Token（例如每季度）
- 如怀疑 Token 泄露，立即在插件配置中更换

### 幂等性

同步接口是幂等的——对同一份数据多次调用，结果一致，不会产生重复数据（通过 `source + external_id` 唯一键保证）。网络超时时可安全重试。

### 监控建议

定期查询同步日志，监控以下指标：

```bash
# 检查最近 24 小时是否有失败的同步任务
curl "${BASE_URL}/logs?source=hr_system&per_page=10" \
  -H "Authorization: Bearer ${SYNC_TOKEN}" | \
  python3 -c "import sys,json; logs=json.load(sys.stdin); \
  [print(f'WARN: {l[\"status\"]} at {l[\"started_at\"]}') for l in logs if l['status'] in ('failed','partial_success')]"
```

---

*文档版本：v0.1.0 | 最后更新：2026-03*
