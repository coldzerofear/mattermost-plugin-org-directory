import React, {useState, useMemo, useEffect} from 'react';
import {useDispatch} from 'react-redux';
import {OrgTreeNode} from '../../types/org_node';
import {OrgDirectoryAPI} from '../../api/client';
import {moveOrgNode} from '../../store/actions';

interface MoveNodeDialogProps {
    nodeId: string;
    nodeName: string;
    onClose: () => void;
}

interface FlatNode {
    id: string;
    name: string;
    depth: number;
    path: string;
}

/** Flatten the tree recursively, collecting id/name/depth for display. */
function flattenTree(nodes: OrgTreeNode[], depth = 0, path = ''): FlatNode[] {
    const result: FlatNode[] = [];
    for (const n of nodes) {
        const nodePath = path ? `${path} > ${n.name}` : n.name;
        result.push({id: n.id, name: n.name, depth, path: nodePath});
        if (n.children && n.children.length > 0) {
            result.push(...flattenTree(n.children, depth + 1, nodePath));
        }
    }
    return result;
}

/** Collect all descendant IDs of a node (inclusive) to exclude from move targets. */
function collectDescendantIds(nodes: OrgTreeNode[], targetId: string): Set<string> {
    const ids = new Set<string>();
    function traverse(list: OrgTreeNode[], inside: boolean) {
        for (const n of list) {
            const isTarget = n.id === targetId;
            if (inside || isTarget) {
                ids.add(n.id);
                if (n.children) {
                    traverse(n.children, true);
                }
            } else if (n.children) {
                traverse(n.children, false);
            }
        }
    }
    traverse(nodes, false);
    return ids;
}

const MoveNodeDialog: React.FC<MoveNodeDialogProps> = ({nodeId, nodeName, onClose}) => {
    const dispatch = useDispatch();

    // Fetch the complete tree on mount — the Redux tree is depth=1 and may be
    // missing deep nodes, so we load afresh with no depth limit.
    const [fullTree, setFullTree] = useState<OrgTreeNode[]>([]);
    const [loadingTree, setLoadingTree] = useState(true);

    useEffect(() => {
        OrgDirectoryAPI.getFullTree().then((t) => {
            setFullTree(t);
        }).catch(() => {
            // fall through with empty list — user will see "没有可用的目标节点"
        }).finally(() => {
            setLoadingTree(false);
        });
    }, []);

    const [selectedParentId, setSelectedParentId] = useState<string | null>(null);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Build flat list, excluding the node itself and all descendants
    const candidates = useMemo(() => {
        const excluded = collectDescendantIds(fullTree, nodeId);
        const flat = flattenTree(fullTree);
        return flat.filter((n) => !excluded.has(n.id));
    }, [fullTree, nodeId]);

    const handleConfirm = async () => {
        if (!selectedParentId) {
            setError('请选择目标父节点');
            return;
        }
        setSaving(true);
        setError(null);
        try {
            await (dispatch as any)(moveOrgNode(nodeId, selectedParentId));
            onClose();
        } catch (err: any) {
            setError(err.message || '移动失败，请重试');
        } finally {
            setSaving(false);
        }
    };

    const inputStyle: React.CSSProperties = {
        padding: '5px 8px',
        borderRadius: '4px',
        border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.24)',
        background: 'var(--center-channel-bg,#fff)',
        color: 'var(--center-channel-text,#3d3c40)',
        fontSize: '12px',
        outline: 'none',
    };

    return (
        <div className='org-directory-user-detail-panel'>
            {/* Header */}
            <div
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    marginBottom: '16px',
                    paddingBottom: '8px',
                    borderBottom: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.12)',
                }}
            >
                <button
                    style={{background: 'none', border: 'none', cursor: 'pointer', fontSize: '18px', color: 'var(--center-channel-text,#3d3c40)', padding: '0 8px 0 0', lineHeight: 1}}
                    onClick={onClose}
                    title={'返回'}
                >{'←'}</button>
                <span style={{fontWeight: 600, fontSize: '15px'}}>{'移动节点'}</span>
                <span style={{color: '#999', fontSize: '12px', marginLeft: '4px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>
                    {'— '}{nodeName}
                </span>
            </div>

            <div style={{marginBottom: '12px', fontSize: '12px', color: '#666'}}>
                {'选择「'}{nodeName}{'」的新父节点：'}
            </div>

            {/* Candidate list */}
            <div
                style={{
                    flex: 1,
                    overflowY: 'auto',
                    maxHeight: '320px',
                    border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.16)',
                    borderRadius: '4px',
                    marginBottom: '12px',
                }}
            >
                {loadingTree && (
                    <div style={{padding: '16px', textAlign: 'center', color: '#aaa', fontSize: '12px'}}>
                        {'加载节点列表…'}
                    </div>
                )}
                {!loadingTree && candidates.length === 0 && (
                    <div style={{padding: '16px', textAlign: 'center', color: '#aaa', fontSize: '12px'}}>
                        {'没有可用的目标节点'}
                    </div>
                )}
                {candidates.map((n) => (
                    <div
                        key={n.id}
                        onClick={() => setSelectedParentId(n.id)}
                        style={{
                            display: 'flex',
                            alignItems: 'center',
                            padding: `6px 8px 6px ${n.depth * 16 + 8}px`,
                            cursor: 'pointer',
                            background: selectedParentId === n.id
                                ? 'rgba(var(--button-bg-rgb,28,88,217),0.1)'
                                : 'transparent',
                            borderLeft: selectedParentId === n.id
                                ? '3px solid var(--button-bg,#1c58d9)'
                                : '3px solid transparent',
                            fontSize: '12px',
                            color: 'var(--center-channel-text,#3d3c40)',
                        }}
                        onMouseEnter={(e) => {
                            if (selectedParentId !== n.id) {
                                (e.currentTarget as HTMLDivElement).style.background = 'rgba(var(--center-channel-text-rgb,63,67,80),0.06)';
                            }
                        }}
                        onMouseLeave={(e) => {
                            if (selectedParentId !== n.id) {
                                (e.currentTarget as HTMLDivElement).style.background = 'transparent';
                            }
                        }}
                    >
                        <span style={{marginRight: '6px'}}>{'📁'}</span>
                        <span style={{flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>{n.name}</span>
                    </div>
                ))}
            </div>

            {/* Selected display */}
            {selectedParentId && (
                <div style={{fontSize: '12px', color: '#666', marginBottom: '8px'}}>
                    {'已选: '}{candidates.find((n) => n.id === selectedParentId)?.path}
                </div>
            )}

            {error && (
                <div style={{color: '#f74343', fontSize: '11px', marginBottom: '8px'}}>{error}</div>
            )}

            {/* Actions */}
            <div style={{display: 'flex', gap: '8px'}}>
                <button
                    onClick={handleConfirm}
                    disabled={saving || !selectedParentId}
                    style={{
                        flex: 1,
                        padding: '7px',
                        borderRadius: '4px',
                        border: 'none',
                        background: 'var(--button-bg,#1c58d9)',
                        color: '#fff',
                        fontSize: '13px',
                        fontWeight: 600,
                        cursor: saving || !selectedParentId ? 'not-allowed' : 'pointer',
                        opacity: saving || !selectedParentId ? 0.6 : 1,
                    }}
                >{saving ? '移动中...' : '确认移动'}</button>
                <button
                    onClick={onClose}
                    style={{...inputStyle, padding: '7px 14px', cursor: 'pointer'}}
                >{'取消'}</button>
            </div>
        </div>
    );
};

export default MoveNodeDialog;
