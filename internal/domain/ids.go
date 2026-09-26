package domain

type RunID string
type RoomID string
type SeatID int
type EntityID string
type AssetID string
type UtteranceID string
type RecordingID string
type MoveID string
type TokenID string
type TraceID string

type Cell struct {
	C int `json:"c"`
	R int `json:"r"`
}

type DiceRoll struct {
	Die    int `json:"die"`
	Result int `json:"result"`
}

type RollRecord struct {
	Die      string `json:"die"`
	Faces    []int  `json:"faces,omitempty"`
	Total    int    `json:"total"`
	Modifier int    `json:"modifier"`
}

type DamageRoll struct {
	Dice   string `json:"dice"`
	Faces  []int  `json:"faces,omitempty"`
	Bonus  int    `json:"bonus"`
	Total  int    `json:"total"`
	Type   string `json:"type"`
	Source string `json:"source"`
}

type Term struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}
