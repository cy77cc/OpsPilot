// Package ucloud 提供 UCLOUD UHost 实例查询适配器实现。
package ucloud

import "testing"

// TestGetRegionLocalName 测试地域中文名称获取。
func TestGetRegionLocalName(t *testing.T) {
	tests := []struct {
		name     string
		regionId string
		expected string
	}{
		{"华北二（北京）", "cn-bj2", "华北二（北京）"},
		{"华东二（上海）", "cn-sh2", "华东二（上海）"},
		{"华南一（广州）", "cn-gd", "华南一（广州）"},
		{"香港", "hk", "香港"},
		{"亚太一（新加坡）", "sg", "亚太一（新加坡）"},
		{"美国西（洛杉矶）", "us-ca", "美国西（洛杉矶）"},
		{"未知", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRegionLocalName(tt.regionId)
			if result != tt.expected {
				t.Errorf("getRegionLocalName(%q) = %q, want %q", tt.regionId, result, tt.expected)
			}
		})
	}
}

// TestGetZoneLocalName 测试可用区中文名称获取。
func TestGetZoneLocalName(t *testing.T) {
	tests := []struct {
		name     string
		zoneId   string
		expected string
	}{
		{"华北二（北京）可用区A", "cn-bj2-01", "华北二（北京）可用区A"},
		{"华北二（北京）可用区B", "cn-bj2-02", "华北二（北京）可用区B"},
		{"华东二（上海）可用区A", "cn-sh2-01", "华东二（上海）可用区A"},
		{"香港可用区A", "hk-01", "香港可用区A"},
		{"动态生成-华南一（广州）可用区B", "cn-gd-02", "华南一（广州）可用区B"},
		{"未知", "unknown-zone", "unknown-zone"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getZoneLocalName(tt.zoneId)
			if result != tt.expected {
				t.Errorf("getZoneLocalName(%q) = %q, want %q", tt.zoneId, result, tt.expected)
			}
		})
	}
}
