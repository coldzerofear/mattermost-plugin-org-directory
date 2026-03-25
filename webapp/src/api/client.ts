import manifest from '../manifest';
import {OrgNode, OrgMember, OrgTreeNode, SearchResult} from '../types/org_node';
import {SyncLog, UserMapping} from '../types/sync';

const API_BASE = `/plugins/${manifest.id}/api/v1`;

// Read the Mattermost CSRF token from the MMCSRF cookie (not httpOnly).
// Combined with the MMAUTHTOKEN session cookie (httpOnly, sent automatically by
// the browser on same-origin requests), this satisfies Mattermost's session
// auth + CSRF protection for plugin HTTP requests.
function getCSRFToken(): string {
    const match = document.cookie.match(/(?:^|;\s*)MMCSRF=([^;]*)/);
    return match ? decodeURIComponent(match[1]) : '';
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const res = await fetch(`${API_BASE}${path}`, {
        method,
        credentials: 'same-origin',
        headers: {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
            'X-CSRF-Token': getCSRFToken(),
        },
        body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) {
        const err = await res.json().catch(() => ({error: res.statusText}));
        throw new Error(err.error || res.statusText);
    }
    return res.json() as Promise<T>;
}

function buildQuery(params: Record<string, string | number | boolean | undefined>): string {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
        if (value === undefined || value === null || value === '') {
            return;
        }
        searchParams.set(key, String(value));
    });

    const query = searchParams.toString();
    return query ? `?${query}` : '';
}

export const OrgDirectoryAPI = {
    // Tree
    getFullTree: (depth?: number): Promise<OrgTreeNode[]> =>
        request('GET', `/tree${depth !== undefined ? `?depth=${depth}` : ''}`),

    getSubTree: (nodeId: string, depth?: number): Promise<OrgTreeNode> =>
        request('GET', `/tree/${nodeId}${depth !== undefined ? `?depth=${depth}` : ''}`),

    getRoots: (): Promise<OrgNode[]> =>
        request('GET', '/roots'),

    // Nodes
    createNode: (data: {name: string; parent_id: string; description?: string; icon?: string}): Promise<OrgNode> =>
        request('POST', '/nodes', data),

    getNode: (id: string): Promise<OrgNode> =>
        request('GET', `/nodes/${id}`),

    updateNode: (id: string, data: {name?: string; description?: string; icon?: string}): Promise<OrgNode> =>
        request('PUT', `/nodes/${id}`, data),

    deleteNode: (id: string, cascadeStrategy?: string): Promise<void> =>
        request('DELETE', `/nodes/${id}${cascadeStrategy ? `?cascade_strategy=${cascadeStrategy}` : ''}`),

    getChildren: (id: string): Promise<OrgNode[]> =>
        request('GET', `/nodes/${id}/children`),

    moveNode: (id: string, newParentId: string): Promise<OrgNode> =>
        request('POST', `/nodes/${id}/move`, {new_parent_id: newParentId}),

    reorderNodes: (parentId: string, nodeIds: string[]): Promise<void> =>
        request('POST', `/nodes/${parentId}/reorder`, {node_ids: nodeIds}),

    reorderMembers: (nodeId: string, userIds: string[]): Promise<void> =>
        request('POST', `/nodes/${nodeId}/members/reorder`, {user_ids: userIds}),

    getNodeStats: (id: string, recursive = false): Promise<{member_count: number}> =>
        request('GET', `/nodes/${id}/stats?recursive=${recursive}`),

    // Members
    getMembers: (nodeId: string, page = 0, perPage?: number): Promise<OrgMember[]> =>
        request('GET', `/nodes/${nodeId}/members${buildQuery({page, per_page: perPage})}`),

    addMember: (nodeId: string, data: {user_id: string; role?: string; position?: string}): Promise<OrgMember> =>
        request('POST', `/nodes/${nodeId}/members`, data),

    removeMember: (nodeId: string, userId: string): Promise<void> =>
        request('DELETE', `/nodes/${nodeId}/members/${userId}`),

    updateMemberRole: (nodeId: string, userId: string, role: string): Promise<void> =>
        request('PUT', `/nodes/${nodeId}/members/${userId}/role`, {role}),

    updateMemberPosition: (nodeId: string, userId: string, position: string): Promise<void> =>
        request('PUT', `/nodes/${nodeId}/members/${userId}/position`, {position}),

    // Search
    searchUsers: (query: string, page = 0, perPage?: number): Promise<SearchResult[]> =>
        request('GET', `/search/users${buildQuery({q: query, page, per_page: perPage})}`),

    searchNodes: (query: string): Promise<OrgNode[]> =>
        request('GET', `/search/nodes?q=${encodeURIComponent(query)}`),

    // User
    getUserNodes: (userId: string): Promise<OrgNode[]> =>
        request('GET', `/users/${userId}/nodes`),

    // Sync logs (admin — uses session auth, not sync token)
    getSyncLogs: (source = '', page = 0, perPage?: number): Promise<SyncLog[]> =>
        request('GET', `/admin/sync/logs${buildQuery({source, page, per_page: perPage})}`),

    getSyncLog: (id: string): Promise<SyncLog> =>
        request('GET', `/admin/sync/logs/${id}`),

    // User mappings (admin — uses session auth, not sync token)
    getUserMappings: (source: string, page = 0, perPage?: number): Promise<UserMapping[]> =>
        request('GET', `/admin/sync/user-mappings/${encodeURIComponent(source)}${buildQuery({page, per_page: perPage})}`),
};
