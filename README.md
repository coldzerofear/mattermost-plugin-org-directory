# Mattermost 组织通讯录插件

基于树形组织结构的企业通讯录 Mattermost 插件，支持多层级部门管理、用户搜索、WebSocket 实时同步，以及与外部系统（HR/OA/LDAP）的双向数据同步。

---

## 功能特性

| 功能 | 描述 |
|------|------|
| 树形组织管理 | 任意层级组织节点的创建/编辑/删除/移动 |
| 用户挂靠 | 将 Mattermost 用户挂靠到指定组织节点，支持多节点归属 |
| 可折叠树形展示 | 右侧边栏懒加载树形组件，节点展开时按需请求数据 |
| 全文搜索 | 按姓名/用户名/邮箱/部门名称模糊搜索，300ms 防抖 |
| 实时同步 | 组织变更通过 WebSocket 推送到所有在线用户 |
| 权限分级 | 系统管理员 → 节点 Admin → 节点 Manager → Member |
| 外部系统同步 | 通过 REST API 从 HR/OA/LDAP 同步组织树和人员关系 |
| 审计日志 | 所有写操作记录操作人、时间和变更内容 |
| Slash 命令 | `/org search`、`/org info`、`/org tree` |

---

## 技术要求

| 组件 | 版本 |
|------|------|
| Mattermost Server | ≥ 9.0.0 |
| 数据库 | PostgreSQL（推荐）|
| Go | 1.21+ |
| Node.js | 18+ |

> **注意：** 本插件使用 `$1/$2` 风格的 SQL 参数占位符，仅支持 PostgreSQL。

---

## 快速开始

### 构建

```bash
# 安装前端依赖
cd webapp && npm install && cd ..

# 构建插件（生成 dist/com.example.org-directory-0.1.0.tar.gz）
make dist
```

### 安装

1. 登录 Mattermost 系统控制台 → **插件管理** → **上传插件**
2. 上传 `dist/com.example.org-directory-0.1.0.tar.gz`
3. 启用插件

### 配置

在系统控制台 → **插件** → **组织通讯录** 中配置以下参数：

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| 启用审计日志 | 记录所有组织变更操作 | `true` |
| 允许所有用户搜索 | `false` 时仅管理员可搜索 | `true` |
| 最大组织层级深度 | `0` 表示不限制 | `10` |
| 默认分页大小 | 每页成员数量 | `50` |
| **外部同步 API Token** | 外部系统调用同步 API 的鉴权 Token（建议 32 位以上随机字符串） | 空（禁用同步 API）|
| 同步用户匹配策略 | 见[用户匹配说明](#用户匹配策略) | `mapping_email_username` |
| 同步时保护本地数据 | 外部全量同步不影响手动创建的节点 | `true` |
| 全量同步节点删除策略 | `cascade_delete`（级联删除）或 `move_to_parent`（子树上移）| `cascade_delete` |

---

## 使用说明

### 查看组织通讯录

点击 Mattermost 界面右侧 App Bar 的 **🏢** 图标，打开组织通讯录侧边栏。

### 管理员操作

系统管理员在侧边栏底部可看到管理工具栏：

- **+ 新建根节点** — 创建顶层组织节点
- **↺ 同步日志** — 查看外部系统同步历史和用户映射

每个节点的 **⚙** 按钮提供：
- ✏️ 编辑节点（名称/描述）
- 👥 管理成员（添加/移除/修改角色）
- ➕ 新建子节点
- 🗑️ 删除节点（级联删除子节点和成员关系）

### Slash 命令

```
/org search <关键词>    # 搜索用户（姓名/用户名/邮箱/部门）
/org info @用户名        # 查看用户所属组织
/org tree               # 以文本格式输出组织树
```

---

## 权限模型

```
系统管理员 (System Admin)
  └─ 所有节点的完整管理权限

节点 Admin
  └─ 指定节点及其子树的管理权限（含成员管理）

节点 Manager
  └─ 指定节点的成员管理权限（不可删除节点）

Member
  └─ 查看权限（默认所有登录用户）
```

角色权限沿树向上继承：若用户是某节点的 Manager，则自动对该节点的所有子节点也拥有 Manager 权限。

---

## 外部系统同步

详见 [外部同步 API 对接指南](docs/sync-api-guide.md)。

---

## 数据库 Schema

插件首次激活时自动创建以下表：

- `org_directory_nodes` — 组织节点（邻接表 + 物化路径）
- `org_directory_members` — 用户-节点关联关系
- `org_directory_user_mappings` — 外部用户 ID → Mattermost 用户 ID 映射
- `org_directory_sync_logs` — 外部同步任务日志
- `org_directory_audit_log` — 操作审计日志

所有表使用软删除（`delete_at` 字段），数据在插件禁用/卸载后保留。

---

## 开发

```bash
# 运行 Go 单元测试
cd server
go test ./model/...                                           # model 层测试（11 个用例）
go test -run "TestIsRoleSufficient|TestTopologicalSortNodes" . # 权限+同步测试（19 个用例）

# TypeScript 类型检查
cd webapp && npx tsc --noEmit

# 开发构建（监听模式）
cd webapp && npm run build
```

---

## License

MIT
