import React, {useEffect} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {OrgMember, OrgNode} from '../../types/org_node';
import {selectUser, fetchUserNodes} from '../../store/actions';
import {getUserNodesForUser} from '../../store/selectors';

interface UserDetailPanelProps {
    member: OrgMember;
}

const UserDetailPanel: React.FC<UserDetailPanelProps> = ({member}) => {
    const dispatch = useDispatch();
    const userNodes: OrgNode[] | undefined = useSelector((state: any) => getUserNodesForUser(state, member.user_id));

    useEffect(() => {
        if (!userNodes) {
            dispatch(fetchUserNodes(member.user_id) as any);
        }
    }, [member.user_id]);

    const displayName = [member.first_name, member.last_name].filter(Boolean).join(' ') || member.username;

    const handleClose = () => {
        dispatch(selectUser(null));
    };

    // Build DM URL: /<teamName>/messages/@<username>
    const teamMatch = window.location.pathname.match(/^\/([^/]+)\//);
    const dmHref = teamMatch ? `/${teamMatch[1]}/messages/@${member.username}` : '';

    const renderOrgPath = (node: OrgNode) => {
        // path looks like "/rootId/parentId/selfId" — show node name hierarchy
        return node.name;
    };

    const statusDot = (status: string) => {
        const colors: Record<string, string> = {
            online: '#3db887',
            away: '#ffbc1f',
            dnd: '#f74343',
            offline: '#aaa',
        };
        const labels: Record<string, string> = {
            online: '在线',
            away: '离开',
            dnd: '勿扰',
            offline: '离线',
        };
        return (
            <span>
                <span
                    style={{
                        display: 'inline-block',
                        width: '8px',
                        height: '8px',
                        borderRadius: '50%',
                        backgroundColor: colors[status] || '#aaa',
                        marginRight: '4px',
                    }}
                />
                {labels[status] || '离线'}
            </span>
        );
    };

    return (
        <div className='org-directory-user-detail-panel'>
            {/* Header with back button */}
            <div
                className='org-directory-user-detail-header'
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    marginBottom: '16px',
                    paddingBottom: '8px',
                    borderBottom: '1px solid rgba(var(--center-channel-text-rgb, 63,67,80),0.12)',
                }}
            >
                <button
                    className='org-directory-back-btn'
                    style={{
                        background: 'none',
                        border: 'none',
                        cursor: 'pointer',
                        fontSize: '18px',
                        color: 'var(--center-channel-text, #3d3c40)',
                        padding: '0 8px 0 0',
                        lineHeight: 1,
                    }}
                    onClick={handleClose}
                    title={'返回'}
                >
                    {'←'}
                </button>
                <span style={{fontWeight: 600, fontSize: '15px'}}>{'用户详情'}</span>
            </div>

            {/* Avatar + name */}
            <div style={{textAlign: 'center', marginBottom: '16px'}}>
                <div
                    style={{
                        width: '64px',
                        height: '64px',
                        borderRadius: '50%',
                        background: 'var(--button-bg, #1c58d9)',
                        color: '#fff',
                        fontSize: '24px',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        margin: '0 auto 8px',
                        fontWeight: 700,
                    }}
                >
                    {displayName.charAt(0).toUpperCase()}
                </div>
                <div style={{fontWeight: 700, fontSize: '16px'}}>{displayName}</div>
                <div style={{color: '#999', fontSize: '13px', marginTop: '2px'}}>{'@'}{member.username}</div>
                {member.status && (
                    <div style={{fontSize: '12px', marginTop: '4px', color: '#666'}}>
                        {statusDot(member.status)}
                    </div>
                )}
            </div>

            {/* Info rows */}
            <div className='org-directory-user-detail-info'>
                {member.email && (
                    <InfoRow icon={'📧'} value={member.email}/>
                )}
                {(member.position || member.mm_position) && (
                    <InfoRow icon={'💼'} value={member.position || member.mm_position}/>
                )}

                {/* Org paths */}
                {userNodes && userNodes.length > 0 && (
                    <div style={{marginTop: '8px'}}>
                        <div style={{fontSize: '11px', color: '#999', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: '4px'}}>
                            {'所属组织'}
                        </div>
                        {userNodes.map((node) => (
                            <div
                                key={node.id}
                                style={{
                                    fontSize: '12px',
                                    color: 'var(--center-channel-text, #3d3c40)',
                                    padding: '3px 0',
                                    display: 'flex',
                                    alignItems: 'flex-start',
                                    gap: '4px',
                                }}
                            >
                                <span>{'🏢'}</span>
                                <span>{renderOrgPath(node)}</span>
                            </div>
                        ))}
                    </div>
                )}
                {userNodes && userNodes.length === 0 && (
                    <div style={{fontSize: '12px', color: '#aaa', marginTop: '8px'}}>{'暂无组织归属'}</div>
                )}
                {!userNodes && (
                    <div style={{fontSize: '12px', color: '#aaa', marginTop: '8px'}}>{'加载中...'}</div>
                )}
            </div>

            {/* Action buttons */}
            <div
                style={{
                    marginTop: '20px',
                    display: 'flex',
                    gap: '8px',
                }}
            >
                {dmHref && (
                    <a
                        className='org-directory-btn-primary'
                        href={dmHref}
                        style={{
                            flex: 1,
                            padding: '8px 12px',
                            borderRadius: '4px',
                            border: 'none',
                            background: 'var(--button-bg, #1c58d9)',
                            color: '#fff',
                            fontWeight: 600,
                            fontSize: '13px',
                            cursor: 'pointer',
                            textAlign: 'center',
                            textDecoration: 'none',
                        }}
                    >
                        {'发送消息'}
                    </a>
                )}
            </div>
        </div>
    );
};

interface InfoRowProps {
    icon: string;
    value: string;
}

const InfoRow: React.FC<InfoRowProps> = ({icon, value}) => (
    <div
        style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            padding: '4px 0',
            fontSize: '13px',
            color: 'var(--center-channel-text, #3d3c40)',
        }}
    >
        <span style={{flexShrink: 0}}>{icon}</span>
        <span style={{overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>{value}</span>
    </div>
);

export default UserDetailPanel;
