package main

import (
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
)

type LoginPageModel struct {
	BotName string
}

func contentHandler(authService *AuthService, reverseProxy *httputil.ReverseProxy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ticket := AuthTicketFromCookie(r)

		switch authService.CheckAccess(ticket) {
		case HasAccess:
			reverseProxy.ServeHTTP(w, r)

		case NoAccess:
			authService.Logout(w)
			renderTemplateSafe(w, http.StatusForbidden, "403.html", &LoginPageModel{
				BotName: authService.BotName(),
			})

		default:
			renderTemplateSafe(w, http.StatusUnauthorized, "401.html", &LoginPageModel{
				BotName: authService.BotName(),
			})
		}
	})
}

func loginHandler(authService *AuthService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ticket := AuthTicketFromURL(r)
		switch authService.CheckAccess(ticket) {
		case HasAccess:
			authService.Login(w, ticket)
			location := r.URL.Query().Get("return_url")
			if location == "" {
				location = "/"
			}

			w.Header().Set("Location", location)
			w.WriteHeader(http.StatusFound)

		default:
			renderTemplateSafe(w, http.StatusForbidden, "403.html", &LoginPageModel{
				BotName: authService.BotName(),
			})
		}
	})
}

func logoutHandler(authService *AuthService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authService.Logout(w)

		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)
	})
}

func renderTemplateSafe(w http.ResponseWriter, statusCode int, name string, model interface{}) {
	err := renderTemplate(w, statusCode, name, model)
	if err != nil {
		log.Printf("unable to render template \"%s\": %s", name, err)
		err = renderTemplate(w, http.StatusInternalServerError, "500.html", nil)
		if err != nil {
			log.Printf("unable to render template \"%s\": %s", "500.html", err)
			panic(err)
		}
	}
}

func renderTemplate(w http.ResponseWriter, statusCode int, name string, model interface{}) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	path := filepath.Join(wd, "www", name)

	t := template.New(name)
	t, err = t.ParseFiles(path)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; chartset=utf-8")
	w.WriteHeader(statusCode)
	err = t.Execute(w, model)
	if err != nil {
		return err
	}

	return nil
}
