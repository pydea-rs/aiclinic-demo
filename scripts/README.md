# AI-Clinic data seeder

A small, dependency-free **Go** program that floods a running AI-Clinic backend
with realistic, randomized data so the app feels "lived-in" for manual testing:
patients, doctors (with verified profiles), nurses, patient medical profiles,
and doctor→nurse assignments.

It talks to the backend exactly like a browser would — cookie sessions + CSRF
double-submit token — so it exercises the real API surface end-to-end.

## What it creates

- **Patients** — registered accounts + random medical profiles (history,
  allergies, medications, surgeries, family history).
- **Doctors** — registered accounts + full doctor profiles (specialty, visit
  methods/types, clinic, languages, license…), then **admin-verified** so they
  show up in the public doctor listings.
- **Nurses** — registered accounts, promoted to the `NURSE` role by the admin,
  then invited by one or more doctors with random permissions; each nurse
  accepts the invitation (`POST /nurse/invitations/:id/accept`), since an
  invitation grants nothing until it is accepted.
- **Relations** — every nurse is guaranteed to be assigned to at least one
  doctor; each doctor gets a configurable random number of nurses.

## Requirements

- Go 1.21+
- A running AI-Clinic backend with its database migrated and the **superadmin
  seeded** (default `admin@ai-clinic.com` / `SuperAdmin123!`).
- **Run the backend with `APP_ENV=development`** (or `NODE_ENV=development`).
  This disables the strict auth rate limit (5 req/min) so seeding is fast. In
  production the limit is enforced and the seeder will fall back to slow,
  backed-off retries.

## Usage

```bash
cd scripts
cp .env.example .env      # then edit as needed
go run .                  # or: go build -o seeder . && ./seeder
```

Point it at a different env file with `-env`:

```bash
go run . -env ./.env.staging
```

## Configuration (`.env`)

| Key | Default | Meaning |
|-----|---------|---------|
| `SERVER_URL` | `http://localhost:8080` | Backend base URL |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | superadmin creds | Used for role management + doctor verification |
| `PATIENT_COUNT` / `DOCTOR_COUNT` / `NURSE_COUNT` | 80 / 20 / 30 | How many of each to create |
| `NURSES_PER_DOCTOR_MIN` / `_MAX` | 1 / 3 | Random nurses assigned per doctor |
| `CREATE_DOCTOR_PROFILES` | true | Forced on when `NURSE_COUNT > 0` |
| `VERIFY_DOCTORS` | true | Admin-verify doctor profiles |
| `CREATE_PATIENT_PROFILES` | true | Create patient medical profiles |
| `EMAIL_DOMAIN` | `seed.aiclinic.local` | Domain for generated emails |
| `PASSWORD_LENGTH` | 12 | Generated password length (≥8) |
| `RANDOM_SEED` | 0 | `0` = time-based; any other value = reproducible run |
| `CONCURRENCY` | 10 | Max concurrent in-flight requests |
| `REQUEST_TIMEOUT_SEC` | 30 | Per-request timeout |
| `MAX_RETRIES` | 6 | Retries on 429/5xx (exponential backoff) |
| `OUTPUT_DIR` | `.` | Where to write the result files |

Any value can also be provided as a real environment variable (which overrides
the `.env` file), e.g. `PATIENT_COUNT=200 go run .`.

## Output files

Written to `OUTPUT_DIR` (the script folder by default). Each record includes the
full credentials (`email`, `username`, `password`, `role`, ids, …) so you can
log in as any seeded user.

| File | Contents |
|------|----------|
| `seeded_users.json` | Every successfully created user |
| `seeded_patients.json` | Patients only |
| `seeded_doctors.json` | Doctors only (with profile id, specialty, verified, assigned nurse ids) |
| `seeded_nurses.json` | Nurses only (with the doctor ids they belong to) |
| `seeded_doctor_nurses.json` | Each doctor with the full records of its assigned nurses |

> ⚠️ The output files contain plaintext passwords. They are for local testing —
> don't commit them or use them against a production database.

## How it works (concurrency)

The identity generator is a small goroutine/channel pipeline: one goroutine
emits random `first,last` candidates on a channel while another generates
rule-compliant passwords; a collector deduplicates usernames/emails (applying an
age/number "append technique" on collisions) and zips each identity with a
password by index. Registration, profile creation, verification, promotion and
assignment each run through a bounded worker pool, all sharing per-user cookie
jars so sessions and CSRF tokens behave exactly like real browser clients.
