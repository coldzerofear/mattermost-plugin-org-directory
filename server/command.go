package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"

	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
)

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "org",
		DisplayName:      "组织通讯录",
		Description:      "管理组织结构和查询通讯录",
		AutoComplete:     true,
		AutoCompleteDesc: "组织通讯录命令",
		AutoCompleteHint: "[search|info|tree]",
	}
}

// ExecuteCommand handles /org slash commands.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	parts := strings.Fields(args.Command)
	if len(parts) < 2 {
		return p.showCommandHelp(), nil
	}

	switch parts[1] {
	case "search":
		return p.cmdSearch(args, parts[2:])
	case "info":
		return p.cmdInfo(args, parts[2:])
	case "tree":
		return p.cmdTree(args, parts[2:])
	default:
		return p.showCommandHelp(), nil
	}
}

func (p *Plugin) showCommandHelp() *model.CommandResponse {
	text := `**组织通讯录命令帮助**
* ` + "`/org search <关键词>`" + ` — 搜索用户
* ` + "`/org info @用户名`" + ` — 查看用户所属组织
* ` + "`/org tree`" + ` — 以文本形式输出组织树`
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}
}

func (p *Plugin) cmdSearch(args *model.CommandArgs, params []string) (*model.CommandResponse, *model.AppError) {
	if len(params) == 0 {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "请提供搜索关键词。用法: `/org search <关键词>`",
		}, nil
	}
	query := strings.Join(params, " ")
	results, err := p.store.SearchMembers(query, 0, 10)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "搜索失败: " + err.Error(),
		}, nil
	}
	if len(results) == 0 {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("未找到与 **%s** 相关的用户。", query),
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**搜索结果 (%d)**\n\n", len(results)))
	for _, r := range results {
		name := r.User.FirstName + " " + r.User.LastName
		if strings.TrimSpace(name) == "" {
			name = r.User.Username
		}
		sb.WriteString(fmt.Sprintf("- **%s** (@%s) — %s — `%s`\n",
			name, r.User.Username, r.NodeName, r.User.Position))
	}
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         sb.String(),
	}, nil
}

func (p *Plugin) cmdInfo(args *model.CommandArgs, params []string) (*model.CommandResponse, *model.AppError) {
	if len(params) == 0 {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "请提供用户名。用法: `/org info @用户名`",
		}, nil
	}
	username := strings.TrimPrefix(params[0], "@")
	user, appErr := p.API.GetUserByUsername(username)
	if appErr != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "未找到用户 @" + username,
		}, nil
	}

	nodes, err := p.store.GetUserNodes(user.Id)
	if err != nil || len(nodes) == 0 {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("用户 @%s 未挂靠任何组织节点。", username),
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**@%s 所属组织**\n\n", username))
	for _, n := range nodes {
		sb.WriteString(fmt.Sprintf("- **%s** (路径: `%s`)\n", n.Name, n.Path))
	}
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         sb.String(),
	}, nil
}

func (p *Plugin) cmdTree(args *model.CommandArgs, params []string) (*model.CommandResponse, *model.AppError) {
	roots, err := p.store.GetRootNodes()
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "获取组织树失败: " + err.Error(),
		}, nil
	}
	if len(roots) == 0 {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "当前没有组织结构数据。",
		}, nil
	}

	var sb strings.Builder
	sb.WriteString("**组织结构**\n```\n")
	for _, root := range roots {
		p.renderTreeText(&sb, root, "", true)
	}
	sb.WriteString("```")
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         sb.String(),
	}, nil
}

func (p *Plugin) renderTreeText(sb *strings.Builder, node *pluginmodel.OrgNode, prefix string, isLast bool) {
	connector := "├── "
	childPrefix := prefix + "│   "
	if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	}
	sb.WriteString(prefix + connector + node.Name + "\n")
	children, err := p.store.GetChildNodes(node.ID)
	if err != nil {
		return
	}
	for i, child := range children {
		p.renderTreeText(sb, child, childPrefix, i == len(children)-1)
	}
}
