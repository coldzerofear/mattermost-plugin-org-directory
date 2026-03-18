package main

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
)

// handleGetFullTree handles GET /api/v1/tree
// Returns the full organization tree up to an optional depth.
func (p *Plugin) handleGetFullTree(w http.ResponseWriter, r *http.Request) {
	depth := -1 // -1 means unlimited
	if d := r.URL.Query().Get("depth"); d != "" {
		if v, err := strconv.Atoi(d); err == nil {
			depth = v
		}
	}

	roots, err := p.store.GetRootNodes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get tree")
		return
	}

	tree := make([]*pluginmodel.OrgTreeNode, 0, len(roots))
	for _, root := range roots {
		treeNode := p.buildTreeNode(root, depth, 0)
		tree = append(tree, treeNode)
	}

	writeJSON(w, http.StatusOK, tree)
}

// handleGetSubTree handles GET /api/v1/tree/{id}
func (p *Plugin) handleGetSubTree(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	depth := -1
	if d := r.URL.Query().Get("depth"); d != "" {
		if v, err := strconv.Atoi(d); err == nil {
			depth = v
		}
	}

	node, err := p.store.GetNode(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	treeNode := p.buildTreeNode(node, depth, 0)
	writeJSON(w, http.StatusOK, treeNode)
}

// handleGetRoots handles GET /api/v1/roots
func (p *Plugin) handleGetRoots(w http.ResponseWriter, r *http.Request) {
	roots, err := p.store.GetRootNodes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get roots")
		return
	}

	// Attach has_children and member count for each root
	for _, root := range roots {
		children, _ := p.store.GetChildNodes(root.ID)
		root.HasChildren = len(children) > 0
		root.MemberCount, _ = p.store.GetNodeMemberCount(root.ID, false)
	}

	writeJSON(w, http.StatusOK, roots)
}

// buildTreeNode recursively constructs a tree node with children up to the given depth.
func (p *Plugin) buildTreeNode(node *pluginmodel.OrgNode, maxDepth, currentDepth int) *pluginmodel.OrgTreeNode {
	treeNode := &pluginmodel.OrgTreeNode{
		OrgNode:  node,
		Children: []*pluginmodel.OrgTreeNode{},
	}

	// Count members at this node
	treeNode.MemberCount, _ = p.store.GetNodeMemberCount(node.ID, false)

	// Load children if within depth limit
	if maxDepth < 0 || currentDepth < maxDepth {
		children, err := p.store.GetChildNodes(node.ID)
		if err == nil {
			for _, child := range children {
				childTree := p.buildTreeNode(child, maxDepth, currentDepth+1)
				treeNode.Children = append(treeNode.Children, childTree)
			}
		}
	} else {
		// Check whether children exist without loading them
		children, _ := p.store.GetChildNodes(node.ID)
		node.HasChildren = len(children) > 0
	}

	return treeNode
}
