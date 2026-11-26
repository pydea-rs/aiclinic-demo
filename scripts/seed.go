package main

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// SeededUser is the record persisted to the output files. Unexported fields
// (session, mu) are ignored by encoding/json.
type SeededUser struct {
	ID        string `json:"id"`
	FullName  string `json:"fullName"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Role      string `json:"role"`

	// Doctor-only
	DoctorProfileID  int      `json:"doctorProfileId,omitempty"`
	Specialty        string   `json:"specialty,omitempty"`
	Verified         bool     `json:"verified,omitempty"`
	AssignedNurseIDs []string `json:"assignedNurseIds,omitempty"`

	// Nurse-only
	AssignedByDoctorIDs []string `json:"assignedByDoctorIds,omitempty"`

	session *Session
	mu      sync.Mutex
}

// Seeder drives the whole seeding process.
type Seeder struct {
	cfg   *Config
	admin *Session

	patients []*SeededUser
	doctors  []*SeededUser
	nurses   []*SeededUser
	all      []*SeededUser
}

func NewSeeder(cfg *Config) *Seeder { return &Seeder{cfg: cfg} }

// Run executes every phase in order and returns the successfully seeded users.
func (s *Seeder) Run() error {
	// 0. Admin login (needed for role management + doctor verification).
	admin, err := newSession(s.cfg)
	if err != nil {
		return err
	}
	if err := admin.prime(); err != nil {
		return fmt.Errorf("priming admin session: %w", err)
	}
	if _, err := admin.login(s.cfg.AdminEmail, s.cfg.AdminPassword); err != nil {
		return fmt.Errorf("admin login failed (check ADMIN_EMAIL/ADMIN_PASSWORD): %w", err)
	}
	s.admin = admin
	log.Printf("✓ admin authenticated as %s", s.cfg.AdminEmail)

	// 1. Generate unique identities and split into role buckets.
	total := s.cfg.TotalUsers()
	log.Printf("generating %d unique identities…", total)
	ids := generateIdentities(s.cfg, total)

	p, d := s.cfg.PatientCount, s.cfg.DoctorCount
	patientIDs := ids[:p]
	doctorIDs := ids[p : p+d]
	nurseIDs := ids[p+d:]

	// 2. Register everyone (patients + nurses as PATIENT, doctors as DOCTOR).
	log.Printf("registering users (patients=%d doctors=%d nurses=%d)…", p, d, s.cfg.NurseCount)
	s.patients = s.registerBatch(patientIDs, "PATIENT")
	s.doctors = s.registerBatch(doctorIDs, "DOCTOR")
	s.nurses = s.registerBatch(nurseIDs, "PATIENT") // promoted to NURSE below
	for _, u := range s.nurses {
		u.Role = "NURSE" // intended final role
	}
	s.all = append(append(append([]*SeededUser{}, s.patients...), s.doctors...), s.nurses...)
	log.Printf("✓ registered %d/%d users", len(s.all), total)

	// 3. Doctor profiles (required for nurse assignment) + verification.
	if s.cfg.CreateDoctorProfiles {
		s.createDoctorProfiles()
	}
	if s.cfg.VerifyDoctors {
		s.verifyDoctors()
	}

	// 4. Patient medical profiles.
	if s.cfg.CreatePatientProfiles {
		s.createPatientProfiles()
	}

	// 5. Promote nurse users to the NURSE role (admin), so every nurse has the
	//    correct role even if some don't end up assigned.
	s.promoteNurses()

	// 6. Doctor → nurse assignments (creates the relations).
	s.assignNurses()

	return nil
}

// ── Phase implementations ───────────────────────────────────────────────────

func (s *Seeder) registerBatch(ids []Identity, role string) []*SeededUser {
	out := make([]*SeededUser, 0, len(ids))
	var mu sync.Mutex
	var failures int64

	runPool(ids, s.cfg.Concurrency, func(id Identity) {
		sess, err := newSession(s.cfg)
		if err != nil {
			atomic.AddInt64(&failures, 1)
			return
		}
		if err := sess.prime(); err != nil {
			atomic.AddInt64(&failures, 1)
			log.Printf("  ! prime failed for %s: %v", id.Email, err)
			return
		}
		uid, err := sess.register(registerReq{
			Email:     id.Email,
			Firstname: id.FirstName,
			Lastname:  id.LastName,
			Password:  id.Password,
			Role:      role,
		})
		if err != nil {
			atomic.AddInt64(&failures, 1)
			log.Printf("  ! register failed for %s: %v", id.Email, err)
			return
		}
		u := &SeededUser{
			ID: uid, FullName: id.FullName, FirstName: id.FirstName, LastName: id.LastName,
			Username: id.Username, Email: id.Email, Password: id.Password, Role: role,
			session: sess,
		}
		mu.Lock()
		out = append(out, u)
		mu.Unlock()
	})

	if failures > 0 {
		log.Printf("  (%d %s registration(s) failed)", failures, role)
	}
	return out
}

func (s *Seeder) createDoctorProfiles() {
	log.Printf("creating %d doctor profiles…", len(s.doctors))
	var ok int64
	runPool(s.doctors, s.cfg.Concurrency, func(u *SeededUser) {
		req := randomDoctorProfile()
		pid, err := u.session.createDoctorProfile(req)
		if err != nil {
			log.Printf("  ! doctor profile failed for %s: %v", u.Email, err)
			return
		}
		u.DoctorProfileID = pid
		u.Specialty = req.Specialty
		atomic.AddInt64(&ok, 1)
	})
	log.Printf("✓ created %d/%d doctor profiles", ok, len(s.doctors))
}

func (s *Seeder) verifyDoctors() {
	targets := make([]*SeededUser, 0, len(s.doctors))
	for _, u := range s.doctors {
		if u.DoctorProfileID > 0 {
			targets = append(targets, u)
		}
	}
	log.Printf("verifying %d doctors (admin)…", len(targets))
	var ok int64
	runPool(targets, s.cfg.Concurrency, func(u *SeededUser) {
		if err := s.admin.verifyDoctor(u.DoctorProfileID); err != nil {
			log.Printf("  ! verify failed for %s: %v", u.Email, err)
			return
		}
		u.Verified = true
		atomic.AddInt64(&ok, 1)
	})
	log.Printf("✓ verified %d/%d doctors", ok, len(targets))
}

func (s *Seeder) createPatientProfiles() {
	log.Printf("creating %d patient profiles…", len(s.patients))
	var ok int64
	runPool(s.patients, s.cfg.Concurrency, func(u *SeededUser) {
		if err := u.session.createPatientProfile(randomPatientProfile()); err != nil {
			log.Printf("  ! patient profile failed for %s: %v", u.Email, err)
			return
		}
		atomic.AddInt64(&ok, 1)
	})
	log.Printf("✓ created %d/%d patient profiles", ok, len(s.patients))
}

func (s *Seeder) promoteNurses() {
	if len(s.nurses) == 0 {
		return
	}
	log.Printf("promoting %d users to NURSE (admin)…", len(s.nurses))
	var ok int64
	runPool(s.nurses, s.cfg.Concurrency, func(u *SeededUser) {
		if err := s.admin.setUserRole(u.ID, "NURSE"); err != nil {
			log.Printf("  ! promote failed for %s: %v", u.Email, err)
			return
		}
		atomic.AddInt64(&ok, 1)
	})
	log.Printf("✓ promoted %d/%d nurses", ok, len(s.nurses))
}

// nurseAssignmentPlan pairs a doctor with the nurses it should be assigned.
type nurseAssignmentPlan struct {
	doctor *SeededUser
	nurses []*SeededUser
}

func (s *Seeder) assignNurses() {
	docs := s.doctorsWithProfiles()
	if len(docs) == 0 || len(s.nurses) == 0 {
		return
	}

	plans := s.buildAssignmentPlans(docs)

	total := 0
	for _, pl := range plans {
		total += len(pl.nurses)
	}
	log.Printf("assigning nurses to doctors (%d relations)…", total)

	var nurseMu sync.Mutex // guards nurses' AssignedByDoctorIDs (multi-writer)
	var ok int64
	runPool(plans, s.cfg.Concurrency, func(pl nurseAssignmentPlan) {
		for _, nurse := range pl.nurses {
			perms := subset(rng, nursePermissions, 1, 4)
			assignmentID, err := pl.doctor.session.assignNurse(nurse.ID, perms)
			if err != nil {
				if !isConflict(err) {
					log.Printf("  ! assign %s→%s failed: %v", pl.doctor.Email, nurse.Email, err)
				}
				continue
			}
			// A conflict here means a retried request already accepted it.
			if err := nurse.session.acceptNurseInvitation(assignmentID); err != nil && !isConflict(err) {
				log.Printf("  ! %s accepting %s's invitation failed: %v", nurse.Email, pl.doctor.Email, err)
				continue
			}
			// doctor's slice: only this doctor's goroutine writes it → safe
			pl.doctor.AssignedNurseIDs = append(pl.doctor.AssignedNurseIDs, nurse.ID)
			nurseMu.Lock()
			nurse.AssignedByDoctorIDs = append(nurse.AssignedByDoctorIDs, pl.doctor.ID)
			nurseMu.Unlock()
			atomic.AddInt64(&ok, 1)
		}
	})
	log.Printf("✓ created %d/%d nurse assignments", ok, total)
}

func (s *Seeder) doctorsWithProfiles() []*SeededUser {
	out := make([]*SeededUser, 0, len(s.doctors))
	for _, u := range s.doctors {
		if u.DoctorProfileID > 0 {
			out = append(out, u)
		}
	}
	return out
}

// buildAssignmentPlans guarantees every nurse is assigned to at least one
// doctor, then tops each doctor up to a random count in the configured range.
func (s *Seeder) buildAssignmentPlans(docs []*SeededUser) []nurseAssignmentPlan {
	planByDoc := make(map[*SeededUser]map[*SeededUser]bool, len(docs))
	for _, d := range docs {
		planByDoc[d] = map[*SeededUser]bool{}
	}

	// Coverage: each nurse → one random doctor.
	for _, n := range s.nurses {
		d := pick(rng, docs)
		planByDoc[d][n] = true
	}

	// Top-up: each doctor gets a random target within [min,max].
	for _, d := range docs {
		target := rng.between(s.cfg.NursesPerDoctorMin, s.cfg.NursesPerDoctorMax)
		guard := 0
		for len(planByDoc[d]) < target && guard < len(s.nurses)*2 {
			n := pick(rng, s.nurses)
			planByDoc[d][n] = true
			guard++
		}
	}

	plans := make([]nurseAssignmentPlan, 0, len(docs))
	for _, d := range docs {
		nurses := make([]*SeededUser, 0, len(planByDoc[d]))
		for n := range planByDoc[d] {
			nurses = append(nurses, n)
		}
		plans = append(plans, nurseAssignmentPlan{doctor: d, nurses: nurses})
	}
	return plans
}

// ── Random profile builders ─────────────────────────────────────────────────

func randomDoctorProfile() doctorProfileReq {
	yearsAgo := rng.between(1, 30)
	started := time.Now().AddDate(-yearsAgo, 0, 0).Format(time.RFC3339)
	primary := pick(rng, specialties)
	// Secondary specialties are drawn from the pool minus the primary, so a
	// doctor never lists the same specialty twice.
	others := make([]string, 0, len(specialties)-1)
	for _, s := range specialties {
		if s != primary {
			others = append(others, s)
		}
	}
	return doctorProfileReq{
		StartedAt:            started,
		Specialty:            primary,
		SecondarySpecialties: subset(rng, others, 0, 2),
		ClinicLocation:       pick(rng, cities),
		Bio:                  pick(rng, bioPhrasesDoctor),
		University:           pick(rng, universities),
		VisitMethods:         subset(rng, visitMethods, 1, 4),
		VisitTypes:           subset(rng, visitTypes, 1, 3),
		PhoneNumber:          randomPhone(),
		Languages:            subset(rng, languages, 1, 3),
		LicenseNumber:        fmt.Sprintf("MD-%06d", rng.intn(1000000)),
	}
}

func randomPatientProfile() patientProfileReq {
	return patientProfileReq{
		Location:       pick(rng, cities),
		Bio:            pick(rng, bioPhrasesPatient),
		MedicalHistory: subset(rng, medicalHistoryPool, 0, 3),
		Allergies:      subset(rng, allergyPool, 0, 2),
		Medications:    subset(rng, medicationPool, 0, 2),
		Surgeries:      subset(rng, surgeryPool, 0, 2),
		FamilyHistory:  subset(rng, familyHistoryPool, 0, 2),
	}
}

func randomPhone() string {
	return fmt.Sprintf("+1-%03d-%03d-%04d", rng.between(200, 989), rng.intn(1000), rng.intn(10000))
}

// runPool runs fn over items using `concurrency` workers.
func runPool[T any](items []T, concurrency int, fn func(T)) {
	if len(items) == 0 {
		return
	}
	if concurrency < 1 {
		concurrency = 1
	}
	ch := make(chan T)
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for it := range ch {
				fn(it)
			}
		}()
	}
	for _, it := range items {
		ch <- it
	}
	close(ch)
	wg.Wait()
}
