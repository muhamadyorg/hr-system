package goserver

import (
        "encoding/json"
        "encoding/xml"
        "fmt"
        "io"
        "log"
        "mime"
        "mime/multipart"
        "net/http"
        "strconv"
        "strings"
        "time"

        "golang.org/x/crypto/bcrypt"
)

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
        respondJSON(w, status, map[string]string{"message": message})
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                _, ok := GetSessionUserID(r)
                if !ok {
                        respondError(w, 401, "Tizimga kiring")
                        return
                }
                next(w, r)
        }
}

func requireSudo(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                userID, ok := GetSessionUserID(r)
                if !ok {
                        respondError(w, 401, "Tizimga kiring")
                        return
                }
                user, err := GetUser(userID)
                if err != nil || user == nil || user.Role != "sudo" {
                        respondError(w, 403, "Ruxsat yo'q. Faqat Sudo uchun")
                        return
                }
                next(w, r)
        }
}

func getCurrentUser(r *http.Request) *User {
        userID, ok := GetSessionUserID(r)
        if !ok {
                return nil
        }
        user, _ := GetUser(userID)
        return user
}

func getPathParam(path, prefix string) string {
        return strings.TrimPrefix(path, prefix)
}

func RegisterRoutes(mux *http.ServeMux) {
        mux.HandleFunc("/api/auth/login", handleLogin)
        mux.HandleFunc("/api/auth/me", handleMe)
        mux.HandleFunc("/api/auth/logout", handleLogout)
        mux.HandleFunc("/api/dashboard/stats", requireAuth(handleDashboardStats))
        mux.HandleFunc("/api/admins", requireAuth(handleAdmins))
        mux.HandleFunc("/api/admins/", requireAuth(handleAdminByID))
        mux.HandleFunc("/api/groups/join", requireAuth(handleGroupJoin))
        mux.HandleFunc("/api/groups/leave", requireAuth(handleGroupLeave))
        mux.HandleFunc("/api/my-groups", requireAuth(handleMyGroups))
        mux.HandleFunc("/api/groups/", requireAuth(handleGroupByID))
        mux.HandleFunc("/api/groups", requireAuth(handleGroups))
        mux.HandleFunc("/api/employees/", requireAuth(handleEmployeeByID))
        mux.HandleFunc("/api/employees", requireAuth(handleEmployees))
        mux.HandleFunc("/api/attendance/report", requireAuth(handleAttendanceReport))
        mux.HandleFunc("/api/attendance/employee/", requireAuth(handleAttendanceByEmployee))
        mux.HandleFunc("/api/attendance", requireAuth(handleAttendance))
        mux.HandleFunc("/api/hikvision/event", handleHikvisionEvent)
        mux.HandleFunc("/api/hikvision/settings", requireAuth(handleHikvisionSettings))
        mux.HandleFunc("/api/hikvision/test", requireSudo(handleHikvisionTest))
        mux.HandleFunc("/api/hikvision/sync", requireSudo(handleHikvisionSync))
        mux.HandleFunc("/api/hikvision/camera-users", requireSudo(handleHikvisionCameraUsers))
        mux.HandleFunc("/api/telegram/token", requireAuth(handleTelegramToken))
        mux.HandleFunc("/api/telegram/panel", requireSudo(handleTelegramPanel))
        mux.HandleFunc("/api/telegram/admin/", requireSudo(handleTelegramAdmin))
        mux.HandleFunc("/api/telegram/employee/", requireSudo(handleTelegramEmployee))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
                respondError(w, 405, "Method not allowed")
                return
        }
        var req LoginRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                respondError(w, 400, "Foydalanuvchi nomi va parol kiritilishi shart")
                return
        }
        if req.Username == "" || req.Password == "" {
                respondError(w, 400, "Foydalanuvchi nomi va parol kiritilishi shart")
                return
        }
        user, err := GetUserByUsername(req.Username)
        if err != nil || user == nil || !user.IsActive {
                respondError(w, 401, "Noto'g'ri foydalanuvchi nomi yoki parol")
                return
        }
        if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
                respondError(w, 401, "Noto'g'ri foydalanuvchi nomi yoki parol")
                return
        }
        if err := CreateSession(w, user.ID); err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        respondJSON(w, 200, map[string]interface{}{
                "id": user.ID, "username": user.Username, "fullName": user.FullName,
                "role": user.Role, "createdBy": user.CreatedBy, "isActive": user.IsActive,
                "telegramUserId": user.TelegramUserID,
        })
}

func handleMe(w http.ResponseWriter, r *http.Request) {
        if r.Method != "GET" {
                respondError(w, 405, "Method not allowed")
                return
        }
        userID, ok := GetSessionUserID(r)
        if !ok {
                respondError(w, 401, "Tizimga kiring")
                return
        }
        user, err := GetUser(userID)
        if err != nil || user == nil || !user.IsActive {
                respondError(w, 401, "Foydalanuvchi topilmadi")
                return
        }
        respondJSON(w, 200, map[string]interface{}{
                "id": user.ID, "username": user.Username, "fullName": user.FullName,
                "role": user.Role, "createdBy": user.CreatedBy, "isActive": user.IsActive,
                "telegramUserId": user.TelegramUserID,
        })
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
                respondError(w, 405, "Method not allowed")
                return
        }
        DestroySession(w, r)
        respondJSON(w, 200, map[string]string{"message": "Chiqildi"})
}

func handleDashboardStats(w http.ResponseWriter, r *http.Request) {
        if r.Method != "GET" {
                respondError(w, 405, "Method not allowed")
                return
        }
        currentUser := getCurrentUser(r)
        if currentUser == nil {
                respondError(w, 401, "Tizimga kiring")
                return
        }

        var accessIds []int
        if currentUser.Role != "sudo" {
                var err error
                accessIds, err = GetAdminGroupAccess(currentUser.ID)
                if err != nil {
                        respondError(w, 500, "Server xatoligi: "+err.Error())
                        return
                }
        }

        stats, err := GetDashboardStats(accessIds)
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        respondJSON(w, 200, stats)
}

func handleAdmins(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case "GET":
                requireSudo(handleGetAdmins)(w, r)
        case "POST":
                requireSudo(handleCreateAdmin)(w, r)
        default:
                respondError(w, 405, "Method not allowed")
        }
}

func handleGetAdmins(w http.ResponseWriter, r *http.Request) {
        admins, err := GetAdmins()
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        result := []map[string]interface{}{}
        for _, a := range admins {
                plainPw, _ := GetSetting(fmt.Sprintf("admin_password_%d", a.ID))
                entry := map[string]interface{}{
                        "id": a.ID, "username": a.Username, "fullName": a.FullName,
                        "role": a.Role, "createdBy": a.CreatedBy, "isActive": a.IsActive,
                        "plainPassword": plainPw,
                }
                result = append(result, entry)
        }
        respondJSON(w, 200, result)
}

func handleCreateAdmin(w http.ResponseWriter, r *http.Request) {
        var req CreateAdminRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                respondError(w, 400, "Ma'lumotlar noto'g'ri")
                return
        }
        if req.Username == "" {
                respondError(w, 400, "Foydalanuvchi nomi kiritilishi shart")
                return
        }
        if len(req.Password) < 4 {
                respondError(w, 400, "Parol kamida 4 ta belgidan iborat bo'lishi shart")
                return
        }
        if req.FullName == "" {
                respondError(w, 400, "To'liq ism kiritilishi shart")
                return
        }
        existing, _ := GetUserByUsername(req.Username)
        if existing != nil {
                respondError(w, 400, "Bu foydalanuvchi nomi allaqachon mavjud")
                return
        }

        currentUser := getCurrentUser(r)
        hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
        user, err := CreateUser(req.Username, string(hashed), req.FullName, "admin", &currentUser.ID, true)
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        SetSetting(fmt.Sprintf("admin_password_%d", user.ID), req.Password)
        respondJSON(w, 200, map[string]interface{}{
                "id": user.ID, "username": user.Username, "fullName": user.FullName,
                "role": user.Role, "createdBy": user.CreatedBy, "isActive": user.IsActive,
        })
}

func handleAdminByID(w http.ResponseWriter, r *http.Request) {
        path := r.URL.Path
        parts := strings.Split(strings.TrimPrefix(path, "/api/admins/"), "/")
        if len(parts) == 0 {
                respondError(w, 400, "ID kiritilishi shart")
                return
        }
        id, err := strconv.Atoi(parts[0])
        if err != nil {
                respondError(w, 400, "Noto'g'ri ID")
                return
        }

        if len(parts) >= 2 && parts[1] == "password" && r.Method == "PATCH" {
                requireSudo(func(w http.ResponseWriter, r *http.Request) {
                        handleUpdateAdminPassword(w, r, id)
                })(w, r)
                return
        }

        if r.Method == "DELETE" {
                requireSudo(func(w http.ResponseWriter, r *http.Request) {
                        handleDeleteAdmin(w, r, id)
                })(w, r)
                return
        }

        respondError(w, 405, "Method not allowed")
}

func handleUpdateAdminPassword(w http.ResponseWriter, r *http.Request, id int) {
        var body struct {
                Password string `json:"password"`
        }
        json.NewDecoder(r.Body).Decode(&body)
        if len(body.Password) < 3 {
                respondError(w, 400, "Parol kamida 3 ta belgidan iborat bo'lishi kerak")
                return
        }
        user, err := GetUser(id)
        if err != nil || user == nil || user.Role == "sudo" {
                respondError(w, 400, "Admin topilmadi")
                return
        }
        hashed, _ := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
        UpdateUserPassword(id, string(hashed))
        SetSetting(fmt.Sprintf("admin_password_%d", id), body.Password)
        respondJSON(w, 200, map[string]string{"message": "Parol yangilandi"})
}

func handleDeleteAdmin(w http.ResponseWriter, r *http.Request, id int) {
        user, err := GetUser(id)
        if err != nil || user == nil {
                respondError(w, 404, "Admin topilmadi")
                return
        }
        if user.Role == "sudo" {
                respondError(w, 400, "Sudo hisobini o'chirish mumkin emas")
                return
        }
        DeleteUser(id)
        respondJSON(w, 200, map[string]string{"message": "Admin o'chirildi"})
}

func handleGroups(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case "GET":
                requireAuth(handleGetGroups)(w, r)
        case "POST":
                requireSudo(handleCreateGroup)(w, r)
        default:
                respondError(w, 405, "Method not allowed")
        }
}

func handleGetGroups(w http.ResponseWriter, r *http.Request) {
        currentUser := getCurrentUser(r)
        allGroups, err := GetGroups()
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        result := make([]map[string]interface{}, 0, len(allGroups))
        for _, g := range allGroups {
                entry := map[string]interface{}{
                        "id": g.ID, "name": g.Name, "description": g.Description,
                        "createdBy": g.CreatedBy, "assignedAdminId": g.AssignedAdminID,
                        "isActive": g.IsActive, "employeeCount": g.EmployeeCount,
                        "assignedAdminName": g.AssignedAdminName,
                }
                if currentUser != nil && currentUser.Role == "sudo" {
                        entry["login"] = g.Login
                        entry["password"] = g.Password
                } else {
                        entry["login"] = "***"
                        entry["password"] = "***"
                }
                result = append(result, entry)
        }
        respondJSON(w, 200, result)
}

func handleCreateGroup(w http.ResponseWriter, r *http.Request) {
        var req CreateGroupRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                respondError(w, 400, "Ma'lumotlar noto'g'ri")
                return
        }
        if req.Name == "" {
                respondError(w, 400, "Guruh nomi kiritilishi shart")
                return
        }
        if req.Login == "" {
                respondError(w, 400, "Login kiritilishi shart")
                return
        }
        if req.Password == "" {
                respondError(w, 400, "Parol kiritilishi shart")
                return
        }
        currentUser := getCurrentUser(r)
        grp, err := CreateGroup(req.Name, &req.Login, &req.Password, req.Description, currentUser.ID, nil, true)
        if err != nil {
                if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
                        respondError(w, 400, "Bu guruh nomi allaqachon mavjud")
                        return
                }
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        respondJSON(w, 200, grp)
}

func handleGroupByID(w http.ResponseWriter, r *http.Request) {
        path := r.URL.Path
        rest := strings.TrimPrefix(path, "/api/groups/")

        parts := strings.Split(rest, "/")
        if len(parts) == 0 || parts[0] == "" {
                respondError(w, 400, "ID kiritilishi shart")
                return
        }

        id, err := strconv.Atoi(parts[0])
        if err != nil {
                respondError(w, 400, "Noto'g'ri ID")
                return
        }

        if len(parts) >= 2 && parts[1] == "employees" {
                requireAuth(func(w http.ResponseWriter, r *http.Request) {
                        handleGroupEmployees(w, r, id)
                })(w, r)
                return
        }

        switch r.Method {
        case "PUT":
                requireSudo(func(w http.ResponseWriter, r *http.Request) {
                        handleUpdateGroup(w, r, id)
                })(w, r)
        case "DELETE":
                requireSudo(func(w http.ResponseWriter, r *http.Request) {
                        handleDeleteGroup(w, r, id)
                })(w, r)
        default:
                respondError(w, 405, "Method not allowed")
        }
}

func handleGroupEmployees(w http.ResponseWriter, r *http.Request, id int) {
        currentUser := getCurrentUser(r)
        if currentUser == nil {
                respondError(w, 401, "Tizimga kiring")
                return
        }
        if currentUser.Role != "sudo" {
                accessIds, _ := GetAdminGroupAccess(currentUser.ID)
                found := false
                for _, aid := range accessIds {
                        if aid == id {
                                found = true
                                break
                        }
                }
                if !found {
                        respondError(w, 403, "Bu guruhga ruxsatingiz yo'q")
                        return
                }
        }
        emps, err := GetGroupEmployees(id)
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        if emps == nil {
                emps = []Employee{}
        }
        respondJSON(w, 200, emps)
}

func handleUpdateGroup(w http.ResponseWriter, r *http.Request, id int) {
        var body map[string]interface{}
        json.NewDecoder(r.Body).Decode(&body)

        data := map[string]interface{}{}
        passwordChanged := false
        if v, ok := body["name"]; ok {
                data["name"] = v
        }
        if v, ok := body["description"]; ok {
                data["description"] = v
        }
        if v, ok := body["login"]; ok {
                data["login"] = v
                passwordChanged = true
        }
        if v, ok := body["password"]; ok {
                data["password"] = v
                passwordChanged = true
        }
        if v, ok := body["assignedAdminId"]; ok {
                if v == nil {
                        data["assigned_admin_id"] = nil
                } else {
                        data["assigned_admin_id"] = int(v.(float64))
                }
        }

        dbData := map[string]interface{}{}
        for k, v := range data {
                dbKey := k
                switch k {
                case "assignedAdminId":
                        dbKey = "assigned_admin_id"
                }
                dbData[dbKey] = v
        }

        updated, err := UpdateGroup(id, dbData)
        if err != nil {
                if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
                        respondError(w, 400, "Bu nom yoki login allaqachon mavjud")
                        return
                }
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        if updated == nil {
                respondError(w, 404, "Guruh topilmadi")
                return
        }
        if passwordChanged {
                RemoveGroupAccess(id)
        }
        respondJSON(w, 200, updated)
}

func handleDeleteGroup(w http.ResponseWriter, r *http.Request, id int) {
        DeleteGroup(id)
        respondJSON(w, 200, map[string]string{"message": "Guruh o'chirildi"})
}

func handleGroupJoin(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
                respondError(w, 405, "Method not allowed")
                return
        }
        var req JoinGroupRequest
        json.NewDecoder(r.Body).Decode(&req)
        if req.GroupID == 0 || req.Login == "" || req.Password == "" {
                respondError(w, 400, "Guruh ID, login va parol kiritilishi shart")
                return
        }
        group, err := GetGroup(req.GroupID)
        if err != nil || group == nil || !group.IsActive {
                respondError(w, 404, "Guruh topilmadi")
                return
        }
        if group.Login == nil || group.Password == nil || *group.Login != req.Login || *group.Password != req.Password {
                respondError(w, 401, "Login yoki parol noto'g'ri")
                return
        }
        currentUser := getCurrentUser(r)
        AddAdminGroupAccess(currentUser.ID, req.GroupID)
        respondJSON(w, 200, map[string]interface{}{"success": true, "message": "Guruhga qo'shildingiz"})
}

func handleGroupLeave(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
                respondError(w, 405, "Method not allowed")
                return
        }
        var req LeaveGroupRequest
        json.NewDecoder(r.Body).Decode(&req)
        if req.GroupID == 0 {
                respondError(w, 400, "Guruh ID kiritilishi shart")
                return
        }
        currentUser := getCurrentUser(r)
        RemoveAdminGroupAccessForAdmin(currentUser.ID, req.GroupID)
        respondJSON(w, 200, map[string]interface{}{"success": true, "message": "Guruhdan chiqildingiz"})
}

func handleMyGroups(w http.ResponseWriter, r *http.Request) {
        if r.Method != "GET" {
                respondError(w, 405, "Method not allowed")
                return
        }
        currentUser := getCurrentUser(r)
        if currentUser == nil {
                respondError(w, 401, "Tizimga kiring")
                return
        }
        allGroups, err := GetGroups()
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        if currentUser.Role == "sudo" {
                respondJSON(w, 200, allGroups)
                return
        }
        accessIds, _ := GetAdminGroupAccess(currentUser.ID)
        accessSet := map[int]bool{}
        for _, id := range accessIds {
                accessSet[id] = true
        }
        var myGroups []map[string]interface{}
        for _, g := range allGroups {
                if accessSet[g.ID] {
                        myGroups = append(myGroups, map[string]interface{}{
                                "id": g.ID, "name": g.Name, "description": g.Description,
                                "createdBy": g.CreatedBy, "assignedAdminId": g.AssignedAdminID,
                                "isActive": g.IsActive, "employeeCount": g.EmployeeCount,
                                "assignedAdminName": g.AssignedAdminName,
                                "login": "***", "password": "***",
                        })
                }
        }
        if myGroups == nil {
                myGroups = []map[string]interface{}{}
        }
        respondJSON(w, 200, myGroups)
}

func handleEmployees(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case "GET":
                requireAuth(handleGetEmployees)(w, r)
        case "POST":
                requireAuth(handleCreateEmployee)(w, r)
        default:
                respondError(w, 405, "Method not allowed")
        }
}

func handleGetEmployees(w http.ResponseWriter, r *http.Request) {
        currentUser := getCurrentUser(r)
        if currentUser == nil {
                respondError(w, 401, "Tizimga kiring")
                return
        }
        emps, err := GetEmployees()
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        if currentUser.Role == "sudo" {
                if emps == nil {
                        emps = []EmployeeWithGroup{}
                }
                respondJSON(w, 200, emps)
                return
        }
        accessIds, _ := GetAdminGroupAccess(currentUser.ID)
        if len(accessIds) == 0 {
                respondJSON(w, 200, []interface{}{})
                return
        }
        accessSet := map[int]bool{}
        for _, id := range accessIds {
                accessSet[id] = true
        }
        var filtered []EmployeeWithGroup
        for _, e := range emps {
                if e.GroupID != nil && accessSet[*e.GroupID] {
                        filtered = append(filtered, e)
                }
        }
        if filtered == nil {
                filtered = []EmployeeWithGroup{}
        }
        respondJSON(w, 200, filtered)
}

func handleCreateEmployee(w http.ResponseWriter, r *http.Request) {
        currentUser := getCurrentUser(r)
        if currentUser == nil {
                respondError(w, 401, "Tizimga kiring")
                return
        }
        var req CreateEmployeeRequest
        json.NewDecoder(r.Body).Decode(&req)
        if req.EmployeeNo == "" {
                respondError(w, 400, "Xodim raqami kiritilishi shart")
                return
        }
        if req.FullName == "" {
                respondError(w, 400, "Xodim ismi kiritilishi shart")
                return
        }
        if currentUser.Role != "sudo" && req.GroupID != nil {
                accessIds, _ := GetAdminGroupAccess(currentUser.ID)
                found := false
                for _, id := range accessIds {
                        if id == *req.GroupID {
                                found = true
                                break
                        }
                }
                if !found {
                        respondError(w, 403, "Bu guruhga ruxsatingiz yo'q")
                        return
                }
        }
        existing, _ := GetEmployeeByNo(req.EmployeeNo)
        if existing != nil {
                respondError(w, 400, "Bu xodim raqami allaqachon mavjud")
                return
        }
        if req.GroupID != nil {
                group, _ := GetGroup(*req.GroupID)
                if group == nil || !group.IsActive {
                        respondError(w, 400, "Tanlangan guruh topilmadi yoki faol emas")
                        return
                }
        }
        emp, err := CreateEmployee(req.EmployeeNo, req.FullName, req.Position, req.GroupID, req.Phone, true, false)
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        respondJSON(w, 200, emp)
}

func handleEmployeeByID(w http.ResponseWriter, r *http.Request) {
        idStr := strings.TrimPrefix(r.URL.Path, "/api/employees/")
        id, err := strconv.Atoi(idStr)
        if err != nil {
                respondError(w, 400, "Noto'g'ri ID")
                return
        }
        switch r.Method {
        case "GET":
                requireAuth(func(w http.ResponseWriter, r *http.Request) {
                        handleGetEmployee(w, r, id)
                })(w, r)
        case "PATCH":
                requireAuth(func(w http.ResponseWriter, r *http.Request) {
                        handleUpdateEmployee(w, r, id)
                })(w, r)
        case "DELETE":
                requireAuth(func(w http.ResponseWriter, r *http.Request) {
                        handleDeleteEmployee(w, r, id)
                })(w, r)
        default:
                respondError(w, 405, "Method not allowed")
        }
}

func handleGetEmployee(w http.ResponseWriter, r *http.Request, id int) {
        currentUser := getCurrentUser(r)
        if currentUser == nil {
                respondError(w, 401, "Tizimga kiring")
                return
        }
        emp, err := GetEmployee(id)
        if err != nil || emp == nil {
                respondError(w, 404, "Xodim topilmadi")
                return
        }
        if currentUser.Role != "sudo" {
                accessIds, _ := GetAdminGroupAccess(currentUser.ID)
                if emp.GroupID == nil {
                        respondError(w, 403, "Bu xodimga ruxsatingiz yo'q")
                        return
                }
                found := false
                for _, aid := range accessIds {
                        if aid == *emp.GroupID {
                                found = true
                                break
                        }
                }
                if !found {
                        respondError(w, 403, "Bu xodimga ruxsatingiz yo'q")
                        return
                }
        }
        respondJSON(w, 200, emp)
}

func handleUpdateEmployee(w http.ResponseWriter, r *http.Request, id int) {
        currentUser := getCurrentUser(r)
        if currentUser == nil {
                respondError(w, 401, "Tizimga kiring")
                return
        }

        var body map[string]interface{}
        json.NewDecoder(r.Body).Decode(&body)

        if currentUser.Role != "sudo" {
                emp, _ := GetEmployee(id)
                if emp == nil {
                        respondError(w, 404, "Xodim topilmadi")
                        return
                }
                accessIds, _ := GetAdminGroupAccess(currentUser.ID)
                if emp.GroupID == nil {
                        respondError(w, 403, "Bu xodimga ruxsatingiz yo'q")
                        return
                }
                found := false
                for _, aid := range accessIds {
                        if aid == *emp.GroupID {
                                found = true
                                break
                        }
                }
                if !found {
                        respondError(w, 403, "Bu xodimga ruxsatingiz yo'q")
                        return
                }
                if newGID, ok := body["groupId"]; ok && newGID != nil {
                        newGIDInt := int(newGID.(float64))
                        gFound := false
                        for _, aid := range accessIds {
                                if aid == newGIDInt {
                                        gFound = true
                                        break
                                }
                        }
                        if !gFound {
                                respondError(w, 403, "Bu guruhga ruxsatingiz yo'q")
                                return
                        }
                }
        }

        dbData := map[string]interface{}{}
        fieldMap := map[string]string{
                "fullName":   "full_name",
                "employeeNo": "employee_no",
                "position":   "position",
                "groupId":    "group_id",
                "phone":      "phone",
                "photoUrl":   "photo_url",
                "isActive":   "is_active",
                "hikvisionSynced": "hikvision_synced",
                "telegramUserId": "telegram_user_id",
        }
        for k, v := range body {
                if dbKey, ok := fieldMap[k]; ok {
                        dbData[dbKey] = v
                }
        }

        emp, err := UpdateEmployee(id, dbData)
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        if emp == nil {
                respondError(w, 404, "Xodim topilmadi")
                return
        }
        respondJSON(w, 200, emp)
}

func handleDeleteEmployee(w http.ResponseWriter, r *http.Request, id int) {
        currentUser := getCurrentUser(r)
        if currentUser == nil {
                respondError(w, 401, "Tizimga kiring")
                return
        }
        if currentUser.Role != "sudo" {
                emp, _ := GetEmployee(id)
                if emp == nil {
                        respondError(w, 404, "Xodim topilmadi")
                        return
                }
                accessIds, _ := GetAdminGroupAccess(currentUser.ID)
                if emp.GroupID == nil {
                        respondError(w, 403, "Bu xodimni o'chirishga ruxsatingiz yo'q")
                        return
                }
                found := false
                for _, aid := range accessIds {
                        if aid == *emp.GroupID {
                                found = true
                                break
                        }
                }
                if !found {
                        respondError(w, 403, "Bu xodimni o'chirishga ruxsatingiz yo'q")
                        return
                }
        }
        DeleteEmployee(id)
        respondJSON(w, 200, map[string]string{"message": "Xodim o'chirildi"})
}

func handleAttendance(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case "GET":
                requireAuth(handleGetAttendance)(w, r)
        case "POST":
                requireAuth(handleCreateAttendance)(w, r)
        default:
                respondError(w, 405, "Method not allowed")
        }
}

func handleGetAttendance(w http.ResponseWriter, r *http.Request) {
        currentUser := getCurrentUser(r)
        if currentUser == nil {
                respondError(w, 401, "Tizimga kiring")
                return
        }
        loc := time.FixedZone("UZ", 5*60*60)
        date := r.URL.Query().Get("date")
        if date == "" {
                date = time.Now().In(loc).Format("2006-01-02")
        }
        var groupIDPtr *int
        if gid := r.URL.Query().Get("groupId"); gid != "" {
                v, _ := strconv.Atoi(gid)
                groupIDPtr = &v
        }
        records, err := GetAttendanceByDate(date, groupIDPtr)
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        if currentUser.Role == "sudo" {
                if records == nil {
                        records = []AttendanceWithEmployee{}
                }
                respondJSON(w, 200, records)
                return
        }
        accessIds, _ := GetAdminGroupAccess(currentUser.ID)
        if len(accessIds) == 0 {
                respondJSON(w, 200, []interface{}{})
                return
        }
        accessSet := map[int]bool{}
        for _, id := range accessIds {
                accessSet[id] = true
        }
        var filtered []AttendanceWithEmployee
        for _, rec := range records {
                if rec.GroupID != nil && accessSet[*rec.GroupID] {
                        filtered = append(filtered, rec)
                }
        }
        if filtered == nil {
                filtered = []AttendanceWithEmployee{}
        }
        respondJSON(w, 200, filtered)
}

func handleCreateAttendance(w http.ResponseWriter, r *http.Request) {
        var req ManualAttendanceRequest
        json.NewDecoder(r.Body).Decode(&req)
        if req.EmployeeID == 0 || req.EmployeeNo == "" {
                respondError(w, 400, "Xodim ID va raqami kiritilishi shart")
                return
        }
        status := req.Status
        if status == "" {
                status = "check_in"
        }
        deviceName := "Manual"
        record, err := CreateAttendance(req.EmployeeID, req.EmployeeNo, time.Now(), status, nil, nil, &deviceName)
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        respondJSON(w, 200, record)
}

func handleAttendanceReport(w http.ResponseWriter, r *http.Request) {
        if r.Method != "GET" {
                respondError(w, 405, "Method not allowed")
                return
        }
        currentUser := getCurrentUser(r)
        if currentUser == nil {
                respondError(w, 401, "Tizimga kiring")
                return
        }
        startDate := r.URL.Query().Get("startDate")
        endDate := r.URL.Query().Get("endDate")
        if startDate == "" || endDate == "" {
                respondError(w, 400, "startDate va endDate talab qilinadi")
                return
        }

        var accessGroupIds []int
        if currentUser.Role != "sudo" {
                var err error
                accessGroupIds, err = GetAdminGroupAccess(currentUser.ID)
                if err != nil || len(accessGroupIds) == 0 {
                        respondJSON(w, 200, map[string]interface{}{"groups": []interface{}{}, "employees": []interface{}{}, "attendance": []interface{}{}})
                        return
                }
        }

        allGroups, _ := GetGroups()
        allEmployees, _ := GetEmployees()

        accessSet := map[int]bool{}
        if accessGroupIds != nil {
                for _, id := range accessGroupIds {
                        accessSet[id] = true
                }
        }

        var filteredGroups []AttendanceReportGroup
        for _, g := range allGroups {
                if accessGroupIds == nil || accessSet[g.ID] {
                        filteredGroups = append(filteredGroups, AttendanceReportGroup{ID: g.ID, Name: g.Name})
                }
        }

        var filteredEmployees []AttendanceReportEmployee
        for _, e := range allEmployees {
                if accessGroupIds == nil || (e.GroupID != nil && accessSet[*e.GroupID]) {
                        filteredEmployees = append(filteredEmployees, AttendanceReportEmployee{ID: e.ID, FullName: e.FullName, GroupID: e.GroupID})
                }
        }

        records, _ := GetAttendanceByDateRange(startDate, endDate, accessGroupIds)
        var attRecs []AttendanceReportRecord
        for _, rec := range records {
                attRecs = append(attRecs, AttendanceReportRecord{
                        EmployeeID: rec.EmployeeID,
                        EventTime:  rec.EventTime.Format(time.RFC3339),
                        Status:     rec.Status,
                })
        }

        if filteredGroups == nil {
                filteredGroups = []AttendanceReportGroup{}
        }
        if filteredEmployees == nil {
                filteredEmployees = []AttendanceReportEmployee{}
        }
        if attRecs == nil {
                attRecs = []AttendanceReportRecord{}
        }

        respondJSON(w, 200, AttendanceReportResponse{
                Groups:     filteredGroups,
                Employees:  filteredEmployees,
                Attendance: attRecs,
        })
}

func handleAttendanceByEmployee(w http.ResponseWriter, r *http.Request) {
        if r.Method != "GET" {
                respondError(w, 405, "Method not allowed")
                return
        }
        idStr := strings.TrimPrefix(r.URL.Path, "/api/attendance/employee/")
        id, err := strconv.Atoi(idStr)
        if err != nil {
                respondError(w, 400, "Noto'g'ri ID")
                return
        }
        currentUser := getCurrentUser(r)
        if currentUser == nil {
                respondError(w, 401, "Tizimga kiring")
                return
        }
        if currentUser.Role != "sudo" {
                emp, _ := GetEmployee(id)
                if emp == nil {
                        respondError(w, 404, "Xodim topilmadi")
                        return
                }
                accessIds, _ := GetAdminGroupAccess(currentUser.ID)
                if emp.GroupID == nil {
                        respondError(w, 403, "Bu xodimga ruxsatingiz yo'q")
                        return
                }
                found := false
                for _, aid := range accessIds {
                        if aid == *emp.GroupID {
                                found = true
                                break
                        }
                }
                if !found {
                        respondError(w, 403, "Bu xodimga ruxsatingiz yo'q")
                        return
                }
        }
        records, err := GetAttendanceByEmployee(id)
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        if records == nil {
                records = []AttendanceRecord{}
        }
        respondJSON(w, 200, records)
}

var statusMap = map[string]string{
        "checkIn": "check_in", "checkOut": "check_out",
        "breakOut": "break_out", "breakIn": "break_in",
        "overtimeIn": "overtime_in", "overtimeOut": "overtime_out",
        "0": "check_in", "1": "check_out", "2": "break_out",
        "3": "break_in", "4": "overtime_in", "5": "overtime_out",
}

type eventData struct {
        EmployeeNoString string
        AttendanceStatus string
        EventTime        string
        PictureURL       string
        DeviceIP         string
        DeviceName       string
}

func processAttendanceEvent(data eventData) (interface{}, error) {
        emp, err := GetEmployeeByNo(data.EmployeeNoString)
        if err != nil || emp == nil {
                log.Printf("[Hikvision Event] Xodim topilmadi: %s", data.EmployeeNoString)
                return nil, nil
        }

        var eventDate time.Time
        if data.EventTime != "" {
                eventDate, err = time.Parse(time.RFC3339, data.EventTime)
                if err != nil {
                        eventDate, err = time.Parse("2006-01-02T15:04:05", data.EventTime)
                        if err != nil {
                                eventDate, err = time.Parse("2006-01-02T15:04:05+05:00", data.EventTime)
                                if err != nil {
                                        eventDate = time.Now()
                                }
                        }
                }
        } else {
                eventDate = time.Now()
        }

        todayRecords, _ := GetTodayAttendanceForEmployee(emp.ID)
        isFirstToday := len(todayRecords) == 0

        if !isFirstToday {
                if emp.TelegramUserID != nil && *emp.TelegramUserID != "" {
                        var firstTime *time.Time
                        if len(todayRecords) > 0 {
                                firstTime = &todayRecords[0].EventTime
                        }
                        go SendEmployeeNotification(*emp.TelegramUserID, emp.FullName, eventDate, false, firstTime, nil)
                }
                return "duplicate", nil
        }

        status := statusMap[data.AttendanceStatus]
        if status == "" {
                status = "check_in"
        }

        var photoURL, deviceIP, deviceName *string
        if data.PictureURL != "" {
                photoURL = &data.PictureURL
        }
        if data.DeviceIP != "" {
                deviceIP = &data.DeviceIP
        }
        if data.DeviceName != "" {
                deviceName = &data.DeviceName
        }

        record, err := CreateAttendance(emp.ID, data.EmployeeNoString, eventDate, status, photoURL, deviceIP, deviceName)
        if err != nil {
                return nil, err
        }

        if !emp.HikvisionSynced {
                UpdateEmployee(emp.ID, map[string]interface{}{"hikvision_synced": true})
        }

        var groupName string
        if emp.GroupID != nil {
                group, _ := GetGroup(*emp.GroupID)
                if group != nil {
                        groupName = group.Name
                }
        }

        go SendAttendanceNotification(emp.FullName, data.EmployeeNoString, status, record.EventTime, groupName)

        if emp.TelegramUserID != nil && *emp.TelegramUserID != "" {
                periodStats := get10DayStats(emp.ID)
                go SendEmployeeNotification(*emp.TelegramUserID, emp.FullName, record.EventTime, true, nil, periodStats)
        }

        if emp.GroupID != nil {
                go sendAdminGroupNotification(emp, *emp.GroupID, groupName)
        }

        return record, nil
}

func get10DayStats(employeeID int) *struct{ Came, Total int } {
        loc := time.FixedZone("UZ", 5*60*60)
        nowUz := time.Now().In(loc)
        today := time.Date(nowUz.Year(), nowUz.Month(), nowUz.Day(), 0, 0, 0, 0, loc)

        total := 0
        came := 0
        for i := 0; i < 10; i++ {
                d := today.AddDate(0, 0, -i)
                if d.Weekday() == time.Sunday {
                        continue
                }
                total++
                dayEnd := d.Add(24*time.Hour - time.Millisecond)
                var cnt int
                DB.QueryRow(`SELECT COUNT(*) FROM attendance_records WHERE employee_id = $1 AND event_time >= $2 AND event_time <= $3`,
                        employeeID, d, dayEnd).Scan(&cnt)
                if cnt > 0 {
                        came++
                }
        }
        return &struct{ Came, Total int }{Came: came, Total: total}
}

func sendAdminGroupNotification(emp *Employee, groupID int, groupName string) {
        group, _ := GetGroup(groupID)
        if group == nil || group.AssignedAdminID == nil {
                return
        }
        admin, _ := GetUser(*group.AssignedAdminID)
        if admin == nil || admin.TelegramUserID == nil || *admin.TelegramUserID == "" {
                return
        }

        groupEmps, _ := GetGroupEmployees(groupID)

        loc := time.FixedZone("UZ", 5*60*60)
        nowUz := time.Now().In(loc)
        uzDateStr := nowUz.Format("2006-01-02")
        todayStart, _ := time.Parse("2006-01-02T15:04:05-07:00", uzDateStr+"T00:00:00+05:00")
        todayEnd, _ := time.Parse("2006-01-02T15:04:05-07:00", uzDateStr+"T23:59:59+05:00")

        rows, err := DB.Query(`SELECT DISTINCT employee_id FROM attendance_records WHERE event_time >= $1 AND event_time <= $2`, todayStart, todayEnd)
        if err != nil {
                return
        }
        defer rows.Close()
        presentIDs := map[int]bool{}
        for rows.Next() {
                var eid int
                rows.Scan(&eid)
                presentIDs[eid] = true
        }

        var cameList, notCameList []string
        for _, e := range groupEmps {
                if presentIDs[e.ID] {
                        cameList = append(cameList, e.FullName)
                } else {
                        notCameList = append(notCameList, e.FullName)
                }
        }

        SendAdminNotification(*admin.TelegramUserID, emp.FullName, groupName, cameList, notCameList)
}

type xmlEventAlert struct {
        XMLName              xml.Name             `xml:"EventNotificationAlert"`
        EventType            string               `xml:"eventType"`
        DateTime             string               `xml:"dateTime"`
        IPAddress            string               `xml:"ipAddress"`
        AccessControllerEvent *xmlACEvent          `xml:"AccessControllerEvent"`
}

type xmlACEvent struct {
        EmployeeNoString string `xml:"employeeNoString"`
        AttendanceStatus string `xml:"attendanceStatus"`
        DeviceName       string `xml:"deviceName"`
}

func extractFromXML(data []byte) *eventData {
        var alert xmlEventAlert
        if err := xml.Unmarshal(data, &alert); err != nil {
                return nil
        }
        if alert.EventType != "AccessControllerEvent" {
                return nil
        }
        if alert.AccessControllerEvent == nil || alert.AccessControllerEvent.EmployeeNoString == "" {
                return nil
        }
        return &eventData{
                EmployeeNoString: alert.AccessControllerEvent.EmployeeNoString,
                AttendanceStatus: alert.AccessControllerEvent.AttendanceStatus,
                EventTime:        alert.DateTime,
                DeviceIP:         alert.IPAddress,
                DeviceName:       alert.AccessControllerEvent.DeviceName,
        }
}

func extractFromJSON(body map[string]interface{}) *eventData {
        if et, ok := body["eventType"].(string); ok && et == "heartBeat" {
                return nil
        }

        if alert, ok := body["EventNotificationAlert"].(map[string]interface{}); ok {
                if et, ok := alert["eventType"].(string); ok && et == "heartBeat" {
                        return nil
                }
                acEvent, ok := alert["AccessControllerEvent"].(map[string]interface{})
                if !ok || acEvent["employeeNoString"] == nil {
                        return nil
                }
                return &eventData{
                        EmployeeNoString: fmt.Sprintf("%v", acEvent["employeeNoString"]),
                        AttendanceStatus: fmt.Sprintf("%v", acEvent["attendanceStatus"]),
                        EventTime:        fmt.Sprintf("%v", alert["dateTime"]),
                        DeviceIP:         fmt.Sprintf("%v", alert["ipAddress"]),
                        DeviceName:       fmt.Sprintf("%v", acEvent["deviceName"]),
                }
        }

        if et, ok := body["eventType"].(string); ok && et == "AccessControllerEvent" {
                acEvent, ok := body["AccessControllerEvent"].(map[string]interface{})
                if !ok || acEvent["employeeNoString"] == nil {
                        return nil
                }
                return &eventData{
                        EmployeeNoString: fmt.Sprintf("%v", acEvent["employeeNoString"]),
                        AttendanceStatus: fmt.Sprintf("%v", acEvent["attendanceStatus"]),
                        EventTime:        fmt.Sprintf("%v", body["dateTime"]),
                        DeviceIP:         fmt.Sprintf("%v", body["ipAddress"]),
                        DeviceName:       fmt.Sprintf("%v", acEvent["deviceName"]),
                }
        }

        if empNo, ok := body["employeeNoString"]; ok {
                ed := &eventData{EmployeeNoString: fmt.Sprintf("%v", empNo)}
                if v, ok := body["attendanceStatus"]; ok {
                        ed.AttendanceStatus = fmt.Sprintf("%v", v)
                }
                if v, ok := body["eventTime"]; ok {
                        ed.EventTime = fmt.Sprintf("%v", v)
                } else if v, ok := body["dateTime"]; ok {
                        ed.EventTime = fmt.Sprintf("%v", v)
                }
                if v, ok := body["deviceIp"]; ok {
                        ed.DeviceIP = fmt.Sprintf("%v", v)
                } else if v, ok := body["ipAddress"]; ok {
                        ed.DeviceIP = fmt.Sprintf("%v", v)
                }
                if v, ok := body["deviceName"]; ok {
                        ed.DeviceName = fmt.Sprintf("%v", v)
                }
                return ed
        }

        return nil
}

func handleHikvisionEvent(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
                respondError(w, 405, "Method not allowed")
                return
        }

        contentType := r.Header.Get("Content-Type")
        var ed *eventData
        isHeartbeat := false

        if strings.Contains(contentType, "multipart") {
                mediaType, params, err := mime.ParseMediaType(contentType)
                if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
                        respondJSON(w, 200, map[string]string{"message": "Event qabul qilindi (tanilmagan format)"})
                        return
                }
                reader := multipart.NewReader(r.Body, params["boundary"])
                for {
                        part, err := reader.NextPart()
                        if err == io.EOF {
                                break
                        }
                        if err != nil {
                                break
                        }
                        partCT := part.Header.Get("Content-Type")
                        data, _ := io.ReadAll(part)
                        part.Close()

                        if strings.Contains(partCT, "xml") || strings.Contains(partCT, "text") {
                                ed = extractFromXML(data)
                                if ed != nil {
                                        break
                                }
                        } else if strings.Contains(partCT, "json") {
                                var body map[string]interface{}
                                if json.Unmarshal(data, &body) == nil {
                                        if et, ok := body["eventType"].(string); ok && et == "heartBeat" {
                                                isHeartbeat = true
                                                continue
                                        }
                                        if et, ok := body["eventType"].(string); ok && et == "AccessControllerEvent" {
                                                if ac, ok := body["AccessControllerEvent"].(map[string]interface{}); ok {
                                                        if ac["employeeNoString"] == nil {
                                                                isHeartbeat = true
                                                                continue
                                                        }
                                                }
                                        }
                                        ed = extractFromJSON(body)
                                        if ed != nil {
                                                break
                                        }
                                }
                        }

                        dataStr := string(data)
                        if strings.TrimSpace(dataStr) != "" && strings.HasPrefix(strings.TrimSpace(dataStr), "<") {
                                ed = extractFromXML(data)
                                if ed != nil {
                                        break
                                }
                        }
                        if strings.TrimSpace(dataStr) != "" && strings.HasPrefix(strings.TrimSpace(dataStr), "{") {
                                var body map[string]interface{}
                                if json.Unmarshal(data, &body) == nil {
                                        ed = extractFromJSON(body)
                                        if ed != nil {
                                                break
                                        }
                                }
                        }
                }
        } else {
                rawBody, _ := io.ReadAll(r.Body)
                bodyStr := string(rawBody)

                if strings.Contains(contentType, "xml") || (strings.TrimSpace(bodyStr) != "" && strings.HasPrefix(strings.TrimSpace(bodyStr), "<")) {
                        ed = extractFromXML(rawBody)
                }

                if ed == nil {
                        var body map[string]interface{}
                        if json.Unmarshal(rawBody, &body) == nil {
                                ed = extractFromJSON(body)
                        }
                }
        }

        if isHeartbeat && ed == nil {
                respondJSON(w, 200, map[string]string{"message": "OK"})
                return
        }

        if ed == nil || ed.EmployeeNoString == "" {
                log.Println("[Hikvision Event] Event parse qilib bo'lmadi")
                respondJSON(w, 200, map[string]string{"message": "Event qabul qilindi (tanilmagan format)"})
                return
        }

        log.Printf("[Hikvision Event] Xodim: %s, Status: %s", ed.EmployeeNoString, ed.AttendanceStatus)

        result, err := processAttendanceEvent(*ed)
        if err != nil {
                log.Printf("[Hikvision Event] Xatolik: %v", err)
                respondJSON(w, 200, map[string]string{"message": "Event qabul qilindi"})
                return
        }
        if result == nil {
                respondJSON(w, 200, map[string]string{"message": "Xodim topilmadi"})
                return
        }
        if result == "duplicate" {
                respondJSON(w, 200, map[string]string{"message": "Takroriy event - oldin qayd etilgan"})
                return
        }
        respondJSON(w, 200, map[string]interface{}{"message": "Davomat qayd etildi", "record": result})
}

func handleHikvisionSettings(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case "GET":
                requireSudo(handleGetHikvisionSettings)(w, r)
        case "POST":
                requireSudo(handleSaveHikvisionSettings)(w, r)
        default:
                respondError(w, 405, "Method not allowed")
        }
}

func handleGetHikvisionSettings(w http.ResponseWriter, r *http.Request) {
        ip, _ := GetSetting("hikvision_ip")
        username, _ := GetSetting("hikvision_username")
        hasPw, _ := GetSetting("hikvision_password")
        autoSync, _ := GetSetting("hikvision_auto_sync")

        ipVal := ""
        if ip != nil {
                ipVal = *ip
        }
        usernameVal := ""
        if username != nil {
                usernameVal = *username
        }

        respondJSON(w, 200, HikvisionSettingsResponse{
                IP:          ipVal,
                Username:    usernameVal,
                HasPassword: hasPw != nil,
                AutoSync:    autoSync != nil && *autoSync == "true",
        })
}

func handleSaveHikvisionSettings(w http.ResponseWriter, r *http.Request) {
        var req HikvisionSettingsRequest
        json.NewDecoder(r.Body).Decode(&req)
        if req.IP == "" {
                respondError(w, 400, "Kamera IP manzili kiritilishi shart")
                return
        }
        if req.Username == "" {
                respondError(w, 400, "Login kiritilishi shart")
                return
        }
        SetSetting("hikvision_ip", req.IP)
        SetSetting("hikvision_username", req.Username)
        if req.Password != "" {
                SetSetting("hikvision_password", req.Password)
        }
        if req.AutoSync != nil {
                SetSetting("hikvision_auto_sync", fmt.Sprintf("%v", *req.AutoSync))
                if *req.AutoSync {
                        StartAutoSync(10)
                } else {
                        StopAutoSync()
                }
        }
        respondJSON(w, 200, map[string]string{"message": "Kamera sozlamalari saqlandi"})
}

func handleHikvisionTest(w http.ResponseWriter, r *http.Request) {
        if r.Method != "GET" {
                respondError(w, 405, "Method not allowed")
                return
        }
        result := TestCameraConnection()
        respondJSON(w, 200, result)
}

func handleHikvisionSync(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
                respondError(w, 405, "Method not allowed")
                return
        }
        result := SyncEmployeesWithCamera()
        respondJSON(w, 200, result)
}

func handleHikvisionCameraUsers(w http.ResponseWriter, r *http.Request) {
        if r.Method != "GET" {
                respondError(w, 405, "Method not allowed")
                return
        }
        users, err := GetCameraUsers()
        if err != nil {
                respondError(w, 500, "Server xatoligi: "+err.Error())
                return
        }
        if users == nil {
                users = []HikvisionUser{}
        }
        respondJSON(w, 200, users)
}

func handleTelegramToken(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case "GET":
                requireSudo(handleGetTelegramToken)(w, r)
        case "POST":
                requireSudo(handleSetTelegramToken)(w, r)
        default:
                respondError(w, 405, "Method not allowed")
        }
}

func handleGetTelegramToken(w http.ResponseWriter, r *http.Request) {
        token, _ := GetSetting("telegram_bot_token")
        val := ""
        if token != nil {
                val = *token
        }
        respondJSON(w, 200, map[string]string{"token": val})
}

func handleSetTelegramToken(w http.ResponseWriter, r *http.Request) {
        var req TelegramTokenRequest
        json.NewDecoder(r.Body).Decode(&req)
        if len(req.Token) < 10 {
                respondError(w, 400, "Bot token noto'g'ri")
                return
        }
        SetSetting("telegram_bot_token", strings.TrimSpace(req.Token))
        RestartTelegramBot()
        respondJSON(w, 200, map[string]string{"message": "Bot token saqlandi va bot qayta ishga tushdi"})
}

func handleTelegramPanel(w http.ResponseWriter, r *http.Request) {
        if r.Method != "GET" {
                respondError(w, 405, "Method not allowed")
                return
        }
        allGroups, _ := GetGroups()
        allEmployees, _ := GetEmployees()
        allAdmins, _ := GetAdmins()

        loc := time.FixedZone("UZ", 5*60*60)
        nowUz := time.Now().In(loc)
        uzDateStr := nowUz.Format("2006-01-02")
        todayStart, _ := time.Parse("2006-01-02T15:04:05-07:00", uzDateStr+"T00:00:00+05:00")
        todayEnd, _ := time.Parse("2006-01-02T15:04:05-07:00", uzDateStr+"T23:59:59+05:00")

        rows, _ := DB.Query(`SELECT DISTINCT employee_id FROM attendance_records WHERE event_time >= $1 AND event_time <= $2`, todayStart, todayEnd)
        presentIDs := map[int]bool{}
        if rows != nil {
                for rows.Next() {
                        var eid int
                        rows.Scan(&eid)
                        presentIDs[eid] = true
                }
                rows.Close()
        }

        var groupsData []TelegramPanelGroup
        for _, g := range allGroups {
                pg := TelegramPanelGroup{ID: g.ID, Name: g.Name, Came: []TelegramPanelEmployee{}, NotCame: []TelegramPanelEmployee{}, Employees: []TelegramPanelEmployee2{}}

                if g.AssignedAdminID != nil {
                        for _, a := range allAdmins {
                                if a.ID == *g.AssignedAdminID {
                                        pg.ResponsibleAdmin = &TelegramPanelAdmin{ID: a.ID, FullName: a.FullName, TelegramUserID: a.TelegramUserID}
                                        break
                                }
                        }
                }

                for _, e := range allEmployees {
                        if e.GroupID == nil || *e.GroupID != g.ID {
                                continue
                        }
                        pg.TotalEmployees++
                        pg.Employees = append(pg.Employees, TelegramPanelEmployee2{ID: e.ID, FullName: e.FullName, EmployeeNo: e.EmployeeNo, TelegramUserID: e.TelegramUserID})
                        if presentIDs[e.ID] {
                                pg.CameCount++
                                pg.Came = append(pg.Came, TelegramPanelEmployee{ID: e.ID, FullName: e.FullName, TelegramUserID: e.TelegramUserID})
                        } else {
                                pg.NotCameCount++
                                pg.NotCame = append(pg.NotCame, TelegramPanelEmployee{ID: e.ID, FullName: e.FullName, TelegramUserID: e.TelegramUserID})
                        }
                }
                groupsData = append(groupsData, pg)
        }

        if groupsData == nil {
                groupsData = []TelegramPanelGroup{}
        }

        adminsList := make([]map[string]interface{}, 0)
        for _, a := range allAdmins {
                adminsList = append(adminsList, map[string]interface{}{
                        "id": a.ID, "fullName": a.FullName, "telegramUserId": a.TelegramUserID,
                })
        }

        respondJSON(w, 200, map[string]interface{}{"groups": groupsData, "admins": adminsList})
}

func handleTelegramAdmin(w http.ResponseWriter, r *http.Request) {
        if r.Method != "PATCH" {
                respondError(w, 405, "Method not allowed")
                return
        }
        idStr := strings.TrimPrefix(r.URL.Path, "/api/telegram/admin/")
        id, err := strconv.Atoi(idStr)
        if err != nil {
                respondError(w, 400, "Noto'g'ri ID")
                return
        }
        var body struct {
                TelegramUserID *string `json:"telegramUserId"`
        }
        json.NewDecoder(r.Body).Decode(&body)
        UpdateUserTelegramID(id, body.TelegramUserID)
        respondJSON(w, 200, map[string]string{"message": "Admin Telegram ID yangilandi"})
}

func handleTelegramEmployee(w http.ResponseWriter, r *http.Request) {
        if r.Method != "PATCH" {
                respondError(w, 405, "Method not allowed")
                return
        }
        idStr := strings.TrimPrefix(r.URL.Path, "/api/telegram/employee/")
        id, err := strconv.Atoi(idStr)
        if err != nil {
                respondError(w, 400, "Noto'g'ri ID")
                return
        }
        var body struct {
                TelegramUserID *string `json:"telegramUserId"`
        }
        json.NewDecoder(r.Body).Decode(&body)
        UpdateEmployee(id, map[string]interface{}{"telegram_user_id": body.TelegramUserID})
        respondJSON(w, 200, map[string]string{"message": "Xodim Telegram ID yangilandi"})
}
