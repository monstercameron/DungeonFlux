package domain

type Run struct {
	ID   RunID  `json:"id"`
	Room RoomID `json:"room"`
	Seed []byte `json:"seed"`
	Mode string `json:"mode"`
}

type Character struct {
	ID                 EntityID `json:"id"`
	Name               string   `json:"name"`
	Class              string   `json:"class"`
	Species            string   `json:"species"`
	Gender             string   `json:"gender"`
	Hook               string   `json:"hook,omitempty"`
	Portrait           AssetID  `json:"portrait,omitempty"`
	PersuasionModifier int      `json:"persuasion_modifier"`
	HP                 int      `json:"hp"`
	MaxHP              int      `json:"max_hp"`
	AC                 int      `json:"ac"`
}

type BuildCard struct {
	Name         string  `json:"name"`
	Class        string  `json:"class"`
	Portrait     AssetID `json:"portrait,omitempty"`
	PlayerNumber int     `json:"player_number"`
}

type Seat struct {
	ID           SeatID     `json:"id"`
	PlayerNumber int        `json:"player_number"`
	Token        string     `json:"token,omitempty"`
	Character    *Character `json:"character,omitempty"`
	Connected    bool       `json:"connected"`
}

type NPC struct {
	ID          EntityID `json:"id"`
	Name        string   `json:"name"`
	Personality string   `json:"personality,omitempty"`
	VoiceID     string   `json:"voice_id,omitempty"`
}

type Creature struct {
	ID       EntityID `json:"id"`
	Name     string   `json:"name"`
	HP       int      `json:"hp"`
	MaxHP    int      `json:"max_hp"`
	AC       int      `json:"ac"`
	Cell     Cell     `json:"cell"`
	Statuses []string `json:"statuses,omitempty"`
}

type Asset struct {
	ID         AssetID           `json:"id"`
	SHA256     string            `json:"sha256"`
	Kind       string            `json:"kind"`
	MIME       string            `json:"mime"`
	URL        string            `json:"url,omitempty"`
	DurationMS int               `json:"duration_ms,omitempty"`
	Meta       map[string]string `json:"meta,omitempty"`
}

type Recording struct {
	ID    RecordingID `json:"id"`
	Text  string      `json:"text,omitempty"`
	Audio []byte      `json:"audio,omitempty"`
	MIME  string      `json:"mime,omitempty"`
}

type Ack struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
}

type Flavor struct {
	Name string `json:"name"`
	Look string `json:"look"`
	Hook string `json:"hook"`
}
