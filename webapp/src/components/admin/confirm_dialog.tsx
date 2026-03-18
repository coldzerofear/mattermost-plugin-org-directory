import React from 'react';

interface ConfirmDialogProps {
    title: string;
    message: string;
    confirmText?: string;
    cancelText?: string;
    dangerous?: boolean;
    onConfirm: () => void;
    onCancel: () => void;
}

const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
    title,
    message,
    confirmText = '确认',
    cancelText = '取消',
    dangerous,
    onConfirm,
    onCancel,
}) => (
    <div
        style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(0,0,0,0.45)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1000,
        }}
        onClick={onCancel}
    >
        <div
            style={{
                background: 'var(--center-channel-bg,#fff)',
                borderRadius: '8px',
                padding: '20px',
                width: '280px',
                boxShadow: '0 4px 20px rgba(0,0,0,0.18)',
            }}
            onClick={(e) => e.stopPropagation()}
        >
            <div style={{fontWeight: 600, fontSize: '15px', marginBottom: '8px', color: 'var(--center-channel-text,#3d3c40)'}}>
                {title}
            </div>
            <div style={{fontSize: '13px', color: '#666', marginBottom: '20px', lineHeight: 1.5}}>
                {message}
            </div>
            <div style={{display: 'flex', gap: '8px'}}>
                <button
                    onClick={onCancel}
                    style={{flex: 1, padding: '8px', borderRadius: '4px', border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.24)', background: 'transparent', color: 'var(--center-channel-text,#3d3c40)', fontSize: '13px', cursor: 'pointer'}}
                >{cancelText}</button>
                <button
                    onClick={onConfirm}
                    style={{flex: 1, padding: '8px', borderRadius: '4px', border: 'none', background: dangerous ? '#f74343' : 'var(--button-bg,#1c58d9)', color: '#fff', fontWeight: 600, fontSize: '13px', cursor: 'pointer'}}
                >{confirmText}</button>
            </div>
        </div>
    </div>
);

export default ConfirmDialog;
