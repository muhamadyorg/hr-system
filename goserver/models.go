package goserver

import "time"

type User struct {
	ID             int     `json:"id" db:"id"`
	Username       string  `json:"username" db:"username"`
	Password       string  `json:"-" db:"password"`
	FullName       string  `json:"fullName" db:"full_name"`
	Role           string  `json:"role" db:"role"`
	CreatedBy      *int    `json:"createdBy" db:"created_by"`
	IsActive       bool    `json:"isActive" db:"is_active"`
	TelegramUserID *string `json:"telegramUserId" db:"telegram_user_id"`
}

type UserWithPassword struct {
	User
	PlainPassword *string `json:"plainPassword,omitempty"`
}

type Group struct {
	ID              int     `json:"id" db:"id"`
	Name            string  `json:"name" db:"name"`
	Login           *string `json:"login" db:"login"`
	Password        *string `json:"password" db:"password"`
	Description     *string `json:"description" db:"description"`
	CreatedBy       int     `json:"createdBy" db:"created_by"`
	AssignedAdminID *int    `json:"assignedAdminId" db:"assigned_admin_id"`
	IsActive        bool    `json:"isActive" db:"is_active"`
}

type GroupWithCount struct {
	Group
	EmployeeCount     int     `json:"employeeCount"`
	AssignedAdminName *string `json:"assignedAdminName,omitempty"`
}

type Employee struct {
	ID              int     `json:"id" db:"id"`
	EmployeeNo      string  `json:"employeeNo" db:"employee_no"`
	FullName        string  `json:"fullName" db:"full_name"`
	Position        *string `json:"position" db:"position"`
	GroupID         *int    `json:"groupId" db:"group_id"`
	PhotoURL        *string `json:"photoUrl" db:"photo_url"`
	Phone           *string `json:"phone" db:"phone"`
	IsActive        bool    `json:"isActive" db:"is_active"`
	HikvisionSynced bool    `json:"hikvisionSynced" db:"hikvision_synced"`
	TelegramUserID  *string `json:"telegramUserId" db:"telegram_user_id"`
}

type EmployeeWithGroup struct {
	Employee
	GroupName *string `json:"groupName"`
}

type AttendanceRecord struct {
	ID         int       `json:"id" db:"id"`
	EmployeeID int       `json:"employeeId" db:"employee_id"`
	EmployeeNo string    `json:"employeeNo" db:"employee_no"`
	EventTime  time.Time `json:"eventTime" db:"event_time"`
	Status     string    `json:"status" db:"status"`
	PhotoURL   *string   `json:"photoUrl" db:"photo_url"`
	DeviceIP   *string   `json:"deviceIp" db:"device_ip"`
	DeviceName *string   `json:"deviceName" db:"device_name"`
}

type AttendanceWithEmployee struct {
	AttendanceRecord
	FullName  string  `json:"fullName"`
	GroupName *string `json:"groupName"`
	GroupID   *int    `json:"groupId"`
}

type AdminGroupAccess struct {
	ID      int `json:"id" db:"id"`
	AdminID int `json:"adminId" db:"admin_id"`
	GroupID int `json:"groupId" db:"group_id"`
}

type Setting struct {
	Key   string `json:"key" db:"key"`
	Value string `json:"value" db:"value"`
}

type DashboardStats struct {
	TotalEmployees   int                    `json:"totalEmployees"`
	TotalGroups      int                    `json:"totalGroups"`
	TodayPresent     int                    `json:"todayPresent"`
	TodayAbsent      int                    `json:"todayAbsent"`
	RecentAttendance []RecentAttendanceItem `json:"recentAttendance"`
	TopEmployees     []TopEmployeeItem      `json:"topEmployees"`
}

type RecentAttendanceItem struct {
	ID         int    `json:"id"`
	EmployeeNo string `json:"employeeNo"`
	FullName   string `json:"fullName"`
	EventTime  string `json:"eventTime"`
	Status     string `json:"status"`
}

type TopEmployeeItem struct {
	ID         int     `json:"id"`
	EmployeeNo string  `json:"employeeNo"`
	FullName   string  `json:"fullName"`
	Position   *string `json:"position"`
	GroupName  *string `json:"groupName"`
	PhotoURL   *string `json:"photoUrl"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateAdminRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	FullName string `json:"fullName"`
}

type CreateEmployeeRequest struct {
	EmployeeNo string  `json:"employeeNo"`
	FullName   string  `json:"fullName"`
	Position   *string `json:"position"`
	GroupID    *int    `json:"groupId"`
	Phone      *string `json:"phone"`
}

type CreateGroupRequest struct {
	Name        string  `json:"name"`
	Login       string  `json:"login"`
	Password    string  `json:"password"`
	Description *string `json:"description"`
}

type UpdateGroupRequest struct {
	Name            *string `json:"name,omitempty"`
	Login           *string `json:"login,omitempty"`
	Password        *string `json:"password,omitempty"`
	Description     *string `json:"description,omitempty"`
	AssignedAdminID *int    `json:"assignedAdminId,omitempty"`
}

type JoinGroupRequest struct {
	GroupID  int    `json:"groupId"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LeaveGroupRequest struct {
	GroupID int `json:"groupId"`
}

type ManualAttendanceRequest struct {
	EmployeeID int    `json:"employeeId"`
	EmployeeNo string `json:"employeeNo"`
	Status     string `json:"status"`
}

type HikvisionSettingsRequest struct {
	IP       string `json:"ip"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	AutoSync *bool  `json:"autoSync,omitempty"`
}

type HikvisionSettingsResponse struct {
	IP          string `json:"ip"`
	Username    string `json:"username"`
	HasPassword bool   `json:"hasPassword"`
	AutoSync    bool   `json:"autoSync"`
}

type TelegramTokenRequest struct {
	Token string `json:"token"`
}

type TelegramPanelGroup struct {
	ID               int                      `json:"id"`
	Name             string                   `json:"name"`
	ResponsibleAdmin *TelegramPanelAdmin      `json:"responsibleAdmin"`
	TotalEmployees   int                      `json:"totalEmployees"`
	CameCount        int                      `json:"cameCount"`
	NotCameCount     int                      `json:"notCameCount"`
	Came             []TelegramPanelEmployee  `json:"came"`
	NotCame          []TelegramPanelEmployee  `json:"notCame"`
	Employees        []TelegramPanelEmployee2 `json:"employees"`
}

type TelegramPanelAdmin struct {
	ID             int     `json:"id"`
	FullName       string  `json:"fullName"`
	TelegramUserID *string `json:"telegramUserId"`
}

type TelegramPanelEmployee struct {
	ID             int     `json:"id"`
	FullName       string  `json:"fullName"`
	TelegramUserID *string `json:"telegramUserId"`
}

type TelegramPanelEmployee2 struct {
	ID             int     `json:"id"`
	FullName       string  `json:"fullName"`
	EmployeeNo     string  `json:"employeeNo"`
	TelegramUserID *string `json:"telegramUserId"`
}

type AttendanceReportResponse struct {
	Groups     []AttendanceReportGroup    `json:"groups"`
	Employees  []AttendanceReportEmployee `json:"employees"`
	Attendance []AttendanceReportRecord   `json:"attendance"`
}

type AttendanceReportGroup struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type AttendanceReportEmployee struct {
	ID       int    `json:"id"`
	FullName string `json:"fullName"`
	GroupID  *int   `json:"groupId"`
}

type AttendanceReportRecord struct {
	EmployeeID int    `json:"employeeId"`
	EventTime  string `json:"eventTime"`
	Status     string `json:"status"`
}
