package app

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

//go:embed schema.sql templates/* web/*
var files embed.FS

var (
	tmpl = template.Must(template.New("root").Funcs(template.FuncMap{
		"money": func(v float64) string { return fmt.Sprintf("%.2f AZN", v) },
		"pct": func(current, target float64) int {
			if target <= 0 {
				return 0
			}
			n := int(current / target * 100)
			if n > 100 {
				return 100
			}
			if n < 0 {
				return 0
			}
			return n
		},
		"account": func(v string) string {
			switch v {
			case "nagd":
				return "Nağd"
			case "kredit_kart":
				return "Kredit kartı"
			case "debet_kart":
				return "Debet kartı"
			default:
				return v
			}
		},
		"kind": func(v string) string {
			if v == "gelir" {
				return "Gəlir"
			}
			return "Xərc"
		},
		"nowDate": func() string { return time.Now().Format("2006-01-02") },
		"month": func(v string) string {
			t, e := time.Parse("2006-01", v)
			if e != nil {
				return v
			}
			names := []string{"Yanvar", "Fevral", "Mart", "Aprel", "May", "İyun", "İyul", "Avqust", "Sentyabr", "Oktyabr", "Noyabr", "Dekabr"}
			return fmt.Sprintf("%s %d", names[int(t.Month())-1], t.Year())
		},
	}).ParseFS(files, "templates/*.html"))
	initOnce sync.Once
	initErr  error
)

type User struct {
	ID                int64
	First, Last       string
	Name, Email, Role string
}
type Category struct {
	ID         int64
	Name, Type string
	Limit      float64
}
type Tx struct {
	ID                            int64
	Category, Account, Note, Date string
	Amount                        float64
	Kind                          string
}
type Goal struct {
	ID                                 int64
	Name, Status, Priority, Start, End string
	Target, Current                    float64
}
type Page struct {
	Title, Active, Name, Email, Error, Success, CSRF, Month string
	User                                                    *User
	Income, Expense, Balance                                float64
	Transactions                                            []Tx
	Categories                                              []Category
	Goals                                                   []Goal
	Items                                                   []map[string]any
	IsAdmin                                                 bool
	FormType                                                string
}

func HandlerFunc() http.Handler { return http.HandlerFunc(Handler) }
func Handler(w http.ResponseWriter, r *http.Request) {
	p := routePath(r)
	if strings.HasPrefix(p, "/assets/") {
		serveAsset(w, r, p)
		return
	}
	if p == "/favicon.ico" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method == http.MethodGet && (p == "/health" || p == "/api/health") {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return
	}
	db, err := database(r.Context())
	if err != nil {
		renderConfig(w, err)
		return
	}
	defer db.Close()
	csrf := csrfToken(w, r)
	user := currentUser(db, r)
	if r.Method == http.MethodPost {
		if !validCSRF(r) {
			http.Error(w, "Təhlükəsizlik tokeni yanlışdır. Səhifəni yeniləyin.", http.StatusForbidden)
			return
		}
		if p == "/login" {
			login(w, r, db)
			return
		}
		if p == "/register" {
			register(w, r, db)
			return
		}
		if p == "/logout" {
			logout(w, r)
			return
		}
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if handlePost(w, r, db, user, p) {
			return
		}
	}
	if user == nil && p != "/login" && p != "/register" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if (p == "/login" || p == "/register") && user != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if user == nil {
		renderAuth(w, r, csrf, p, "")
		return
	}
	if p == "/admin" && user.Role != "admin" {
		http.Error(w, "Qadağandır", http.StatusForbidden)
		return
	}
	if p != "/" && p != "/dashboard" && p != "/income" && p != "/expenses" && p != "/categories" && p != "/goals" && p != "/reports" && p != "/settings" && p != "/admin" {
		http.NotFound(w, r)
		return
	}
	page := Page{User: user, Name: user.Name, Email: user.Email, CSRF: csrf, Active: p, Month: r.URL.Query().Get("month")}
	if page.Month == "" {
		page.Month = time.Now().Format("2006-01")
	}
	loadPage(db, r, &page, p)
	page.Success = r.URL.Query().Get("ok")
	page.Error = r.URL.Query().Get("err")
	if err := tmpl.ExecuteTemplate(w, "page.html", page); err != nil {
		log.Printf("template: %v", err)
	}
}
func routePath(r *http.Request) string {
	if v := r.URL.Query().Get("path"); v != "" {
		if !strings.HasPrefix(v, "/") {
			v = "/" + v
		}
		v = strings.TrimSuffix(v, "/")
		if v == "" {
			return "/"
		}
		return v
	}
	p := strings.TrimSuffix(r.URL.Path, "/")
	if p == "" {
		p = "/"
	}
	return p
}
func database(ctx context.Context) (*sql.DB, error) {
	if os.Getenv("SESSION_SECRET") == "" && os.Getenv("VERCEL") == "1" {
		return nil, fmt.Errorf("SESSION_SECRET mühit dəyişəni təyin edilməyib")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL təyin edilməyib")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	db := stdlib.OpenDB(*config)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Minute)
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("Supabase PostgreSQL bağlantısı alınmadı: %w", err)
	}
	initOnce.Do(func() {
		for _, statement := range strings.Split(string(mustRead("schema.sql")), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, initErr = db.ExecContext(ctx, statement); initErr != nil {
				break
			}
		}
	})
	if initErr != nil {
		db.Close()
		return nil, fmt.Errorf("DB sxemi yaradılmadı: %w", initErr)
	}
	return db, nil
}
func mustRead(name string) []byte {
	b, e := files.ReadFile(name)
	if e != nil {
		panic(e)
	}
	return b
}
func serveAsset(w http.ResponseWriter, r *http.Request, p string) {
	name := strings.TrimPrefix(p, "/assets/")
	b, e := files.ReadFile("web/" + name)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	switch {
	case strings.HasSuffix(name, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(name, ".woff2"):
		w.Header().Set("Content-Type", "font/woff2")
	default:
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(b)
}
func renderConfig(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = fmt.Fprintf(w, `<!doctype html><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>BlPay sazlanması</title><style>body{font:16px system-ui;background:#f5f8f5;color:#17251f;max-width:680px;margin:10vh auto;padding:24px}main{background:white;border-radius:18px;padding:30px;box-shadow:0 15px 45px #1232}code{background:#edf3ee;padding:4px 7px;border-radius:5px}</style><main><h1>BlPay bazaya qoşula bilmir</h1><p>%s</p><p>Vercel layihəsinin Environment Variables bölməsində Supabase PostgreSQL connection string-i <code>DATABASE_URL</code> adı ilə əlavə edin. Sonra səhifəni yeniləyin.</p></main>`, template.HTMLEscapeString(err.Error()))
}
func secret() []byte {
	s := os.Getenv("SESSION_SECRET")
	if s == "" {
		s = "local-development-secret-change-before-deploy"
	}
	return []byte(s)
}
func sign(s string) string {
	m := hmac.New(sha256.New, secret())
	m.Write([]byte(s))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
func setSession(w http.ResponseWriter, r *http.Request, id int64) {
	value := strconv.FormatInt(id, 10)
	http.SetCookie(w, &http.Cookie{Name: "blpay_session", Value: value + "." + sign(value), Path: "/", HttpOnly: true, Secure: r.TLS != nil || r.Header.Get("x-forwarded-proto") == "https", SameSite: http.SameSiteLaxMode, MaxAge: 30 * 24 * 3600})
}
func sessionID(r *http.Request) int64 {
	c, e := r.Cookie("blpay_session")
	if e != nil {
		return 0
	}
	a, b, ok := strings.Cut(c.Value, ".")
	if !ok || !hmac.Equal([]byte(b), []byte(sign(a))) {
		return 0
	}
	id, _ := strconv.ParseInt(a, 10, 64)
	return id
}
func currentUser(db *sql.DB, r *http.Request) *User {
	id := sessionID(r)
	if id == 0 {
		return nil
	}
	var u User
	var a, s string
	if db.QueryRowContext(r.Context(), `SELECT id,ad,soyad,email,rol FROM istifadeciler WHERE id=$1 AND status='aktiv'`, id).Scan(&u.ID, &a, &s, &u.Email, &u.Role) != nil {
		return nil
	}
	u.First, u.Last = a, s
	u.Name = strings.TrimSpace(a + " " + s)
	return &u
}
func csrfToken(w http.ResponseWriter, r *http.Request) string {
	c, e := r.Cookie("blpay_csrf")
	if e == nil && len(c.Value) > 20 {
		return c.Value
	}
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	v := base64.RawURLEncoding.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{Name: "blpay_csrf", Value: v, Path: "/", Secure: r.TLS != nil || r.Header.Get("x-forwarded-proto") == "https", SameSite: http.SameSiteLaxMode, MaxAge: 86400})
	return v
}
func validCSRF(r *http.Request) bool {
	c, e := r.Cookie("blpay_csrf")
	if e != nil {
		return false
	}
	return hmac.Equal([]byte(c.Value), []byte(r.FormValue("csrf")))
}
func redirect(w http.ResponseWriter, r *http.Request, path, key, msg string) {
	http.Redirect(w, r, path+"?"+key+"="+urlQuery(msg), http.StatusSeeOther)
}
func urlQuery(s string) string {
	return url.QueryEscape(s)
}
func login(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	var id int64
	var hash, status string
	err := db.QueryRowContext(r.Context(), `SELECT id,sifre,status FROM istifadeciler WHERE lower(email)=lower($1)`, email).Scan(&id, &hash, &status)
	if err != nil || status != "aktiv" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		renderAuth(w, r, csrfToken(w, r), "/login", "E-poçt və ya şifrə yanlışdır.")
		return
	}
	setSession(w, r, id)
	_, _ = db.ExecContext(r.Context(), `UPDATE istifadeciler SET son_giris=NOW() WHERE id=$1`, id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func register(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	name := strings.TrimSpace(r.FormValue("name"))
	surname := strings.TrimSpace(r.FormValue("surname"))
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	confirm := r.FormValue("confirm")
	if name == "" || surname == "" || len(name) > 50 || len(surname) > 50 {
		renderAuth(w, r, csrfToken(w, r), "/register", "Ad və soyad daxil edin.")
		return
	}
	if _, e := mail.ParseAddress(email); e != nil {
		renderAuth(w, r, csrfToken(w, r), "/register", "Düzgün e-poçt ünvanı daxil edin.")
		return
	}
	if !validPassword(password) || password != confirm {
		renderAuth(w, r, csrfToken(w, r), "/register", "Şifrə 8–72 simvol olmalı, böyük və kiçik hərf, rəqəm və xüsusi simvol saxlamalıdır; təsdiqi də uyğun olmalıdır.")
		return
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		http.Error(w, "Qeydiyyat alınmadı", 500)
		return
	}
	tx, e := db.BeginTx(r.Context(), nil)
	if e != nil {
		http.Error(w, "Qeydiyyat alınmadı", 500)
		return
	}
	var id int64
	e = tx.QueryRowContext(r.Context(), `INSERT INTO istifadeciler(ad,soyad,email,sifre) VALUES($1,$2,$3,$4) RETURNING id`, name, surname, email, string(hash)).Scan(&id)
	if e == nil {
		_, e = tx.ExecContext(r.Context(), `INSERT INTO kateqoriyalar(istifadeci_id,ad,tip,limit_mebleq) SELECT $1,ad,tip,limit_mebleq FROM kateqoriyalar WHERE standart=TRUE`, id)
	}
	if e != nil {
		_ = tx.Rollback()
		renderAuth(w, r, csrfToken(w, r), "/register", "Bu e-poçt artıq istifadə olunur və ya qeydiyyat baş tutmadı.")
		return
	}
	if e = tx.Commit(); e != nil {
		http.Error(w, "Qeydiyyat alınmadı", 500)
		return
	}
	setSession(w, r, id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func validPassword(password string) bool {
	if len(password) < 8 || len(password) > 72 {
		return false
	}
	var upper, lower, digit, special bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		case !unicode.IsLetter(r):
			special = true
		}
	}
	return upper && lower && digit && special
}
func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "blpay_session", Value: "", Path: "/", HttpOnly: true, MaxAge: -1, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
func renderAuth(w http.ResponseWriter, r *http.Request, csrf, path, err string) {
	kind := "login"
	if path == "/register" {
		kind = "register"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "auth.html", map[string]any{"Kind": kind, "CSRF": csrf, "Error": err})
}
func handlePost(w http.ResponseWriter, r *http.Request, db *sql.DB, u *User, path string) bool {
	ctx := r.Context()
	q := func(query string, args ...any) error { _, e := db.ExecContext(ctx, query, args...); return e }
	f := r.FormValue
	switch path {
	case "/income", "/expenses":
		table, typ := "gelirler", "gelir"
		if path == "/expenses" {
			table, typ = "xercler", "xerc"
		}
		action := f("action")
		id, _ := strconv.ParseInt(f("id"), 10, 64)
		if action == "delete" {
			_ = q(`DELETE FROM `+table+` WHERE id=$1 AND istifadeci_id=$2`, id, u.ID)
		} else {
			amount, e := strconv.ParseFloat(f("amount"), 64)
			cat, _ := strconv.ParseInt(f("category"), 10, 64)
			date := f("date")
			account := f("account")
			note := f("note")
			_, dateErr := time.Parse("2006-01-02", date)
			var own int
			if (action != "add" && action != "edit") || e != nil || dateErr != nil || amount <= 0 || (account != "nagd" && account != "kredit_kart" && account != "debet_kart") || db.QueryRowContext(ctx, `SELECT count(*) FROM kateqoriyalar WHERE id=$1 AND istifadeci_id=$2 AND tip=$3`, cat, u.ID, typ).Scan(&own) != nil || own == 0 {
				redirect(w, r, path, "err", "Məlumatları düzgün doldurun.")
				return true
			}
			if typ == "xerc" {
				var limit, spent float64
				_ = db.QueryRowContext(ctx, `SELECT COALESCE(limit_mebleq,0) FROM kateqoriyalar WHERE id=$1 AND istifadeci_id=$2`, cat, u.ID).Scan(&limit)
				if limit > 0 {
					_ = db.QueryRowContext(ctx, `SELECT COALESCE(SUM(mebleq),0) FROM xercler WHERE istifadeci_id=$1 AND kateqoriya_id=$2 AND to_char(tarix,'YYYY-MM')=$3 AND ($4=0 OR id<>$4)`, u.ID, cat, date[:7], id).Scan(&spent)
					if spent+amount > limit {
						redirect(w, r, path, "err", "Bu əməliyyat kateqoriyanın aylıq limitini keçəcək.")
						return true
					}
				}
			}
			if action == "edit" {
				_ = q(`UPDATE `+table+` SET mebleq=$1,kateqoriya_id=$2,tarix=$3,hesab_novu=$4,qeyd=$5 WHERE id=$6 AND istifadeci_id=$7`, amount, cat, date, account, note, id, u.ID)
			} else {
				_ = q(`INSERT INTO `+table+`(istifadeci_id,mebleq,kateqoriya_id,tarix,hesab_novu,qeyd) VALUES($1,$2,$3,$4,$5,$6)`, u.ID, amount, cat, date, account, note)
			}
		}
		redirect(w, r, path, "ok", "Yadda saxlanıldı")
		return true
	case "/categories":
		id, _ := strconv.ParseInt(f("id"), 10, 64)
		switch f("action") {
		case "delete":
			_ = q(`DELETE FROM kateqoriyalar k WHERE id=$1 AND istifadeci_id=$2 AND NOT EXISTS(SELECT 1 FROM gelirler WHERE kateqoriya_id=k.id) AND NOT EXISTS(SELECT 1 FROM xercler WHERE kateqoriya_id=k.id)`, id, u.ID)
		case "add", "edit":
			name := strings.TrimSpace(f("name"))
			typ := f("type")
			limit, _ := strconv.ParseFloat(f("limit"), 64)
			if name == "" || (typ != "gelir" && typ != "xerc") {
				redirect(w, r, path, "err", "Kateqoriya məlumatlarını yoxlayın.")
				return true
			}
			var lim any
			if typ == "xerc" && limit > 0 {
				lim = limit
			}
			if f("action") == "edit" {
				var currentType string
				_ = db.QueryRowContext(ctx, `SELECT tip FROM kateqoriyalar WHERE id=$1 AND istifadeci_id=$2`, id, u.ID).Scan(&currentType)
				if currentType != typ {
					var used int
					_ = db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM gelirler WHERE kateqoriya_id=$1)+(SELECT count(*) FROM xercler WHERE kateqoriya_id=$1)`, id).Scan(&used)
					if used > 0 {
						redirect(w, r, path, "err", "Əməliyyatlarda istifadə olunan kateqoriyanın tipini dəyişmək olmur.")
						return true
					}
				}
				_ = q(`UPDATE kateqoriyalar SET ad=$1,tip=$2,limit_mebleq=$3 WHERE id=$4 AND istifadeci_id=$5`, name, typ, lim, id, u.ID)
			} else {
				_ = q(`INSERT INTO kateqoriyalar(istifadeci_id,ad,tip,limit_mebleq) VALUES($1,$2,$3,$4)`, u.ID, name, typ, lim)
			}
		}
		redirect(w, r, path, "ok", "Kateqoriya yeniləndi")
		return true
	case "/goals":
		id, _ := strconv.ParseInt(f("id"), 10, 64)
		switch f("action") {
		case "delete":
			_ = q(`DELETE FROM hedefler WHERE id=$1 AND istifadeci_id=$2`, id, u.ID)
		case "add":
			target, _ := strconv.ParseFloat(f("target"), 64)
			if target <= 0 || strings.TrimSpace(f("name")) == "" {
				redirect(w, r, path, "err", "Hədəf məlumatlarını yoxlayın.")
				return true
			}
			_, _ = db.ExecContext(ctx, `INSERT INTO hedefler(istifadeci_id,ad,hedef_mebleq,baslangic_tarixi,hedef_tarixi,prioritet,qeyd) VALUES($1,$2,$3,$4,$5,$6,$7)`, u.ID, f("name"), target, f("start"), f("end"), f("priority"), f("note"))
		case "cancel":
			_ = q(`UPDATE hedefler SET status='legv_edilib' WHERE id=$1 AND istifadeci_id=$2 AND status='aktiv'`, id, u.ID)
		case "contribute":
			amount, _ := strconv.ParseFloat(f("amount"), 64)
			kind := f("kind")
			if amount > 0 && (kind == "artirma" || kind == "azaltma") {
				tx, _ := db.BeginTx(ctx, nil)
				if tx != nil {
					var goalID int64
					var cur float64
					err := tx.QueryRowContext(ctx, `SELECT id,cari_mebleq FROM hedefler WHERE id=$1 AND istifadeci_id=$2 AND status='aktiv' FOR UPDATE`, id, u.ID).Scan(&goalID, &cur)
					if err == nil {
						next := cur + amount
						if kind == "azaltma" {
							next = cur - amount
						}
						if next < 0 {
							next = 0
						}
						_, err = tx.ExecContext(ctx, `UPDATE hedefler SET cari_mebleq=$1,status=CASE WHEN $1>=hedef_mebleq THEN 'tamamlanib' ELSE status END WHERE id=$2`, next, id)
						if err == nil {
							_, err = tx.ExecContext(ctx, `INSERT INTO hedef_emeliyyatlar(hedef_id,istifadeci_id,mebleq,emeliyyat_novu,tarix) VALUES($1,$2,$3,$4,CURRENT_DATE)`, id, u.ID, amount, kind)
						}
					}
					if err == nil {
						_ = tx.Commit()
					} else {
						_ = tx.Rollback()
					}
				}
			}
		}
		redirect(w, r, path, "ok", "Hədəflər yeniləndi")
		return true
	case "/settings":
		name := strings.TrimSpace(f("name"))
		surname := strings.TrimSpace(f("surname"))
		if name != "" && surname != "" {
			_ = q(`UPDATE istifadeciler SET ad=$1,soyad=$2 WHERE id=$3`, name, surname, u.ID)
		}
		redirect(w, r, path, "ok", "Profil yeniləndi")
		return true
	case "/admin":
		if u.Role != "admin" {
			http.Error(w, "Qadağandır", http.StatusForbidden)
			return true
		}
		id, _ := strconv.ParseInt(f("id"), 10, 64)
		if id != u.ID {
			switch f("action") {
			case "role":
				role := f("role")
				if role == "user" || role == "admin" {
					_ = q(`UPDATE istifadeciler SET rol=$1 WHERE id=$2`, role, id)
				}
			case "status":
				status := f("status")
				if status == "aktiv" || status == "deaktiv" {
					_ = q(`UPDATE istifadeciler SET status=$1 WHERE id=$2`, status, id)
				}
			case "delete":
				_ = q(`DELETE FROM istifadeciler WHERE id=$1`, id)
			}
		}
		redirect(w, r, path, "ok", "İstifadəçi yeniləndi")
		return true
	}
	return false
}

func loadPage(db *sql.DB, r *http.Request, p *Page, path string) {
	ctx := r.Context()
	p.IsAdmin = p.User.Role == "admin"
	month := p.Month
	if len(month) != 7 || month[4] != '-' {
		month = time.Now().Format("2006-01")
		p.Month = month
	}
	if path == "/" || path == "/dashboard" {
		p.Title = "İcmal"
		_ = db.QueryRowContext(ctx, `SELECT COALESCE((SELECT SUM(mebleq) FROM gelirler WHERE istifadeci_id=$1 AND to_char(tarix,'YYYY-MM')=$2),0),COALESCE((SELECT SUM(mebleq) FROM xercler WHERE istifadeci_id=$1 AND to_char(tarix,'YYYY-MM')=$2),0)`, p.User.ID, month).Scan(&p.Income, &p.Expense)
		p.Balance = p.Income - p.Expense
		rows, e := db.QueryContext(ctx, `SELECT x.id,k.ad,x.mebleq,x.tarix::text,x.hesab_novu,x.qeyd,x.kind FROM (SELECT id,kateqoriya_id,mebleq,tarix,hesab_novu,qeyd,'gelir' kind,istifadeci_id FROM gelirler UNION ALL SELECT id,kateqoriya_id,mebleq,tarix,hesab_novu,qeyd,'xerc' kind,istifadeci_id FROM xercler) x JOIN kateqoriyalar k ON k.id=x.kateqoriya_id WHERE x.istifadeci_id=$1 ORDER BY x.tarix DESC,x.id DESC LIMIT 8`, p.User.ID)
		if e == nil {
			defer rows.Close()
			for rows.Next() {
				var x Tx
				_ = rows.Scan(&x.ID, &x.Category, &x.Amount, &x.Date, &x.Account, &x.Note, &x.Kind)
				p.Transactions = append(p.Transactions, x)
			}
		}
		goals(db, ctx, p, false)
		return
	}
	switch path {
	case "/income", "/expenses":
		isIncome := path == "/income"
		table, typ, title := "xercler", "xerc", "Xərclər"
		if isIncome {
			table, typ, title = "gelirler", "gelir", "Gəlirlər"
		}
		p.Title = title
		p.FormType = typ
		rows, _ := db.QueryContext(ctx, `SELECT id,ad,tip,COALESCE(limit_mebleq,0) FROM kateqoriyalar WHERE istifadeci_id=$1 AND tip=$2 ORDER BY ad`, p.User.ID, typ)
		if rows != nil {
			for rows.Next() {
				var c Category
				_ = rows.Scan(&c.ID, &c.Name, &c.Type, &c.Limit)
				p.Categories = append(p.Categories, c)
			}
			rows.Close()
		}
		rows, _ = db.QueryContext(ctx, `SELECT t.id,k.ad,t.mebleq,t.tarix::text,t.hesab_novu,t.qeyd FROM `+table+` t JOIN kateqoriyalar k ON k.id=t.kateqoriya_id WHERE t.istifadeci_id=$1 AND to_char(t.tarix,'YYYY-MM')=$2 ORDER BY t.tarix DESC,t.id DESC`, p.User.ID, month)
		if rows != nil {
			for rows.Next() {
				var x Tx
				x.Kind = typ
				_ = rows.Scan(&x.ID, &x.Category, &x.Amount, &x.Date, &x.Account, &x.Note)
				p.Transactions = append(p.Transactions, x)
			}
			rows.Close()
		}
	case "/categories":
		p.Title = "Kateqoriyalar"
		rows, _ := db.QueryContext(ctx, `SELECT id,ad,tip,COALESCE(limit_mebleq,0) FROM kateqoriyalar WHERE istifadeci_id=$1 ORDER BY tip,ad`, p.User.ID)
		if rows != nil {
			for rows.Next() {
				var c Category
				_ = rows.Scan(&c.ID, &c.Name, &c.Type, &c.Limit)
				p.Categories = append(p.Categories, c)
			}
			rows.Close()
		}
	case "/goals":
		p.Title = "Maliyyə hədəfləri"
		goals(db, ctx, p, true)
	case "/reports":
		p.Title = "Hesabatlar"
		_ = db.QueryRowContext(ctx, `SELECT COALESCE((SELECT SUM(mebleq) FROM gelirler WHERE istifadeci_id=$1 AND to_char(tarix,'YYYY-MM')=$2),0),COALESCE((SELECT SUM(mebleq) FROM xercler WHERE istifadeci_id=$1 AND to_char(tarix,'YYYY-MM')=$2),0)`, p.User.ID, month).Scan(&p.Income, &p.Expense)
		p.Balance = p.Income - p.Expense
		rows, _ := db.QueryContext(ctx, `SELECT k.ad,COALESCE(SUM(x.mebleq),0) FROM kateqoriyalar k LEFT JOIN xercler x ON x.kateqoriya_id=k.id AND x.istifadeci_id=$1 AND to_char(x.tarix,'YYYY-MM')=$2 WHERE k.istifadeci_id=$1 AND k.tip='xerc' GROUP BY k.id ORDER BY 2 DESC`, p.User.ID, month)
		if rows != nil {
			for rows.Next() {
				var name string
				var val float64
				_ = rows.Scan(&name, &val)
				p.Items = append(p.Items, map[string]any{"Name": name, "Amount": val})
			}
			rows.Close()
		}
	case "/settings":
		p.Title = "Hesab parametrləri"
	case "/admin":
		p.Title = "İdarəetmə"
		if !p.IsAdmin {
			return
		}
		var users, transactions, cats int
		_ = db.QueryRowContext(ctx, `SELECT count(*) FROM istifadeciler`).Scan(&users)
		_ = db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM gelirler)+(SELECT count(*) FROM xercler)`).Scan(&transactions)
		_ = db.QueryRowContext(ctx, `SELECT count(*) FROM kateqoriyalar`).Scan(&cats)
		p.Items = append(p.Items, map[string]any{"Name": "İstifadəçilər", "Amount": users}, map[string]any{"Name": "Əməliyyatlar", "Amount": transactions}, map[string]any{"Name": "Kateqoriyalar", "Amount": cats})
		rows, _ := db.QueryContext(ctx, `SELECT id,ad,soyad,email,rol,status FROM istifadeciler ORDER BY yaradilma_tarixi DESC`)
		if rows != nil {
			for rows.Next() {
				var id int64
				var name, surname, email, role, status string
				_ = rows.Scan(&id, &name, &surname, &email, &role, &status)
				p.Items = append(p.Items, map[string]any{"ID": id, "Name": name + " " + surname, "Email": email, "Role": role, "Status": status})
			}
			rows.Close()
		}
	default:
		p.Title = "İcmal"
		p.Active = "/"
	}
}
func goals(db *sql.DB, ctx context.Context, p *Page, all bool) {
	query := `SELECT id,ad,status,prioritet,baslangic_tarixi::text,hedef_tarixi::text,hedef_mebleq,cari_mebleq FROM hedefler WHERE istifadeci_id=$1`
	if !all {
		query += ` AND status='aktiv'`
	}
	query += ` ORDER BY CASE status WHEN 'aktiv' THEN 0 ELSE 1 END,hedef_tarixi LIMIT 10`
	rows, e := db.QueryContext(ctx, query, p.User.ID)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var g Goal
		_ = rows.Scan(&g.ID, &g.Name, &g.Status, &g.Priority, &g.Start, &g.End, &g.Target, &g.Current)
		p.Goals = append(p.Goals, g)
	}
}
