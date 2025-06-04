package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

type Config struct {
	AdminWachtwoord string `json:"adminWachtwoord"`
	Mysql           struct {
		Gebruiker      string `json:"gebruiker"`
		Wachtwoord     string `json:"wachtwoord"`
		Host           string `json:"host"`
		Database       string `json:"database"`
		CertificaatPad string `json:"certificaatPad"`
	} `json:"mysql"`
	Server struct {
		Poort string `json:"poort"`
		HTTPS bool   `json:"https"`
	} `json:"server"`
	CookieSecret string `json:"cookieSecret"`
}

type Reservering struct {
	ID            int
	Voornaam      string
	Tussenvoegsel string
	Achternaam    string
	BeginDatum    string
	EindDatum     string
	Kenteken      string
	Email         string
	Telefoon      string
	Accommodatie  string
}

var db *sql.DB
var store *sessions.CookieStore
var config Config

func main() {
	// Logbestand openen of aanmaken
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Kan logbestand niet openen: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	// Logging configureren
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("=== Applicatie wordt gestart ===")

	// Config.json inladen
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("Fout bij openen config.json: %v", err)
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		log.Fatalf("Fout bij lezen config.json: %v", err)
	}

	// TLS certificaat voor DB laden
	rootCertPool := x509.NewCertPool()
	pem, err := os.ReadFile(config.Mysql.CertificaatPad)
	if err != nil {
		log.Fatalf("Kan certificaat niet lezen: %v", err)
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		log.Fatal("Kon rootcertificaat niet toevoegen")
	}

	tlsConfig := &tls.Config{
		RootCAs:    rootCertPool,
		ServerName: config.Mysql.Host,
	}
	mysql.RegisterTLSConfig("azure", tlsConfig)

	// DB verbinding
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?tls=azure",
		config.Mysql.Gebruiker, config.Mysql.Wachtwoord,
		config.Mysql.Host, config.Mysql.Database)

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Database open-fout: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Database verbindingsfout: %v", err)
	}
	log.Println("Database succesvol verbonden")

	// Sessions
	store = sessions.NewCookieStore([]byte(config.CookieSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   config.Server.HTTPS,
		SameSite: http.SameSiteLaxMode,
	}

	// Router
	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/accommodaties", accommodatiesHandler)
	r.HandleFunc("/reserveren", reserveerHandler)
	r.HandleFunc("/contact", contactHandler)
	r.HandleFunc("/over", overHandler)

	r.HandleFunc("/admin/login", adminLoginHandler)
	r.HandleFunc("/admin/dashboard", adminDashboardHandler)
	r.HandleFunc("/admin/reserveringen", adminReserveringenHandler)
	r.HandleFunc("/admin/accommodaties", adminAccommodatiesHandler)

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start server
	log.Printf("Server wordt gestart op poort %s (HTTPS: %v)\n", config.Server.Poort, config.Server.HTTPS)
	if config.Server.HTTPS {
		log.Fatal(http.ListenAndServeTLS(":"+config.Server.Poort, "certs/cert.pem", "certs/privkey.pem", r))
	} else {
		log.Fatal(http.ListenAndServe(":"+config.Server.Poort, r))
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, err := template.ParseFiles("templates/base.html", "templates/"+tmpl+".html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	t.ExecuteTemplate(w, "base", data)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "home", nil)
}

func accommodatiesHandler(w http.ResponseWriter, r *http.Request) {
	accommodaties, err := getAccommodaties()
	if err != nil {
		http.Error(w, "Kan accommodaties niet laden: "+err.Error(), 500)
		return
	}
	renderTemplate(w, "accommodaties", accommodaties)
}

func contactHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "contact", nil)
}

func overHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "over", nil)
}

func reserveerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		accommodaties, err := getAccommodaties()
		if err != nil {
			http.Error(w, "Kan accommodaties niet laden: "+err.Error(), 500)
			return
		}
		data := struct {
			Accommodaties []string
			Vandaag       string
		}{
			Accommodaties: accommodaties,
			Vandaag:       time.Now().Format("2006-01-02"),
		}
		renderTemplate(w, "reserveren", data)

	case http.MethodPost:
		r.ParseForm()
		_, err := db.Exec(`
			INSERT INTO reserveringen (
				voornaam, tussenvoegsel, achternaam, begindatum, einddatum,
				kenteken, email, telefoon, accommodatie
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.FormValue("voornaam"), r.FormValue("tussenvoegsel"),
			r.FormValue("achternaam"), r.FormValue("begindatum"), r.FormValue("einddatum"),
			r.FormValue("kenteken"), r.FormValue("email"), r.FormValue("telefoon"),
			r.FormValue("accommodatie"),
		)
		if err != nil {
			http.Error(w, "Fout bij opslaan reservering: "+err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func adminLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if r.FormValue("wachtwoord") == config.AdminWachtwoord {
			session, _ := store.Get(r, "session")
			session.Values["isAdmin"] = true
			session.Save(r, w)
			http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
			return
		}
		http.Error(w, "Onjuist wachtwoord", http.StatusUnauthorized)
		return
	}
	renderTemplate(w, "admin_login", nil)
}

func checkAdmin(w http.ResponseWriter, r *http.Request) bool {
	session, _ := store.Get(r, "session")
	if auth, ok := session.Values["isAdmin"].(bool); !ok || !auth {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return false
	}
	return true
}

func adminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdmin(w, r) {
		return
	}
	renderTemplate(w, "admin_dashboard", nil)
}

func adminReserveringenHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdmin(w, r) {
		return
	}
	rows, err := db.Query(`
		SELECT id, voornaam, tussenvoegsel, achternaam, begindatum, einddatum,
		       kenteken, email, telefoon, accommodatie
		FROM reserveringen
	`)
	if err != nil {
		http.Error(w, "Tabel 'reserveringen' ontbreekt in de database.", 500)
		return
	}
	defer rows.Close()

	var reserveringen []Reservering
	for rows.Next() {
		var r Reservering
		err := rows.Scan(
			&r.ID, &r.Voornaam, &r.Tussenvoegsel, &r.Achternaam,
			&r.BeginDatum, &r.EindDatum, &r.Kenteken,
			&r.Email, &r.Telefoon, &r.Accommodatie,
		)
		if err == nil {
			reserveringen = append(reserveringen, r)
		}
	}
	renderTemplate(w, "admin_reserveringen", reserveringen)
}

func adminAccommodatiesHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdmin(w, r) {
		return
	}
	switch r.Method {
	case http.MethodPost:
		actie := r.FormValue("actie")
		naam := r.FormValue("naam")
		if actie == "toevoegen" && naam != "" {
			_, err := db.Exec("INSERT INTO accommodaties (naam) VALUES (?)", naam)
			if err != nil {
				http.Error(w, "Fout bij toevoegen accommodatie: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else if actie == "verwijderen" && naam != "" {
			_, err := db.Exec("DELETE FROM accommodaties WHERE naam = ?", naam)
			if err != nil {
				http.Error(w, "Fout bij verwijderen accommodatie: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
		http.Redirect(w, r, "/admin/accommodaties", http.StatusSeeOther)

	case http.MethodGet:
		rows, err := db.Query("SELECT naam FROM accommodaties")
		if err != nil {
			http.Error(w, "Tabel 'accommodaties' ontbreekt in de database.", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var accommodaties []string
		for rows.Next() {
			var naam string
			if err := rows.Scan(&naam); err == nil {
				accommodaties = append(accommodaties, naam)
			}
		}
		renderTemplate(w, "admin_accommodaties", accommodaties)
	}
}

func getAccommodaties() ([]string, error) {
	rows, err := db.Query("SELECT naam FROM accommodaties")
	if err != nil {
		return nil, fmt.Errorf("tabel 'accommodaties' ontbreekt in de database")
	}
	defer rows.Close()

	var lijst []string
	for rows.Next() {
		var naam string
		if err := rows.Scan(&naam); err == nil {
			lijst = append(lijst, naam)
		}
	}
	return lijst, nil
}