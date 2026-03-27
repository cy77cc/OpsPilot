package v1

type AssignRoleBody struct {
	RoleIds []uint64 `json:"roleIds"` // 角色ID列表
}

type CheckPermissionReq struct {
	Resource string `json:"resource"` // 资源
	Action   string `json:"action"`   // 动作
}

type CheckPermissionResp struct {
	Allowed bool `json:"allowed"` // 是否允许
}

type GetHealthResp struct {
	Status string `json:"status"` // 状态
}

type GrantPermissionBody struct {
	PermissionIds []uint64 `json:"permissionIds"` // 权限ID列表
}

type LoginReq struct {
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码
}

type LogoutReq struct {
	RefreshToken string `json:"refreshToken"` // 刷新令牌
}

type PermissionCreateReq struct {
	Name        string `json:"name"`        // 权限名称
	Code        string `json:"code"`        // 权限代码
	Type        int32  `json:"type"`        // 类型：1菜单，2按钮，3API
	Resource    string `json:"resource"`    // 资源路径
	Action      string `json:"action"`      // 动作方法 (GET, POST...)
	Description string `json:"description"` // 描述
}

type PermissionListReq struct {
	Page     int32 `form:"page"`     // 页码
	PageSize int32 `form:"pageSize"` // 每页数量
}

type PermissionListResp struct {
	Total int64            `json:"total"` // 总数
	List  []PermissionResp `json:"list"`  // 权限列表
}

type PermissionResp struct {
	Id          uint64 `json:"id"`          // 权限ID
	Name        string `json:"name"`        // 权限名称
	Code        string `json:"code"`        // 权限代码
	Type        int32  `json:"type"`        // 类型
	Resource    string `json:"resource"`    // 资源路径
	Action      string `json:"action"`      // 动作方法
	Description string `json:"description"` // 描述
	Status      int32  `json:"status"`      // 状态
	CreateTime  int64  `json:"createTime"`  // 创建时间
	UpdateTime  int64  `json:"updateTime"`  // 更新时间
}

type PermissionUpdateReq struct {
	Id          uint64 `path:"id"`          // 权限ID
	Name        string `json:"name"`        // 权限名称
	Description string `json:"description"` // 描述
	Status      int32  `json:"status"`      // 状态
	Resource    string `json:"resource"`    // 资源路径
	Action      string `json:"action"`      // 动作方法
	Type        int32  `json:"type"`        // 类型
}

type RefreshReq struct {
	RefreshToken string `json:"refreshToken"` // 刷新令牌
}

type RevokePermissionBody struct {
	PermissionIds []uint64 `json:"permissionIds"` // 权限ID列表
}

type RevokeRoleBody struct {
	RoleIds []uint64 `json:"roleIds"` // 角色ID列表
}

type RoleCreateReq struct {
	Name        string `json:"name"`        // 角色名称
	Code        string `json:"code"`        // 角色代码
	Description string `json:"description"` // 描述
}

type RoleListReq struct {
	Page     int32 `form:"page"`     // 页码
	PageSize int32 `form:"pageSize"` // 每页数量
}

type RoleListResp struct {
	Total int64      `json:"total"` // 总数
	List  []RoleResp `json:"list"`  // 角色列表
}

type RoleResp struct {
	Id          uint64 `json:"id"`          // 角色ID
	Name        string `json:"name"`        // 角色名称
	Code        string `json:"code"`        // 角色代码
	Description string `json:"description"` // 描述
	Status      int32  `json:"status"`      // 状态
	CreateTime  int64  `json:"createTime"`  // 创建时间
	UpdateTime  int64  `json:"updateTime"`  // 更新时间
}

type RoleUpdateReq struct {
	Id          uint64 `path:"id"`          // 角色ID
	Name        string `json:"name"`        // 角色名称
	Description string `json:"description"` // 描述
	Status      int32  `json:"status"`      // 状态
}

type TokenResp struct {
	AccessToken  string    `json:"accessToken"`  // 访问令牌
	RefreshToken string    `json:"refreshToken"` // 刷新令牌
	Expires      int64     `json:"expires"`      // 过期时间（秒）
	Uid          uint64    `json:"uid"`          // 用户ID
	Roles        []string  `json:"roles"`        // 用户角色列表
	User         *AuthUser `json:"user,omitempty"`
	Permissions  []string  `json:"permissions,omitempty"`
}

type AuthUser struct {
	Id          uint64   `json:"id"`
	Username    string   `json:"username"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type UserCreateReq struct {
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码
	Email    string `json:"email"`    // 邮箱
	Phone    string `json:"phone"`    // 手机号
	Avatar   string `json:"avatar"`   // 头像地址
}

type UserListReq struct {
	Page     int32  `form:"page"`     // 页码
	PageSize int32  `form:"pageSize"` // 每页数量
	Query    string `form:"form"`     // 搜索关键词
}

type UserListResp struct {
	Total int64      `json:"total"` // 总数
	List  []UserResp `json:"list"`  // 用户列表
}

type UserResp struct {
	Id            uint64 `json:"id"`            // 用户ID
	Username      string `json:"username"`      // 用户名
	Email         string `json:"email"`         // 邮箱
	Phone         string `json:"phone"`         // 手机号
	Avatar        string `json:"avatar"`        // 头像地址
	Status        int32  `json:"status"`        // 状态
	CreateTime    int64  `json:"createTime"`    // 创建时间
	UpdateTime    int64  `json:"updateTime"`    // 更新时间
	LastLoginTime int64  `json:"lastLoginTime"` // 最后登录时间
}

type UserUpdateReq struct {
	Id     uint64 `path:"id"`     // 用户ID
	Email  string `json:"email"`  // 邮箱
	Phone  string `json:"phone"`  // 手机号
	Avatar string `json:"avatar"` // 头像地址
	Status int32  `json:"status"` // 状态：1正常，0禁用
}
