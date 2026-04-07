package ulighthost

// 地域和可用区名称映射（复用 UHost 的定义）
var regionNames = map[string]string{
	"cn-bj2":  "华北二（北京）",
	"cn-sh2":  "华东二（上海）",
	"cn-gd":   "华南一（广州）",
	"hk":      "香港",
	"tw-tp":   "台北",
	"sg":      "亚太一（新加坡）",
	"us-ca":   "美国西（洛杉矶）",
	"us-ws":   "美国东（华盛顿）",
	"ge-fra":  "欧洲（法兰克福）",
}

var zoneNames = map[string]string{
	"hk-01": "香港可用区A",
	"hk-02": "香港可用区B",
	"sg-01": "亚太一（新加坡）可用区A",
	"sg-02": "亚太一（新加坡）可用区B",
}

func getRegionLocalName(regionId string) string {
	if name, ok := regionNames[regionId]; ok {
		return name
	}
	return regionId
}

func getZoneLocalName(zoneId string) string {
	if name, ok := zoneNames[zoneId]; ok {
		return name
	}
	return zoneId
}