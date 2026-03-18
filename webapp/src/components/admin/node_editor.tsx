import React, {useState} from 'react';
import {useDispatch} from 'react-redux';
import {OrgTreeNode} from '../../types/org_node';
import {createOrgNode, updateOrgNode} from '../../store/actions';

interface NodeEditorProps {
    mode: 'create' | 'edit';
    parentId: string | null;   // for create mode (null = root)
    node?: OrgTreeNode;        // for edit mode
    onClose: () => void;
    onSaved: () => void;
}

const NodeEditor: React.FC<NodeEditorProps> = ({mode, parentId, node, onClose, onSaved}) => {
    const dispatch = useDispatch();
    const [name, setName] = useState(mode === 'edit' ? (node?.name || '') : '');
    const [description, setDescription] = useState(mode === 'edit' ? (node?.description || '') : '');
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleSave = async () => {
        if (!name.trim()) {
            setError('节点名称不能为空');
            return;
        }
        setSaving(true);
        setError(null);
        try {
            if (mode === 'create') {
                await (dispatch as any)(createOrgNode({
                    name: name.trim(),
                    parent_id: parentId || '',
                    description: description.trim() || undefined,
                }));
            } else if (node) {
                await (dispatch as any)(updateOrgNode(node.id, {
                    name: name.trim(),
                    description: description.trim() || undefined,
                }));
            }
            onSaved();
        } catch (err: any) {
            setError(err.message || '操作失败');
            setSaving(false);
        }
    };

    const inputStyle: React.CSSProperties = {
        width: '100%',
        padding: '6px 8px',
        borderRadius: '4px',
        border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.24)',
        background: 'var(--center-channel-bg,#fff)',
        color: 'var(--center-channel-text,#3d3c40)',
        fontSize: '13px',
        boxSizing: 'border-box',
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
                <span style={{fontWeight: 600, fontSize: '15px'}}>
                    {mode === 'create' ? '新建节点' : '编辑节点'}
                </span>
            </div>

            {/* Form */}
            <div style={{display: 'flex', flexDirection: 'column', gap: '12px'}}>
                <div>
                    <div style={{marginBottom: '4px', fontSize: '12px', fontWeight: 600, color: 'var(--center-channel-text,#3d3c40)'}}>
                        {'节点名称 *'}
                    </div>
                    <input
                        autoFocus={true}
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        onKeyDown={(e) => e.key === 'Enter' && handleSave()}
                        placeholder={'请输入节点名称'}
                        style={inputStyle}
                    />
                </div>

                <div>
                    <div style={{marginBottom: '4px', fontSize: '12px', fontWeight: 600, color: 'var(--center-channel-text,#3d3c40)'}}>
                        {'描述'}
                    </div>
                    <textarea
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        placeholder={'可选描述'}
                        rows={3}
                        style={{...inputStyle, resize: 'vertical', fontFamily: 'inherit'}}
                    />
                </div>

                {error && (
                    <div style={{color: '#f74343', fontSize: '12px'}}>{error}</div>
                )}
            </div>

            {/* Buttons */}
            <div style={{display: 'flex', gap: '8px', marginTop: '20px'}}>
                <button
                    onClick={onClose}
                    style={{flex: 1, padding: '8px', borderRadius: '4px', border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.24)', background: 'transparent', color: 'var(--center-channel-text,#3d3c40)', fontSize: '13px', cursor: 'pointer'}}
                >{'取消'}</button>
                <button
                    onClick={handleSave}
                    disabled={saving}
                    style={{flex: 1, padding: '8px', borderRadius: '4px', border: 'none', background: 'var(--button-bg,#1c58d9)', color: '#fff', fontWeight: 600, fontSize: '13px', cursor: saving ? 'wait' : 'pointer', opacity: saving ? 0.7 : 1}}
                >{saving ? '保存中...' : '保存'}</button>
            </div>
        </div>
    );
};

export default NodeEditor;
