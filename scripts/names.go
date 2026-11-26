package main

// Wide pools of real first and last names. Full names, usernames and emails are
// built by randomly combining one of each (see identity.go).

var firstNames = []string{
	"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda",
	"William", "Elizabeth", "David", "Barbara", "Richard", "Susan", "Joseph", "Jessica",
	"Thomas", "Sarah", "Charles", "Karen", "Christopher", "Nancy", "Daniel", "Lisa",
	"Matthew", "Margaret", "Anthony", "Betty", "Mark", "Sandra", "Donald", "Ashley",
	"Steven", "Kimberly", "Paul", "Emily", "Andrew", "Donna", "Joshua", "Michelle",
	"Kenneth", "Carol", "Kevin", "Amanda", "Brian", "Dorothy", "George", "Melissa",
	"Timothy", "Deborah", "Ronald", "Stephanie", "Edward", "Rebecca", "Jason", "Sharon",
	"Jeffrey", "Laura", "Ryan", "Cynthia", "Jacob", "Kathleen", "Gary", "Amy",
	"Nicholas", "Angela", "Eric", "Shirley", "Jonathan", "Anna", "Stephen", "Brenda",
	"Larry", "Pamela", "Justin", "Emma", "Scott", "Nicole", "Brandon", "Helen",
	"Benjamin", "Samantha", "Samuel", "Katherine", "Gregory", "Christine", "Alexander", "Debra",
	"Patrick", "Rachel", "Frank", "Carolyn", "Raymond", "Janet", "Jack", "Maria",
	"Dennis", "Olivia", "Jerry", "Heather", "Tyler", "Diane", "Aaron", "Julie",
	"Jose", "Joyce", "Adam", "Victoria", "Nathan", "Kelly", "Henry", "Christina",
	"Zachary", "Lauren", "Douglas", "Joan", "Peter", "Evelyn", "Kyle", "Judith",
	"Noah", "Megan", "Ethan", "Cheryl", "Jeremy", "Andrea", "Walter", "Hannah",
	"Christian", "Martha", "Keith", "Jacqueline", "Roger", "Frances", "Terry", "Gloria",
	"Sean", "Ann", "Gerald", "Teresa", "Harold", "Kathryn", "Carl", "Sophia",
	"Arthur", "Grace", "Lawrence", "Sara", "Dylan", "Isabella", "Amir", "Layla",
	"Omar", "Yasmin", "Ibrahim", "Fatima", "Chen", "Mei", "Hiroshi", "Aiko",
	"Diego", "Camila", "Mateo", "Valentina", "Priya", "Arjun", "Ananya", "Rohan",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
	"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas",
	"Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson", "White",
	"Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson", "Walker", "Young",
	"Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores",
	"Green", "Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell",
	"Carter", "Roberts", "Gomez", "Phillips", "Evans", "Turner", "Diaz", "Parker",
	"Cruz", "Edwards", "Collins", "Reyes", "Stewart", "Morris", "Morales", "Murphy",
	"Cook", "Rogers", "Gutierrez", "Ortiz", "Morgan", "Cooper", "Peterson", "Bailey",
	"Reed", "Kelly", "Howard", "Ramos", "Kim", "Cox", "Ward", "Richardson",
	"Watson", "Brooks", "Chavez", "Wood", "James", "Bennett", "Gray", "Mendoza",
	"Ruiz", "Hughes", "Price", "Alvarez", "Castillo", "Sanders", "Patel", "Myers",
	"Long", "Ross", "Foster", "Jimenez", "Powell", "Jenkins", "Perry", "Russell",
	"Sullivan", "Bell", "Coleman", "Butler", "Henderson", "Barnes", "Gonzales", "Fisher",
	"Vasquez", "Simmons", "Romero", "Jordan", "Patterson", "Alexander", "Hamilton", "Graham",
	"Reynolds", "Griffin", "Wallace", "Moreno", "West", "Cole", "Hayes", "Bryant",
	"Herrera", "Gibson", "Ellis", "Tran", "Medina", "Aguilar", "Stevens", "Murray",
	"Ford", "Castro", "Marshall", "Owens", "Harrison", "Fernandez", "Mcdonald", "Woods",
	"Washington", "Kennedy", "Wells", "Vargas", "Henry", "Chen", "Freeman", "Webb",
	"Tucker", "Guzman", "Burns", "Crawford", "Olson", "Simpson", "Porter", "Hunter",
	"Gordon", "Mendez", "Silva", "Shaw", "Snyder", "Mason", "Dixon", "Munoz",
}

// ── Random pools for rich profiles ──────────────────────────────────────────

var specialties = []string{
	"CARDIOLOGY", "DERMATOLOGY", "ENT", "GASTROENTEROLOGY", "GYNECOLOGY",
	"NEUROLOGY", "ONCOLOGY", "ORTHOPEDICS", "PEDIATRICS", "PSYCHIATRY",
	"UROLOGY", "GENERAL", "OTHER",
}

var visitMethods = []string{"CHAT", "VOICE_CALL", "VIDEO_CALL", "ON_SITE"}

var visitTypes = []string{
	"CONSULTATION", "EXAMINATION", "SURGERY", "LABORATORY", "RADIOLOGY",
	"PHARMACY", "DENTISTRY", "THERAPY", "NUTRITION", "OTHER",
}

var nursePermissions = []string{
	"VIEW_PATIENTS", "VIEW_SOAPS", "CHAT_WITH_PATIENTS",
	"MANAGE_SCHEDULE", "MANAGE_APPOINTMENTS", "VIEW_CONSULTATION_NOTES",
}

var languages = []string{
	"English", "Spanish", "French", "German", "Arabic", "Mandarin",
	"Hindi", "Portuguese", "Persian", "Japanese", "Russian", "Italian",
}

var universities = []string{
	"Johns Hopkins University", "Harvard Medical School", "Stanford University",
	"University of Oxford", "University of Toronto", "Karolinska Institute",
	"University of Melbourne", "Tehran University of Medical Sciences",
	"Charité Berlin", "Kyoto University", "McGill University", "UCL Medical School",
}

var cities = []string{
	"New York, NY", "Los Angeles, CA", "Chicago, IL", "Houston, TX", "Phoenix, AZ",
	"Toronto, ON", "London, UK", "Berlin, DE", "Tokyo, JP", "Sydney, AU",
	"Dubai, AE", "Tehran, IR", "Paris, FR", "Madrid, ES", "Sao Paulo, BR",
}

var medicalHistoryPool = []string{
	"Hypertension", "Type 2 Diabetes", "Asthma", "High cholesterol", "Migraine",
	"Hypothyroidism", "GERD", "Anemia", "Osteoarthritis", "Depression", "Anxiety",
	"Chronic sinusitis", "Eczema", "Seasonal allergies", "Kidney stones",
}

var allergyPool = []string{
	"Penicillin", "Peanuts", "Shellfish", "Latex", "Aspirin", "Pollen",
	"Dust mites", "Bee stings", "Eggs", "Soy", "Sulfa drugs", "Cats",
}

var medicationPool = []string{
	"Lisinopril 10mg", "Metformin 500mg", "Atorvastatin 20mg", "Levothyroxine 50mcg",
	"Omeprazole 20mg", "Albuterol inhaler", "Amlodipine 5mg", "Sertraline 50mg",
	"Ibuprofen 400mg", "Vitamin D3 2000IU",
}

var surgeryPool = []string{
	"Appendectomy 2018", "Tonsillectomy 2005", "Knee arthroscopy 2020",
	"Cholecystectomy 2019", "Cataract surgery 2021", "C-section 2017",
	"Wisdom teeth removal 2015", "Hernia repair 2016",
}

var familyHistoryPool = []string{
	"Father - Heart disease", "Mother - Type 2 diabetes", "Grandfather - Stroke",
	"Mother - Breast cancer", "Father - Hypertension", "Sibling - Asthma",
	"Grandmother - Osteoporosis", "Father - High cholesterol",
}

var bioPhrasesPatient = []string{
	"Generally healthy, looking for routine check-ups.",
	"Managing a chronic condition and seeking ongoing care.",
	"New patient interested in preventive medicine.",
	"Prefers online consultations when possible.",
	"Active lifestyle, occasional sports injuries.",
}

var bioPhrasesDoctor = []string{
	"Board-certified physician passionate about patient-centered care.",
	"Over a decade of clinical experience with a focus on preventive health.",
	"Dedicated to evidence-based medicine and clear communication.",
	"Committed to accessible, compassionate healthcare for all.",
	"Experienced specialist offering both in-person and telehealth visits.",
}
