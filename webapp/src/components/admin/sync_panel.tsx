import React, {useState, useEffect, useCallback} from 'react';
import {OrgDirectoryAPI} from '../../api/client';
import {SyncLog, UserMapping} from '../../types/sync';

interface SyncPanelProps {
    onClose: () => void;
}

type Tab = 'logs' | 'mappings';

// ── helpers ──────────────────────────────────────────────────────────────────

function formatTime(ms: number): string {
    if (!ms) {
        return '—';
    }
    return new Date(ms).toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
    });
}

function formatDuration(startMs: number, endMs: number): string {
    if (!startMs || !endMs) {
        return '—';
    }
    const sec = Math.round((endMs - startMs) / 1000);
    if (sec < 60) {
        return `${sec}s`;
    }
    return `${Math.floor(sec / 60)}m ${sec % 60}s`;
}

const STATUS_CONFIG: Record<string, {label: string; color: string; bg: string}> = {
    success: {label: '成功', color: '#3db887', bg: 'rgba(61,184,135,0.12)'},
    partial_success: {label: '部分成功', color: '#ffbc1f', bg: 'rgba(255,188,31,0.12)'},
    failed: {label: '失败', color: '#f74343', bg: 'rgba(247,67,67,0.12)'},
    running: {label: '运行中', color: '#1c58d9', bg: 'rgba(28,88,217,0.12)'},
};

const StatusBadge: React.FC<{status: string}> = ({status}) => {
    const cfg = STATUS_CONFIG[status] || {label: status, color: '#aaa', bg: 'rgba(170,170,170,0.12)'};
    return (
        <span
            style={{
                display: 'inline-block',
                padding: '1px 7px',
                borderRadius: '10px',
                fontSize: '11px',
                fontWeight: 600,
                color: cfg.color,
                background: cfg.bg,
                whiteSpace: 'nowrap',
            }}
        >{cfg.label}</span>
    );
};

function CountChip({n, kind}: {n: number; kind: 'create' | 'update' | 'delete' | 'skip'}) {
    if (n === 0) {
        return null;
    }
    const colors = {create: '#3db887', update: '#1c58d9', delete: '#f74343', skip: '#ffbc1f'};
    const symbols = {create: '+', update: '~', delete: '-', skip: '!'};
    return (
        <span style={{color: colors[kind], fontSize: '11px', marginRight: '3px'}}>
            {symbols[kind]}{n}
        </span>
    );
}

// ── Sync Log Detail ───────────────────────────────────────────────────────────

const SyncLogDetail: React.FC<{log: SyncLog}> = ({log}) => (
    <div
        style={{
            padding: '8px 10px',
            background: 'rgba(var(--center-channel-text-rgb,63,67,80),0.04)',
            borderRadius: '4px',
            fontSize: '12px',
            marginTop: '4px',
        }}
    >
        <div style={{display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '4px 12px', marginBottom: '8px'}}>
            <StatRow label={'开始时间'} value={formatTime(log.started_at)}/>
            <StatRow label={'结束时间'} value={formatTime(log.finished_at)}/>
            <StatRow label={'耗时'} value={formatDuration(log.started_at, log.finished_at)}/>
            <StatRow label={'触发者'} value={log.triggered_by || '外部系统'}/>
        </div>

        <div style={{marginBottom: '6px'}}>
            <div style={{fontWeight: 600, marginBottom: '3px', color: 'var(--center-channel-text,#3d3c40)'}}>{'节点'}</div>
            <div>
                {'总计 '}<strong>{log.total_nodes}</strong>
                {' | '}
                <span style={{color: '#3db887'}}>{'新增 '}<strong>{log.created_nodes}</strong></span>
                {' | '}
                <span style={{color: '#1c58d9'}}>{'更新 '}<strong>{log.updated_nodes}</strong></span>
                {' | '}
                <span style={{color: '#f74343'}}>{'删除 '}<strong>{log.deleted_nodes}</strong></span>
            </div>
        </div>

        <div style={{marginBottom: '6px'}}>
            <div style={{fontWeight: 600, marginBottom: '3px', color: 'var(--center-channel-text,#3d3c40)'}}>{'成员'}</div>
            <div>
                {'总计 '}<strong>{log.total_members}</strong>
                {' | '}
                <span style={{color: '#3db887'}}>{'新增 '}<strong>{log.created_members}</strong></span>
                {' | '}
                <span style={{color: '#1c58d9'}}>{'更新 '}<strong>{log.updated_members}</strong></span>
                {' | '}
                <span style={{color: '#f74343'}}>{'删除 '}<strong>{log.deleted_members}</strong></span>
                {log.skipped_users > 0 && (
                    <span style={{color: '#ffbc1f'}}>
                        {' | '}{'跳过 '}<strong>{log.skipped_users}</strong>
                    </span>
                )}
            </div>
        </div>

        {log.error_message && (
            <div style={{color: '#f74343', marginTop: '6px', wordBreak: 'break-all'}}>
                <span style={{fontWeight: 600}}>{'错误: '}</span>{log.error_message}
            </div>
        )}
    </div>
);

const StatRow: React.FC<{label: string; value: string}> = ({label, value}) => (
    <div>
        <span style={{color: '#999'}}>{label}: </span>
        <span style={{color: 'var(--center-channel-text,#3d3c40)'}}>{value}</span>
    </div>
);

// ── Sync Logs Tab ─────────────────────────────────────────────────────────────

const SyncLogsTab: React.FC = () => {
    const [logs, setLogs] = useState<SyncLog[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [page, setPage] = useState(0);
    const [hasMore, setHasMore] = useState(false);
    const [sourceFilter, setSourceFilter] = useState('');
    const [expandedId, setExpandedId] = useState<string | null>(null);

    const perPage = 15;

    const loadLogs = useCallback(async (p: number, src: string) => {
        setLoading(true);
        setError(null);
        try {
            const data = await OrgDirectoryAPI.getSyncLogs(src, p, perPage);
            setLogs(data || []);
            setHasMore((data || []).length === perPage);
        } catch (err: any) {
            setError(err.message || '加载失败');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        loadLogs(0, '');
    }, []);

    const handleFilterApply = () => {
        setPage(0);
        setExpandedId(null);
        loadLogs(0, sourceFilter);
    };

    const handlePage = (delta: number) => {
        const next = page + delta;
        setPage(next);
        setExpandedId(null);
        loadLogs(next, sourceFilter);
    };

    return (
        <div>
            {/* Filter bar */}
            <div style={{display: 'flex', gap: '6px', marginBottom: '10px'}}>
                <input
                    value={sourceFilter}
                    onChange={(e) => setSourceFilter(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleFilterApply()}
                    placeholder={'来源筛选（留空=全部）'}
                    style={{
                        flex: 1,
                        padding: '5px 8px',
                        borderRadius: '4px',
                        border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.24)',
                        background: 'var(--center-channel-bg,#fff)',
                        color: 'var(--center-channel-text,#3d3c40)',
                        fontSize: '12px',
                        outline: 'none',
                    }}
                />
                <button
                    onClick={handleFilterApply}
                    style={{padding: '5px 10px', borderRadius: '4px', border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.24)', background: 'transparent', color: 'var(--center-channel-text,#3d3c40)', fontSize: '12px', cursor: 'pointer', flexShrink: 0}}
                >{'筛选'}</button>
                <button
                    onClick={() => loadLogs(page, sourceFilter)}
                    title={'刷新'}
                    style={{padding: '5px 8px', borderRadius: '4px', border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.24)', background: 'transparent', color: 'var(--center-channel-text,#3d3c40)', fontSize: '12px', cursor: 'pointer', flexShrink: 0}}
                >{'↺'}</button>
            </div>

            {/* Log list */}
            {loading && <div style={{color: '#999', fontSize: '12px', padding: '12px 0', textAlign: 'center'}}>{'加载中...'}</div>}
            {error && <div style={{color: '#f74343', fontSize: '12px', marginBottom: '8px'}}>{error}</div>}
            {!loading && logs.length === 0 && (
                <div style={{color: '#aaa', fontSize: '12px', textAlign: 'center', padding: '20px 0'}}>{'暂无同步记录'}</div>
            )}

            {logs.map((log) => {
                const isExpanded = expandedId === log.id;
                return (
                    <div
                        key={log.id}
                        style={{
                            marginBottom: '6px',
                            border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.1)',
                            borderRadius: '4px',
                            overflow: 'hidden',
                        }}
                    >
                        {/* Row */}
                        <div
                            style={{
                                display: 'flex',
                                alignItems: 'center',
                                gap: '6px',
                                padding: '7px 8px',
                                cursor: 'pointer',
                                background: isExpanded ? 'rgba(var(--center-channel-text-rgb,63,67,80),0.04)' : undefined,
                            }}
                            onClick={() => setExpandedId(isExpanded ? null : log.id)}
                        >
                            <StatusBadge status={log.status}/>
                            <span style={{flex: 1, fontSize: '12px', fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: 'var(--center-channel-text,#3d3c40)'}}>
                                {log.source}
                            </span>
                            <span style={{fontSize: '11px', color: '#aaa', flexShrink: 0}}>
                                {log.sync_type === 'full' ? '全量' : '增量'}
                            </span>
                            <span style={{fontSize: '11px', flexShrink: 0}}>
                                <CountChip n={log.created_nodes} kind={'create'}/>
                                <CountChip n={log.updated_nodes} kind={'update'}/>
                                <CountChip n={log.deleted_nodes} kind={'delete'}/>
                                <span style={{color: '#aaa', margin: '0 3px'}}>{'|'}</span>
                                <CountChip n={log.created_members} kind={'create'}/>
                                <CountChip n={log.updated_members} kind={'update'}/>
                                <CountChip n={log.deleted_members} kind={'delete'}/>
                                {log.skipped_users > 0 && <CountChip n={log.skipped_users} kind={'skip'}/>}
                            </span>
                            <span style={{fontSize: '11px', color: '#aaa', flexShrink: 0}}>{formatTime(log.started_at)}</span>
                            <span style={{fontSize: '11px', color: '#aaa', flexShrink: 0}}>{isExpanded ? '▲' : '▼'}</span>
                        </div>

                        {/* Expanded detail */}
                        {isExpanded && (
                            <div style={{padding: '0 8px 8px'}}>
                                <SyncLogDetail log={log}/>
                            </div>
                        )}
                    </div>
                );
            })}

            {/* Pagination */}
            {(page > 0 || hasMore) && (
                <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '8px'}}>
                    <button
                        onClick={() => handlePage(-1)}
                        disabled={page === 0}
                        style={{padding: '4px 10px', borderRadius: '4px', border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.24)', background: 'transparent', color: 'var(--center-channel-text,#3d3c40)', fontSize: '12px', cursor: page === 0 ? 'default' : 'pointer', opacity: page === 0 ? 0.4 : 1}}
                    >{'← 上一页'}</button>
                    <span style={{fontSize: '12px', color: '#999'}}>{'第 '}{page + 1}{' 页'}</span>
                    <button
                        onClick={() => handlePage(1)}
                        disabled={!hasMore}
                        style={{padding: '4px 10px', borderRadius: '4px', border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.24)', background: 'transparent', color: 'var(--center-channel-text,#3d3c40)', fontSize: '12px', cursor: !hasMore ? 'default' : 'pointer', opacity: !hasMore ? 0.4 : 1}}
                    >{'下一页 →'}</button>
                </div>
            )}
        </div>
    );
};

// ── User Mappings Tab ─────────────────────────────────────────────────────────

const UserMappingsTab: React.FC = () => {
    const [source, setSource] = useState('');
    const [mappings, setMappings] = useState<UserMapping[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [page, setPage] = useState(0);
    const [hasMore, setHasMore] = useState(false);
    const [queried, setQueried] = useState(false);

    const perPage = 30;

    const loadMappings = useCallback(async (src: string, p: number) => {
        if (!src.trim()) {
            return;
        }
        setLoading(true);
        setError(null);
        try {
            const data = await OrgDirectoryAPI.getUserMappings(src.trim(), p, perPage);
            setMappings(data || []);
            setHasMore((data || []).length === perPage);
            setQueried(true);
        } catch (err: any) {
            setError(err.message || '加载失败');
        } finally {
            setLoading(false);
        }
    }, []);

    const handleQuery = () => {
        setPage(0);
        loadMappings(source, 0);
    };

    const handlePage = (delta: number) => {
        const next = page + delta;
        setPage(next);
        loadMappings(source, next);
    };

    return (
        <div>
            {/* Source input */}
            <div style={{display: 'flex', gap: '6px', marginBottom: '10px'}}>
                <input
                    value={source}
                    onChange={(e) => setSource(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleQuery()}
                    placeholder={'输入来源名称（如 hr_system）'}
                    style={{
                        flex: 1,
                        padding: '5px 8px',
                        borderRadius: '4px',
                        border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.24)',
                        background: 'var(--center-channel-bg,#fff)',
                        color: 'var(--center-channel-text,#3d3c40)',
                        fontSize: '12px',
                        outline: 'none',
                    }}
                />
                <button
                    onClick={handleQuery}
                    disabled={loading || !source.trim()}
                    style={{padding: '5px 10px', borderRadius: '4px', border: 'none', background: 'var(--button-bg,#1c58d9)', color: '#fff', fontSize: '12px', fontWeight: 600, cursor: loading || !source.trim() ? 'default' : 'pointer', opacity: loading || !source.trim() ? 0.6 : 1, flexShrink: 0}}
                >{'查询'}</button>
            </div>

            {loading && <div style={{color: '#999', fontSize: '12px', textAlign: 'center', padding: '12px 0'}}>{'加载中...'}</div>}
            {error && <div style={{color: '#f74343', fontSize: '12px', marginBottom: '8px'}}>{error}</div>}
            {queried && !loading && mappings.length === 0 && (
                <div style={{color: '#aaa', fontSize: '12px', textAlign: 'center', padding: '20px 0'}}>{'该来源暂无用户映射'}</div>
            )}

            {mappings.length > 0 && (
                <>
                    {/* Header */}
                    <div
                        style={{
                            display: 'grid',
                            gridTemplateColumns: '1fr 1fr 1fr',
                            gap: '4px',
                            padding: '4px 6px',
                            fontSize: '10px',
                            color: '#999',
                            textTransform: 'uppercase',
                            letterSpacing: '0.5px',
                            borderBottom: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.1)',
                            marginBottom: '4px',
                        }}
                    >
                        <span>{'外部 ID'}</span>
                        <span>{'外部用户名'}</span>
                        <span>{'MM 用户 ID'}</span>
                    </div>

                    {mappings.map((m) => (
                        <div
                            key={m.id}
                            style={{
                                display: 'grid',
                                gridTemplateColumns: '1fr 1fr 1fr',
                                gap: '4px',
                                padding: '5px 6px',
                                fontSize: '11px',
                                color: 'var(--center-channel-text,#3d3c40)',
                                borderBottom: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.06)',
                            }}
                        >
                            <span style={{overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}} title={m.external_user_id}>{m.external_user_id || '—'}</span>
                            <span style={{overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}} title={m.external_username}>{m.external_username || m.external_email || '—'}</span>
                            <span style={{overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: '#666', fontFamily: 'monospace'}} title={m.mm_user_id}>{m.mm_user_id.substring(0, 8)}{'...'}</span>
                        </div>
                    ))}

                    {/* Pagination */}
                    {(page > 0 || hasMore) && (
                        <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '8px'}}>
                            <button
                                onClick={() => handlePage(-1)}
                                disabled={page === 0}
                                style={{padding: '4px 10px', borderRadius: '4px', border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.24)', background: 'transparent', color: 'var(--center-channel-text,#3d3c40)', fontSize: '12px', cursor: page === 0 ? 'default' : 'pointer', opacity: page === 0 ? 0.4 : 1}}
                            >{'← 上一页'}</button>
                            <span style={{fontSize: '12px', color: '#999'}}>{'第 '}{page + 1}{' 页'}</span>
                            <button
                                onClick={() => handlePage(1)}
                                disabled={!hasMore}
                                style={{padding: '4px 10px', borderRadius: '4px', border: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.24)', background: 'transparent', color: 'var(--center-channel-text,#3d3c40)', fontSize: '12px', cursor: !hasMore ? 'default' : 'pointer', opacity: !hasMore ? 0.4 : 1}}
                            >{'下一页 →'}</button>
                        </div>
                    )}
                </>
            )}
        </div>
    );
};

// ── Main SyncPanel ────────────────────────────────────────────────────────────

const SyncPanel: React.FC<SyncPanelProps> = ({onClose}) => {
    const [tab, setTab] = useState<Tab>('logs');

    const tabStyle = (active: boolean): React.CSSProperties => ({
        padding: '5px 12px',
        fontSize: '12px',
        fontWeight: active ? 600 : 400,
        cursor: 'pointer',
        color: active ? 'var(--button-bg,#1c58d9)' : '#999',
        background: 'none',
        borderTop: 'none',
        borderLeft: 'none',
        borderRight: 'none',
        borderBottom: active ? '2px solid var(--button-bg,#1c58d9)' : '2px solid transparent',
        outline: 'none',
    });

    return (
        <div className='org-directory-user-detail-panel'>
            {/* Header */}
            <div
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    marginBottom: '0',
                    paddingBottom: '8px',
                    borderBottom: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.12)',
                }}
            >
                <button
                    style={{background: 'none', border: 'none', cursor: 'pointer', fontSize: '18px', color: 'var(--center-channel-text,#3d3c40)', padding: '0 8px 0 0', lineHeight: 1}}
                    onClick={onClose}
                    title={'返回'}
                >{'←'}</button>
                <span style={{fontWeight: 600, fontSize: '15px'}}>{'同步管理'}</span>
            </div>

            {/* Tabs */}
            <div
                style={{
                    display: 'flex',
                    gap: '0',
                    borderBottom: '1px solid rgba(var(--center-channel-text-rgb,63,67,80),0.12)',
                    marginBottom: '12px',
                    marginLeft: '-12px',
                    marginRight: '-12px',
                    paddingLeft: '12px',
                }}
            >
                <button style={tabStyle(tab === 'logs')} onClick={() => setTab('logs')}>{'同步日志'}</button>
                <button style={tabStyle(tab === 'mappings')} onClick={() => setTab('mappings')}>{'用户映射'}</button>
            </div>

            {/* Tab content */}
            {tab === 'logs' && <SyncLogsTab/>}
            {tab === 'mappings' && <UserMappingsTab/>}
        </div>
    );
};

export default SyncPanel;
