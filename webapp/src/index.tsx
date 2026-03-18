import React from 'react';

import manifest from './manifest';
import SidebarRight from './components/sidebar_right';
import reducer, {OrgDirectoryState} from './store/reducer';
import {handleTreeUpdate, handleMemberUpdate} from './store/actions';
import './styles/org_directory.css';

const {id: pluginId} = manifest;

/**
 * Extract plugin reducer state from the Mattermost Redux store.
 * registry.registerReducer() internally registers at key 'plugins-<pluginId>'.
 */
function extractPluginState(storeState: any): OrgDirectoryState | null {
    return storeState?.['plugins-' + pluginId] ?? null;
}

export default class OrgDirectoryPlugin {
    initialize(registry: any, store: any) {
        // Register Redux reducer
        registry.registerReducer(reducer);

        // Wrapper component: uses store.subscribe() so we never depend on
        // useSelector finding the correct state path.
        const SidebarWrapper = (props: any) => {
            const [pluginState, setPluginState] = React.useState<OrgDirectoryState | null>(
                () => extractPluginState(store.getState()),
            );
            const [currentUser, setCurrentUser] = React.useState<string>(() => {
                return store.getState()?.entities?.users?.currentUserId || '';
            });
            const [isAdmin, setIsAdmin] = React.useState<boolean>(() => {
                const s = store.getState();
                const uid = s?.entities?.users?.currentUserId || '';
                return !!(s?.entities?.users?.profiles?.[uid]?.roles?.includes('system_admin'));
            });

            React.useEffect(() => {
                const unsubscribe = store.subscribe(() => {
                    const newState = store.getState();
                    setPluginState(extractPluginState(newState));

                    const uid = newState?.entities?.users?.currentUserId || '';
                    setCurrentUser(uid);
                    setIsAdmin(!!(newState?.entities?.users?.profiles?.[uid]?.roles?.includes('system_admin')));
                });

                return unsubscribe;
            }, []);

            return (
                <SidebarRight
                    {...props}
                    currentUserId={currentUser}
                    isAdmin={isAdmin}
                    pluginStateOverride={pluginState}
                />
            );
        };

        // Register right-hand sidebar component
        const {toggleRHSPlugin} = registry.registerRightHandSidebarComponent(
            SidebarWrapper,
            '组织通讯录',
        );

        // Register App Bar button.
        const iconURL = `/plugins/${pluginId}/icon`;
        registry.registerAppBarComponent(
            iconURL,
            () => store.dispatch(toggleRHSPlugin),
            '组织通讯录',
        );

        // Register WebSocket event handlers (plain action objects — no thunk needed)
        registry.registerWebSocketEventHandler(
            `custom_${pluginId}_tree_update`,
            (event: any) => store.dispatch(handleTreeUpdate(event.data)),
        );
        registry.registerWebSocketEventHandler(
            `custom_${pluginId}_member_update`,
            (event: any) => store.dispatch(handleMemberUpdate(event.data)),
        );

        // On reconnect: mark tree stale so SidebarRight's useEffect refetches
        registry.registerReconnectHandler(() => {
            store.dispatch(handleTreeUpdate({}));
        });
    }

    uninitialize() {
        // Cleanup if needed
    }
}

(window as any).registerPlugin(pluginId, new OrgDirectoryPlugin());
