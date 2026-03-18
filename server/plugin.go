package main

import (
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	root "github.com/your-org/mattermost-plugin-org-directory"
	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
	"github.com/your-org/mattermost-plugin-org-directory/server/store"
)

var manifest = root.Manifest

// Plugin implements the interface expected by the Mattermost server to communicate between the
// server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin
	client *pluginapi.Client

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration.
	configuration *configuration

	router    *mux.Router
	BotUserID string
	store     store.Store
}

// OnActivate is invoked when the plugin is activated.
func (p *Plugin) OnActivate() error {
	// 1. Initialize pluginapi client
	if p.client == nil {
		p.client = pluginapi.NewClient(p.API, p.Driver)
	}

	// 2. Ensure Bot account exists
	botID, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    "org-directory",
		DisplayName: "组织通讯录",
		Description: "组织通讯录插件机器人",
	})
	if err != nil {
		return err
	}
	p.BotUserID = botID

	// 3. Initialize data store layer (create tables, run migrations)
	sqlStore, err := store.NewSQLStore(p.client, p.API)
	if err != nil {
		return err
	}
	p.store = sqlStore

	// 4. Initialize HTTP router
	p.initializeAPI()

	// 5. Register Slash command
	return p.API.RegisterCommand(getCommand())
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	if p.store != nil {
		return p.store.Close()
	}
	return nil
}

// ServeHTTP handles HTTP requests routed to the plugin.
func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.router.ServeHTTP(w, r)
}

// isSystemAdmin checks if the given user is a system administrator.
func (p *Plugin) isSystemAdmin(userID string) bool {
	user, appErr := p.API.GetUser(userID)
	if appErr != nil {
		return false
	}
	return user.IsInRole(model.SystemAdminRoleId)
}

// broadcastTreeUpdate publishes a WebSocket event for tree structure changes.
func (p *Plugin) broadcastTreeUpdate(action string, node *pluginmodel.OrgNode) {
	data := map[string]interface{}{
		"action": action,
	}
	if node != nil {
		data["node_id"] = node.ID
		data["parent_id"] = node.ParentID
		data["path"] = node.Path
	}
	p.API.PublishWebSocketEvent(
		WSEventTreeUpdate,
		data,
		&model.WebsocketBroadcast{},
	)
}

// broadcastMemberUpdate publishes a WebSocket event for member changes.
func (p *Plugin) broadcastMemberUpdate(action, nodeID, userID string) {
	p.API.PublishWebSocketEvent(
		WSEventMemberUpdate,
		map[string]interface{}{
			"action":  action,
			"node_id": nodeID,
			"user_id": userID,
		},
		&model.WebsocketBroadcast{},
	)
}

const (
	WSEventTreeUpdate   = "tree_update"
	WSEventMemberUpdate = "member_update"
)
