package domain

type Beat struct {
	ID    string   `json:"id"`
	Text  string   `json:"text"`
	Leads []string `json:"leads,omitempty"`
}
type OneShot struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	NPCs      []NPC          `json:"npcs"`
	Beats     []Beat         `json:"beats"`
	Encounter Encounter      `json:"encounter"`
	Music     MusicCatalogue `json:"music"`
}
