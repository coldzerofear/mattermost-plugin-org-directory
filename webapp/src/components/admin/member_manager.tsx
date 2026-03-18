import React, {useState, useEffect} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {OrgMember} from '../../types/org_node';
import {getMembersCache} from '../../store/selectors';
import {addOrgMember, removeOrgMember, updateOrgMemberRole, reorderOrgMembers, reloadNodeMembers} from '../../store/actions';

interface MemberManagerProps {
    nodeId: string;
    nodeName: string;
    onClose: () => void;
}

const ROLES = ['member', 'manager', 'admin'];

const MemberManager: React.FC<MemberManagerProps> = ({nodeId, nodeName, onClose}) => {
    const dispatch = useDispatch();
    const membersCache = useSelector(getMembersCache);
    const members: OrgMember[] = membersCache[nodeId] || [];

    // Load members on mount (cache may be empty if node was never expanded)
    useEffect(() => {
        (dispatch as any)(reloadNodeMembers(nodeId));
    }, [nodeId]);

    const [addUsername, setAddUsername] = useState('');
    const [addResult, setAddResult] = useState<{id: string; username: string; display: string} | null>(null);
    const [addRole, setAddRole] = useState('member');
    const [addPosition, setAddPosition] = useState('');
    const [searching, setSearching] = useState(false);
    const [addError, setAddError] = useState<string | null>(null);
    const [saving, setSaving] = useState(false);
    const [removingId, setRemovingId] = useState<string | null>(null);

    const handleSearchUser = async () => {
        if (!addUsername.trim()) {
            return;
        }
        setSearching(true);
        setAddResult(null);
        setAddError(null);
        try {
            const resp = await fetch(`/api/v4/users/username/${encodeURIComponent(addUsername.trim())}`, {
                credentials: 'same-origin',
                headers: {'X-Requested-With': 'XMLHttpRequest'},
            });
            if (!resp.ok) {
                setAddError('未找到该用户');
            } else {
                const u = await resp.json();
                const display = [u.first_name, u.last_name].filter(Boolean).join(' ') || u.username;
                setAddResult({id: u.id, username: u.username, display});
            }
        } catch {
            setAddError('搜索失败，请检查网络');
        } finally {
            setSearching(false);
        }
    };

    const handleAddMember = async () => {
        if (!addResult) {
            return;
        }
        setSaving(true);
        setAddError(null);
        try {
            await (dispatch as any)(addOrgMember(nodeId, {
                user_id: addResult.id,
                role: addRole,
                position: addPosition.trim() || undefined,
            }));
            setAddUsername('');
            setAddResult(null);
            setAddRole('member');
            setAddPosition('');
        } catch (err: any) {
            setAddError(err.message || '添加失败');
        } finally {
            setSaving(false);
        }
    };

    const handleRemove = async (userId: string) => {
        setRemovingId(userId);
        try {
            await (dispatch as any)(removeOrgMember(nodeId, userId));
        } finally {
            setRemovingId(null);
        }
    };

    const handleRoleChange = async (userId: string, role: string) => {
        await (dispatch as any)(updateOrgMemberRole(nodeId, userId, role));
    };

    const handleMoveUp = async (idx: number) => {
        if (idx === 0) {
            return;
        }
        const newOrder = members.map((m) => m.user_id);
        [newOrder[idx - 1], newOrder[idx]] = [newOrder[idx], newOrder[idx - 1]];
        await (dispatch as any)(reorderOrgMembers(nodeId, newOrder));
    };

    const handleMoveDown = async (idx: number) => {
        if (idx >= members.length - 1) {
            return;
        }
        const newOrder = members.map((m) => m.user_id);
        [newOrder[idx], newOrder[idx + 1]] = [newOrder[idx + 1], newOrder[idx]];
        await (dispatch as any)(reorderOrgMembers(nodeId, newOrder));
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
                <span style={{fontWeight: 600, fontSize: '15px'}}>{'管理成员'}</span>
                <span style={{color: '#999', fontSize: '12px', marginLeft: '4px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>
                    {'— '}{nodeName}
                </span>
            </div>

            {/* Current members list */}
            <div style={{marginBottom: '16px'}}>
                <div style={{fontSize: '11px', color: '#999', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: '8px'}}>
                    {'当前成员 ('}{members.length}{')'}
                </div>
                {members.length === 0 && (
                    <div style={{fontSize: '12px', color: '#aaa', padding: '8px 0'}}>{'暂无成员'}</div>
                )}
                {members.map((m, idx) => {
                    const display = [m.first_name, m.last_name].filter(Boolean).join(' ') || m.username;
                    return (
                        <div
                            key={m.user_id}
                            style={{display: 'flex', alignItems: 'center', gap: '4px', padding: '6px 0', borderBottom: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.06)'}}
                        >
                            {/* Reorder buttons */}
                            <div style={{display: 'flex', flexDirection: 'column', gap: '1px', flexShrink: 0}}>
                                <button
                                    onClick={() => handleMoveUp(idx)}
                                    disabled={idx === 0}
                                    title={'上移'}
                                    style={{background: 'none', border: 'none', cursor: idx === 0 ? 'default' : 'pointer', color: '#aaa', fontSize: '10px', padding: '0 2px', lineHeight: 1, opacity: idx === 0 ? 0.3 : 1}}
                                >{'▲'}</button>
                                <button
                                    onClick={() => handleMoveDown(idx)}
                                    disabled={idx >= members.length - 1}
                                    title={'下移'}
                                    style={{background: 'none', border: 'none', cursor: idx >= members.length - 1 ? 'default' : 'pointer', color: '#aaa', fontSize: '10px', padding: '0 2px', lineHeight: 1, opacity: idx >= members.length - 1 ? 0.3 : 1}}
                                >{'▼'}</button>
                            </div>
                            <span style={{flex: 1, fontSize: '12px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>
                                {display}
                                <span style={{color: '#aaa', fontSize: '11px', marginLeft: '4px'}}>{'@'}{m.username}</span>
                            </span>
                            <select
                                value={m.role}
                                onChange={(e) => handleRoleChange(m.user_id, e.target.value)}
                                style={{...inputStyle, padding: '2px 4px', cursor: 'pointer', flexShrink: 0}}
                            >
                                {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
                            </select>
                            <button
                                onClick={() => handleRemove(m.user_id)}
                                disabled={removingId === m.user_id}
                                title={'移除成员'}
                                style={{background: 'none', border: 'none', cursor: 'pointer', color: '#f74343', fontSize: '14px', padding: '2px', lineHeight: 1, flexShrink: 0, opacity: removingId === m.user_id ? 0.4 : 1}}
                            >{'✕'}</button>
                        </div>
                    );
                })}
            </div>

            {/* Add member */}
            <div style={{borderTop: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.12)', paddingTop: '12px'}}>
                <div style={{fontSize: '11px', color: '#999', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: '8px'}}>
                    {'添加成员'}
                </div>
                <div style={{display: 'flex', gap: '6px', marginBottom: '6px'}}>
                    <input
                        value={addUsername}
                        onChange={(e) => setAddUsername(e.target.value)}
                        onKeyDown={(e) => e.key === 'Enter' && handleSearchUser()}
                        placeholder={'输入用户名查找'}
                        style={{...inputStyle, flex: 1}}
                    />
                    <button
                        onClick={handleSearchUser}
                        disabled={searching}
                        style={{...inputStyle, padding: '5px 10px', cursor: 'pointer', flexShrink: 0}}
                    >{searching ? '...' : '查找'}</button>
                </div>

                {addError && (
                    <div style={{color: '#f74343', fontSize: '11px', marginBottom: '6px'}}>{addError}</div>
                )}

                {addResult && (
                    <div
                        style={{
                            padding: '8px',
                            background: 'rgba(var(--center-channel-text-rgb,63,67,80),0.04)',
                            borderRadius: '4px',
                            marginBottom: '6px',
                        }}
                    >
                        <div style={{fontSize: '12px', marginBottom: '8px'}}>
                            {'👤 '}{addResult.display}
                            <span style={{color: '#aaa', marginLeft: '4px'}}>{'@'}{addResult.username}</span>
                        </div>
                        <div style={{display: 'flex', gap: '6px', marginBottom: '6px'}}>
                            <select
                                value={addRole}
                                onChange={(e) => setAddRole(e.target.value)}
                                style={{...inputStyle, flex: 1, cursor: 'pointer'}}
                            >
                                {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
                            </select>
                            <input
                                value={addPosition}
                                onChange={(e) => setAddPosition(e.target.value)}
                                placeholder={'职位（可选）'}
                                style={{...inputStyle, flex: 1}}
                            />
                        </div>
                        <button
                            onClick={handleAddMember}
                            disabled={saving}
                            style={{width: '100%', padding: '6px', borderRadius: '4px', border: 'none', background: 'var(--button-bg,#1c58d9)', color: '#fff', fontSize: '12px', fontWeight: 600, cursor: saving ? 'wait' : 'pointer', opacity: saving ? 0.7 : 1}}
                        >{saving ? '添加中...' : '确认添加'}</button>
                    </div>
                )}
            </div>
        </div>
    );
};

export default MemberManager;
