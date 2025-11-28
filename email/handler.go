package email

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var emailService *EmailService // just for demo

type RequestPayload struct {
	Email string `json:"email"`
}

type Response struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// For demo only, in production, use redis, db or other services
type TwoFADetails struct {
	Code      string
	ExpiresAt time.Time
}

var store = struct {
	sync.RWMutex
	codes map[string]TwoFADetails // email is key
}{
	codes: make(map[string]TwoFADetails),
}

func init() {
	s, err := NewEmailService()
	if err != nil {
		log.Fatal(err)
	}
	emailService = s
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, payload Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "email/index.html")
}

func send2FAHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, Response{Error: "Method not allowed"})
		return
	}

	var payload RequestPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil || payload.Email == "" {
		writeJSONResponse(w, http.StatusBadRequest, Response{Error: "Invalid request body"})
		return
	}

	code, err := emailService.Generate2FACode()
	if err != nil {
		log.Printf("Error generating 2FA code: %v", err)
		writeJSONResponse(w, http.StatusInternalServerError, Response{Error: "Failed to generate code"})
		return
	}

	// store auth code which expires in 5mins
	store.Lock()
	store.codes[payload.Email] = TwoFADetails{
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	store.Unlock()

	err = emailService.Send2FACode(payload.Email, code)
	if err != nil {
		log.Printf("Error sending 2FA email to %s: %v", payload.Email, err)
		writeJSONResponse(w, http.StatusInternalServerError, Response{Error: "Failed to send email"})
		return
	}

	writeJSONResponse(w, http.StatusOK, Response{Message: "OK"})
}

func sendResetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, Response{Error: "Method not allowed"})
		return
	}

	var payload RequestPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil || payload.Email == "" {
		writeJSONResponse(w, http.StatusBadRequest, Response{Error: "Invalid request body"})
		return
	}

	// simulate user name/id
	name := "test"
	err = emailService.SendPasswordReset(payload.Email, name)
	if err != nil {
		log.Printf("Error sending reset email to %s: %v", payload.Email, err)
		writeJSONResponse(w, http.StatusInternalServerError, Response{Error: "Failed to send email"})
		return
	}

	log.Printf("Successfully sent password reset link to %s", payload.Email)
	writeJSONResponse(w, http.StatusOK, Response{Message: "OK"})
}

func verify2FAHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, Response{Error: "Method not allowed"})
		return
	}

	var payload struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil || payload.Email == "" || payload.Code == "" {
		writeJSONResponse(w, http.StatusBadRequest, Response{Error: "Invalid request body"})
		return
	}

	store.RLock()
	details, found := store.codes[payload.Email]
	store.RUnlock()

	if !found {
		writeJSONResponse(w, http.StatusBadRequest, Response{Error: "Code not exist or expired"})
		return
	}

	if time.Now().After(details.ExpiresAt) {
		// delete expired auth code
		store.Lock()
		delete(store.codes, payload.Email)
		store.Unlock()
		writeJSONResponse(w, http.StatusBadRequest, Response{Error: "Code expired"})
		return
	}

	if payload.Code != details.Code {
		writeJSONResponse(w, http.StatusUnauthorized, Response{Error: "Invalid code"})
		return
	}

	// verify successfully, delete code
	store.Lock()
	delete(store.codes, payload.Email)
	store.Unlock()

	log.Printf("2FA verification successful for %s", payload.Email)
	writeJSONResponse(w, http.StatusOK, Response{Message: "Success"})
}

func showResetPasswordFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		http.Error(w, "Token is missing", http.StatusBadRequest)
		return
	}

	jwtSecret := emailService.jwtSecret // just for demo, not production
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	// if token is valid, allow user reset password
	tmpl, err := template.ParseFiles("email/reset_password_form.html")
	if err != nil {
		log.Printf("Error parsing reset password form template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// pass token as hidden input into template
	tmpl.Execute(w, struct{ Token string }{Token: tokenString})
}

func performResetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	newPassword := r.FormValue("newPassword")
	tokenString := r.FormValue("token")

	if newPassword == "" || tokenString == "" {
		http.Error(w, "Missing new password or token", http.StatusBadRequest)
		return
	}

	// Double check token for more reliable
	jwtSecret := emailService.jwtSecret // just for demo, not in production
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	// Lấy user_id từ claims (trong production, bạn sẽ dùng ID này để update trong DB)
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		userID := claims["user_id"]
		log.Printf("Password reset requested for user_id: %v. New password would be: %s", userID, newPassword)
		// TODO: update new password into DB
		// Example: db.UpdatePassword(userID, newPassword)
	} else {
		http.Error(w, "Could not parse token claims", http.StatusInternalServerError)
		return
	}

	// Send success
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("<h1>Mật khẩu của bạn đã được đặt lại thành công!</h1><p>Bạn có thể đóng tab này.</p>"))
}

func RunUseCaseExample(skip bool) {
	if skip {
		return
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/send-2fa", send2FAHandler)
	http.HandleFunc("/send-reset", sendResetHandler)
	http.HandleFunc("/verify-2fa", verify2FAHandler)
	// can group into more complex router, this implement just for demo
	// chi route | gorilla/mux
	http.HandleFunc("/reset-password", showResetPasswordFormHandler)
	http.HandleFunc("/reset-password-action", performResetHandler)

	port := ":8080"
	log.Printf("Server starting on http://localhost%s\n", port)
	server := &http.Server{
		Addr:         port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}
}
