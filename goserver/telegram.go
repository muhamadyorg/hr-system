package goserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	telegramBotToken string
	subscribedChats  = map[int64]bool{}
	chatsMu          sync.RWMutex
	telegramRunning  bool
	telegramStopCh   chan bool
)

func InitTelegramBotFromDb() {
	token, _ := GetSetting("telegram_bot_token")
	if token != nil && len(*token) > 10 {
		startTelegramBot(*token)
	} else {
		log.Println("[Telegram] Bot token topilmadi (DB), bot ishga tushmadi")
	}
}

func RestartTelegramBot() {
	stopTelegramBot()
	InitTelegramBotFromDb()
}

func stopTelegramBot() {
	if telegramRunning && telegramStopCh != nil {
		select {
		case telegramStopCh <- true:
		default:
		}
	}
	telegramRunning = false
	chatsMu.Lock()
	subscribedChats = map[int64]bool{}
	chatsMu.Unlock()
}

func startTelegramBot(token string) {
	telegramBotToken = token
	telegramRunning = true
	telegramStopCh = make(chan bool, 1)

	log.Println("[Telegram] Bot ishga tushdi")

	loadSavedChatIds()

	go pollTelegram(token)
}

func pollTelegram(token string) {
	offset := 0
	client := &http.Client{Timeout: 35 * time.Second}

	for {
		select {
		case <-telegramStopCh:
			return
		default:
		}

		apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", token, offset)
		resp, err := client.Get(apiURL)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		var result struct {
			OK     bool `json:"ok"`
			Result []struct {
				UpdateID int `json:"update_id"`
				Message  *struct {
					Chat struct {
						ID int64 `json:"id"`
					} `json:"chat"`
					Text string `json:"text"`
				} `json:"message"`
			} `json:"result"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		resp.Body.Close()

		if !result.OK {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range result.Result {
			offset = update.UpdateID + 1
			if update.Message == nil {
				continue
			}

			chatID := update.Message.Chat.ID
			text := strings.TrimSpace(update.Message.Text)

			switch {
			case strings.HasPrefix(text, "/start"):
				chatsMu.Lock()
				subscribedChats[chatID] = true
				chatsMu.Unlock()
				saveChatId(chatID)
				sendTelegramMessage(token, chatID,
					"Assalomu alaykum! HR Davomat Bot ga xush kelibsiz.\n\n"+
						"Bu bot orqali xodimlar davomati haqida bildirishnomalar olasiz.\n\n"+
						"Buyruqlar:\n"+
						"/start - Botni ishga tushirish\n"+
						"/stats - Bugungi statistika\n"+
						"/today - Bugungi davomat ro'yxati\n"+
						"/help - Yordam\n"+
						"/stop - Bildirishnomalarni to'xtatish")

			case strings.HasPrefix(text, "/stop"):
				chatsMu.Lock()
				delete(subscribedChats, chatID)
				chatsMu.Unlock()
				removeChatId(chatID)
				sendTelegramMessage(token, chatID, "Bildirishnomalar to'xtatildi. Qayta boshlash uchun /start buyrug'ini yuboring.")

			case strings.HasPrefix(text, "/stats"):
				handleStatsCommand(token, chatID)

			case strings.HasPrefix(text, "/today"):
				handleTodayCommand(token, chatID)

			case strings.HasPrefix(text, "/help"):
				sendTelegramMessage(token, chatID,
					"HR Davomat Bot - Yordam\n\n"+
						"Bu bot Hikvision kamera orqali xodimlarning davomat ma'lumotlarini real vaqtda yuboradi.\n\n"+
						"Buyruqlar:\n"+
						"/start - Bildirishnomalarni yoqish\n"+
						"/stats - Bugungi statistika\n"+
						"/today - Bugungi davomat ro'yxati\n"+
						"/stop - Bildirishnomalarni to'xtatish\n"+
						"/help - Ushbu yordam\n\n"+
						"Bildirishnomalar:\n"+
						"Xodim kamera orqali yuzini skanerlasa, darhol xabar keladi.")
			}
		}
	}
}

func handleStatsCommand(token string, chatID int64) {
	stats, err := GetDashboardStats(nil)
	if err != nil {
		sendTelegramMessage(token, chatID, "Statistikani olishda xatolik yuz berdi.")
		return
	}

	loc := time.FixedZone("UZ", 5*60*60)
	today := time.Now().In(loc).Format("2006-01-02")

	msg := fmt.Sprintf("Bugungi statistika (%s)\n\nJami xodimlar: %d\nGuruhlar soni: %d\nBugun kelganlar: %d\nKelmagan: %d",
		today, stats.TotalEmployees, stats.TotalGroups, stats.TodayPresent, stats.TodayAbsent)

	sendTelegramMessage(token, chatID, msg)
}

func handleTodayCommand(token string, chatID int64) {
	loc := time.FixedZone("UZ", 5*60*60)
	today := time.Now().In(loc).Format("2006-01-02")
	records, err := GetAttendanceByDate(today, nil)
	if err != nil {
		sendTelegramMessage(token, chatID, "Davomat ma'lumotlarini olishda xatolik yuz berdi.")
		return
	}

	if len(records) == 0 {
		sendTelegramMessage(token, chatID, "Bugun hali hech kim kelmagan.")
		return
	}

	statusLabels := map[string]string{
		"check_in":     "Kirdi",
		"check_out":    "Chiqdi",
		"break_out":    "Tanaffus",
		"break_in":     "Tanaffusdan qaytdi",
		"overtime_in":  "Qo'shimcha ish",
		"overtime_out": "Qo'shimcha ishdan chiqdi",
	}

	grouped := map[string][]string{}
	order := []string{}
	for _, rec := range records {
		name := rec.FullName
		t := rec.EventTime.In(loc).Format("15:04")
		status := statusLabels[rec.Status]
		if status == "" {
			status = rec.Status
		}
		entry := fmt.Sprintf("%s - %s", t, status)
		if _, exists := grouped[name]; !exists {
			order = append(order, name)
		}
		grouped[name] = append(grouped[name], entry)
	}

	msg := fmt.Sprintf("Bugungi davomat (%s)\n\n", today)
	for _, name := range order {
		msg += fmt.Sprintf("%s\n", name)
		for _, event := range grouped[name] {
			msg += fmt.Sprintf("   %s\n", event)
		}
		msg += "\n"
	}
	msg += fmt.Sprintf("Jami: %d ta xodim", len(grouped))

	if len(msg) > 4000 {
		msg = msg[:4000] + "\n\n...davomi qisqartirildi"
	}

	sendTelegramMessage(token, chatID, msg)
}

func sendTelegramMessage(token string, chatID int64, text string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	data := url.Values{
		"chat_id": {strconv.FormatInt(chatID, 10)},
		"text":    {text},
	}
	http.PostForm(apiURL, data)
}

func SendAttendanceNotification(employeeName, employeeNo, status string, eventTime time.Time, groupName string) {
	if telegramBotToken == "" {
		return
	}

	chatsMu.RLock()
	chats := make([]int64, 0, len(subscribedChats))
	for id := range subscribedChats {
		chats = append(chats, id)
	}
	chatsMu.RUnlock()

	if len(chats) == 0 {
		return
	}

	statusLabels := map[string]string{
		"check_in":     "Ishga keldi",
		"check_out":    "Ishdan ketdi",
		"break_out":    "Tanaffusga chiqdi",
		"break_in":     "Tanaffusdan qaytdi",
		"overtime_in":  "Qo'shimcha ishga keldi",
		"overtime_out": "Qo'shimcha ishdan ketdi",
	}

	loc := time.FixedZone("UZ", 5*60*60)
	statusText := statusLabels[status]
	if statusText == "" {
		statusText = status
	}
	t := eventTime.In(loc).Format("15:04:05")
	d := eventTime.In(loc).Format("2006-01-02")

	msg := fmt.Sprintf("%s\n\n%s\n", statusText, employeeName)
	if groupName != "" {
		msg += fmt.Sprintf("Bo'lim: %s\n", groupName)
	}
	msg += fmt.Sprintf("Vaqt: %s\nSana: %s", t, d)

	for _, chatID := range chats {
		sendTelegramMessage(telegramBotToken, chatID, msg)
	}
}

func SendEmployeeNotification(telegramUserID string, employeeName string, eventTime time.Time, isFirstToday bool, firstEventTime *time.Time, periodStats *struct{ Came, Total int }) {
	if telegramBotToken == "" {
		return
	}

	chatID, err := strconv.ParseInt(telegramUserID, 10, 64)
	if err != nil {
		return
	}

	loc := time.FixedZone("UZ", 5*60*60)
	t := eventTime.In(loc).Format("15:04")

	if isFirstToday {
		msg := fmt.Sprintf("Assalomu alaykum, %s!\nIshxonaga xush kelibsiz!\nSoat: %s\n", employeeName, t)
		if periodStats != nil {
			msg += fmt.Sprintf("\nSo'nggi 10 kun:\nKelgan: %d kun\nKelmagan: %d kun", periodStats.Came, periodStats.Total-periodStats.Came)
		}
		sendTelegramMessage(telegramBotToken, chatID, msg)
	} else {
		firstTime := "—"
		if firstEventTime != nil {
			firstTime = firstEventTime.In(loc).Format("15:04")
		}
		msg := fmt.Sprintf("%s, bugun siz soat %s da o'tgansiz.", employeeName, firstTime)
		sendTelegramMessage(telegramBotToken, chatID, msg)
	}
}

func SendAdminNotification(adminTelegramUserID string, employeeName string, groupName string, cameList []string, notCameList []string) {
	if telegramBotToken == "" {
		return
	}

	chatID, err := strconv.ParseInt(adminTelegramUserID, 10, 64)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("%s guruhida\n%s keldi!\n\nBugungi holat:\n", groupName, employeeName)
	msg += fmt.Sprintf("Kelganlar (%d): %s\n", len(cameList), strings.Join(cameList, ", "))
	msg += fmt.Sprintf("Kelmaganlar (%d): %s", len(notCameList), strings.Join(notCameList, ", "))

	if len(msg) > 4000 {
		msg = msg[:4000] + "\n..."
	}

	sendTelegramMessage(telegramBotToken, chatID, msg)
}

const chatIDsKey = "telegram_chat_ids"

func saveChatId(chatID int64) {
	ids := loadChatIdsFromDb()
	for _, id := range ids {
		if id == chatID {
			return
		}
	}
	ids = append(ids, chatID)
	data, _ := json.Marshal(ids)
	SetSetting(chatIDsKey, string(data))
}

func removeChatId(chatID int64) {
	ids := loadChatIdsFromDb()
	var filtered []int64
	for _, id := range ids {
		if id != chatID {
			filtered = append(filtered, id)
		}
	}
	data, _ := json.Marshal(filtered)
	SetSetting(chatIDsKey, string(data))
}

func loadChatIdsFromDb() []int64 {
	val, _ := GetSetting(chatIDsKey)
	if val == nil {
		return nil
	}
	var ids []int64
	json.Unmarshal([]byte(*val), &ids)
	return ids
}

func loadSavedChatIds() {
	ids := loadChatIdsFromDb()
	chatsMu.Lock()
	for _, id := range ids {
		subscribedChats[id] = true
	}
	chatsMu.Unlock()
	if len(ids) > 0 {
		log.Printf("[Telegram] %d ta saqlangan chat yuklandi", len(ids))
	}
}
