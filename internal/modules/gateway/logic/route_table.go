package logic

import (
	"fmt"
	"sync"

	gatewaymodel "github.com/cy77cc/OpsPilot/internal/modules/gateway/model"
	"gorm.io/gorm"
)

// RouteTable 管理主机路由，内存缓存 + DB 持久化。
type RouteTable struct {
	cache sync.Map // hostID(uint64) -> *gatewaymodel.HostRoute
	db    *gorm.DB
}

// NewRouteTable 创建路由表实例。
func NewRouteTable(db *gorm.DB) *RouteTable {
	return &RouteTable{db: db}
}

// LoadFromDB 从数据库加载所有路由到内存缓存。
func (rt *RouteTable) LoadFromDB() error {
	if rt.db == nil {
		return fmt.Errorf("db is nil")
	}
	var routes []gatewaymodel.HostRoute
	if err := rt.db.Find(&routes).Error; err != nil {
		return fmt.Errorf("load host routes: %w", err)
	}
	for i := range routes {
		rt.cache.Store(routes[i].HostID, &routes[i])
	}
	return nil
}

// Get 从缓存获取路由，返回 nil 表示未找到。
func (rt *RouteTable) Get(hostID uint64) *gatewaymodel.HostRoute {
	val, ok := rt.cache.Load(hostID)
	if !ok {
		return nil
	}
	route, _ := val.(*gatewaymodel.HostRoute)
	return route
}

// Set 写入路由到 DB 并更新缓存。
func (rt *RouteTable) Set(route gatewaymodel.HostRoute) error {
	if rt.db == nil {
		return fmt.Errorf("db is nil")
	}
	err := rt.db.Where("host_id = ?", route.HostID).
		Assign(route).
		FirstOrCreate(&route).Error
	if err != nil {
		return fmt.Errorf("upsert host route: %w", err)
	}
	rt.cache.Store(route.HostID, &route)
	return nil
}

// Delete 从 DB 和缓存中删除路由。
func (rt *RouteTable) Delete(hostID uint64) error {
	if rt.db != nil {
		if err := rt.db.Where("host_id = ?", hostID).Delete(&gatewaymodel.HostRoute{}).Error; err != nil {
			return fmt.Errorf("delete host route: %w", err)
		}
	}
	rt.cache.Delete(hostID)
	return nil
}

// UpdateTunnelID 更新指定主机的隧道 ID。
func (rt *RouteTable) UpdateTunnelID(hostID uint64, tunnelID string) error {
	if rt.db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := rt.db.Model(&gatewaymodel.HostRoute{}).
		Where("host_id = ?", hostID).
		Update("tunnel_id", tunnelID).Error; err != nil {
		return fmt.Errorf("update tunnel id: %w", err)
	}
	if val, ok := rt.cache.Load(hostID); ok {
		route := val.(*gatewaymodel.HostRoute)
		route.TunnelID = tunnelID
	}
	return nil
}
