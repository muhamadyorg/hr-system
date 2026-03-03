package goserver

import (
        "database/sql"
        "fmt"
        "strings"
        "time"
)

func GetUser(id int) (*User, error) {
        u := &User{}
        err := DB.QueryRow(`SELECT id, username, password, full_name, role, created_by, is_active, telegram_user_id FROM users WHERE id = $1`, id).
                Scan(&u.ID, &u.Username, &u.Password, &u.FullName, &u.Role, &u.CreatedBy, &u.IsActive, &u.TelegramUserID)
        if err == sql.ErrNoRows {
                return nil, nil
        }
        if err != nil {
                return nil, err
        }
        return u, nil
}

func GetUserByUsername(username string) (*User, error) {
        u := &User{}
        err := DB.QueryRow(`SELECT id, username, password, full_name, role, created_by, is_active, telegram_user_id FROM users WHERE username = $1`, username).
                Scan(&u.ID, &u.Username, &u.Password, &u.FullName, &u.Role, &u.CreatedBy, &u.IsActive, &u.TelegramUserID)
        if err == sql.ErrNoRows {
                return nil, nil
        }
        if err != nil {
                return nil, err
        }
        return u, nil
}

func CreateUser(username, password, fullName, role string, createdBy *int, isActive bool) (*User, error) {
        u := &User{}
        err := DB.QueryRow(`INSERT INTO users (username, password, full_name, role, created_by, is_active) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, username, password, full_name, role, created_by, is_active, telegram_user_id`,
                username, password, fullName, role, createdBy, isActive).
                Scan(&u.ID, &u.Username, &u.Password, &u.FullName, &u.Role, &u.CreatedBy, &u.IsActive, &u.TelegramUserID)
        if err != nil {
                return nil, err
        }
        return u, nil
}

func GetAdmins() ([]User, error) {
        rows, err := DB.Query(`SELECT id, username, password, full_name, role, created_by, is_active, telegram_user_id FROM users WHERE role = 'admin' AND is_active = true`)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var admins []User
        for rows.Next() {
                var u User
                if err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.FullName, &u.Role, &u.CreatedBy, &u.IsActive, &u.TelegramUserID); err != nil {
                        return nil, err
                }
                admins = append(admins, u)
        }
        return admins, nil
}

func UpdateUserPassword(id int, hashedPassword string) error {
        _, err := DB.Exec(`UPDATE users SET password = $1 WHERE id = $2`, hashedPassword, id)
        return err
}

func DeleteUser(id int) error {
        _, err := DB.Exec(`UPDATE users SET is_active = false WHERE id = $1`, id)
        return err
}

func UpdateUserTelegramID(id int, telegramUserID *string) error {
        _, err := DB.Exec(`UPDATE users SET telegram_user_id = $1 WHERE id = $2`, telegramUserID, id)
        return err
}

func GetGroups() ([]GroupWithCount, error) {
        rows, err := DB.Query(`SELECT id, name, login, password, description, created_by, assigned_admin_id, is_active FROM groups WHERE is_active = true`)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var groups []GroupWithCount
        for rows.Next() {
                var g Group
                if err := rows.Scan(&g.ID, &g.Name, &g.Login, &g.Password, &g.Description, &g.CreatedBy, &g.AssignedAdminID, &g.IsActive); err != nil {
                        return nil, err
                }

                var cnt int
                DB.QueryRow(`SELECT COUNT(*) FROM employees WHERE group_id = $1 AND is_active = true`, g.ID).Scan(&cnt)

                var assignedAdminName *string
                if g.AssignedAdminID != nil {
                        var name string
                        err := DB.QueryRow(`SELECT full_name FROM users WHERE id = $1`, *g.AssignedAdminID).Scan(&name)
                        if err == nil {
                                assignedAdminName = &name
                        }
                }

                groups = append(groups, GroupWithCount{Group: g, EmployeeCount: cnt, AssignedAdminName: assignedAdminName})
        }
        return groups, nil
}

func GetGroup(id int) (*Group, error) {
        g := &Group{}
        err := DB.QueryRow(`SELECT id, name, login, password, description, created_by, assigned_admin_id, is_active FROM groups WHERE id = $1`, id).
                Scan(&g.ID, &g.Name, &g.Login, &g.Password, &g.Description, &g.CreatedBy, &g.AssignedAdminID, &g.IsActive)
        if err == sql.ErrNoRows {
                return nil, nil
        }
        if err != nil {
                return nil, err
        }
        return g, nil
}

func CreateGroup(name string, login, password *string, description *string, createdBy int, assignedAdminID *int, isActive bool) (*Group, error) {
        g := &Group{}
        err := DB.QueryRow(`INSERT INTO groups (name, login, password, description, created_by, assigned_admin_id, is_active) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, name, login, password, description, created_by, assigned_admin_id, is_active`,
                name, login, password, description, createdBy, assignedAdminID, isActive).
                Scan(&g.ID, &g.Name, &g.Login, &g.Password, &g.Description, &g.CreatedBy, &g.AssignedAdminID, &g.IsActive)
        if err != nil {
                return nil, err
        }
        return g, nil
}

var allowedGroupColumns = map[string]bool{
        "name": true, "login": true, "password": true, "description": true,
        "assigned_admin_id": true, "is_active": true,
}

func UpdateGroup(id int, data map[string]interface{}) (*Group, error) {
        setParts := []string{}
        args := []interface{}{}
        i := 1
        for k, v := range data {
                if !allowedGroupColumns[k] {
                        continue
                }
                setParts = append(setParts, fmt.Sprintf("%s = $%d", k, i))
                args = append(args, v)
                i++
        }
        if len(setParts) == 0 {
                return GetGroup(id)
        }
        args = append(args, id)
        query := fmt.Sprintf(`UPDATE groups SET %s WHERE id = $%d RETURNING id, name, login, password, description, created_by, assigned_admin_id, is_active`, strings.Join(setParts, ", "), i)

        g := &Group{}
        err := DB.QueryRow(query, args...).
                Scan(&g.ID, &g.Name, &g.Login, &g.Password, &g.Description, &g.CreatedBy, &g.AssignedAdminID, &g.IsActive)
        if err == sql.ErrNoRows {
                return nil, nil
        }
        if err != nil {
                return nil, err
        }
        return g, nil
}

func DeleteGroup(id int) error {
        _, err := DB.Exec(`UPDATE employees SET group_id = NULL WHERE group_id = $1`, id)
        if err != nil {
                return err
        }
        _, err = DB.Exec(`DELETE FROM admin_group_access WHERE group_id = $1`, id)
        if err != nil {
                return err
        }
        _, err = DB.Exec(`UPDATE groups SET is_active = false WHERE id = $1`, id)
        return err
}

func GetGroupEmployees(groupID int) ([]Employee, error) {
        rows, err := DB.Query(`SELECT id, employee_no, full_name, position, group_id, photo_url, phone, is_active, hikvision_synced, telegram_user_id FROM employees WHERE group_id = $1 AND is_active = true`, groupID)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var emps []Employee
        for rows.Next() {
                var e Employee
                if err := rows.Scan(&e.ID, &e.EmployeeNo, &e.FullName, &e.Position, &e.GroupID, &e.PhotoURL, &e.Phone, &e.IsActive, &e.HikvisionSynced, &e.TelegramUserID); err != nil {
                        return nil, err
                }
                emps = append(emps, e)
        }
        return emps, nil
}

func GetEmployees() ([]EmployeeWithGroup, error) {
        rows, err := DB.Query(`SELECT e.id, e.employee_no, e.full_name, e.position, e.group_id, e.photo_url, e.phone, e.is_active, e.hikvision_synced, e.telegram_user_id, g.name
                FROM employees e LEFT JOIN groups g ON e.group_id = g.id WHERE e.is_active = true`)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var emps []EmployeeWithGroup
        for rows.Next() {
                var e EmployeeWithGroup
                if err := rows.Scan(&e.ID, &e.EmployeeNo, &e.FullName, &e.Position, &e.GroupID, &e.PhotoURL, &e.Phone, &e.IsActive, &e.HikvisionSynced, &e.TelegramUserID, &e.GroupName); err != nil {
                        return nil, err
                }
                emps = append(emps, e)
        }
        return emps, nil
}

func GetEmployee(id int) (*EmployeeWithGroup, error) {
        e := &EmployeeWithGroup{}
        err := DB.QueryRow(`SELECT e.id, e.employee_no, e.full_name, e.position, e.group_id, e.photo_url, e.phone, e.is_active, e.hikvision_synced, e.telegram_user_id, g.name
                FROM employees e LEFT JOIN groups g ON e.group_id = g.id WHERE e.id = $1`, id).
                Scan(&e.ID, &e.EmployeeNo, &e.FullName, &e.Position, &e.GroupID, &e.PhotoURL, &e.Phone, &e.IsActive, &e.HikvisionSynced, &e.TelegramUserID, &e.GroupName)
        if err == sql.ErrNoRows {
                return nil, nil
        }
        if err != nil {
                return nil, err
        }
        return e, nil
}

func GetEmployeeByNo(employeeNo string) (*Employee, error) {
        e := &Employee{}
        err := DB.QueryRow(`SELECT id, employee_no, full_name, position, group_id, photo_url, phone, is_active, hikvision_synced, telegram_user_id FROM employees WHERE employee_no = $1`, employeeNo).
                Scan(&e.ID, &e.EmployeeNo, &e.FullName, &e.Position, &e.GroupID, &e.PhotoURL, &e.Phone, &e.IsActive, &e.HikvisionSynced, &e.TelegramUserID)
        if err == sql.ErrNoRows {
                return nil, nil
        }
        if err != nil {
                return nil, err
        }
        return e, nil
}

func CreateEmployee(employeeNo, fullName string, position *string, groupID *int, phone *string, isActive bool, hikvisionSynced bool) (*Employee, error) {
        e := &Employee{}
        err := DB.QueryRow(`INSERT INTO employees (employee_no, full_name, position, group_id, phone, is_active, hikvision_synced) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, employee_no, full_name, position, group_id, photo_url, phone, is_active, hikvision_synced, telegram_user_id`,
                employeeNo, fullName, position, groupID, phone, isActive, hikvisionSynced).
                Scan(&e.ID, &e.EmployeeNo, &e.FullName, &e.Position, &e.GroupID, &e.PhotoURL, &e.Phone, &e.IsActive, &e.HikvisionSynced, &e.TelegramUserID)
        if err != nil {
                return nil, err
        }
        return e, nil
}

var allowedEmployeeColumns = map[string]bool{
        "employee_no": true, "full_name": true, "position": true, "group_id": true,
        "photo_url": true, "phone": true, "is_active": true, "hikvision_synced": true,
        "telegram_user_id": true,
}

func UpdateEmployee(id int, data map[string]interface{}) (*Employee, error) {
        setParts := []string{}
        args := []interface{}{}
        i := 1
        for k, v := range data {
                if !allowedEmployeeColumns[k] {
                        continue
                }
                setParts = append(setParts, fmt.Sprintf("%s = $%d", k, i))
                args = append(args, v)
                i++
        }
        if len(setParts) == 0 {
                return nil, nil
        }
        args = append(args, id)
        query := fmt.Sprintf(`UPDATE employees SET %s WHERE id = $%d RETURNING id, employee_no, full_name, position, group_id, photo_url, phone, is_active, hikvision_synced, telegram_user_id`, strings.Join(setParts, ", "), i)

        e := &Employee{}
        err := DB.QueryRow(query, args...).
                Scan(&e.ID, &e.EmployeeNo, &e.FullName, &e.Position, &e.GroupID, &e.PhotoURL, &e.Phone, &e.IsActive, &e.HikvisionSynced, &e.TelegramUserID)
        if err == sql.ErrNoRows {
                return nil, nil
        }
        if err != nil {
                return nil, err
        }
        return e, nil
}

func DeleteEmployee(id int) error {
        _, err := DB.Exec(`UPDATE employees SET is_active = false WHERE id = $1`, id)
        return err
}

func GetAttendanceByDate(date string, groupID *int) ([]AttendanceWithEmployee, error) {
        startDate := date + "T00:00:00+05:00"
        endDate := date + "T23:59:59.999+05:00"

        rows, err := DB.Query(`SELECT ar.id, ar.employee_id, ar.employee_no, ar.event_time, ar.status, ar.photo_url, ar.device_ip, ar.device_name
                FROM attendance_records ar WHERE ar.event_time >= $1::timestamptz AND ar.event_time <= $2::timestamptz ORDER BY ar.event_time DESC`, startDate, endDate)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var result []AttendanceWithEmployee
        for rows.Next() {
                var ar AttendanceRecord
                if err := rows.Scan(&ar.ID, &ar.EmployeeID, &ar.EmployeeNo, &ar.EventTime, &ar.Status, &ar.PhotoURL, &ar.DeviceIP, &ar.DeviceName); err != nil {
                        return nil, err
                }
                emp, err := GetEmployee(ar.EmployeeID)
                if err != nil || emp == nil || !emp.IsActive {
                        continue
                }
                if groupID != nil && (emp.GroupID == nil || *emp.GroupID != *groupID) {
                        continue
                }
                result = append(result, AttendanceWithEmployee{
                        AttendanceRecord: ar,
                        FullName:         emp.FullName,
                        GroupName:        emp.GroupName,
                        GroupID:          emp.GroupID,
                })
        }
        return result, nil
}

func GetAttendanceByDateRange(startDate, endDate string, groupIDs []int) ([]AttendanceRecord, error) {
        start := startDate + "T00:00:00+05:00"
        end := endDate + "T23:59:59.999+05:00"

        rows, err := DB.Query(`SELECT id, employee_id, employee_no, event_time, status, photo_url, device_ip, device_name
                FROM attendance_records WHERE event_time >= $1::timestamptz AND event_time <= $2::timestamptz ORDER BY event_time`, start, end)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var records []AttendanceRecord
        for rows.Next() {
                var ar AttendanceRecord
                if err := rows.Scan(&ar.ID, &ar.EmployeeID, &ar.EmployeeNo, &ar.EventTime, &ar.Status, &ar.PhotoURL, &ar.DeviceIP, &ar.DeviceName); err != nil {
                        return nil, err
                }
                records = append(records, ar)
        }

        if len(groupIDs) == 0 {
                return records, nil
        }

        groupSet := map[int]bool{}
        for _, id := range groupIDs {
                groupSet[id] = true
        }

        var filtered []AttendanceRecord
        for _, rec := range records {
                emp, err := GetEmployee(rec.EmployeeID)
                if err != nil || emp == nil {
                        continue
                }
                if emp.GroupID != nil && groupSet[*emp.GroupID] {
                        filtered = append(filtered, rec)
                }
        }
        return filtered, nil
}

func GetAttendanceByEmployee(employeeID int) ([]AttendanceRecord, error) {
        rows, err := DB.Query(`SELECT id, employee_id, employee_no, event_time, status, photo_url, device_ip, device_name
                FROM attendance_records WHERE employee_id = $1 ORDER BY event_time DESC LIMIT 50`, employeeID)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var records []AttendanceRecord
        for rows.Next() {
                var ar AttendanceRecord
                if err := rows.Scan(&ar.ID, &ar.EmployeeID, &ar.EmployeeNo, &ar.EventTime, &ar.Status, &ar.PhotoURL, &ar.DeviceIP, &ar.DeviceName); err != nil {
                        return nil, err
                }
                records = append(records, ar)
        }
        return records, nil
}

func CreateAttendance(employeeID int, employeeNo string, eventTime time.Time, status string, photoURL, deviceIP, deviceName *string) (*AttendanceRecord, error) {
        ar := &AttendanceRecord{}
        err := DB.QueryRow(`INSERT INTO attendance_records (employee_id, employee_no, event_time, status, photo_url, device_ip, device_name) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, employee_id, employee_no, event_time, status, photo_url, device_ip, device_name`,
                employeeID, employeeNo, eventTime, status, photoURL, deviceIP, deviceName).
                Scan(&ar.ID, &ar.EmployeeID, &ar.EmployeeNo, &ar.EventTime, &ar.Status, &ar.PhotoURL, &ar.DeviceIP, &ar.DeviceName)
        if err != nil {
                return nil, err
        }
        return ar, nil
}

func GetDashboardStats(accessGroupIDs []int) (*DashboardStats, error) {
        allEmps, err := GetEmployees()
        if err != nil {
                return nil, err
        }
        allGroups, err := GetGroups()
        if err != nil {
                return nil, err
        }

        accessSet := map[int]bool{}
        hasFilter := accessGroupIDs != nil
        for _, id := range accessGroupIDs {
                accessSet[id] = true
        }

        var filteredEmps []EmployeeWithGroup
        for _, e := range allEmps {
                if hasFilter {
                        if e.GroupID != nil && accessSet[*e.GroupID] {
                                filteredEmps = append(filteredEmps, e)
                        }
                } else {
                        filteredEmps = append(filteredEmps, e)
                }
        }

        var filteredGroups []GroupWithCount
        for _, g := range allGroups {
                if hasFilter {
                        if accessSet[g.ID] {
                                filteredGroups = append(filteredGroups, g)
                        }
                } else {
                        filteredGroups = append(filteredGroups, g)
                }
        }

        loc := time.FixedZone("UZ", 5*60*60)
        nowUz := time.Now().In(loc)
        uzDateStr := nowUz.Format("2006-01-02")
        todayStart, _ := time.Parse("2006-01-02T15:04:05-07:00", uzDateStr+"T00:00:00+05:00")
        todayEnd, _ := time.Parse("2006-01-02T15:04:05-07:00", uzDateStr+"T23:59:59+05:00")

        rows, err := DB.Query(`SELECT id, employee_id, employee_no, event_time, status FROM attendance_records WHERE event_time >= $1 AND event_time <= $2 ORDER BY event_time DESC`, todayStart, todayEnd)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        type todayRec struct {
                ID         int
                EmployeeID int
                EmployeeNo string
                EventTime  time.Time
                Status     string
        }
        var todayRecords []todayRec
        for rows.Next() {
                var r todayRec
                rows.Scan(&r.ID, &r.EmployeeID, &r.EmployeeNo, &r.EventTime, &r.Status)
                todayRecords = append(todayRecords, r)
        }

        empIDs := map[int]bool{}
        for _, e := range filteredEmps {
                empIDs[e.ID] = true
        }

        var filteredRecords []todayRec
        for _, r := range todayRecords {
                if empIDs[r.EmployeeID] {
                        filteredRecords = append(filteredRecords, r)
                }
        }

        presentIDs := map[int]bool{}
        for _, r := range filteredRecords {
                presentIDs[r.EmployeeID] = true
        }

        var recentAttendance []RecentAttendanceItem
        for i, rec := range filteredRecords {
                if i >= 10 {
                        break
                }
                for _, emp := range filteredEmps {
                        if emp.ID == rec.EmployeeID {
                                recentAttendance = append(recentAttendance, RecentAttendanceItem{
                                        ID:         rec.ID,
                                        EmployeeNo: rec.EmployeeNo,
                                        FullName:   emp.FullName,
                                        EventTime:  rec.EventTime.Format(time.RFC3339),
                                        Status:     rec.Status,
                                })
                                break
                        }
                }
        }

        var topEmployees []TopEmployeeItem
        for i, emp := range filteredEmps {
                if i >= 5 {
                        break
                }
                topEmployees = append(topEmployees, TopEmployeeItem{
                        ID:         emp.ID,
                        EmployeeNo: emp.EmployeeNo,
                        FullName:   emp.FullName,
                        Position:   emp.Position,
                        GroupName:  emp.GroupName,
                        PhotoURL:   emp.PhotoURL,
                })
        }

        if recentAttendance == nil {
                recentAttendance = []RecentAttendanceItem{}
        }
        if topEmployees == nil {
                topEmployees = []TopEmployeeItem{}
        }

        return &DashboardStats{
                TotalEmployees:   len(filteredEmps),
                TotalGroups:      len(filteredGroups),
                TodayPresent:     len(presentIDs),
                TodayAbsent:      len(filteredEmps) - len(presentIDs),
                RecentAttendance: recentAttendance,
                TopEmployees:     topEmployees,
        }, nil
}

func GetAdminGroupAccess(adminID int) ([]int, error) {
        rows, err := DB.Query(`SELECT group_id FROM admin_group_access WHERE admin_id = $1`, adminID)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var ids []int
        for rows.Next() {
                var id int
                rows.Scan(&id)
                ids = append(ids, id)
        }
        return ids, nil
}

func AddAdminGroupAccess(adminID, groupID int) error {
        var exists int
        err := DB.QueryRow(`SELECT COUNT(*) FROM admin_group_access WHERE admin_id = $1 AND group_id = $2`, adminID, groupID).Scan(&exists)
        if err != nil {
                return err
        }
        if exists == 0 {
                _, err = DB.Exec(`INSERT INTO admin_group_access (admin_id, group_id) VALUES ($1, $2)`, adminID, groupID)
        }
        return err
}

func RemoveGroupAccess(groupID int) error {
        _, err := DB.Exec(`DELETE FROM admin_group_access WHERE group_id = $1`, groupID)
        return err
}

func RemoveAdminGroupAccessForAdmin(adminID, groupID int) error {
        _, err := DB.Exec(`DELETE FROM admin_group_access WHERE admin_id = $1 AND group_id = $2`, adminID, groupID)
        return err
}

func SetSetting(key, value string) error {
        _, err := DB.Exec(`INSERT INTO settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2`, key, value)
        return err
}

func GetSetting(key string) (*string, error) {
        var val string
        err := DB.QueryRow(`SELECT value FROM settings WHERE key = $1`, key).Scan(&val)
        if err == sql.ErrNoRows {
                return nil, nil
        }
        if err != nil {
                return nil, err
        }
        return &val, nil
}

func GetTodayAttendanceForEmployee(employeeID int) ([]AttendanceRecord, error) {
        loc := time.FixedZone("UZ", 5*60*60)
        nowUz := time.Now().In(loc)
        uzDateStr := nowUz.Format("2006-01-02")
        todayStart, _ := time.Parse("2006-01-02T15:04:05-07:00", uzDateStr+"T00:00:00+05:00")
        todayEnd, _ := time.Parse("2006-01-02T15:04:05-07:00", uzDateStr+"T23:59:59+05:00")

        rows, err := DB.Query(`SELECT id, employee_id, employee_no, event_time, status, photo_url, device_ip, device_name
                FROM attendance_records WHERE employee_id = $1 AND event_time >= $2 AND event_time <= $3 ORDER BY event_time`, employeeID, todayStart, todayEnd)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var records []AttendanceRecord
        for rows.Next() {
                var ar AttendanceRecord
                rows.Scan(&ar.ID, &ar.EmployeeID, &ar.EmployeeNo, &ar.EventTime, &ar.Status, &ar.PhotoURL, &ar.DeviceIP, &ar.DeviceName)
                records = append(records, ar)
        }
        return records, nil
}
