package handler

import (
	"errors"
	"strings"

	v1 "github.com/cy77cc/k8s-manage/api/node/v1"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	hostlogic "github.com/cy77cc/k8s-manage/internal/service/host/logic"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

type NodeHandler struct {
	svcCtx *svc.ServiceContext
}

func NewNodeHandler(svcCtx *svc.ServiceContext) *NodeHandler {
	return &NodeHandler{
		svcCtx: svcCtx,
	}
}

func (n *NodeHandler) Add(c *gin.Context) {
	var req v1.CreateNodeReq
	if err := c.BindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	createReq := hostlogic.CreateReq{
		Name:        req.Name,
		IP:          req.IP,
		Port:        req.Port,
		Username:    req.SSHUser,
		Password:    req.SSHPassword,
		Description: req.Description,
		Role:        req.Role,
		ClusterID:   uint(req.ClusterID),
		Force:       true,
		Status:      "offline",
	}
	if req.SSHKeyID != 0 {
		keyID := uint64(req.SSHKeyID)
		createReq.SSHKeyID = &keyID
	}
	hostSvc := hostlogic.NewHostService(n.svcCtx)
	probeReq := hostlogic.ProbeReq{
		Name:     req.Name,
		IP:       req.IP,
		Port:     req.Port,
		AuthType: "password",
		Username: req.SSHUser,
		Password: req.SSHPassword,
	}
	if createReq.SSHKeyID != nil {
		probeReq.AuthType = "key"
		probeReq.SSHKeyID = createReq.SSHKeyID
	}
	probeResp, err := hostSvc.Probe(c.Request.Context(), 0, probeReq)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	if probeResp == nil || !probeResp.Reachable {
		msg := "ssh probe failed"
		if probeResp != nil && strings.TrimSpace(probeResp.Message) != "" {
			msg = probeResp.Message
		}
		httpx.Fail(c, xcode.ServerError, msg)
		return
	}
	createReq.ProbeToken = probeResp.ProbeToken
	node, err := hostSvc.CreateWithProbe(c.Request.Context(), 0, true, createReq)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	resp := v1.NodeResp{
		ID:          node.ID,
		Name:        node.Name,
		Hostname:    node.Hostname,
		Description: node.Description,
		IP:          node.IP,
		Port:        node.Port,
		SSHUser:     node.SSHUser,
		OS:          node.OS,
		Arch:        node.Arch,
		Kernel:      node.Kernel,
		CPUCores:    node.CpuCores,
		MemoryMB:    node.MemoryMB,
		DiskGB:      node.DiskGB,
		Status:      node.Status,
		Role:        node.Role,
		ClusterID:   int64(node.ClusterID),
		Labels:      node.Labels,
		LastCheckAt: node.LastCheckAt,
		CreatedAt:   node.CreatedAt,
		UpdatedAt:   node.UpdatedAt,
	}
	httpx.OK(c, resp)
}

func (n *NodeHandler) Get(c *gin.Context) {
	httpx.Fail(c, xcode.ServerError, errors.New("not implemented").Error())
}
