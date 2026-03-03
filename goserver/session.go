package goserver

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

const sessionCookieName = "hr_session_id"

type SessionData struct {
	UserID int `json:"userId"`
}

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func CreateSession(w http.ResponseWriter, userID int) error {
	sid := generateSessionID()
	sessData, _ := json.Marshal(SessionData{UserID: userID})
	expire := time.Now().Add(30 * 24 * time.Hour)

	_, err := DB.Exec(`INSERT INTO "session" (sid, sess, expire) VALUES ($1, $2, $3) ON CONFLICT (sid) DO UPDATE SET sess = $2, expire = $3`,
		sid, string(sessData), expire)
	if err != nil {
		return err
	}

	secure := false
	if os.Getenv("NODE_ENV") == "production" {
		secure = true
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sid,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func GetSessionUserID(r *http.Request) (int, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return 0, false
	}

	var sessJSON string
	var expire time.Time
	err = DB.QueryRow(`SELECT sess, expire FROM "session" WHERE sid = $1`, cookie.Value).Scan(&sessJSON, &expire)
	if err == sql.ErrNoRows {
		return 0, false
	}
	if err != nil {
		return 0, false
	}

	if time.Now().After(expire) {
		DB.Exec(`DELETE FROM "session" WHERE sid = $1`, cookie.Value)
		return 0, false
	}

	var data SessionData
	if err := json.Unmarshal([]byte(sessJSON), &data); err != nil {
		return 0, false
	}

	if data.UserID == 0 {
		return 0, false
	}

	return data.UserID, true
}

func DestroySession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		DB.Exec(`DELETE FROM "session" WHERE sid = $1`, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
