package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// doctorWithNurses is the relational view: a doctor plus the full records of the
// nurses assigned to them.
type doctorWithNurses struct {
	Doctor *SeededUser   `json:"doctor"`
	Nurses []*SeededUser `json:"nurses"`
}

// WriteOutputs writes the five result files into cfg.OutputDir.
func (s *Seeder) WriteOutputs() ([]string, error) {
	dir := s.cfg.OutputDir
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	files := map[string]any{
		"seeded_users.json":         s.all,
		"seeded_patients.json":      s.patients,
		"seeded_doctors.json":       s.doctors,
		"seeded_nurses.json":        s.nurses,
		"seeded_doctor_nurses.json": s.doctorNurseView(),
	}

	written := make([]string, 0, len(files))
	for name, data := range files {
		path := filepath.Join(dir, name)
		if err := writeJSON(path, data); err != nil {
			return written, fmt.Errorf("writing %s: %w", name, err)
		}
		written = append(written, path)
	}
	return written, nil
}

func (s *Seeder) doctorNurseView() []doctorWithNurses {
	byID := make(map[string]*SeededUser, len(s.nurses))
	for _, n := range s.nurses {
		byID[n.ID] = n
	}
	out := make([]doctorWithNurses, 0, len(s.doctors))
	for _, d := range s.doctors {
		nurses := make([]*SeededUser, 0, len(d.AssignedNurseIDs))
		for _, nid := range d.AssignedNurseIDs {
			if n := byID[nid]; n != nil {
				nurses = append(nurses, n)
			}
		}
		out = append(out, doctorWithNurses{Doctor: d, Nurses: nurses})
	}
	return out
}

func writeJSON(path string, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}
