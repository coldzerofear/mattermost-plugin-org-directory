import React from 'react';
import {OrgTreeNode, OrgMember} from '../../types/org_node';
import TreeNode, {AdminAction} from './tree_node';

interface OrgTreeProps {
    nodes: OrgTreeNode[];
    expandedNodes: Record<string, boolean>;
    membersCache: Record<string, OrgMember[]>;
    loadingNodes: Record<string, boolean>;
    onToggleExpand: (nodeId: string) => void;
    onSelectNode: (nodeId: string) => void;
    onUserClick: (userId: string) => void;
    onAdminAction?: (action: AdminAction, nodeId: string, node: OrgTreeNode) => void;
    isAdmin: boolean;
}

const OrgTree: React.FC<OrgTreeProps> = ({
    nodes,
    expandedNodes,
    membersCache,
    loadingNodes,
    onToggleExpand,
    onSelectNode,
    onUserClick,
    onAdminAction,
    isAdmin,
}) => {
    if (!nodes || nodes.length === 0) {
        return (
            <div style={{padding: '20px', color: '#999', textAlign: 'center'}}>
                {'暂无组织结构数据'}
            </div>
        );
    }

    return (
        <div className='org-directory-tree'>
            {nodes.map((node) => (
                <TreeNode
                    key={node.id}
                    node={node}
                    level={0}
                    expanded={expandedNodes[node.id] || false}
                    members={membersCache[node.id] || []}
                    isLoading={loadingNodes[node.id] || false}
                    expandedNodes={expandedNodes}
                    membersCache={membersCache}
                    loadingNodes={loadingNodes}
                    onToggle={onToggleExpand}
                    onSelectNode={onSelectNode}
                    onUserClick={onUserClick}
                    onAdminAction={onAdminAction}
                    isAdmin={isAdmin}
                />
            ))}
        </div>
    );
};

export default OrgTree;
