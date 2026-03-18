import React, {useState, useEffect, useRef} from 'react';
import {OrgTreeNode, OrgMember} from '../../types/org_node';
import UserLeaf from './user_leaf';

export type AdminAction = 'edit' | 'members' | 'create-child' | 'move' | 'delete';

interface TreeNodeProps {
    node: OrgTreeNode;
    level: number;
    expanded: boolean;
    members: OrgMember[];
    isLoading: boolean;
    expandedNodes: Record<string, boolean>;
    membersCache: Record<string, OrgMember[]>;
    loadingNodes: Record<string, boolean>;
    onToggle: (nodeId: string) => void;
    onSelectNode: (nodeId: string) => void;
    onUserClick: (userId: string) => void;
    onAdminAction?: (action: AdminAction, nodeId: string, node: OrgTreeNode) => void;
    isAdmin: boolean;
}

const TreeNode: React.FC<TreeNodeProps> = ({
    node,
    level,
    expanded,
    members,
    isLoading,
    expandedNodes,
    membersCache,
    loadingNodes,
    onToggle,
    onSelectNode,
    onUserClick,
    onAdminAction,
    isAdmin,
}) => {
    const hasChildren = node.has_children || (node.children && node.children.length > 0);
    const indent = level * 20;
    const [showMenu, setShowMenu] = useState(false);
    const menuRef = useRef<HTMLDivElement>(null);

    // Close menu on outside click
    useEffect(() => {
        if (!showMenu) {
            return undefined;
        }
        const handleClick = (e: MouseEvent) => {
            if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
                setShowMenu(false);
            }
        };
        document.addEventListener('mousedown', handleClick);
        return () => document.removeEventListener('mousedown', handleClick);
    }, [showMenu]);

    const handleAdminAction = (action: AdminAction) => {
        setShowMenu(false);
        onAdminAction?.(action, node.id, node);
    };

    return (
        <div className='org-directory-tree-node'>
            {/* Node row */}
            <div
                className='org-directory-node-row'
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    padding: `6px 8px 6px ${indent + 8}px`,
                    cursor: 'pointer',
                    borderRadius: '4px',
                    userSelect: 'none',
                    position: 'relative',
                }}
                onClick={() => onToggle(node.id)}
                onContextMenu={(e) => {
                    e.preventDefault();
                    onSelectNode(node.id);
                }}
            >
                {/* Expand/collapse chevron or loading spinner */}
                <span
                    style={{
                        display: 'inline-block',
                        width: '16px',
                        fontSize: '10px',
                        color: '#aaa',
                        flexShrink: 0,
                    }}
                >
                    {isLoading ? (
                        <span className='org-directory-node-loading'>{'⟳'}</span>
                    ) : hasChildren ? (
                        <span
                            style={{
                                display: 'inline-block',
                                transition: 'transform 0.15s',
                                transform: expanded ? 'rotate(90deg)' : 'rotate(0deg)',
                            }}
                        >
                            {'▶'}
                        </span>
                    ) : null}
                </span>

                {/* Folder icon */}
                <span style={{marginRight: '6px', flexShrink: 0}}>
                    {expanded ? '📂' : '📁'}
                </span>

                {/* Node name */}
                <span style={{flex: 1, fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>
                    {node.name}
                </span>

                {/* Member count */}
                {node.member_count !== undefined && node.member_count > 0 && (
                    <span style={{color: '#999', fontSize: '12px', marginLeft: '4px', flexShrink: 0}}>
                        {'('}{node.member_count}{'人)'}
                    </span>
                )}

                {/* Admin actions */}
                {isAdmin && (
                    <span
                        className='org-directory-node-actions'
                        style={{position: 'relative', marginLeft: '4px', flexShrink: 0}}
                        ref={menuRef}
                    >
                        <span
                            style={{color: '#aaa', fontSize: '14px', cursor: 'pointer', padding: '2px 4px', borderRadius: '3px'}}
                            onClick={(e) => {
                                e.stopPropagation();
                                setShowMenu((v) => !v);
                            }}
                            title={'节点操作'}
                        >{'⚙'}</span>

                        {/* Dropdown menu */}
                        {showMenu && (
                            <div
                                style={{
                                    position: 'absolute',
                                    right: 0,
                                    top: '20px',
                                    background: 'var(--center-channel-bg,#fff)',
                                    border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.15)',
                                    borderRadius: '4px',
                                    boxShadow: '0 4px 12px rgba(0,0,0,0.12)',
                                    zIndex: 100,
                                    minWidth: '110px',
                                    overflow: 'hidden',
                                }}
                                onClick={(e) => e.stopPropagation()}
                            >
                                {[
                                    {action: 'edit' as AdminAction, label: '✏️ 编辑节点'},
                                    {action: 'members' as AdminAction, label: '👥 管理成员'},
                                    {action: 'create-child' as AdminAction, label: '➕ 新建子节点'},
                                    {action: 'move' as AdminAction, label: '📦 移动节点'},
                                    {action: 'delete' as AdminAction, label: '🗑️ 删除节点', danger: true},
                                ].map(({action, label, danger}) => (
                                    <div
                                        key={action}
                                        style={{
                                            padding: '7px 12px',
                                            fontSize: '12px',
                                            cursor: 'pointer',
                                            color: danger ? '#f74343' : 'var(--center-channel-text,#3d3c40)',
                                            borderTop: action === 'delete' ? '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.1)' : 'none',
                                        }}
                                        onMouseEnter={(e) => {
                                            (e.currentTarget as HTMLDivElement).style.background = 'rgba(var(--center-channel-text-rgb,63,67,80),0.08)';
                                        }}
                                        onMouseLeave={(e) => {
                                            (e.currentTarget as HTMLDivElement).style.background = '';
                                        }}
                                        onClick={() => handleAdminAction(action)}
                                    >{label}</div>
                                ))}
                            </div>
                        )}
                    </span>
                )}
            </div>

            {/* Children and members (shown when expanded) */}
            {expanded && (
                <div className='org-directory-tree-children'>
                    {/* Child nodes */}
                    {node.children && node.children.map((child) => (
                        <TreeNode
                            key={child.id}
                            node={child}
                            level={level + 1}
                            expanded={expandedNodes[child.id] || false}
                            members={membersCache[child.id] || []}
                            isLoading={loadingNodes[child.id] || false}
                            expandedNodes={expandedNodes}
                            membersCache={membersCache}
                            loadingNodes={loadingNodes}
                            onToggle={onToggle}
                            onSelectNode={onSelectNode}
                            onUserClick={onUserClick}
                            onAdminAction={onAdminAction}
                            isAdmin={isAdmin}
                        />
                    ))}

                    {/* Member leaves for this node */}
                    {members && members.map((member) => (
                        <UserLeaf
                            key={member.id}
                            member={member}
                            level={level + 1}
                            onUserClick={onUserClick}
                        />
                    ))}
                </div>
            )}
        </div>
    );
};

export default TreeNode;
