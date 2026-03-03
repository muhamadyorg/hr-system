package goserver

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type HikvisionConfig struct {
	IP       string
	Username string
	Password string
}

type HikvisionUser struct {
	EmployeeNo string `json:"employeeNo"`
	Name       string `json:"name"`
}

type SyncResult struct {
	Added           []string `json:"added"`
	Removed         []string `json:"removed"`
	Existing        []string `json:"existing"`
	Errors          []string `json:"errors"`
	CameraConnected bool     `json:"cameraConnected"`
}

var (
	isSyncing       bool
	syncMu          sync.Mutex
	autoSyncTicker  *time.Ticker
	autoSyncDone    chan bool
)

func getHikvisionConfig() (*HikvisionConfig, error) {
	ip, _ := GetSetting("hikvision_ip")
	username, _ := GetSetting("hikvision_username")
	password, _ := GetSetting("hikvision_password")

	if ip == nil || username == nil || password == nil {
		return nil, nil
	}
	return &HikvisionConfig{IP: *ip, Username: *username, Password: *password}, nil
}

func makeBasicAuth(username, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

func fetchWithAuth(url string, config *HikvisionConfig, method string, body []byte) (*http.Response, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", makeBasicAuth(config.Username, config.Password))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 401 {
		resp.Body.Close()
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}
		req2, _ := http.NewRequest(method, url, bodyReader)
		req2.Header.Set("Content-Type", "application/json")
		return client.Do(req2)
	}

	return resp, nil
}

func TestCameraConnection() map[string]interface{} {
	config, err := getHikvisionConfig()
	if err != nil || config == nil {
		return map[string]interface{}{"connected": false, "message": "Kamera sozlamalari topilmadi. IP, login va parolni kiriting."}
	}

	resp, err := fetchWithAuth(fmt.Sprintf("http://%s/ISAPI/System/deviceInfo?format=json", config.IP), config, "GET", nil)
	if err != nil {
		return map[string]interface{}{"connected": false, "message": fmt.Sprintf("Kameraga ulanib bo'lmadi: %s", err.Error())}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return map[string]interface{}{"connected": false, "message": "Kamera login yoki paroli noto'g'ri"}
	}
	if resp.StatusCode != 200 {
		return map[string]interface{}{"connected": false, "message": fmt.Sprintf("Kamera javob bermadi: %d", resp.StatusCode)}
	}

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)
	deviceInfo := data
	if di, ok := data["DeviceInfo"]; ok {
		deviceInfo = di.(map[string]interface{})
	}

	return map[string]interface{}{"connected": true, "message": "Kameraga muvaffaqiyatli ulandi", "deviceInfo": deviceInfo}
}

func GetCameraUsers() ([]HikvisionUser, error) {
	config, err := getHikvisionConfig()
	if err != nil || config == nil {
		return nil, fmt.Errorf("Kamera sozlamalari topilmadi")
	}

	body, _ := json.Marshal(map[string]interface{}{
		"UserInfoSearchCond": map[string]interface{}{
			"searchID":             "1",
			"searchResultPosition": 0,
			"maxResults":           1000,
		},
	})

	resp, err := fetchWithAuth(fmt.Sprintf("http://%s/ISAPI/AccessControl/UserInfo/Search?format=json", config.IP), config, "POST", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Kamera javob: %d", resp.StatusCode)
	}

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)

	var users []HikvisionUser
	if search, ok := data["UserInfoSearch"].(map[string]interface{}); ok {
		if userList, ok := search["UserInfo"].([]interface{}); ok {
			for _, u := range userList {
				um := u.(map[string]interface{})
				users = append(users, HikvisionUser{
					EmployeeNo: fmt.Sprintf("%v", um["employeeNo"]),
					Name:       fmt.Sprintf("%v", um["name"]),
				})
			}
		}
	}

	return users, nil
}

func addUserToCamera(employeeNo, name string) error {
	config, err := getHikvisionConfig()
	if err != nil || config == nil {
		return fmt.Errorf("Kamera sozlamalari topilmadi")
	}

	body, _ := json.Marshal(map[string]interface{}{
		"UserInfo": map[string]interface{}{
			"employeeNo": employeeNo,
			"name":       name,
			"userType":   "normal",
			"Valid": map[string]interface{}{
				"enable":    true,
				"beginTime": "2024-01-01T00:00:00",
				"endTime":   "2030-12-31T23:59:59",
			},
		},
	})

	resp, err := fetchWithAuth(fmt.Sprintf("http://%s/ISAPI/AccessControl/UserInfo/Record?format=json", config.IP), config, "POST", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Kamera javob: %d - %s", resp.StatusCode, string(b))
	}
	return nil
}

func deleteUserFromCamera(employeeNo string) error {
	config, err := getHikvisionConfig()
	if err != nil || config == nil {
		return fmt.Errorf("Kamera sozlamalari topilmadi")
	}

	body, _ := json.Marshal(map[string]interface{}{
		"UserInfoDelCond": map[string]interface{}{
			"EmployeeNoList": []map[string]string{{"employeeNo": employeeNo}},
		},
	})

	resp, err := fetchWithAuth(fmt.Sprintf("http://%s/ISAPI/AccessControl/UserInfo/Delete?format=json", config.IP), config, "PUT", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Kamera javob: %d - %s", resp.StatusCode, string(b))
	}
	return nil
}

func SyncEmployeesWithCamera() *SyncResult {
	result := &SyncResult{
		Added:    []string{},
		Removed:  []string{},
		Existing: []string{},
		Errors:   []string{},
	}

	syncMu.Lock()
	if isSyncing {
		syncMu.Unlock()
		result.Errors = append(result.Errors, "Sinxronizatsiya allaqachon davom etmoqda, kuting")
		return result
	}
	isSyncing = true
	syncMu.Unlock()

	defer func() {
		syncMu.Lock()
		isSyncing = false
		syncMu.Unlock()
	}()

	connTest := TestCameraConnection()
	if connTest["connected"] != true {
		result.Errors = append(result.Errors, connTest["message"].(string))
		return result
	}
	result.CameraConnected = true

	cameraUsers, err := GetCameraUsers()
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	webEmployees, err := GetEmployees()
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	cameraEmployeeNos := map[string]bool{}
	for _, u := range cameraUsers {
		cameraEmployeeNos[u.EmployeeNo] = true
	}

	webEmployeeNos := map[string]bool{}
	for _, e := range webEmployees {
		webEmployeeNos[e.EmployeeNo] = true
	}

	for _, emp := range webEmployees {
		if cameraEmployeeNos[emp.EmployeeNo] {
			result.Existing = append(result.Existing, emp.EmployeeNo)
			if !emp.HikvisionSynced {
				UpdateEmployee(emp.ID, map[string]interface{}{"hikvision_synced": true})
			}
		} else {
			err := addUserToCamera(emp.EmployeeNo, emp.FullName)
			if err == nil {
				result.Added = append(result.Added, emp.EmployeeNo)
				UpdateEmployee(emp.ID, map[string]interface{}{"hikvision_synced": true})
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): %s", emp.EmployeeNo, emp.FullName, err.Error()))
				if emp.HikvisionSynced {
					UpdateEmployee(emp.ID, map[string]interface{}{"hikvision_synced": false})
				}
			}
		}
	}

	for _, camUser := range cameraUsers {
		if !webEmployeeNos[camUser.EmployeeNo] {
			err := deleteUserFromCamera(camUser.EmployeeNo)
			if err == nil {
				result.Removed = append(result.Removed, camUser.EmployeeNo)
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("O'chirish %s: %s", camUser.EmployeeNo, err.Error()))
			}
		}
	}

	return result
}

func StartAutoSync(intervalMinutes int) {
	StopAutoSync()

	log.Printf("[Hikvision] Avtomatik sinxronizatsiya har %d daqiqada ishga tushadi", intervalMinutes)

	go func() {
		time.Sleep(5 * time.Second)
		config, _ := getHikvisionConfig()
		if config != nil {
			log.Println("[Hikvision] Birinchi sinxronizatsiya boshlanmoqda...")
			result := SyncEmployeesWithCamera()
			log.Printf("[Hikvision] Sinxronizatsiya: +%d qo'shildi, -%d o'chirildi, %d mavjud, %d xatolik",
				len(result.Added), len(result.Removed), len(result.Existing), len(result.Errors))
		}
	}()

	autoSyncTicker = time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	autoSyncDone = make(chan bool)

	go func() {
		for {
			select {
			case <-autoSyncDone:
				return
			case <-autoSyncTicker.C:
				config, _ := getHikvisionConfig()
				if config == nil {
					continue
				}
				result := SyncEmployeesWithCamera()
				if len(result.Added) > 0 || len(result.Removed) > 0 {
					log.Printf("[Hikvision] Avtomatik sinxronizatsiya: +%d, -%d", len(result.Added), len(result.Removed))
				}
			}
		}
	}()
}

func StopAutoSync() {
	if autoSyncTicker != nil {
		autoSyncTicker.Stop()
		autoSyncTicker = nil
	}
	if autoSyncDone != nil {
		select {
		case autoSyncDone <- true:
		default:
		}
		autoSyncDone = nil
	}
	log.Println("[Hikvision] Avtomatik sinxronizatsiya to'xtatildi")
}
