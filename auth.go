package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AuthService struct {
	allowedUsers *AllowedUsers
	secret       []byte
	botName      string
	maxAuthAge   time.Duration
}

func NewAuthService(botToken string, maxAuthAge time.Duration, allowedUsers *AllowedUsers) (*AuthService, error) {
	svc := &AuthService{
		allowedUsers: allowedUsers,
		maxAuthAge:   maxAuthAge,
	}

	h := sha256.New()
	_, err := h.Write([]byte(botToken))
	if err != nil {
		return nil, err
	}

	svc.secret = h.Sum(nil)

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, err
	}

	log.Printf("connected to telegram as %s", bot.Self.UserName)
	svc.botName = bot.Self.UserName

	allowedUsers.Print()

	return svc, nil
}

func (svc *AuthService) BotName() string {
	return svc.botName
}

type CheckAccessResult int

const (
	NoAccess CheckAccessResult = iota
	HasAccess
	NoTicket
	TicketExpired
)

func (svc *AuthService) CheckAccess(ticket *AuthTicket) CheckAccessResult {
	if ticket == nil {
		return NoTicket
	}

	if !ticket.CheckSignature(svc.secret) {
		return NoTicket
	}

	age := time.Now().Sub(ticket.AuthDate())
	if age.Seconds() > svc.maxAuthAge.Seconds() {
		return TicketExpired
	}

	if !svc.allowedUsers.IsAllowed(ticket) {
		return NoAccess
	}

	return HasAccess
}

const AuthCookieName = ".auth"

func (svc *AuthService) Login(w http.ResponseWriter, ticket *AuthTicket) {
	cookie := http.Cookie{
		Name:     AuthCookieName,
		Value:    ticket.String(),
		Path:     "/",
		Expires:  ticket.AuthDate().Add(svc.maxAuthAge),
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)

	log.Printf("logged in as %v @%s", ticket.UserID(), ticket.UserName())
}

func (svc *AuthService) Logout(w http.ResponseWriter) {
	cookie := http.Cookie{
		Name:   AuthCookieName,
		MaxAge: -1,
	}
	http.SetCookie(w, &cookie)
}

type AuthTicket struct {
	parameters map[string]string
}

func AuthTicketFromCookie(req *http.Request) *AuthTicket {
	cookie, err := req.Cookie(AuthCookieName)
	if err != nil {
		return nil
	}
	if cookie == nil {
		return nil
	}

	parameters := make(map[string]string)
	str, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return nil
	}

	err = json.Unmarshal([]byte(str), &parameters)
	if err != nil {
		return nil
	}

	ticket := &AuthTicket{parameters}
	ticket.sanitizeParameters()
	if !ticket.areParametersWellFormed() {
		return nil
	}

	return ticket
}

func AuthTicketFromURL(req *http.Request) *AuthTicket {
	query := req.URL.Query()

	parameters := make(map[string]string)
	for k, v := range query {
		parameters[k] = v[0]
	}

	ticket := &AuthTicket{parameters}
	ticket.sanitizeParameters()
	if !ticket.areParametersWellFormed() {
		return nil
	}

	return ticket
}

func (t *AuthTicket) UserID() int64 {
	s, exists := t.parameters["id"]
	if exists {
		val, err := strconv.ParseInt(s, 10, 0)
		if err == nil {
			return val
		}
	}

	return 0
}

func (t *AuthTicket) UserName() string {
	s, exists := t.parameters["username"]
	if exists {
		return s
	}

	return ""
}

func (t *AuthTicket) AuthDate() time.Time {
	s, exists := t.parameters["auth_date"]
	if exists {
		val, err := strconv.ParseInt(s, 10, 0)
		if err == nil {
			return time.Unix(val, 0)
		}
	}

	return time.Time{}
}

func (t *AuthTicket) dataCheckString() string {
	var dataCheckStringParts []string
	for k, v := range t.parameters {
		if k != "hash" {
			dataCheckStringParts = append(dataCheckStringParts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	sort.Strings(dataCheckStringParts)

	val := strings.Join(dataCheckStringParts, "\n")
	return val
}

func (t *AuthTicket) hash() string {
	s, exists := t.parameters["hash"]
	if exists {
		return s
	}

	return ""
}

func (t *AuthTicket) CheckSignature(secret []byte) bool {
	dataCheckString := t.dataCheckString()
	actualHash := t.hash()

	h := hmac.New(sha256.New, secret)

	_, err := h.Write([]byte(dataCheckString))
	if err != nil {
		panic(err)
	}

	expectedHash := hex.EncodeToString(h.Sum(nil))

	return actualHash == expectedHash
}

func (t *AuthTicket) String() string {
	bs, err := json.Marshal(t.parameters)
	if err != nil {
		panic(err)
	}

	str := string(bs)
	str = url.QueryEscape(str)
	return str
}

func (t *AuthTicket) sanitizeParameters() {
	for k := range t.parameters {
		switch k {
		case "id", "first_name", "last_name", "username", "photo_url", "auth_date", "hash":
			continue

		default:
			delete(t.parameters, k)
		}
	}
}

func (t *AuthTicket) areParametersWellFormed() bool {
	_, exists := t.parameters["id"]
	if !exists {
		return false
	}

	_, exists = t.parameters["username"]
	if !exists {
		return false
	}

	_, exists = t.parameters["auth_date"]
	if !exists {
		return false
	}

	_, exists = t.parameters["hash"]
	if !exists {
		return false
	}

	return true
}

type AllowedUsers struct {
	UserIDs   map[int64]struct{}
	UserNames map[string]struct{}
}

func NewAllowedUsers(raw string) *AllowedUsers {
	users := &AllowedUsers{
		UserIDs:   make(map[int64]struct{}),
		UserNames: make(map[string]struct{}),
	}

	fields := strings.FieldsFunc(raw, func(c rune) bool {
		return c == ',' || c == ';' || c == ' '
	})
	for _, str := range fields {
		str = strings.TrimSpace(str)

		i, err := strconv.ParseInt(str, 10, 0)
		if err == nil {
			users.UserIDs[i] = struct{}{}
			continue
		}

		str = strings.TrimPrefix(str, "@")
		str = strings.TrimPrefix(str, "t.me/")
		str = strings.TrimPrefix(str, "http://t.me/")
		str = strings.TrimPrefix(str, "https://t.me/")

		users.UserNames[str] = struct{}{}
	}

	return users
}

func (users *AllowedUsers) Print() {
	log.Printf("access is configured for user(s):")
	for id := range users.UserIDs {
		log.Printf("- %v", id)
	}

	for name := range users.UserNames {
		log.Printf("- @%v", name)
	}
}

func (users *AllowedUsers) IsAllowed(t *AuthTicket) bool {
	_, exists := users.UserIDs[t.UserID()]
	if exists {
		return true
	}

	_, exists = users.UserNames[t.UserName()]
	if exists {
		return true
	}

	return false
}
