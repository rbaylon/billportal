package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Token struct {
	Name string
	Jwt  string
}

type Sub struct {
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	FramedIp    string    `json:"framed_ip"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Mac         string    `json:"mac"`
	Loc         string    `json:"loc"`
	Downspeed   int       `json:"downspeed"`
	Upspeed     int       `json:"upspeed"`
	Burstspeed  int       `json:"burstspeed"`
	Duration    int       `json:"duration"`
	Gateway     string    `json:"gateway"`
	Priority    int       `json:"priority"`
	DateEnd     time.Time `json:"date_end"`
	DateExpires time.Time `json:"date_expires"`
	PfconfigID  uint      `json:"pfconfig_id"`
	DueDay      int       `json:"due_day"`
	LaterCount  int       `json:"later_count"`
}

func GetEnvVariable(key string) string {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}
	return os.Getenv(key)
}

func ActivateSubByIp(ip string, status string, t *string) string {
	var (
		api_url = GetEnvVariable("API_URL")
	)
	url := api_url + "subs/activate/" + ip + "/" + status
	log.Println(url)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *t))
	res, _ := client.Do(req)
	defer res.Body.Close()
	v := "active"
	if res.StatusCode == 404 {
		v = "NotFound"
	}
	if res.StatusCode == 201 {
		v = "activated"
	}
	if res.StatusCode == 202 {
		v = "updated"
	}
	log.Println("Code validated")
	return v
}

func GetSub(ip string, t *string) (*Sub, error) {
	var (
		api_url = GetEnvVariable("API_URL")
	)
	url := api_url + "subs/getbyip/" + ip
	log.Println(url)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *t))
	res, _ := client.Do(req)
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("ip not found\n")
	}
	defer res.Body.Close()
	responseData, ioerr := ioutil.ReadAll(res.Body)
	if ioerr != nil {
		return nil, ioerr
	}
	var sub Sub
	json.Unmarshal(responseData, &sub)
	return &sub, nil
}

func GetToken() (*string, error) {
	var (
		api_auth = GetEnvVariable("API_AUTH")
		api_url  = GetEnvVariable("API_URL")
	)
	url := api_url + "login"
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", api_auth))
	res, err := client.Do(req)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}
	responseData, ioerr := ioutil.ReadAll(res.Body)
	if ioerr != nil {
		return nil, ioerr
	}

	var t Token
	json.Unmarshal(responseData, &t)
	return &t.Jwt, nil
}

func GetUnixConn() net.Conn {
	c, err := net.Dial("unix", GetEnvVariable("UNIX_SOCK"))
	if err != nil {
		log.Println("Dial error ", err)
	}
	return c
}
