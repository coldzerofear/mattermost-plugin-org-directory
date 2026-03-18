export interface OrgNode {
    id: string;
    name: string;
    parent_id: string;
    path: string;
    depth: number;
    sort_order: number;
    description: string;
    icon: string;
    metadata: string;
    source: string;
    external_id: string;
    create_at: number;
    update_at: number;
    member_count: number;
    has_children: boolean;
    children?: OrgNode[];
}

export interface OrgTreeNode extends OrgNode {
    children: OrgTreeNode[];
    members?: OrgMember[];
    member_count: number;
}

export interface OrgMember {
    id: string;
    node_id: string;
    user_id: string;
    role: string;
    position: string;
    sort_order: number;
    source: string;
    external_id: string;
    create_at: number;
    update_at: number;
    // Joined from Mattermost users table
    username: string;
    first_name: string;
    last_name: string;
    nickname: string;
    email: string;
    mm_position: string;
    status: string;
}

export interface SearchResult {
    user: OrgMember;
    node_name: string;
    node_path: string;
}
