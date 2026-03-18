package main

import (
	"reflect"

	"github.com/pkg/errors"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
type configuration struct {
	EnableAuditLog         bool   `json:"EnableAuditLog"`
	AllowUserSearch        bool   `json:"AllowUserSearch"`
	MaxTreeDepth           string `json:"MaxTreeDepth"`
	DefaultPageSize        string `json:"DefaultPageSize"`
	SyncAPIToken           string `json:"SyncAPIToken"`
	SyncUserMatchStrategy  string `json:"SyncUserMatchStrategy"`
	SyncProtectLocalData   bool   `json:"SyncProtectLocalData"`
	SyncFullDeleteStrategy string `json:"SyncFullDeleteStrategy"`
}

// Clone makes a deep copy of the configuration.
func (c *configuration) Clone() *configuration {
	var clone = *c
	return &clone
}

// IsValid checks if the configuration is valid.
func (c *configuration) IsValid() error {
	switch c.SyncUserMatchStrategy {
	case "mapping_email_username", "mapping_only", "email_only", "":
	default:
		return errors.New("invalid SyncUserMatchStrategy value")
	}
	switch c.SyncFullDeleteStrategy {
	case "cascade_delete", "move_to_parent", "":
	default:
		return errors.New("invalid SyncFullDeleteStrategy value")
	}
	return nil
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex supports no
// reentrancy.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the configuration is exactly the same.
		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	var configuration = new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	if err := configuration.IsValid(); err != nil {
		return errors.Wrap(err, "invalid plugin configuration")
	}

	p.setConfiguration(configuration)

	return nil
}

// setDefaultsIfEmpty fills in default values for empty configuration fields.
func (c *configuration) setDefaultsIfEmpty() {
	if reflect.DeepEqual(c, &configuration{}) {
		c.EnableAuditLog = true
		c.AllowUserSearch = true
		c.MaxTreeDepth = "10"
		c.DefaultPageSize = "50"
		c.SyncUserMatchStrategy = "mapping_email_username"
		c.SyncProtectLocalData = true
		c.SyncFullDeleteStrategy = "cascade_delete"
	}
}
