import {OrgDirectoryAPI} from '../api/client';
import {OrgTreeNode, OrgNode, SearchResult} from '../types/org_node';

// Action type constants
export const ORG_FETCH_TREE_REQUEST = 'ORG_FETCH_TREE_REQUEST';
export const ORG_FETCH_TREE_SUCCESS = 'ORG_FETCH_TREE_SUCCESS';
export const ORG_FETCH_TREE_FAILURE = 'ORG_FETCH_TREE_FAILURE';

export const ORG_LOAD_CHILDREN_REQUEST = 'ORG_LOAD_CHILDREN_REQUEST';
export const ORG_LOAD_CHILDREN_SUCCESS = 'ORG_LOAD_CHILDREN_SUCCESS';
export const ORG_LOAD_CHILDREN_FAILURE = 'ORG_LOAD_CHILDREN_FAILURE';

export const ORG_TOGGLE_EXPAND = 'ORG_TOGGLE_EXPAND';
export const ORG_SELECT_NODE = 'ORG_SELECT_NODE';

export const ORG_SEARCH_REQUEST = 'ORG_SEARCH_REQUEST';
export const ORG_SEARCH_SUCCESS = 'ORG_SEARCH_SUCCESS';
export const ORG_SEARCH_FAILURE = 'ORG_SEARCH_FAILURE';
export const ORG_CLEAR_SEARCH = 'ORG_CLEAR_SEARCH';

export const ORG_TREE_UPDATE = 'ORG_TREE_UPDATE';
export const ORG_MEMBER_UPDATE = 'ORG_MEMBER_UPDATE';

export const ORG_SELECT_USER = 'ORG_SELECT_USER';
export const ORG_FETCH_USER_NODES_REQUEST = 'ORG_FETCH_USER_NODES_REQUEST';
export const ORG_FETCH_USER_NODES_SUCCESS = 'ORG_FETCH_USER_NODES_SUCCESS';
export const ORG_FETCH_USER_NODES_FAILURE = 'ORG_FETCH_USER_NODES_FAILURE';
export const ORG_INVALIDATE_USER_NODES = 'ORG_INVALIDATE_USER_NODES';

// Thunk: fetch the top-level roots on initial load
export const fetchOrgTree = () => async (dispatch: any) => {
    dispatch({type: ORG_FETCH_TREE_REQUEST});
    try {
        const tree: OrgTreeNode[] = await OrgDirectoryAPI.getFullTree(1);
        dispatch({type: ORG_FETCH_TREE_SUCCESS, tree});
    } catch (err: any) {
        dispatch({type: ORG_FETCH_TREE_FAILURE, error: err.message});
    }
};

// Thunk: expand or collapse a node while keeping async loading and toggle state consistent
export const expandNode = (nodeId: string) => async (dispatch: any, getState: any) => {
    const state = getState();
    const pluginState = state['plugins-com.example.org-directory'];
    const alreadyLoaded = pluginState?.loadedNodes?.[nodeId];
    const isLoading = pluginState?.loadingNodes?.[nodeId];
    const isExpanded = pluginState?.expandedNodes?.[nodeId];

    if (isExpanded) {
        dispatch({type: ORG_TOGGLE_EXPAND, nodeId, expanded: false});
        return;
    }

    if (!alreadyLoaded && !isLoading) {
        dispatch({type: ORG_LOAD_CHILDREN_REQUEST, nodeId});
        try {
            const [subTree, members] = await Promise.all([
                OrgDirectoryAPI.getSubTree(nodeId, 1),
                OrgDirectoryAPI.getMembers(nodeId),
            ]);
            dispatch({type: ORG_LOAD_CHILDREN_SUCCESS, nodeId, children: subTree.children || [], members});
        } catch (err: any) {
            dispatch({type: ORG_LOAD_CHILDREN_FAILURE, nodeId, error: err.message});
            return;
        }
    } else if (isLoading) {
        return;
    }

    const nextState = getState();
    const nextPluginState = nextState['plugins-com.example.org-directory'];
    if (!nextPluginState?.expandedNodes?.[nodeId]) {
        dispatch({type: ORG_TOGGLE_EXPAND, nodeId, expanded: true});
    }
};

// Thunk: search users/nodes
export const searchOrg = (query: string) => async (dispatch: any) => {
    if (!query.trim()) {
        dispatch({type: ORG_CLEAR_SEARCH});
        return;
    }
    dispatch({type: ORG_SEARCH_REQUEST, query});
    try {
        const results: SearchResult[] = await OrgDirectoryAPI.searchUsers(query);
        dispatch({type: ORG_SEARCH_SUCCESS, results, query});
    } catch (err: any) {
        dispatch({type: ORG_SEARCH_FAILURE, error: err.message});
    }
};

// Action: select/deselect a user for detail view
export const selectUser = (userId: string | null) => ({
    type: ORG_SELECT_USER,
    userId,
});

// Thunk: fetch all org nodes a user belongs to
export const fetchUserNodes = (userId: string, force = false) => async (dispatch: any, getState: any) => {
    const pluginState = getState()['plugins-com.example.org-directory'];
    if (!force && pluginState?.userNodes?.[userId]) {
        return;
    }

    dispatch({type: ORG_FETCH_USER_NODES_REQUEST, userId});
    try {
        const nodes: OrgNode[] = await OrgDirectoryAPI.getUserNodes(userId);
        dispatch({type: ORG_FETCH_USER_NODES_SUCCESS, userId, nodes});
    } catch (err: any) {
        dispatch({type: ORG_FETCH_USER_NODES_FAILURE, userId, error: err.message});
    }
};

export const invalidateUserNodes = (userIds: string[]) => ({
    type: ORG_INVALIDATE_USER_NODES,
    userIds,
});

// WebSocket event handlers
export const handleTreeUpdate = (data: any) => ({
    type: ORG_TREE_UPDATE,
    data,
});

export const handleMemberUpdate = (data: any) => ({
    type: ORG_MEMBER_UPDATE,
    data,
});

// Admin action types
export const ORG_RELOAD_MEMBERS_SUCCESS = 'ORG_RELOAD_MEMBERS_SUCCESS';

// Reload members for a node and update cache
export const reloadNodeMembers = (nodeId: string) => async (dispatch: any) => {
    try {
        const members = await OrgDirectoryAPI.getMembers(nodeId);
        dispatch({type: ORG_RELOAD_MEMBERS_SUCCESS, nodeId, members});
    } catch (_) { /* silent */ }
};

// Admin Node CRUD
export const createOrgNode = (data: {name: string; parent_id: string; description?: string}) =>
    async (dispatch: any): Promise<void> => {
        await OrgDirectoryAPI.createNode(data);
        await dispatch(fetchOrgTree() as any);
    };

export const updateOrgNode = (nodeId: string, data: {name?: string; description?: string}) =>
    async (dispatch: any): Promise<void> => {
        await OrgDirectoryAPI.updateNode(nodeId, data);
        await dispatch(fetchOrgTree() as any);
    };

export const deleteOrgNode = (nodeId: string) =>
    async (dispatch: any): Promise<void> => {
        await OrgDirectoryAPI.deleteNode(nodeId);
        await dispatch(fetchOrgTree() as any);
    };

// Admin Member management
export const addOrgMember = (nodeId: string, data: {user_id: string; role?: string; position?: string}) =>
    async (dispatch: any): Promise<void> => {
        await OrgDirectoryAPI.addMember(nodeId, data);
        dispatch(invalidateUserNodes([data.user_id]));
        dispatch(reloadNodeMembers(nodeId) as any);
    };

export const removeOrgMember = (nodeId: string, userId: string) =>
    async (dispatch: any): Promise<void> => {
        await OrgDirectoryAPI.removeMember(nodeId, userId);
        dispatch(invalidateUserNodes([userId]));
        dispatch(reloadNodeMembers(nodeId) as any);
    };

export const updateOrgMemberRole = (nodeId: string, userId: string, role: string) =>
    async (dispatch: any): Promise<void> => {
        await OrgDirectoryAPI.updateMemberRole(nodeId, userId, role);
        dispatch(invalidateUserNodes([userId]));
        dispatch(reloadNodeMembers(nodeId) as any);
    };

export const moveOrgNode = (nodeId: string, newParentId: string) =>
    async (dispatch: any): Promise<void> => {
        await OrgDirectoryAPI.moveNode(nodeId, newParentId);
        await dispatch(fetchOrgTree() as any);
    };

export const reorderOrgMembers = (nodeId: string, userIds: string[]) =>
    async (dispatch: any): Promise<void> => {
        await OrgDirectoryAPI.reorderMembers(nodeId, userIds);
        dispatch(reloadNodeMembers(nodeId) as any);
    };
