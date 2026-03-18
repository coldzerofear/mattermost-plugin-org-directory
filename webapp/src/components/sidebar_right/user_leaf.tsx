import React from 'react';
import {OrgMember} from '../../types/org_node';

interface UserLeafProps {
    member: OrgMember;
    level: number;
    onUserClick: (userId: string) => void;
}

const UserLeaf: React.FC<UserLeafProps> = React.memo(({member, level, onUserClick}) => {
    const displayName = [member.first_name, member.last_name].filter(Boolean).join(' ') || member.username;
    const indent = level * 20;

    const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            onUserClick(member.user_id);
        }
    };

    return (
        <div
            className='org-directory-user-leaf'
            role='button'
            tabIndex={0}
            style={{paddingLeft: `${indent + 24}px`, padding: `4px 8px 4px ${indent + 24}px`, cursor: 'pointer'}}
            onClick={() => onUserClick(member.user_id)}
            onKeyDown={handleKeyDown}
            aria-label={`${displayName} (@${member.username})`}
        >
            <span className='org-directory-user-icon' style={{marginRight: '6px'}}>{'👤'}</span>
            <span className='org-directory-user-name' style={{fontWeight: 500}}>{displayName}</span>
            {member.position && (
                <span className='org-directory-user-position' style={{color: '#999', fontSize: '12px', marginLeft: '6px'}}>
                    {' - '}{member.position}
                </span>
            )}
            <span className='org-directory-user-username' style={{color: '#aaa', fontSize: '12px', marginLeft: '6px'}}>
                {'@'}{member.username}
            </span>
        </div>
    );
});

UserLeaf.displayName = 'UserLeaf';

export default UserLeaf;
