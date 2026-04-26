package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "github.com/cy77cc/OpsPilot/api/host/v1"
	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	monitormodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"github.com/gin-gonic/gin"
)

// ListProcesses 获取主机进程列表。
func (h *Handler) ListProcesses(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	node, err := h.hostService.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return
	}
	privateKey, passphrase, err := h.loadNodePrivateKey(c, node)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, fmt.Errorf("failed to load SSH key: %w", err).Error())
		return
	}
	password := strings.TrimSpace(h.hostService.ResolveNodeSSHPassword(node))
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		if writeHostKeyPayloadIfNeeded(c, err) {
			return
		}
		httpx.OK(c, gin.H{"reachable": false, "message": err.Error()})
		return
	}
	defer cli.Close()

	out, err := sshclient.RunCommand(cli, "ps aux --no-headers")
	if err != nil {
		httpx.Fail(c, xcode.ServerError, fmt.Sprintf("failed to list processes: %s", out))
		return
	}

	processes := parsePsOutput(out)
	httpx.OK(c, processes)
}

// parsePsOutput parses `ps aux --no-headers` output into ProcessItem slices.
func parsePsOutput(out string) []v1.ProcessItem {
	if out == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	processes := make([]v1.ProcessItem, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		pid, _ := strconv.Atoi(fields[1])
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		vsz, _ := strconv.ParseUint(fields[4], 10, 64)
		rss, _ := strconv.ParseUint(fields[5], 10, 64)
		command := strings.Join(fields[10:], " ")
		processes = append(processes, v1.ProcessItem{
			PID:     pid,
			User:    fields[0],
			CPU:     cpu,
			Memory:  mem,
			VSZ:     vsz,
			RSS:     rss,
			State:   fields[7],
			Start:   fields[8],
			Time:    fields[9],
			Command: command,
		})
	}
	return processes
}

// KillProcess 终止主机进程。
func (h *Handler) KillProcess(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:execute", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	pidStr := c.Param("pid")
	pid, err := strconv.ParseUint(pidStr, 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid pid")
		return
	}

	// Optional signal query param, default to 15 (SIGTERM)
	signal := c.DefaultQuery("signal", "15")

	node, err := h.hostService.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return
	}
	privateKey, passphrase, err := h.loadNodePrivateKey(c, node)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, fmt.Errorf("failed to load SSH key: %w", err).Error())
		return
	}
	password := strings.TrimSpace(h.hostService.ResolveNodeSSHPassword(node))
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		if writeHostKeyPayloadIfNeeded(c, err) {
			return
		}
		httpx.OK(c, gin.H{"reachable": false, "message": err.Error()})
		return
	}
	defer cli.Close()

	cmd := fmt.Sprintf("kill -%s %d", signal, pid)
	out, err := sshclient.RunCommand(cli, cmd)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, fmt.Sprintf("kill failed: %s", out))
		return
	}
	httpx.OK(c, gin.H{"message": fmt.Sprintf("process %d terminated with signal %s", pid, signal), "output": out})
}

// ListServices 获取主机服务列表。
func (h *Handler) ListServices(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	out, err := h.runSSHCommandOnHost(c, id, "systemctl list-units --type=service --all --no-pager --no-legend")
	if err != nil {
		return // error already written
	}
	httpx.OK(c, parseSystemctlOutput(out))
}

// parseSystemctlOutput parses `systemctl list-units --type=service --all --no-pager --no-legend` output.
func parseSystemctlOutput(out string) []v1.ServiceItem {
	if out == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	services := make([]v1.ServiceItem, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		description := ""
		if len(fields) > 4 {
			description = strings.Join(fields[4:], " ")
		}
		services = append(services, v1.ServiceItem{
			Name:        fields[0],
			Status:      fields[2],
			Startup:     fields[1],
			Description: description,
		})
	}
	return services
}

// ServiceAction 执行服务操作 (start/stop/restart/reload/status).
func (h *Handler) ServiceAction(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:execute", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	name := c.Param("name")
	var req struct {
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	out, err := h.runSSHCommandOnHost(c, id, fmt.Sprintf("systemctl %s %s", req.Action, name))
	if err != nil {
		return // error already written
	}
	httpx.OK(c, gin.H{"message": fmt.Sprintf("service %s %s completed", name, req.Action), "output": out})
}

// ListDisks 获取主机磁盘分区信息。
func (h *Handler) ListDisks(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	out, err := h.runSSHCommandOnHost(c, id, "df -BG --output=source,target,fstype,size,used,avail,pcent --total")
	if err != nil {
		return // error already written
	}
	httpx.OK(c, parseDfOutput(out))
}

// parseDfOutput parses `df -BG --output=source,target,fstype,size,used,avail,pcent --total` output.
func parseDfOutput(out string) []v1.PartitionItem {
	if out == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	disks := make([]v1.PartitionItem, 0, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || i == 0 || strings.HasPrefix(line, "total") {
			continue // skip header and total
		}
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		usagePct := 0.0
		pctStr := strings.TrimSuffix(fields[6], "%")
		if val, err := strconv.ParseFloat(pctStr, 64); err == nil {
			usagePct = val
		}
		disks = append(disks, v1.PartitionItem{
			Filesystem: fields[0],
			Type:       fields[2],
			Size:       strings.TrimSuffix(fields[3], "G") + "GB",
			Used:       strings.TrimSuffix(fields[4], "G") + "GB",
			Available:  strings.TrimSuffix(fields[5], "G") + "GB",
			UsagePct:   usagePct,
			Mounted:    fields[1],
		})
	}
	return disks
}

// ListNetworkInterfaces 获取主机网络接口信息。
func (h *Handler) ListNetworkInterfaces(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}

	// Collect IPv4 addresses
	ipOut, err := h.runSSHCommandOnHost(c, id, "ip -o addr show inet")
	if err != nil {
		return
	}
	ipMap := map[string]string{}
	for _, line := range strings.Split(ipOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			ifaceName := fields[1]
			cidr := fields[3]
			ip := strings.Split(cidr, "/")[0]
			if existing, ok := ipMap[ifaceName]; ok {
				ipMap[ifaceName] = existing + ", " + ip
			} else {
				ipMap[ifaceName] = ip
			}
		}
	}

	// Collect link info (MAC, state, MTU)
	linkOut, err := h.runSSHCommandOnHost(c, id, "ip link show")
	if err != nil {
		return
	}

	// Collect RX/TX bytes from /sys/class/net
	rxTxMap := map[string][2]string{}
	for ifaceName := range ipMap {
		rxOut, _ := h.runSSHCommandOnHost(c, id, fmt.Sprintf("cat /sys/class/net/%s/statistics/rx_bytes 2>/dev/null || echo 0", ifaceName))
		txOut, _ := h.runSSHCommandOnHost(c, id, fmt.Sprintf("cat /sys/class/net/%s/statistics/tx_bytes 2>/dev/null || echo 0", ifaceName))
		rxBytes, _ := strconv.ParseUint(strings.TrimSpace(rxOut), 10, 64)
		txBytes, _ := strconv.ParseUint(strings.TrimSpace(txOut), 10, 64)
		rxTxMap[ifaceName] = [2]string{formatBytes(rxBytes), formatBytes(txBytes)}
	}

	interfaces := parseIpLinkOutput(linkOut, ipMap, rxTxMap)
	httpx.OK(c, interfaces)
}

// parseIpLinkOutput parses `ip link show` output into InterfaceItem slices.
func parseIpLinkOutput(out string, ipMap map[string]string, rxTxMap map[string][2]string) []v1.InterfaceItem {
	if out == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	interfaces := make([]v1.InterfaceItem, 0)

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		// Interface header line: "N: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 ..."
		parts := strings.SplitN(trimmed, ":", 3)
		if len(parts) < 2 {
			continue
		}
		namePart := strings.TrimSpace(parts[1])
		nameFields := strings.Fields(namePart)
		if len(nameFields) == 0 {
			continue
		}
		name := strings.TrimSuffix(nameFields[0], ":")

		// Skip loopback
		if name == "lo" {
			continue
		}

		// Extract MTU
		mtu := 1500
		if len(parts) >= 3 {
			allFields := strings.Fields(parts[2])
			for k, ff := range allFields {
				if ff == "mtu" && k+1 < len(allFields) {
					if val, err := strconv.Atoi(allFields[k+1]); err == nil {
						mtu = val
					}
					break
				}
			}
		}

		// Extract state from angle brackets
		status := "down"
		if strings.Contains(trimmed, "UP") && !strings.Contains(trimmed, "DOWN") {
			status = "up"
		} else if strings.Contains(trimmed, "LOWER_UP") {
			status = "up"
		}

		// Extract MAC from next line if available
		mac := ""
		if i+1 < len(lines) {
			nextFields := strings.Fields(lines[i+1])
			for j, f := range nextFields {
				if f == "link/ether" && j+1 < len(nextFields) {
					mac = nextFields[j+1]
					break
				}
				_ = j
			}
		}

		rx := "0 B"
		tx := "0 B"
		if stats, ok := rxTxMap[name]; ok {
			rx = stats[0]
			tx = stats[1]
		}

		interfaces = append(interfaces, v1.InterfaceItem{
			Name:   name,
			IP:     ipMap[name],
			MAC:    mac,
			Status: status,
			RX:     rx,
			TX:     tx,
			MTU:    mtu,
		})
	}
	return interfaces
}

// formatBytes converts bytes to human-readable string.
func formatBytes(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

// ListNetworkRoutes 获取主机路由表信息。
func (h *Handler) ListNetworkRoutes(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	out, err := h.runSSHCommandOnHost(c, id, "ip route show")
	if err != nil {
		return // error already written
	}
	httpx.OK(c, parseRouteOutput(out))
}

// parseRouteOutput parses `ip route show` output into RouteItem slices.
func parseRouteOutput(out string) []v1.RouteItem {
	if out == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	routes := make([]v1.RouteItem, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		item := v1.RouteItem{
			Gateway:     "0.0.0.0",
			Mask:        "255.255.255.255",
			Flags:       "U",
			Metric:      0,
		}
		fields := strings.Fields(line)

		if fields[0] == "default" {
			// default via 192.168.1.1 dev eth0
			item.Destination = "0.0.0.0"
			item.Mask = "0.0.0.0"
			item.Flags = "UG"
			for i, f := range fields {
				if f == "via" && i+1 < len(fields) {
					item.Gateway = fields[i+1]
				}
				if f == "dev" && i+1 < len(fields) {
					item.Interface = fields[i+1]
				}
				if f == "metric" && i+1 < len(fields) {
					if val, err := strconv.Atoi(fields[i+1]); err == nil {
						item.Metric = val
					}
				}
			}
		} else {
			// 192.168.1.0/24 dev eth0 or 10.0.0.0/8 via 10.0.0.1 dev eth0
			destination := fields[0]
			parts := strings.Split(destination, "/")
			item.Destination = parts[0]
			if len(parts) == 2 {
				item.Mask = cidrToMask(parts[1])
			} else {
				item.Mask = "255.255.255.255"
			}
			for i, f := range fields {
				if f == "via" && i+1 < len(fields) {
					item.Gateway = fields[i+1]
					if item.Flags == "U" {
						item.Flags = "UG"
					}
				}
				if f == "dev" && i+1 < len(fields) {
					item.Interface = fields[i+1]
				}
			}
		}
		routes = append(routes, item)
	}
	return routes
}

// cidrToMask converts a CIDR prefix length to dotted decimal notation.
func cidrToMask(bits string) string {
	prefix, err := strconv.Atoi(bits)
	if err != nil || prefix < 0 || prefix > 32 {
		return "255.255.255.255"
	}
	mask := uint32(0xFFFFFFFF) << (32 - prefix)
	return fmt.Sprintf("%d.%d.%d.%d",
		(mask>>24)&0xFF, (mask>>16)&0xFF, (mask>>8)&0xFF, mask&0xFF)
}

// ListPackages 获取主机已安装软件包。
func (h *Handler) ListPackages(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	// TODO: SSH dpkg -l / rpm -qa
	mockData := []v1.PackageItem{
		{Name: "nginx", Version: "1.18.0-6ubuntu1.4", Arch: "amd64", Status: "installed", Description: "high performance web server"},
		{Name: "openssl", Version: "1.1.1f-1ubuntu2.19", Arch: "amd64", Status: "upgradable", Description: "Secure Sockets Layer toolkit"},
	}
	httpx.OK(c, mockData)
}

// ListAlarms 获取主机告警历史。
func (h *Handler) ListAlarms(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}

	node, err := h.hostService.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return
	}

	var alerts []monitormodel.AlertEvent
	// 根据 IP 过滤告警
	err = h.svcCtx.DB.WithContext(c.Request.Context()).
		Where("source LIKE ?", "%"+node.IP+"%").
		Order("triggered_at DESC").
		Limit(100).
		Find(&alerts).Error
	if err != nil {
		httpx.Fail(c, xcode.DatabaseError, err.Error())
		return
	}

	resp := make([]v1.AlarmHistoryItem, 0, len(alerts))
	for _, a := range alerts {
		duration := "进行中"
		if a.ResolvedAt != nil {
			duration = a.ResolvedAt.Sub(a.TriggeredAt).Round(time.Second).String()
		}
		resp = append(resp, v1.AlarmHistoryItem{
			ID:         fmt.Sprintf("%d", a.ID),
			Level:      a.Severity,
			Title:      a.Title,
			Status:     a.Status,
			StartedAt:  a.TriggeredAt,
			ResolvedAt: a.ResolvedAt,
			Duration:   duration,
			Value:      fmt.Sprintf("%.2f", a.Value),
		})
	}

	// 如果数据库没数据，提供一些 mock 展示
	if len(resp) == 0 {
		now := time.Now()
		resp = []v1.AlarmHistoryItem{
			{ID: "m1", Level: "warning", Title: "磁盘使用率过高", Status: "resolved", StartedAt: now.Add(-1 * time.Hour), Duration: "45m", Value: "85%"},
			{ID: "m2", Level: "critical", Title: "内存使用率超过 90%", Status: "resolved", StartedAt: now.Add(-2 * time.Hour), Duration: "18m", Value: "92%"},
		}
	}

	httpx.OK(c, resp)
}

// runSSHCommandOnHost establishes an SSH connection to a host and runs a command.
func (h *Handler) runSSHCommandOnHost(c *gin.Context, hostID uint64, cmd string) (string, error) {
	node, err := h.hostService.Get(c.Request.Context(), hostID)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return "", err
	}
	privateKey, passphrase, err := h.loadNodePrivateKey(c, node)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, fmt.Errorf("failed to load SSH key: %w", err).Error())
		return "", err
	}
	password := strings.TrimSpace(h.hostService.ResolveNodeSSHPassword(node))
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		if writeHostKeyPayloadIfNeeded(c, err) {
			return "", err
		}
		httpx.OK(c, gin.H{"reachable": false, "message": err.Error()})
		return "", err
	}
	defer cli.Close()

	out, err := sshclient.RunCommand(cli, cmd)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, fmt.Sprintf("command failed: %s", out))
		return "", err
	}
	return out, nil
}
