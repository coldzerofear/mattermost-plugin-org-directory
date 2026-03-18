import {OrgDirectoryState} from './reducer';

const PLUGIN_ID = 'com.example.org-directory';

function getPluginState(state: any): OrgDirectoryState {
    // Mattermost registerReducer stores state at state['plugins-<pluginId>']
    // (NOT state.plugins.plugins[id] — that's the built-in component registry)
    return state['plugins-' + PLUGIN_ID] || {};
}

export const getOrgTree = (state: any) => getPluginState(state).tree || [];
export const getExpandedNodes = (state: any) => getPluginState(state).expandedNodes || {};
export const getLoadedNodes = (state: any) => getPluginState(state).loadedNodes || {};
export const getLoadingNodes = (state: any) => getPluginState(state).loadingNodes || {};
export const getSelectedNodeId = (state: any) => getPluginState(state).selectedNodeId;
export const getMembersCache = (state: any) => getPluginState(state).members || {};
export const getUsersCache = (state: any) => getPluginState(state).usersCache || {};
export const getSelectedUserId = (state: any) => getPluginState(state).selectedUserId || null;
export const getUserNodesForUser = (state: any, userId: string) => getPluginState(state).userNodes?.[userId];
export const getSearchResults = (state: any) => getPluginState(state).searchResults || [];
export const getSearchQuery = (state: any) => getPluginState(state).searchQuery || '';
export const getIsLoading = (state: any) => getPluginState(state).isLoading || false;
export const getIsSearching = (state: any) => getPluginState(state).isSearching || false;
export const getError = (state: any) => getPluginState(state).error;
export const getViewMode = (state: any) => getPluginState(state).viewMode || 'tree';
export const getTreeNeedsRefresh = (state: any) => getPluginState(state).treeNeedsRefresh || false;
export const getNodeMembers = (state: any, nodeId: string) => getPluginState(state).members?.[nodeId];
