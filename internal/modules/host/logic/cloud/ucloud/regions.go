// Package ucloud 提供 UCLOUD UHost 实例查询适配器实现。
package ucloud

// regionNames UCLOUD 地域中文名称映射。
//
// 数据来源: https://docs.ucloud.cn/api/summary/regionlist
var regionNames = map[string]string{
	// 国内地域
	"cn-bj2":  "华北二（北京）",
	"cn-sh2":  "华东二（上海）",
	"cn-gd":   "华南一（广州）",
	"cn-nj":   "华东一（南京）",
	"cn-wlcb": "华北一（乌兰察布）",
	"cn-tj":   "华北三（天津）",
	"cn-qz":   "华南二（泉州）",
	"cn-fsd":  "华南三（佛山）",

	// 港澳台
	"hk":    "香港",
	"tw-tp": "台北",
	"macau": "澳门",

	// 亚太地区
	"sg":          "亚太一（新加坡）",
	"jp-tky":      "日本（东京）",
	"kr-seoul":    "韩国（首尔）",
	"th-bkk":      "泰国（曼谷）",
	"vn-sng":      "越南（胡志明）",
	"idn-jakarta": "印尼（雅加达）",
	"ph-mnl":      "菲律宾（马尼拉）",
	"in-mum":      "印度（孟买）",
	"in-blr":      "印度（班加罗尔）",

	// 北美地区
	"us-ca":  "美国西（洛杉矶）",
	"us-ws":  "美国东（华盛顿）",
	"us-mia": "美国南（迈阿密）",

	// 欧洲地区
	"ge-fra": "欧洲（法兰克福）",
	"uk-lon": "英国（伦敦）",
	"ru-mos": "俄罗斯（莫斯科）",

	// 中东及非洲
	"dubai":    "中东（迪拜）",
	"ng-lagos": "非洲（拉各斯）",

	// 南美
	"bra-saopaulo": "巴西（圣保罗）",
}

// zoneNames UCLOUD 可用区中文名称映射。
//
// 可用区命名规则: {地域}-0{编号}
// 数据来源: https://docs.ucloud.cn/api/summary/regionlist
var zoneNames = map[string]string{
	// 华北二（北京）
	"cn-bj2-01": "华北二（北京）可用区A",
	"cn-bj2-02": "华北二（北京）可用区B",
	"cn-bj2-03": "华北二（北京）可用区C",
	"cn-bj2-04": "华北二（北京）可用区D",
	"cn-bj2-05": "华北二（北京）可用区E",

	// 华东二（上海）
	"cn-sh2-01": "华东二（上海）可用区A",
	"cn-sh2-02": "华东二（上海）可用区B",
	"cn-sh2-03": "华东二（上海）可用区C",

	// 华南一（广州）
	"cn-gd-01": "华南一（广州）可用区A",
	"cn-gd-02": "华南一（广州）可用区B",
	"cn-gd-03": "华南一（广州）可用区C",

	// 香港
	"hk-01": "香港可用区A",
	"hk-02": "香港可用区B",

	// 亚太一（新加坡）
	"sg-01": "亚太一（新加坡）可用区A",
	"sg-02": "亚太一（新加坡）可用区B",

	// 美国西（洛杉矶）
	"us-ca-01": "美国西（洛杉矶）可用区A",
	"us-ca-02": "美国西（洛杉矶）可用区B",

	// 美国东（华盛顿）
	"us-ws-01": "美国东（华盛顿）可用区A",

	// 欧洲（法兰克福）
	"ge-fra-01": "欧洲（法兰克福）可用区A",

	// 日本（东京）
	"jp-tky-01": "日本（东京）可用区A",

	// 韩国（首尔）
	"kr-seoul-01": "韩国（首尔）可用区A",

	// 台北
	"tw-tp-01": "台北可用区A",

	// 泰国（曼谷）
	"th-bkk-01": "泰国（曼谷）可用区A",

	// 印尼（雅加达）
	"idn-jakarta-01": "印尼（雅加达）可用区A",

	// 中东（迪拜）
	"dubai-01": "中东（迪拜）可用区A",
}

// getRegionLocalName 获取地域中文名称。
//
// 参数:
//   - regionId: 地域标识（如 "cn-bj2"）
//
// 返回中文名称，未找到时返回原地域标识。
func getRegionLocalName(regionId string) string {
	if name, ok := regionNames[regionId]; ok {
		return name
	}
	return regionId
}

// getZoneLocalName 获取可用区中文名称。
//
// 参数:
//   - zoneId: 可用区标识（如 "cn-bj2-01"）
//
// 返回中文名称，未找到时尝试根据命名规则生成或返回原标识。
func getZoneLocalName(zoneId string) string {
	if name, ok := zoneNames[zoneId]; ok {
		return name
	}

	// 尝试根据命名规则生成名称: {地域}-0{编号} -> {地域中文}可用区{字母}
	// 例如: cn-bj2-03 -> 华北二（北京）可用区C
	if len(zoneId) >= 3 && zoneId[len(zoneId)-2] == '0' {
		regionId := zoneId[:len(zoneId)-3]
		zoneNum := zoneId[len(zoneId)-1]

		if regionName := getRegionLocalName(regionId); regionName != regionId {
			// 编号转字母: 1->A, 2->B, 3->C...
			if zoneNum >= '1' && zoneNum <= '9' {
				letter := rune('A' + (zoneNum - '1'))
				return regionName + "可用区" + string(letter)
			}
		}
	}

	return zoneId
}
