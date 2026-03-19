import {OrgTreeNode, OrgMember, OrgNode, SearchResult} from '../types/org_node';
import {
    ORG_FETCH_TREE_REQUEST,
    ORG_FETCH_TREE_SUCCESS,
    ORG_FETCH_TREE_FAILURE,
    ORG_LOAD_CHILDREN_REQUEST,
    ORG_LOAD_CHILDREN_SUCCESS,
    ORG_LOAD_CHILDREN_FAILURE,
    ORG_TOGGLE_EXPAND,
    ORG_SELECT_NODE,
    ORG_SEARCH_REQUEST,
    ORG_SEARCH_SUCCESS,
    ORG_SEARCH_FAILURE,
    ORG_CLEAR_SEARCH,
    ORG_TREE_UPDATE,
    ORG_MEMBER_UPDATE,
    ORG_SELECT_USER,
    ORG_FETCH_USER_NODES_REQUEST,
    ORG_FETCH_USER_NODES_SUCCESS,
    ORG_FETCH_USER_NODES_FAILURE,
    ORG_INVALIDATE_USER_NODES,
    ORG_RELOAD_MEMBERS_SUCCESS,
} from './actions';


export interface OrgDirectoryState {
    tree: OrgTreeNode[];
    expandedNodes: Record<string, boolean>;
    loadedNodes: Record<string, boolean>;
    loadingNodes: Record<string, boolean>;
    selectedNodeId: string | null;
    members: Record<string, OrgMember[]>;
    usersCache: Record<string, OrgMember>;
    selectedUserId: string | null;
    userNodes: Record<string, OrgNode[]>;
    searchResults: SearchResult[];
    searchQuery: string;
    isLoading: boolean;
    isSearching: boolean;
    error: string | null;
    viewMode: 'tree' | 'search';
    treeNeedsRefresh: boolean;
}

const initialState: OrgDirectoryState = {
    tree: [],
    expandedNodes: {},
    loadedNodes: {},
    loadingNodes: {},
    selectedNodeId: null,
    members: {},
    usersCache: {},
    selectedUserId: null,
    userNodes: {},
    searchResults: [],
    searchQuery: '',
    isLoading: false,
    isSearching: false,
    error: null,
    viewMode: 'tree',
    treeNeedsRefresh: false,
};

export default function reducer(state = initialState, action: any): OrgDirectoryState {
    switch (action.type) {
    case ORG_FETCH_TREE_REQUEST:
        return {...state, isLoading: true, error: null};

    case ORG_FETCH_TREE_SUCCESS:
        return {
            ...state,
            isLoading: false,
            tree: action.tree,
            loadedNodes: {},   // clear stale cache so re-expanded nodes refetch children
            treeNeedsRefresh: false,
        };

    case ORG_FETCH_TREE_FAILURE:
        return {...state, isLoading: false, error: action.error};

    case ORG_LOAD_CHILDREN_REQUEST:
        return {
            ...state,
            loadingNodes: {...state.loadingNodes, [action.nodeId]: true},
        };

    case ORG_LOAD_CHILDREN_SUCCESS: {
        const newUsersCache = buildUsersCache(state.usersCache, action.members);
        return {
            ...state,
            loadingNodes: {...state.loadingNodes, [action.nodeId]: false},
            loadedNodes: {...state.loadedNodes, [action.nodeId]: true},
            members: {...state.members, [action.nodeId]: action.members || []},
            usersCache: newUsersCache,
            tree: updateTreeChildren(state.tree, action.nodeId, action.children),
        };
    }

    case ORG_LOAD_CHILDREN_FAILURE:
        return {
            ...state,
            loadingNodes: {...state.loadingNodes, [action.nodeId]: false},
        };

    case ORG_TOGGLE_EXPAND:
        return {
            ...state,
            expandedNodes: {
                ...state.expandedNodes,
                [action.nodeId]: action.expanded ?? !state.expandedNodes[action.nodeId],
            },
        };


    case ORG_SELECT_NODE:
        return {...state, selectedNodeId: action.nodeId};

    case ORG_SEARCH_REQUEST:
        return {...state, isSearching: true, searchQuery: action.query, viewMode: 'search', error: null};

    case ORG_SEARCH_SUCCESS: {
        const newUsersCache = buildUsersCacheFromSearch(state.usersCache, action.results);
        return {
            ...state,
            isSearching: false,
            searchResults: action.results,
            searchQuery: action.query,
            viewMode: 'search',
            usersCache: newUsersCache,
        };
    }

    case ORG_SEARCH_FAILURE:
        return {...state, isSearching: false, error: action.error};

    case ORG_CLEAR_SEARCH:
        return {...state, searchResults: [], searchQuery: '', viewMode: 'tree'};

    case ORG_TREE_UPDATE:
        return {...state, treeNeedsRefresh: true};

    case ORG_MEMBER_UPDATE:
        if (action.data?.node_id) {
            const updated = {...state.members};
            delete updated[action.data.node_id];
            const loaded = {...state.loadedNodes};
            delete loaded[action.data.node_id];
            return {...state, members: updated, loadedNodes: loaded};
        }
        return state;

    case ORG_SELECT_USER:
        return {...state, selectedUserId: action.userId};

    case ORG_FETCH_USER_NODES_REQUEST:
        return state;

    case ORG_FETCH_USER_NODES_SUCCESS:
        return {
            ...state,
            userNodes: {...state.userNodes, [action.userId]: action.nodes},
        };

    case ORG_FETCH_USER_NODES_FAILURE:
        return state;

    case ORG_INVALIDATE_USER_NODES: {
        const nextUserNodes = {...state.userNodes};
        for (const userId of action.userIds || []) {
            delete nextUserNodes[userId];
        }
        return {
            ...state,
            userNodes: nextUserNodes,
        };
    }

    case ORG_RELOAD_MEMBERS_SUCCESS: {

        const newUsersCache = buildUsersCache(state.usersCache, action.members);
        return {
            ...state,
            members: {...state.members, [action.nodeId]: action.members || []},
            usersCache: newUsersCache,
        };
    }

    default:
        return state;
    }
}

function updateTreeChildren(tree: OrgTreeNode[], nodeId: string, children: OrgTreeNode[]): OrgTreeNode[] {
    return tree.map((node) => {
        if (node.id === nodeId) {
            return {...node, children: children || [], has_children: (children?.length ?? 0) > 0};
        }
        if (node.children?.length) {
            return {...node, children: updateTreeChildren(node.children, nodeId, children)};
        }
        return node;
    });
}

function buildUsersCache(existing: Record<string, OrgMember>, members?: OrgMember[]): Record<string, OrgMember> {
    if (!members || members.length === 0) {
        return existing;
    }
    const updated = {...existing};
    for (const m of members) {
        if (m.user_id) {
            updated[m.user_id] = m;
        }
    }
    return updated;
}

function buildUsersCacheFromSearch(existing: Record<string, OrgMember>, results?: SearchResult[]): Record<string, OrgMember> {
    if (!results || results.length === 0) {
        return existing;
    }
    const updated = {...existing};
    for (const r of results) {
        if (r.user?.user_id) {
            updated[r.user.user_id] = r.user;
        }
    }
    return updated;
}
