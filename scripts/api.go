package main

import "fmt"

// ── Request payloads (only fields the backend DTOs accept) ──────────────────

type registerReq struct {
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Password  string `json:"password"`
	Role      string `json:"role,omitempty"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type doctorProfileReq struct {
	StartedAt            string   `json:"startedAt"`
	Specialty            string   `json:"specialty"`
	SecondarySpecialties []string `json:"secondarySpecialties,omitempty"`
	ClinicLocation       string   `json:"clinicLocation,omitempty"`
	Bio                  string   `json:"bio,omitempty"`
	University           string   `json:"university,omitempty"`
	VisitMethods         []string `json:"visitMethods,omitempty"`
	VisitTypes           []string `json:"visitTypes,omitempty"`
	PhoneNumber          string   `json:"phoneNumber,omitempty"`
	Languages            []string `json:"languages,omitempty"`
	LicenseNumber        string   `json:"licenseNumber,omitempty"`
}

type patientProfileReq struct {
	Location       string   `json:"location,omitempty"`
	Bio            string   `json:"bio,omitempty"`
	MedicalHistory []string `json:"medicalHistory,omitempty"`
	Allergies      []string `json:"allergies,omitempty"`
	Medications    []string `json:"medications,omitempty"`
	Surgeries      []string `json:"surgeries,omitempty"`
	FamilyHistory  []string `json:"familyHistory,omitempty"`
}

type verifyReq struct {
	Approved bool `json:"approved"`
}

type roleReq struct {
	Role string `json:"role"`
}

type assignReq struct {
	NurseID     string   `json:"nurseId"`
	Permissions []string `json:"permissions,omitempty"`
}

// ── Response shapes we care about ───────────────────────────────────────────

type idStringResp struct {
	ID string `json:"id"`
}
type idIntResp struct {
	ID int `json:"id"`
}

// ── Endpoint wrappers ───────────────────────────────────────────────────────

// register creates a user and (server-side) opens an authenticated session on
// this Session's cookie jar. Returns the new user id.
func (s *Session) register(r registerReq) (string, error) {
	var out idStringResp
	if err := s.do("POST", "/auth/register", r, &out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("register: server returned no id")
	}
	return out.ID, nil
}

// login authenticates this session (used for the admin/superadmin).
func (s *Session) login(email, password string) (string, error) {
	var out idStringResp
	if err := s.do("POST", "/auth/login", loginReq{Email: email, Password: password}, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

// createDoctorProfile creates the caller's doctor profile. Returns profile id.
func (s *Session) createDoctorProfile(r doctorProfileReq) (int, error) {
	var out idIntResp
	if err := s.do("POST", "/doctor", r, &out); err != nil {
		return 0, err
	}
	return out.ID, nil
}

// createPatientProfile creates the caller's patient medical profile.
func (s *Session) createPatientProfile(r patientProfileReq) error {
	return s.do("POST", "/patient/profile", r, nil)
}

// verifyDoctor (admin) approves a doctor profile by its integer id.
func (s *Session) verifyDoctor(profileID int) error {
	return s.do("PATCH", fmt.Sprintf("/admin/doctors/%d/verify", profileID), verifyReq{Approved: true}, nil)
}

// setUserRole (admin) updates a user's role by user uuid.
func (s *Session) setUserRole(userID, role string) error {
	return s.do("PATCH", "/admin/users/"+userID, roleReq{Role: role}, nil)
}

// assignNurse (doctor) invites a user (by uuid) to the caller's team with the
// given permissions. Returns the assignment id; it grants nothing until the
// invitee accepts it (acceptNurseInvitation).
func (s *Session) assignNurse(nurseID string, permissions []string) (int, error) {
	var out idIntResp
	if err := s.do("POST", "/nurse/assign", assignReq{NurseID: nurseID, Permissions: permissions}, &out); err != nil {
		return 0, err
	}
	return out.ID, nil
}

// acceptNurseInvitation (invitee) accepts a pending invitation, which puts the
// assignment in effect (and makes a patient a NURSE).
func (s *Session) acceptNurseInvitation(assignmentID int) error {
	return s.do("POST", fmt.Sprintf("/nurse/invitations/%d/accept", assignmentID), nil, nil)
}
