export interface SyncLog {
    id: string;
    source: string;
    sync_type: string;      // full / incremental
    status: string;         // running / success / partial_success / failed
    total_nodes: number;
    created_nodes: number;
    updated_nodes: number;
    deleted_nodes: number;
    total_members: number;
    created_members: number;
    updated_members: number;
    deleted_members: number;
    skipped_users: number;
    error_message: string;
    details: string;        // JSON string
    started_at: number;     // unix ms
    finished_at: number;    // unix ms (0 if still running)
    triggered_by: string;
}

export interface UserMapping {
    id: string;
    source: string;
    external_user_id: string;
    mm_user_id: string;
    external_username: string;
    external_email: string;
    create_at: number;
    update_at: number;
    delete_at: number;
}
