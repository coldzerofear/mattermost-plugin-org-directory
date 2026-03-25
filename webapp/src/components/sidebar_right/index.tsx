import React, {useEffect, useCallback, useState} from 'react';
import {useDispatch, useSelector} from 'react-redux';

import {
    fetchOrgTree,
    expandNode,
    searchOrg,
    selectUser,
    deleteOrgNode,
} from '../../store/actions';
import {
    getOrgTree,
    getExpandedNodes,
    getMembersCache,
    getLoadingNodes,
    getSearchResults,
    getSearchQuery,
    getIsLoading,
    getIsSearching,
    getViewMode,
    getTreeNeedsRefresh,
    getSelectedUserId,
    getUsersCache,
    getError,
} from '../../store/selectors';
import {SearchResult, OrgMember, OrgTreeNode} from '../../types/org_node';
import {OrgDirectoryState} from '../../store/reducer';
import {AdminAction} from './tree_node';

import OrgTree from './org_tree';
import SearchBar from './search_bar';
import Loading from '../common/loading';
import UserDetailPanel from '../user_detail';
import NodeEditor from '../admin/node_editor';
import MemberManager from '../admin/member_manager';
import MoveNodeDialog from '../admin/move_node_dialog';
import ConfirmDialog from '../admin/confirm_dialog';
import SyncPanel from '../admin/sync_panel';

interface SidebarRightProps {
    currentUserId: string;
    isAdmin: boolean;
    // When provided by index.tsx wrapper via store.subscribe(), all plugin state
    // is sourced from this object instead of useSelector (bypasses state-path issues).
    pluginStateOverride?: OrgDirectoryState | null;
}

type AdminModal =
    | {type: 'create'; parentId: string | null}
    | {type: 'edit'; nodeId: string; node: OrgTreeNode}
    | {type: 'delete'; nodeId: string; nodeName: string}
    | {type: 'members'; nodeId: string; nodeName: string}
    | {type: 'move'; nodeId: string; nodeName: string}
    | {type: 'sync'}
    | null;

const SidebarRight: React.FC<SidebarRightProps> = ({currentUserId, isAdmin, pluginStateOverride}) => {
    const dispatch = useDispatch();

    // Always call useSelector (rules of hooks); override with pluginStateOverride when provided.
    const _sTree = useSelector(getOrgTree);
    const _sExpandedNodes = useSelector(getExpandedNodes);
    const _sMembersCache = useSelector(getMembersCache);
    const _sLoadingNodes = useSelector(getLoadingNodes);
    const _sSearchResults = useSelector(getSearchResults);
    const _sSearchQuery = useSelector(getSearchQuery);
    const _sIsLoading = useSelector(getIsLoading);
    const _sIsSearching = useSelector(getIsSearching);
    const _sViewMode = useSelector(getViewMode);
    const _sTreeNeedsRefresh = useSelector(getTreeNeedsRefresh);
    const _sSelectedUserId = useSelector(getSelectedUserId);
    const _sUsersCache = useSelector(getUsersCache);
    const _sError = useSelector(getError);

    // Use pluginStateOverride fields only when non-undefined; otherwise fall back
    // to useSelector. This handles the case where pluginStateOverride is {} (the
    // initial empty object Mattermost stores before any action fires the reducer).
    const ps = pluginStateOverride;
    const tree = ps?.tree ?? _sTree;
    const expandedNodes = ps?.expandedNodes ?? _sExpandedNodes;
    const membersCache = ps?.members ?? _sMembersCache;
    const loadingNodes = ps?.loadingNodes ?? _sLoadingNodes;
    const searchResults = ps?.searchResults ?? _sSearchResults;
    const searchQuery = ps?.searchQuery ?? _sSearchQuery;
    const isLoading = ps?.isLoading ?? _sIsLoading;
    const isSearching = ps?.isSearching ?? _sIsSearching;
    const viewMode = ps?.viewMode ?? _sViewMode;
    const treeNeedsRefresh = ps?.treeNeedsRefresh ?? _sTreeNeedsRefresh;
    const selectedUserId = ps?.selectedUserId ?? _sSelectedUserId;
    const usersCache = ps?.usersCache ?? _sUsersCache;
    const error = ps?.error ?? _sError;

    const [adminModal, setAdminModal] = useState<AdminModal>(null);
    const [deleting, setDeleting] = useState(false);

    // Initial load
    useEffect(() => {
        dispatch(fetchOrgTree() as any);
    }, []);

    // Refresh when tree is stale (WebSocket event received)
    useEffect(() => {
        if (treeNeedsRefresh) {
            dispatch(fetchOrgTree() as any);
        }
    }, [treeNeedsRefresh]);

    const handleToggleExpand = useCallback((nodeId: string) => {
        dispatch(expandNode(nodeId) as any);
    }, [dispatch]);

    const handleSelectNode = useCallback((_nodeId: string) => {
        // context menu fallback (no-op; admin actions use onAdminAction)
    }, []);

    const handleUserClick = useCallback((userId: string) => {
        dispatch(selectUser(userId));
    }, [dispatch]);

    const handleSearch = useCallback((q: string) => {
        dispatch(searchOrg(q) as any);
    }, [dispatch]);

    const handleClearSearch = useCallback(() => {
        dispatch(searchOrg('') as any);
    }, [dispatch]);

    // Admin action handler — called from TreeNode dropdown
    const handleAdminAction = useCallback((action: AdminAction, nodeId: string, node: OrgTreeNode) => {
        if (action === 'edit') {
            setAdminModal({type: 'edit', nodeId, node});
        } else if (action === 'members') {
            setAdminModal({type: 'members', nodeId, nodeName: node.name});
        } else if (action === 'create-child') {
            setAdminModal({type: 'create', parentId: nodeId});
        } else if (action === 'move') {
            setAdminModal({type: 'move', nodeId, nodeName: node.name});
        } else if (action === 'delete') {
            setAdminModal({type: 'delete', nodeId, nodeName: node.name});
        }
    }, []);

    const handleConfirmDelete = useCallback(async () => {
        if (!adminModal || adminModal.type !== 'delete') {
            return;
        }
        setDeleting(true);
        try {
            await (dispatch as any)(deleteOrgNode(adminModal.nodeId));
        } finally {
            setDeleting(false);
            setAdminModal(null);
        }
    }, [adminModal, dispatch]);

    // Resolve selected user from cache
    const selectedMember: OrgMember | null = selectedUserId ? (usersCache[selectedUserId] || null) : null;

    const closeAdminModal = useCallback(() => setAdminModal(null), []);

    return (
        <div
            className='org-directory-sidebar'
            style={{
                display: 'flex',
                flexDirection: 'column',
                height: '100%',
                minHeight: 0,
                padding: '12px',
                boxSizing: 'border-box',
                position: 'relative',
            }}
        >
            {/* Confirm delete dialog (rendered in portal-like fixed overlay) */}
            {adminModal?.type === 'delete' && (
                <ConfirmDialog
                    title={'删除节点'}
                    message={`确认删除节点「${adminModal.nodeName}」？子节点和成员关系将一并删除，此操作不可撤销。`}
                    confirmText={deleting ? '删除中...' : '删除'}
                    dangerous={true}
                    onConfirm={handleConfirmDelete}
                    onCancel={closeAdminModal}
                />
            )}

            {/* User detail panel overlay */}
            {selectedUserId && selectedMember && (
                <UserDetailPanel member={selectedMember}/>
            )}

            {/* Node editor overlay (create or edit) */}
            {adminModal && (adminModal.type === 'create' || adminModal.type === 'edit') && (
                <NodeEditor
                    mode={adminModal.type}
                    parentId={adminModal.type === 'create' ? adminModal.parentId : null}
                    node={adminModal.type === 'edit' ? adminModal.node : undefined}
                    onClose={closeAdminModal}
                    onSaved={closeAdminModal}
                />
            )}

            {/* Member manager overlay */}
            {adminModal?.type === 'members' && (
                <MemberManager
                    nodeId={adminModal.nodeId}
                    nodeName={adminModal.nodeName}
                    onClose={closeAdminModal}
                />
            )}

            {/* Move node overlay */}
            {adminModal?.type === 'move' && (
                <MoveNodeDialog
                    nodeId={adminModal.nodeId}
                    nodeName={adminModal.nodeName}
                    onClose={closeAdminModal}
                />
            )}

            {/* Sync panel overlay */}
            {adminModal?.type === 'sync' && (
                <SyncPanel onClose={closeAdminModal}/>
            )}

            {/* Main content (hidden when any overlay is open) */}
            {!selectedUserId && !adminModal && (
                <>
                    {/* Header */}
                    <div
                        className='org-directory-header'
                        style={{
                            fontSize: '16px',
                            fontWeight: 600,
                            marginBottom: '12px',
                            paddingBottom: '8px',
                            borderBottom: '1px solid rgba(var(--center-channel-text-rgb, 63,67,80),0.12)',
                        }}
                    >
                        {'🏢 组织通讯录'}
                    </div>

                    {/* Error banner */}
                    {error && (
                        <div className='org-directory-error-banner' role='alert'>
                            {'⚠ '}{error}
                        </div>
                    )}


                    {/* Search bar */}
                    <div style={{marginBottom: '12px'}}>
                        <SearchBar
                            query={searchQuery}
                            onSearch={handleSearch}
                            onClear={handleClearSearch}
                        />
                    </div>

                    {/* Content area */}
                    <div
                        className='org-directory-content'
                        style={{
                            flex: 1,
                            minHeight: 0,
                            overflowY: 'auto',
                        }}
                    >
                        {isLoading || isSearching ? (
                            <Loading/>
                        ) : viewMode === 'search' ? (
                            <SearchResultList
                                results={searchResults}
                                query={searchQuery}
                                onUserClick={handleUserClick}
                            />
                        ) : (
                            <OrgTree
                                nodes={tree}
                                expandedNodes={expandedNodes}
                                membersCache={membersCache}
                                loadingNodes={loadingNodes}
                                onToggleExpand={handleToggleExpand}
                                onSelectNode={handleSelectNode}
                                onUserClick={handleUserClick}
                                onAdminAction={isAdmin ? handleAdminAction : undefined}
                                isAdmin={isAdmin}
                            />
                        )}
                    </div>

                    {/* Admin footer */}
                    {isAdmin && viewMode === 'tree' && (
                        <div
                            className='org-directory-admin-bar'
                            style={{
                                borderTop: '1px solid rgba(var(--center-channel-text-rgb, 63,67,80),0.12)',
                                paddingTop: '8px',
                                marginTop: '8px',
                                display: 'flex',
                                gap: '8px',
                                flexShrink: 0,
                            }}
                        >
                            <button
                                style={{
                                    fontSize: '12px',
                                    padding: '4px 10px',
                                    borderRadius: '4px',
                                    border: '1px solid rgba(var(--center-channel-text-rgb, 63,67,80),0.24)',
                                    cursor: 'pointer',
                                    background: 'transparent',
                                    color: 'var(--center-channel-text, #3d3c40)',
                                }}
                                onClick={() => setAdminModal({type: 'create', parentId: null})}
                            >
                                {'+ 新建根节点'}
                            </button>
                            <button
                                style={{
                                    fontSize: '12px',
                                    padding: '4px 10px',
                                    borderRadius: '4px',
                                    border: '1px solid rgba(var(--center-channel-text-rgb, 63,67,80),0.24)',
                                    cursor: 'pointer',
                                    background: 'transparent',
                                    color: 'var(--center-channel-text, #3d3c40)',
                                }}
                                onClick={() => setAdminModal({type: 'sync'})}
                            >
                                {'↺ 同步日志'}
                            </button>
                        </div>
                    )}
                </>
            )}
        </div>
    );
};

// --- inline search result list component ---
interface SearchResultListProps {
    results: SearchResult[];
    query: string;
    onUserClick: (userId: string) => void;
}

const SearchResultList: React.FC<SearchResultListProps> = ({results, query, onUserClick}) => {
    if (!query) {
        return null;
    }
    if (results.length === 0) {
        return (
            <div style={{padding: '20px', textAlign: 'center', color: '#999'}}>
                {`未找到与 "${query}" 相关的结果`}
            </div>
        );
    }

    return (
        <div className='org-directory-search-results'>
            <div style={{color: '#999', fontSize: '12px', marginBottom: '8px'}}>
                {`搜索结果 (${results.length})`}
            </div>
            {results.map((r, idx) => {
                const displayName = [r.user.first_name, r.user.last_name].filter(Boolean).join(' ') || r.user.username;
                const pathDisplay = r.node_path ? r.node_path.replace(/\//g, ' > ').replace(/^ > /, '') : r.node_name;
                return (
                    <div
                        key={`${r.user.user_id}-${idx}`}
                        className='org-directory-search-result-item'
                        style={{
                            padding: '8px',
                            borderRadius: '4px',
                            marginBottom: '4px',
                            cursor: 'pointer',
                            borderBottom: '1px solid rgba(var(--center-channel-text-rgb, 63,67,80),0.06)',
                        }}
                        onClick={() => onUserClick(r.user.user_id)}
                    >
                        <div style={{display: 'flex', alignItems: 'center', gap: '6px'}}>
                            <span>{'👤'}</span>
                            <span style={{fontWeight: 500}}>{displayName}</span>
                            <span style={{color: '#aaa', fontSize: '12px'}}>{'@'}{r.user.username}</span>
                        </div>
                        <div style={{fontSize: '12px', color: '#999', marginTop: '2px', paddingLeft: '22px'}}>
                            {pathDisplay}
                        </div>
                        {r.user.position && (
                            <div style={{fontSize: '12px', color: '#999', paddingLeft: '22px'}}>
                                {r.user.position}
                            </div>
                        )}
                    </div>
                );
            })}
        </div>
    );
};

export default SidebarRight;
