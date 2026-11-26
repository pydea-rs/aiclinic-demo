package main

import (
	"fmt"
	"strings"
	"sync"
)

// Identity is a fully-formed, unique login identity for one user.
type Identity struct {
	FirstName string
	LastName  string
	FullName  string
	Username  string
	Email     string
	Password  string
}

// generateIdentities builds `count` unique identities using a small concurrent
// pipeline, as follows:
//
//	[nameGen goroutine] ──candidates──▶ [collector] ─▶ unique (username,email)
//	[passwordGen goroutine] ─▶ passwords slice (uniqueness not required)
//
// The collector applies an "append technique" (numbers / age suffixes) whenever
// a generated username/email collides with one already taken. Once `count`
// unique identities and `count` passwords exist, they are zipped by index.
func generateIdentities(cfg *Config, count int) []Identity {
	if count <= 0 {
		return nil
	}

	// ── name candidate generator ────────────────────────────────────────────
	candCh := make(chan [2]string, 256)
	done := make(chan struct{})
	go func() {
		for {
			cand := [2]string{pick(rng, firstNames), pick(rng, lastNames)}
			select {
			case <-done:
				return
			case candCh <- cand:
			}
		}
	}()

	// ── password generator (runs concurrently, no uniqueness needed) ─────────
	passwords := make([]string, count)
	var passWG sync.WaitGroup
	passWG.Add(1)
	go func() {
		defer passWG.Done()
		for i := 0; i < count; i++ {
			passwords[i] = generatePassword(cfg.PasswordLength)
		}
	}()

	// ── collector: dedupe into unique identities ─────────────────────────────
	identities := make([]Identity, 0, count)
	usedUsernames := make(map[string]struct{}, count)

	for len(identities) < count {
		cand := <-candCh
		first, last := cand[0], cand[1]
		baseUser := baseUsername(first, last)

		username := uniquify(baseUser, usedUsernames)
		usedUsernames[username] = struct{}{}

		identities = append(identities, Identity{
			FirstName: first,
			LastName:  last,
			FullName:  fmt.Sprintf("%s %s", first, last),
			Username:  username,
			Email:     username + "@" + cfg.EmailDomain,
		})
	}
	close(done) // stop the generator goroutine

	passWG.Wait()
	for i := range identities {
		identities[i].Password = passwords[i]
	}
	return identities
}

// baseUsername builds a clean "first.last" handle from a name.
func baseUsername(first, last string) string {
	clean := func(s string) string {
		var b strings.Builder
		for _, r := range strings.ToLower(s) {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
	return clean(first) + "." + clean(last)
}

// uniquify returns base if free, otherwise applies the append technique until a
// free handle is found: first a random age (18-90), then incrementing numeric /
// sign suffixes. Guaranteed to terminate.
func uniquify(base string, used map[string]struct{}) string {
	if _, taken := used[base]; !taken {
		return base
	}

	// Try an age-style suffix first (feels natural: jane.doe27).
	age := rng.between(18, 90)
	if cand := fmt.Sprintf("%s%d", base, age); !isTaken(cand, used) {
		return cand
	}

	// Fall back to incrementing numeric suffixes with occasional separators.
	seps := []string{"", ".", "_"}
	for n := 1; ; n++ {
		sep := seps[rng.intn(len(seps))]
		cand := fmt.Sprintf("%s%s%d", base, sep, n)
		if !isTaken(cand, used) {
			return cand
		}
	}
}

func isTaken(s string, used map[string]struct{}) bool {
	_, ok := used[s]
	return ok
}

const (
	lowerChars = "abcdefghijkmnpqrstuvwxyz"
	upperChars = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	digitChars = "23456789"
	allChars   = lowerChars + upperChars + digitChars
)

// generatePassword builds a password that satisfies the backend rule:
// ^(?=.*[a-z])(?=.*[A-Z])(?=.*[0-9]).{8,}$  — at least one lower, upper, digit.
func generatePassword(length int) string {
	if length < 8 {
		length = 8
	}
	b := make([]byte, length)
	// Guarantee one of each required category.
	b[0] = lowerChars[rng.intn(len(lowerChars))]
	b[1] = upperChars[rng.intn(len(upperChars))]
	b[2] = digitChars[rng.intn(len(digitChars))]
	for i := 3; i < length; i++ {
		b[i] = allChars[rng.intn(len(allChars))]
	}
	// Shuffle so the guaranteed chars are not always in front.
	rng.mu.Lock()
	rng.r.Shuffle(len(b), func(i, j int) { b[i], b[j] = b[j], b[i] })
	rng.mu.Unlock()
	return string(b)
}
