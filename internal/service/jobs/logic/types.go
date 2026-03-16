package logic

type CreateJobReq struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type"`
	Command     string `json:"command"`
	HostIDs     string `json:"host_ids"`
	Cron        string `json:"cron"`
	Timeout     int    `json:"timeout"`
	Priority    int    `json:"priority"`
	Description string `json:"description"`
}

type UpdateJobReq struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Command     string `json:"command"`
	HostIDs     string `json:"host_ids"`
	Cron        string `json:"cron"`
	Status      string `json:"status"`
	Timeout     int    `json:"timeout"`
	Priority    int    `json:"priority"`
	Description string `json:"description"`
}

type ListJobsReq struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}
