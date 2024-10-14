package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type Token struct {
	Name string
	Jwt  string
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
