package main

import (
	"billportal/auth"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

var apitoken *string

type Userdata struct {
	Ip    string
	NotDc bool
}

func main() {
	var (
		app_ip   = auth.GetEnvVariable("APP_IP")
		app_port = auth.GetEnvVariable("APP_PORT")
	)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	files := []string{
		"./templates/base.tmpl",
		"./templates/index.tmpl",
		"./templates/errindex.tmpl",
		"./templates/errbase.tmpl",
		"./templates/waitbase.tmpl",
		"./templates/wait.tmpl",
	}

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		log.Print(err.Error())
		return
	}
	http.HandleFunc("/", serveTemplate(tmpl))
	token, err := auth.GetToken()
	apitoken = token
	if err != nil {
		log.Println(err)
	}
	log.Printf("%s:%s", app_ip, app_port)
	err = http.ListenAndServe(fmt.Sprintf("%s:%s", app_ip, app_port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func serveTemplate(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		remote := strings.Split(r.RemoteAddr, ":")
		log.Println("Captured: ", remote[0])
		ip := strings.Replace(remote[0], ".", "", -1)
		data := &Userdata{Ip: ip, NotDc: true}
		if r.Method != http.MethodPost {
			sub, err := auth.GetSub(remote[0], apitoken)
			if err != nil {
				log.Println("Error: ", err)
			} else {
				if sub.LaterCount > 14 {
					data.NotDc = false
				}
			}
			tmpl.ExecuteTemplate(w, "base", data)
			return
		}

		result := auth.ActivateSubByIp(remote[0], "active", apitoken)
		if result == "NotFound" {
			log.Println("Code error: Not Found")
			tmpl.ExecuteTemplate(w, "errbase", nil)
			return
		}

		time.Sleep(1 * time.Second)
		log.Println("activated ", ip)
		//redirect to landing page instead of below
		//http.Redirect(w, r, "https://www.google.com", http.StatusSeeOther)
		tmpl.ExecuteTemplate(w, "waitbase", nil)
	}
}
