package main

import (
	"billportal/auth"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	contentDir    = "./data"
	contentFile   = "./data/content.json"
	gcashQRPath   = "./static/assets/img/gcash.jpg"
	mayaQRPath    = "./static/assets/img/maya.jpg"
	maxUploadSize = 5 << 20 // 5MB per request
	maxMessageLen = 2000
)

const defaultMessage = `This is a gentle reminder about your overdue bill for account# {AccountNumber}. Kindly send payments to 09686980551 gcash or maya. You may scan or save the QR code below. Please send proof of payment/ref number via sms or screenshot to our FB page "CCR Fiber Internet Mambaling". Thank you for your patronage.`

type storedContent struct {
	Message string `json:"message"`
}

var (
	contentMu      sync.RWMutex
	currentMessage = defaultMessage
)

// loadContent reads the admin-editable reminder text from disk at startup,
// seeding the file with the built-in default on first run.
func loadContent() {
	data, err := os.ReadFile(contentFile)
	if err != nil {
		if err := saveContent(defaultMessage); err != nil {
			log.Println("seed content error:", err)
		}
		return
	}
	var c storedContent
	if err := json.Unmarshal(data, &c); err != nil || c.Message == "" {
		log.Println("invalid content file, using default message")
		return
	}
	contentMu.Lock()
	currentMessage = c.Message
	contentMu.Unlock()
}

func saveContent(message string) error {
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(storedContent{Message: message}, "", "  ")
	if err != nil {
		return err
	}
	tmp := contentFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, contentFile)
}

func getMessage() string {
	contentMu.RLock()
	defer contentMu.RUnlock()
	return currentMessage
}

func setMessage(message string) error {
	if err := saveContent(message); err != nil {
		return err
	}
	contentMu.Lock()
	currentMessage = message
	contentMu.Unlock()
	return nil
}

// basicAuth protects the admin page using credentials from .env
// (ADMIN_USER / ADMIN_PASS), compared in constant time.
func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wantUser := auth.GetEnvVariable("ADMIN_USER")
		wantPass := auth.GetEnvVariable("ADMIN_PASS")
		user, pass, ok := r.BasicAuth()
		userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(wantUser)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(wantPass)) == 1
		if !ok || wantUser == "" || wantPass == "" || !userMatch || !passMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="billportal admin"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

type AdminData struct {
	Message   string
	Saved     bool
	Error     string
	CacheBust int64
}

func adminHandler(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := &AdminData{Message: getMessage(), CacheBust: time.Now().Unix()}

		if r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
			if err := r.ParseMultipartForm(maxUploadSize); err != nil {
				data.Error = "Upload too large or invalid form."
				tmpl.ExecuteTemplate(w, "adminbase", data)
				return
			}

			message := strings.TrimSpace(r.FormValue("message"))
			if message == "" {
				data.Error = "Reminder text cannot be empty."
				tmpl.ExecuteTemplate(w, "adminbase", data)
				return
			}
			if len(message) > maxMessageLen {
				data.Error = "Reminder text is too long."
				tmpl.ExecuteTemplate(w, "adminbase", data)
				return
			}
			data.Message = message

			if err := saveUploadedQR(r, "gcash_qr", gcashQRPath); err != nil {
				data.Error = err.Error()
				tmpl.ExecuteTemplate(w, "adminbase", data)
				return
			}
			if err := saveUploadedQR(r, "maya_qr", mayaQRPath); err != nil {
				data.Error = err.Error()
				tmpl.ExecuteTemplate(w, "adminbase", data)
				return
			}

			if err := setMessage(message); err != nil {
				log.Println("save content error:", err)
				data.Error = "Failed to save reminder text."
				tmpl.ExecuteTemplate(w, "adminbase", data)
				return
			}

			data.Saved = true
			data.CacheBust = time.Now().Unix()
		}

		tmpl.ExecuteTemplate(w, "adminbase", data)
	}
}

// saveUploadedQR decodes the uploaded image (if any) and atomically
// overwrites path with it re-encoded as JPEG. A missing file field is not
// an error - it means "keep the current image".
func saveUploadedQR(r *http.Request, field, path string) error {
	file, header, err := r.FormFile(field)
	if err != nil {
		if err == http.ErrMissingFile {
			return nil
		}
		return fmt.Errorf("failed to read %s upload", field)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("%s is not a valid image (jpg/png)", header.Filename)
	}

	tmp := path + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("failed to save %s", header.Filename)
	}

	if err := jpeg.Encode(out, img, &jpeg.Options{Quality: 90}); err != nil {
		out.Close()
		os.Remove(tmp)
		return fmt.Errorf("failed to encode %s", header.Filename)
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to save %s", header.Filename)
	}
	return os.Rename(tmp, path)
}
