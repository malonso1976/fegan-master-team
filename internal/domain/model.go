package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Club struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ShortName   string `json:"short_name,omitempty"`
	MediumName  string `json:"medium_name,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
	SeasonStart int    `json:"season_start,omitempty"`
}
type Swimmer struct {
	License   string `json:"license"`
	Name      string `json:"name"`
	BirthYear int    `json:"birth_year"`
	Gender    string `json:"gender"`
	ClubID    string `json:"club_id"`
	Member    bool   `json:"member"`
	Available bool   `json:"available"`
}
type Mark struct {
	RetrievedAt   string   `json:"retrieved_at,omitempty"`
	ID            string   `json:"id"`
	License       string   `json:"license"`
	ClubID        string   `json:"club_id"`
	Event         string   `json:"event"`
	Pool          int      `json:"pool"`
	Chrono        string   `json:"chrono"`
	Date          string   `json:"date"`
	Location      string   `json:"location"`
	Centiseconds  int      `json:"centiseconds"`
	Source        string   `json:"source"`
	SourceURL     string   `json:"source_url"`
	Verified      bool     `json:"verified"`
	Notes         string   `json:"notes"`
	RelayLicenses []string `json:"relay_licenses,omitempty"`
	Gender        string   `json:"gender"`
	AgeTotal      int      `json:"age_total,omitempty"`
	RelayCategory int      `json:"relay_category,omitempty"`
}
type Data struct {
	Clubs    []Club    `json:"clubs"`
	Swimmers []Swimmer `json:"swimmers"`
	Marks    []Mark    `json:"marks"`
}

var Events = []string{"50L", "100L", "200L", "400L", "800L", "1500L", "50M", "100M", "200M", "50E", "100E", "200E", "50B", "100B", "200B", "100S", "200S", "400S", "4x50L", "4x100L", "4x200L", "4x50S", "4x100S", "2000L", "3000L", "5000L"}
var identifier = regexp.MustCompile(`^[\pL\pN_-]{1,40}$`)

func ValidID(s string) bool { return identifier.MatchString(s) }
func ValidEvent(s string) bool {
	for _, e := range Events {
		if s == e {
			return true
		}
	}
	return false
}
func Relay(s string) bool { return strings.HasPrefix(s, "4x") }
func (c Club) Validate() error {
	if !ValidID(c.ID) || strings.TrimSpace(c.Name) == "" || len(c.Name) > 200 {
		return errors.New("identificador o nombre de club no válido")
	}
	return nil
}
func (s Swimmer) Validate() error {
	if !ValidID(s.License) || !ValidID(s.ClubID) || strings.TrimSpace(s.Name) == "" || len(s.Name) > 200 {
		return errors.New("licencia, nombre o club no válido")
	}
	if s.BirthYear < 1900 || s.BirthYear > time.Now().Year() {
		return errors.New("año de nacimiento no válido")
	}
	if s.Gender != "M" && s.Gender != "F" {
		return errors.New("categoría de nadador: M o F")
	}
	return nil
}
func (m Mark) Validate() error {
	if !ValidID(m.ClubID) || !ValidEvent(m.Event) || (m.Pool != 25 && m.Pool != 50) || (m.Chrono != "E" && m.Chrono != "M") || m.Centiseconds <= 0 || m.Centiseconds > 24*60*60*100 {
		return errors.New("prueba, club, piscina, cronometraje o tiempo no válido")
	}
	date, err := time.Parse("2006-01-02", m.Date)
	if err != nil || date.After(time.Now()) {
		return errors.New("fecha de marca no válida o futura")
	}
	if m.Source != "manual" && m.Source != "fegan" && m.Source != "fegan-pdf" && m.Source != "local-pdf" {
		return errors.New("procedencia no válida")
	}
	if len(m.Location) > 200 || len(m.Notes) > 2000 {
		return errors.New("texto demasiado largo")
	}
	if Relay(m.Event) {
		if m.License != "" || (len(m.RelayLicenses) != 0 && len(m.RelayLicenses) != 4) || !ValidRelayCategory(m.RelayCategory) || (m.AgeTotal != 0 && m.AgeTotal < m.RelayCategory) || (m.AgeTotal != 0 && (m.AgeTotal < 80 || m.AgeTotal > 500)) || (m.Gender != "M" && m.Gender != "F" && m.Gender != "X") {
			return errors.New("relevo: indicar categoría de edad válida; suma de edades (80–500) e integrantes opcionales")
		}
		seen := map[string]bool{}
		for _, l := range m.RelayLicenses {
			if !ValidID(l) || seen[l] {
				return errors.New("las cuatro licencias del relevo deben ser distintas")
			}
			seen[l] = true
		}
	} else if !ValidID(m.License) || (len(m.RelayLicenses) > 0 || m.RelayCategory != 0 || m.AgeTotal != 0) || (m.Gender != "M" && m.Gender != "F") {
		return errors.New("marca individual: licencia y categoría M/F requeridas")
	}
	return nil
}

var clockPattern = regexp.MustCompile(`^(?:(\d{1,3}):)?(\d{1,3})(?:[.,](\d{1,2}))?$`)

func ParseTime(s string) (int, error) {
	p := clockPattern.FindStringSubmatch(strings.TrimSpace(s))
	if p == nil {
		return 0, errors.New("tiempo esperado: 28.45 o 1:02.34")
	}
	sec, _ := strconv.Atoi(p[2])
	min := 0
	if p[1] != "" {
		min, _ = strconv.Atoi(p[1])
		if sec >= 60 {
			return 0, errors.New("segundos deben ser menores de 60")
		}
	}
	frac := p[3]
	if len(frac) == 1 {
		frac += "0"
	}
	cs, _ := strconv.Atoi(frac)
	total := (min*60+sec)*100 + cs
	if total <= 0 {
		return 0, errors.New("el tiempo debe ser positivo")
	}
	return total, nil
}
func FormatTime(cs int) string {
	if cs >= 6000 {
		return fmt.Sprintf("%d:%02d.%02d", cs/6000, (cs/100)%60, cs%100)
	}
	return fmt.Sprintf("%d.%02d", cs/100, cs%100)
}
