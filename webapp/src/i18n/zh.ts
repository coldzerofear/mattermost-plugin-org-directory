/**
 * zh.ts — Simplified Chinese string catalog for mattermost-plugin-org-directory.
 *
 * Usage (future i18n integration):
 *   import {t} from '../i18n';
 *   <span>{t('sidebar.title')}</span>
 *
 * All UI strings are collected here as a single source of truth.
 * To add a new language, create e.g. en.ts with the same keys in English.
 */

const zh = {
    // ── Sidebar header ─────────────────────────────────────────────────────
    'sidebar.title': '🏢 组织通讯录',

    // ── Search bar ─────────────────────────────────────────────────────────
    'search.placeholder': '搜索姓名、用户名、邮箱或部门…',
    'search.clear': '清除',
    'search.results.count': '搜索结果 ({count})',
    'search.results.empty': '未找到与 "{query}" 相关的结果',

    // ── Tree nodes ─────────────────────────────────────────────────────────
    'node.member_count': '({count}人)',
    'node.loading': '加载中…',
    'node.actions.label': '节点操作',
    'node.actions.edit': '✏️ 编辑节点',
    'node.actions.members': '👥 管理成员',
    'node.actions.create_child': '➕ 新建子节点',
    'node.actions.move': '📦 移动节点',
    'node.actions.delete': '🗑️ 删除节点',

    // ── Admin footer ───────────────────────────────────────────────────────
    'admin.create_root': '+ 新建根节点',
    'admin.sync_logs': '↺ 同步日志',

    // ── Node editor ────────────────────────────────────────────────────────
    'node_editor.title.create': '新建节点',
    'node_editor.title.edit': '编辑节点',
    'node_editor.label.name': '节点名称',
    'node_editor.label.description': '描述（可选）',
    'node_editor.placeholder.name': '如：研发部门',
    'node_editor.placeholder.description': '简短描述…',
    'node_editor.button.save': '保存',
    'node_editor.button.saving': '保存中…',
    'node_editor.button.cancel': '取消',
    'node_editor.error.name_required': '节点名称不能为空',

    // ── Confirm dialog ─────────────────────────────────────────────────────
    'confirm.delete_node.title': '删除节点',
    'confirm.delete_node.message': '确认删除节点「{name}」？子节点和成员关系将一并删除，此操作不可撤销。',
    'confirm.button.delete': '删除',
    'confirm.button.deleting': '删除中…',
    'confirm.button.cancel': '取消',

    // ── Move node dialog ───────────────────────────────────────────────────
    'move_node.title': '移动节点',
    'move_node.prompt': '选择「{name}」的新父节点：',
    'move_node.selected': '已选: {path}',
    'move_node.no_targets': '没有可用的目标节点',
    'move_node.button.confirm': '确认移动',
    'move_node.button.moving': '移动中…',
    'move_node.button.cancel': '取消',
    'move_node.error.no_selection': '请选择目标父节点',

    // ── Member manager ─────────────────────────────────────────────────────
    'member_manager.title': '管理成员',
    'member_manager.current_members': '当前成员 ({count})',
    'member_manager.no_members': '暂无成员',
    'member_manager.move_up': '上移',
    'member_manager.move_down': '下移',
    'member_manager.remove': '移除成员',
    'member_manager.add_section': '添加成员',
    'member_manager.search_placeholder': '输入用户名查找',
    'member_manager.search_button': '查找',
    'member_manager.searching': '…',
    'member_manager.add_button': '确认添加',
    'member_manager.adding': '添加中…',
    'member_manager.position_placeholder': '职位（可选）',
    'member_manager.user_not_found': '未找到该用户',
    'member_manager.search_error': '搜索失败，请检查网络',

    // ── Role labels ────────────────────────────────────────────────────────
    'role.member': 'member',
    'role.manager': 'manager',
    'role.admin': 'admin',

    // ── User detail panel ──────────────────────────────────────────────────
    'user_detail.back': '← 返回',
    'user_detail.org_paths': '所属组织',
    'user_detail.no_org': '未加入任何组织节点',
    'user_detail.send_message': '💬 发送消息',
    'user_detail.position': '职位',
    'user_detail.email': '邮箱',

    // ── Sync panel ─────────────────────────────────────────────────────────
    'sync_panel.title': '同步管理',
    'sync_panel.tab.logs': '同步日志',
    'sync_panel.tab.mappings': '用户映射',
    'sync_panel.filter.source': '按来源筛选…',
    'sync_panel.logs.empty': '暂无同步日志',
    'sync_panel.logs.source': '来源',
    'sync_panel.logs.type': '类型',
    'sync_panel.logs.status': '状态',
    'sync_panel.logs.started': '开始时间',
    'sync_panel.logs.nodes': '节点',
    'sync_panel.logs.members': '成员',
    'sync_panel.logs.detail.created': '新建',
    'sync_panel.logs.detail.updated': '更新',
    'sync_panel.logs.detail.deleted': '删除',
    'sync_panel.logs.detail.skipped': '跳过',
    'sync_panel.logs.detail.errors': '错误',
    'sync_panel.mappings.empty': '暂无用户映射记录',
    'sync_panel.mappings.external_id': '外部 ID',
    'sync_panel.mappings.mm_user': 'MM 用户',
    'sync_panel.pagination.prev': '上一页',
    'sync_panel.pagination.next': '下一页',
    'sync_panel.status.success': '成功',
    'sync_panel.status.partial_success': '部分成功',
    'sync_panel.status.failed': '失败',
    'sync_panel.status.running': '运行中',

    // ── Error states ───────────────────────────────────────────────────────
    'error.load_tree': '加载组织树失败',
    'error.load_children': '加载子节点失败',
    'error.search': '搜索失败',
    'error.generic': '操作失败，请重试',

    // ── Slash commands ─────────────────────────────────────────────────────
    'slash.search.hint': '[关键词]',
    'slash.search.desc': '搜索用户（姓名/用户名/邮箱/部门）',
    'slash.info.hint': '[@用户名]',
    'slash.info.desc': '查看用户所属组织',
    'slash.tree.hint': '',
    'slash.tree.desc': '以文本格式输出组织树',
} as const;

export type I18nKey = keyof typeof zh;
export default zh;

/**
 * t() — minimal translation helper using the zh catalog.
 *
 * Supports simple named placeholders: t('key', {name: 'foo'}) replaces {name}.
 *
 * @example
 *   t('node.member_count', {count: 5})  // → "(5人)"
 */
export function t(key: I18nKey, params?: Record<string, string | number>): string {
    let str: string = zh[key];
    if (params) {
        for (const [k, v] of Object.entries(params)) {
            str = str.replace(new RegExp(`\\{${k}\\}`, 'g'), String(v));
        }
    }
    return str;
}
