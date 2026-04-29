package logic

import hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"

var defaultCatalog = []hostpluginmodel.HostPlugin{
	{
		PluginKey:      "opsagent",
		Name:           "OpsAgent",
		Category:       "host-observability",
		Description:    "Host metrics and sandbox execution plugin",
		DefaultVersion: "nodeagentx-dc57fbc-dirty",
		Status:         "active",
	},
}
