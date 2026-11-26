// Command seeder floods a running AI-Clinic backend with realistic random
// users (patients, doctors, nurses), doctor profiles, patient profiles and
// doctor→nurse assignments, then writes categorized result files.
//
// Usage:
//
//	cp .env.example .env   # edit as needed
//	go run .               # or: go build -o seeder . && ./seeder
//
// Flags:
//
//	-env path   path to the .env file (default ".env")
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	envPath := flag.String("env", ".env", "path to the .env configuration file")
	flag.Parse()

	log.SetFlags(log.Ltime)

	cfg, err := LoadConfig(*envPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	initRNG(cfg.RandomSeed)

	printPlan(cfg)

	start := time.Now()
	seeder := NewSeeder(cfg)
	if err := seeder.Run(); err != nil {
		log.Fatalf("seeding failed: %v", err)
	}

	files, err := seeder.WriteOutputs()
	if err != nil {
		log.Fatalf("writing outputs failed: %v", err)
	}

	printSummary(seeder, files, time.Since(start))
}

func printPlan(cfg *Config) {
	fmt.Println("──────────────────────────────────────────────")
	fmt.Println(" AI-Clinic data seeder")
	fmt.Println("──────────────────────────────────────────────")
	fmt.Printf("  server        : %s\n", cfg.ServerURL)
	fmt.Printf("  admin         : %s\n", cfg.AdminEmail)
	fmt.Printf("  patients      : %d\n", cfg.PatientCount)
	fmt.Printf("  doctors       : %d (profiles=%v, verify=%v)\n", cfg.DoctorCount, cfg.CreateDoctorProfiles, cfg.VerifyDoctors)
	fmt.Printf("  nurses        : %d (%d-%d per doctor)\n", cfg.NurseCount, cfg.NursesPerDoctorMin, cfg.NursesPerDoctorMax)
	fmt.Printf("  patient prof. : %v\n", cfg.CreatePatientProfiles)
	fmt.Printf("  concurrency   : %d\n", cfg.Concurrency)
	if cfg.RandomSeed != 0 {
		fmt.Printf("  random seed   : %d (reproducible)\n", cfg.RandomSeed)
	}
	fmt.Println("──────────────────────────────────────────────")
	fmt.Println("  NOTE: /auth is rate-limited to 5 req/min in production.")
	fmt.Println("        Run the server with APP_ENV=development to seed fast.")
	fmt.Println("──────────────────────────────────────────────")
}

func printSummary(s *Seeder, files []string, elapsed time.Duration) {
	verified, assignments, nursesLinked := 0, 0, 0
	for _, d := range s.doctors {
		if d.Verified {
			verified++
		}
		assignments += len(d.AssignedNurseIDs)
	}
	for _, n := range s.nurses {
		if len(n.AssignedByDoctorIDs) > 0 {
			nursesLinked++
		}
	}

	fmt.Println()
	fmt.Println("──────────────────────────────────────────────")
	fmt.Println(" Done ✓")
	fmt.Println("──────────────────────────────────────────────")
	fmt.Printf("  patients registered : %d\n", len(s.patients))
	fmt.Printf("  doctors registered  : %d (verified %d)\n", len(s.doctors), verified)
	fmt.Printf("  nurses registered   : %d (linked %d)\n", len(s.nurses), nursesLinked)
	fmt.Printf("  nurse assignments   : %d\n", assignments)
	fmt.Printf("  elapsed             : %s\n", elapsed.Round(time.Millisecond))
	fmt.Println("  output files:")
	for _, f := range files {
		fmt.Printf("    - %s\n", f)
	}
	fmt.Println("──────────────────────────────────────────────")

	if len(s.all) == 0 {
		os.Exit(1)
	}
}
