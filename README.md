# Go Reserveringsapplicatie

Deze applicatie is een reserveringstool gebouwd in Go. Gebruikers kunnen accommodaties bekijken en reserveringen maken. Beheerders kunnen via een loginomgeving reserveringen en accommodaties beheren.

---

## Installatie

### 1. Vereisten

- Go 1.21+
- MySQL database (in Azure)
- `config.json` bestand
- Root TLS-certificaat (Voor Azure)
- HTML-templates en statische bestanden

### 2. Projectstructuur

```
project/
│
├── main.go
├── config.json
├── certs/
│   └── DigiCertGlobalRootCA.crt.pem
├── static/
│   └── ... (CSS, JS, etc.)
├── templates/
│   ├── base.html
│   ├── home.html
│   ├── reserveren.html
│   ├── admin_login.html
│   └── ...
```

### 3. Configuratiebestand (`config.json`)

```json
{
  "adminWachtwoord": "adminwachtwoord",
  "mysql": {
    "gebruiker": "gebruiker",
    "wachtwoord": "wachtwoord",
    "host": "mysqldb.mysql.database.azure.com",
    "database": "voorbeeldb",
    "certificaatPad": "certs/DigiCertGlobalRootCA.crt.pem"
  },
  "server": {
    "poort": "80",
    "https": false
  },
  "cookieSecret": "iets-geheims-hier"
}
```

---

## Benodigde SQL Queries (Database setup)

Voer deze queries uit in je MySQL-database om de tabellen correct aan te maken:

### 1. Tabel `reserveringen`

```sql
CREATE TABLE reserveringen (
    id INT AUTO_INCREMENT PRIMARY KEY,
    voornaam VARCHAR(100) NOT NULL,
    tussenvoegsel VARCHAR(20),
    achternaam VARCHAR(100) NOT NULL,
    begindatum DATE NOT NULL,
    einddatum DATE NOT NULL,
    kenteken VARCHAR(20),
    email VARCHAR(100),
    telefoon VARCHAR(20),
    accommodatie VARCHAR(100) NOT NULL
);
```

**Uitleg:**
- Slaat alle reserveringsgegevens op van gebruikers.
- De `id` is een automatisch oplopende unieke sleutel.

---

### 2. Tabel `accommodaties`

```sql
CREATE TABLE accommodaties (
    id INT AUTO_INCREMENT PRIMARY KEY,
    naam VARCHAR(100) UNIQUE NOT NULL
);
```

**Uitleg:**
- Bevat de lijst van beschikbare accommodaties.
- De naam moet uniek zijn en wordt gebruikt bij het maken van reserveringen.

---

## Adminpaneel

### URL's:
- `/admin/login` – Loginpagina voor beheerders
- `/admin/dashboard` – Overzicht na inloggen
- `/admin/reserveringen` – Bekijk alle reserveringen
- `/admin/accommodaties` – Beheer de accommodaties (toevoegen/verwijderen)

**Loginwachtwoord**: ingesteld via `config.json > adminWachtwoord`.

---

## Templates

De `templates/` map bevat alle HTML-bestanden. Zorg voor minimaal deze bestanden:

- `base.html` – Bevat de layout (header/footer)
- `home.html` – Welkomstpagina
- `reserveren.html` – Formulier voor reservering
- `accommodaties.html` – Formulier voor reservering
- `admin_login.html` – Inlogpagina admin
- `admin_dashboard.html` – Admin startpagina
- `admin_reserveringen.html` – Overzicht reserveringen
- `admin_accommodaties.html` – Beheer accommodaties

---

## Extra tips

- Zet `https: true` in `config.json` voor productiegebruik (zorg voor geldige TLS-certificaten).
- Plaats de TLS-certificaten in `certs/` en verwijs ernaar via `config.json`.
- De cookie secret in `config.json` moet lang en uniek zijn voor veiligheid.

---

## Start de applicatie

```bash
go run main.go
```

Of compileer:

```bash
go build -o reservering-app
./reservering-app
```

---

## Feedback of hulp nodig?

Neem gerust contact op met de ontwikkelaar of open een issue op GitHub.
