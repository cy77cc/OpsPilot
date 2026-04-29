package logic

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"

	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	pb "github.com/cy77cc/OpsPilot/proto"
)

func NormalizeMetricBatch(hostID uint64, batch *pb.MetricBatch) *hostmodel.HostHealthSnapshot {
	snapshot := &hostmodel.HostHealthSnapshot{
		HostID:             hostID,
		State:              "healthy",
		ConnectivityStatus: "healthy",
		ResourceStatus:     "healthy",
		SystemStatus:       "healthy",
		CheckedAt:          time.Now(),
	}
	if batch == nil || len(batch.GetMetrics()) == 0 {
		return snapshot
	}

	summary := map[string]any{
		"metric_count": len(batch.GetMetrics()),
	}

	for _, metric := range batch.GetMetrics() {
		if metric == nil {
			continue
		}
		if ts := metric.GetTimestampMs(); ts > 0 {
			checkedAt := time.UnixMilli(ts)
			if checkedAt.After(snapshot.CheckedAt) {
				snapshot.CheckedAt = checkedAt
			}
		}
		fields := fieldMap(metric.GetFields())
		switch metric.GetName() {
		case "cpu":
			if usagePercent, ok := floatField(fields, "usage_percent"); ok {
				summary["cpu_usage_percent"] = usagePercent
				snapshot.CpuLoad = usagePercent / 20.0
			}
		case "memory":
			if totalBytes, ok := intField(fields, "total_bytes"); ok {
				snapshot.MemoryTotalMB = int(totalBytes / (1024 * 1024))
			}
			if usedBytes, ok := intField(fields, "used_bytes"); ok {
				snapshot.MemoryUsedMB = int(usedBytes / (1024 * 1024))
			}
			if usedPercent, ok := floatField(fields, "used_percent"); ok {
				summary["memory_used_percent"] = usedPercent
				if snapshot.MemoryTotalMB > 0 && snapshot.MemoryUsedMB == 0 {
					snapshot.MemoryUsedMB = int(math.Round(float64(snapshot.MemoryTotalMB) * usedPercent / 100.0))
				}
			}
		case "disk":
			if usedPercent, ok := floatField(fields, "used_percent"); ok {
				snapshot.DiskUsedPct = usedPercent
				summary["disk_used_percent"] = usedPercent
			}
			if inodeUsedPercent, ok := floatField(fields, "inode_used_percent"); ok {
				snapshot.InodeUsedPct = inodeUsedPercent
				summary["inode_used_percent"] = inodeUsedPercent
			}
			if diskIO, ok := floatField(fields, "io_iops"); ok {
				summary["disk_io_iops"] = diskIO
			} else if diskIO, ok := floatField(fields, "iops"); ok {
				summary["disk_io_iops"] = diskIO
			}
		case "net":
			if rxBytes, ok := intField(fields, "rx_bytes"); ok {
				summary["net_rx_bytes"] = rxBytes
			}
			if txBytes, ok := intField(fields, "tx_bytes"); ok {
				summary["net_tx_bytes"] = txBytes
			}
		case "process":
			if count, ok := intField(fields, "process_count"); ok {
				summary["process_count"] = count
			}
		}
	}

	raw, _ := json.Marshal(summary)
	snapshot.SummaryJSON = string(raw)
	return snapshot
}

func (s *Server) persistMetricBatch(ctx context.Context, instance *hostpluginmodel.HostPluginInstance, batch *pb.MetricBatch) error {
	if instance == nil {
		return nil
	}
	snapshot := NormalizeMetricBatch(instance.HostID, batch)
	db := s.db()
	if db == nil {
		return nil
	}
	if err := db.WithContext(ctx).Create(snapshot).Error; err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&hostmodel.Node{}).
		Where("id = ?", instance.HostID).
		Updates(map[string]any{
			"health_state":  snapshot.State,
			"last_check_at": snapshot.CheckedAt,
		}).Error
}

func fieldMap(fields []*pb.Field) map[string]*pb.Field {
	out := make(map[string]*pb.Field, len(fields))
	for _, field := range fields {
		if field == nil {
			continue
		}
		out[field.GetKey()] = field
	}
	return out
}

func floatField(fields map[string]*pb.Field, key string) (float64, bool) {
	field, ok := fields[key]
	if !ok || field == nil {
		return 0, false
	}
	switch value := field.GetValue().(type) {
	case *pb.Field_DoubleValue:
		return value.DoubleValue, true
	case *pb.Field_IntValue:
		return float64(value.IntValue), true
	default:
		return 0, false
	}
}

func intField(fields map[string]*pb.Field, key string) (int64, bool) {
	field, ok := fields[key]
	if !ok || field == nil {
		return 0, false
	}
	switch value := field.GetValue().(type) {
	case *pb.Field_IntValue:
		return value.IntValue, true
	case *pb.Field_DoubleValue:
		return int64(value.DoubleValue), true
	default:
		return 0, false
	}
}

func parseRevisionVersion(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}
