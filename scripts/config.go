package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds every tunable knob, loaded from the .env file (with sane
// defaults applied for anything missing).
type Config struct {
	ServerURL     string
	AdminEmail    string
	AdminPassword string

	PatientCount int
	DoctorCount  int
	NurseCount   int

	NursesPerDoctorMin int
	NursesPerDoctorMax int

	CreateDoctorProfiles  bool
	VerifyDoctors         bool
	CreatePatientProfiles bool

	EmailDomain    string
	PasswordLength int
	RandomSeed     int64

	Concurrency    int
	RequestTimeout int // seconds
	MaxRetries     int

	OutputDir string
}

// loadDotEnv reads a KEY=VALUE file into a map. Missing file is not an error —
// the caller falls back to process env / defaults. Lines starting with '#' and
// blank lines are ignored; surrounding quotes on values are stripped.
func loadDotEnv(path string) (map[string]string, error) {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		val = strings.Trim(val, `"'`)
		out[key] = val
	}
	return out, sc.Err()
}

// LoadConfig merges .env file values with real environment variables (env wins)
// and applies defaults, then validates.
func LoadConfig(envPath string) (*Config, error) {
	fileVals, err := loadDotEnv(envPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", envPath, err)
	}

	get := func(key, def string) string {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			return v
		}
		if v, ok := fileVals[key]; ok && v != "" {
			return v
		}
		return def
	}
	getInt := func(key string, def int) int {
		if n, err := strconv.Atoi(get(key, "")); err == nil {
			return n
		}
		return def
	}
	getBool := func(key string, def bool) bool {
		v := strings.ToLower(get(key, ""))
		switch v {
		case "true", "1", "yes", "y", "on":
			return true
		case "false", "0", "no", "n", "off":
			return false
		}
		return def
	}

	c := &Config{
		ServerURL:             strings.TrimRight(get("SERVER_URL", "http://localhost:8080"), "/"),
		AdminEmail:            get("ADMIN_EMAIL", "admin@ai-clinic.com"),
		AdminPassword:         get("ADMIN_PASSWORD", "SuperAdmin123!"),
		PatientCount:          getInt("PATIENT_COUNT", 80),
		DoctorCount:           getInt("DOCTOR_COUNT", 20),
		NurseCount:            getInt("NURSE_COUNT", 30),
		NursesPerDoctorMin:    getInt("NURSES_PER_DOCTOR_MIN", 1),
		NursesPerDoctorMax:    getInt("NURSES_PER_DOCTOR_MAX", 3),
		CreateDoctorProfiles:  getBool("CREATE_DOCTOR_PROFILES", true),
		VerifyDoctors:         getBool("VERIFY_DOCTORS", true),
		CreatePatientProfiles: getBool("CREATE_PATIENT_PROFILES", true),
		EmailDomain:           get("EMAIL_DOMAIN", "seed.aiclinic.local"),
		PasswordLength:        getInt("PASSWORD_LENGTH", 12),
		RandomSeed:            int64(getInt("RANDOM_SEED", 0)),
		Concurrency:           getInt("CONCURRENCY", 10),
		RequestTimeout:        getInt("REQUEST_TIMEOUT_SEC", 30),
		MaxRetries:            getInt("MAX_RETRIES", 6),
		OutputDir:             get("OUTPUT_DIR", "."),
	}

	// Nurse assignment needs doctor profiles; force them on.
	if c.NurseCount > 0 {
		c.CreateDoctorProfiles = true
	}

	return c, c.validate()
}

func (c *Config) validate() error {
	if c.ServerURL == "" {
		return fmt.Errorf("SERVER_URL is required")
	}
	if c.PatientCount < 0 || c.DoctorCount < 0 || c.NurseCount < 0 {
		return fmt.Errorf("counts must be non-negative")
	}
	if c.PatientCount+c.DoctorCount+c.NurseCount == 0 {
		return fmt.Errorf("nothing to seed: all counts are zero")
	}
	if c.NurseCount > 0 && c.DoctorCount == 0 {
		return fmt.Errorf("NURSE_COUNT > 0 requires at least one doctor (nurses are assigned by doctors)")
	}
	if c.NursesPerDoctorMin < 0 || c.NursesPerDoctorMax < c.NursesPerDoctorMin {
		return fmt.Errorf("invalid NURSES_PER_DOCTOR range")
	}
	if c.PasswordLength < 8 {
		c.PasswordLength = 8
	}
	if c.Concurrency < 1 {
		c.Concurrency = 1
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}
	return nil
}

func (c *Config) TotalUsers() int { return c.PatientCount + c.DoctorCount + c.NurseCount }
