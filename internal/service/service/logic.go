package service

import "github.com/cy77cc/k8s-manage/internal/svc"

type Logic struct {
	svcCtx *svc.ServiceContext
}

func NewLogic(svcCtx *svc.ServiceContext) *Logic { return &Logic{svcCtx: svcCtx} }
