import React from 'react';

const Loading: React.FC = () => (
    <div
        className='org-directory-loading'
        style={{textAlign: 'center', padding: '20px', color: '#999'}}
        aria-label={'加载中'}
        aria-busy={true}
    >
        <span
            className='org-directory-node-loading'
            style={{fontSize: '22px', display: 'block', marginBottom: '8px'}}
            aria-hidden={true}
        >
            {'⟳'}
        </span>
        <span>{'加载中...'}</span>
    </div>
);

export default Loading;
