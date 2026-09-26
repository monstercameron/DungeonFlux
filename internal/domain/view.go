package domain

import (
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"time"
)

type TimerView struct {
	Name        string `json:"name"`
	RemainingMS int64  `json:"remaining_ms"`
	TotalMS     int64  `json:"total_ms"`
	Frozen      bool   `json:"frozen"`
}
type DiceView struct {
	State             string `json:"state"`
	D20, Modifier, DC int
	Outcome           string
	Kind              string
	VSLLabel          string `json:"vs_label,omitempty"`
	Crit              bool
	Damage            *DamageView
}
type DamageView struct {
	Dice         string
	Faces        []int
	Bonus, Total int
	Type         string
}
type MusicView struct {
	TrackID, URL                string
	LoopStartMS, LoopEndMS, BPM int
	Level, Duck                 float64
	Cue                         string
}
type SlotView struct {
	Name  string
	State string
	Asset AssetID
}
type SceneView struct {
	BackgroundURL       string
	Layers              []string
	Narration, Subtitle string
}

// LocalizedMessage carries a view string as a key plus arguments. Clients
// render it through the catalog for the view locale; Fallback covers keys
// the client's catalog does not know yet.
type LocalizedMessage struct {
	Key      string            `json:"key"`
	Args     map[string]string `json:"args,omitempty"`
	Count    int               `json:"count,omitempty"`
	Fallback string            `json:"fallback,omitempty"`
}

type SeatView struct {
	Seat         SeatID
	PlayerNumber int
	Connected    bool
	Locale       string
	Character    *Character
	Build        *BuildCard
	Moves        []MoveView
	TurnTimer    TimerView
	StatusText   string
	StatusMsg    LocalizedMessage
}
type MoveView struct {
	ID       vocab.MoveID
	Label    string
	Enabled  bool
	Reason   string
	Options  []OptionView
	Preview  *MovePreview
	TargetID EntityID
	Cell     Cell
}
type OptionView struct{ ID, Label string }
type MovePreview struct {
	Modifier, VS int
	PSuccess     float64
	Damage       DamageView
}
type BattlefieldView struct {
	Mode              string
	Visible           bool
	SceneURL, LiteURL string
	Transform         Transform
	Cameras           map[string]CameraDef
	Grid              Grid
	Flat              FlatBattlefield
	Camera            CameraView
	Tokens            []TokenView
	Highlights        []HighlightView
	TurnOrder         []TurnEntry
	Round             int
}
type CameraView struct {
	Preset       string
	FocusTokenID TokenID
	Seq          uint64
}
type TokenView struct {
	ID           TokenID
	Kind, Name   string
	Cell         Cell
	Path         []Cell
	Portrait     AssetID
	Clips        map[string]AssetID
	Anim, Status string
	HP, HPMax    int
	Active       bool
}
type HighlightView struct {
	Kind  string
	Cells []Cell
}
type TurnEntry struct {
	TokenID      TokenID
	Name         string
	Portrait     AssetID
	HP, HPMax    int
	Active, Done bool
}
type CombatView struct {
	Tokens     []TokenView
	Highlights []HighlightView
	TurnOrder  []TurnEntry
	Round      int
	Banner     string
	Contact    TimerView
	Cap        TimerView
}
type View struct {
	Version     uint64
	At          time.Duration
	Path        vocab.StateID
	Paused      bool
	Spotlight   SeatID
	RunMode     vocab.RunMode
	NextD20     int
	Locale      string
	Notice      LocalizedMessage
	Seats       []SeatView
	Scene       SceneView
	Dice        *DiceView
	Battlefield *BattlefieldView
	Combat      *CombatView
	Preload     []string
	Callout     string
	Music       MusicView
	Slots       []SlotView
}

func (v View) DeepCopy() View {
	out := v
	out.Seats = append([]SeatView(nil), v.Seats...)
	out.Preload = append([]string(nil), v.Preload...)
	out.Slots = append([]SlotView(nil), v.Slots...)
	if v.Battlefield != nil {
		b := *v.Battlefield
		b.Tokens = append([]TokenView(nil), v.Battlefield.Tokens...)
		b.Highlights = append([]HighlightView(nil), v.Battlefield.Highlights...)
		out.Battlefield = &b
	}
	if v.Combat != nil {
		c := *v.Combat
		c.Tokens = append([]TokenView(nil), v.Combat.Tokens...)
		c.Highlights = append([]HighlightView(nil), v.Combat.Highlights...)
		out.Combat = &c
	}
	if v.Dice != nil {
		d := *v.Dice
		if v.Dice.Damage != nil {
			x := *v.Dice.Damage
			x.Faces = append([]int(nil), x.Faces...)
			d.Damage = &x
		}
		out.Dice = &d
	}
	return out
}
