package content

import "hash/fnv"

// heroNamesBySpeciesGender is a curated table of dark-fantasy, river-town
// appropriate hero names, one list per (species, gender) pair. It backs the
// deterministic fallback used when character_flavor is unavailable (fake and
// safe mode, and on flavor_failed), so the demo never falls back to a bare
// seat label like "Hero 1".
var heroNamesBySpeciesGender = map[string]map[string][]string{
	"human": {
		"female":    {"Isolde", "Mirena", "Agnes", "Rosalind", "Wrenna", "Sabine"},
		"male":      {"Corvin", "Aldric", "Bram", "Osric", "Tobin", "Merrick"},
		"nonbinary": {"Sael", "Rowan", "Doran", "Larke", "Wynn", "Idris"},
	},
	"elf": {
		"female":    {"Amareth", "Sylvenna", "Ithlien", "Quorra", "Faelyn", "Nimrael"},
		"male":      {"Erevan", "Thalorin", "Silvarion", "Caladrel", "Aerendyl", "Vaelith"},
		"nonbinary": {"Elowyn", "Thessaly", "Aurelune", "Cindris", "Faervel", "Ondrelle"},
	},
	"dwarf": {
		"female":    {"Brynja", "Torvild", "Helka", "Dagny", "Osryn", "Kilda"},
		"male":      {"Borin", "Thrandor", "Gralkin", "Durgan", "Halvard", "Ormund"},
		"nonbinary": {"Skara", "Brogun", "Nordri", "Fyrka", "Vethrik", "Ossandr"},
	},
	"halfling": {
		"female":    {"Marigold", "Pipsy", "Willabeth", "Rosamund", "Tansy", "Nettle"},
		"male":      {"Fennick", "Todbury", "Wilmer", "Robin", "Doby", "Alric"},
		"nonbinary": {"Peregrin", "Sorrel", "Bramble", "Juniper", "Sparrow", "Cricket"},
	},
	"orc": {
		"female":    {"Ghurka", "Vokka", "Drava", "Uzruk", "Mokara", "Skarra"},
		"male":      {"Groth", "Uzgash", "Draknor", "Morkuz", "Vaskul", "Throgar"},
		"nonbinary": {"Ruzka", "Vothok", "Gharn", "Zulmar", "Krosk", "Hurrak"},
	},
	"tiefling": {
		"female":    {"Zerith", "Morrigrave", "Ashael", "Vexanya", "Kaldris", "Nyxara"},
		"male":      {"Kairon", "Malvorne", "Ashvael", "Vorash", "Dravik", "Zenros"},
		"nonbinary": {"Cinderel", "Sableth", "Nocturis", "Ravelock", "Emberith", "Duskaerin"},
	},
	"dragonborn": {
		"female":    {"Thessaleth", "Ravaneth", "Korrathi", "Vyrandra", "Sathira", "Nyrassa"},
		"male":      {"Balasar", "Kriv", "Torvash", "Drennok", "Arjhan", "Vrondir"},
		"nonbinary": {"Ondrik", "Sethrek", "Kaldrun", "Vaelros", "Threska", "Morvath"},
	},
	"gnome": {
		"female":    {"Ottilie", "Nissa", "Bramblewick", "Fennela", "Corrin", "Tazzle"},
		"male":      {"Fizwick", "Boddyn", "Nackle", "Perrin", "Wobbin", "Ambrose"},
		"nonbinary": {"Tinker", "Glimmer", "Pockets", "Whistle", "Cogsworth", "Bristle"},
	},
	"goliath": {
		"female":    {"Kavaha", "Thurra", "Vondeka", "Ranneka", "Isseka", "Morava"},
		"male":      {"Vondar", "Threska", "Kalmuk", "Orgeth", "Bavrok", "Denneth"},
		"nonbinary": {"Ashka", "Torveq", "Nallek", "Ombrek", "Serrak", "Uvanek"},
	},
}

// heroNameFallback is used when the species or gender is unknown, missing,
// or not in the curated table.
var heroNameFallback = []string{
	"Wren", "Corvin", "Sael", "Ashka", "Marrow", "Thistle", "Brael", "Ondine",
}

// FallbackHeroName deterministically picks a curated hero name for the given
// species and gender, using seed to select among the curated list. The same
// species, gender, and seed always yield the same name, so fake mode, safe
// mode, and a flavor_failed fallback never show a bare seat label.
func FallbackHeroName(species, gender string, seed []byte) string {
	list := heroNameList(species, gender)
	if len(list) == 0 {
		list = heroNameFallback
	}
	return list[seedIndex(seed, len(list))]
}

func heroNameList(species, gender string) []string {
	genders, ok := heroNamesBySpeciesGender[species]
	if !ok {
		return nil
	}
	return genders[gender]
}

// seedIndex derives a stable index in [0, n) from seed bytes using FNV-1a.
// It never uses a randomness source, so results are reproducible from the
// same input, which archtest's purity rule for content requires.
func seedIndex(seed []byte, n int) int {
	if n <= 0 {
		return 0
	}
	hash := fnv.New32a()
	_, _ = hash.Write(seed)
	return int(hash.Sum32() % uint32(n))
}
