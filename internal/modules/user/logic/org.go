package logic

import (
	"context"

	v1 "github.com/cy77cc/OpsPilot/api/user/v1"
	dao "github.com/cy77cc/OpsPilot/internal/modules/user/dao"
	"github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

type OrgLogic struct {
	svcCtx  *svc.ServiceContext
	deptDAO *dao.DepartmentDAO
}

func NewOrgLogic(svcCtx *svc.ServiceContext) *OrgLogic {
	return &OrgLogic{
		svcCtx:  svcCtx,
		deptDAO: dao.NewDepartmentDAO(svcCtx.DB, svcCtx.Cache, svcCtx.Rdb),
	}
}

func (l *OrgLogic) GetDepartmentTree(ctx context.Context) ([]v1.DepartmentResp, error) {
	depts, err := l.deptDAO.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	return buildDeptTree(depts, 0), nil
}

func (l *OrgLogic) CreateDepartment(ctx context.Context, req v1.DepartmentCreateReq) error {
	dept := &model.Department{
		Name:     req.Name,
		ParentID: req.ParentId,
		LeaderID: req.LeaderId,
		Status:   1,
	}
	return l.deptDAO.Create(ctx, dept)
}

func (l *OrgLogic) UpdateDepartment(ctx context.Context, id int64, req v1.DepartmentUpdateReq) error {
	dept, err := l.deptDAO.FindByID(ctx, model.UserID(id))
	if err != nil {
		return err
	}
	if req.Name != "" {
		dept.Name = req.Name
	}
	dept.ParentID = req.ParentId
	dept.LeaderID = req.LeaderId
	dept.Status = req.Status
	return l.deptDAO.Update(ctx, dept)
}

func (l *OrgLogic) DeleteDepartment(ctx context.Context, id int64) error {
	return l.deptDAO.Delete(ctx, model.UserID(id))
}

func (l *OrgLogic) TransferMember(ctx context.Context, req v1.MemberTransferReq) error {
	// Implement member transfer logic here.
	// We can use a transaction to ensure atomicity.
	return l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Delete old association if exists
		if req.OldDeptId > 0 {
			if err := tx.Where("user_id = ? AND dept_id = ?", req.UserId, req.OldDeptId).Delete(&model.DepartmentMember{}).Error; err != nil {
				return err
			}
		} else {
			// If OldDeptId is not provided, we might want to delete ALL old associations for this user
			// depending on the business requirement. For now, let's just delete ALL.
			if err := tx.Where("user_id = ?", req.UserId).Delete(&model.DepartmentMember{}).Error; err != nil {
				return err
			}
		}

		// 2. Add new association
		member := &model.DepartmentMember{
			UserID: req.UserId,
			DeptID: req.NewDeptId,
		}
		return tx.Create(member).Error
	})
}

func (l *OrgLogic) GetDepartmentMembers(ctx context.Context, deptId int64) ([]v1.UserResp, error) {
	var users []model.User
	err := l.svcCtx.DB.WithContext(ctx).
		Table("users").
		Select("users.*").
		Joins("JOIN department_members dm ON users.id = dm.user_id").
		Where("dm.dept_id = ?", deptId).
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	var resp []v1.UserResp
	for _, u := range users {
		resp = append(resp, v1.UserResp{
			Id:            uint64(u.ID),
			Username:      u.Username,
			Email:         u.Email,
			Phone:         u.Phone,
			Avatar:        u.Avatar,
			Status:        int32(u.Status),
			CreateTime:    u.CreateTime,
			UpdateTime:    u.UpdateTime,
			LastLoginTime: u.LastLoginTime,
		})
	}
	return resp, nil
}

func (l *OrgLogic) GetDepartmentRoles(ctx context.Context, deptId int64) ([]uint64, error) {
	var roleIds []uint64
	err := l.svcCtx.DB.WithContext(ctx).
		Model(&model.DepartmentRole{}).
		Where("dept_id = ?", deptId).
		Pluck("role_id", &roleIds).Error
	return roleIds, err
}

func (l *OrgLogic) UpdateDepartmentRoles(ctx context.Context, deptId int64, req v1.DepartmentRolesReq) error {
	return l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Clear existing roles
		if err := tx.Where("dept_id = ?", deptId).Delete(&model.DepartmentRole{}).Error; err != nil {
			return err
		}

		// 2. Add new roles
		for _, rid := range req.RoleIds {
			dr := &model.DepartmentRole{
				DeptID: deptId,
				RoleID: int64(rid),
			}
			if err := tx.Create(dr).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func buildDeptTree(depts []model.Department, parentId int64) []v1.DepartmentResp {
	var tree []v1.DepartmentResp
	for _, dept := range depts {
		if dept.ParentID == parentId {
			node := v1.DepartmentResp{
				Id:         int64(dept.ID),
				Name:       dept.Name,
				ParentId:   dept.ParentID,
				LeaderId:   dept.LeaderID,
				Status:     dept.Status,
				CreateTime: dept.CreateTime,
				UpdateTime: dept.UpdateTime,
				Children:   buildDeptTree(depts, int64(dept.ID)),
			}
			tree = append(tree, node)
		}
	}
	return tree
}
