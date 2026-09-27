package api

import (
	"encoding/json"
	"maps"
	"math"
	"sort"
	"strconv"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// Project converts a domain view to the protobuf view for one client kind.
func Project(view domain.View, kind df.ClientKind, seat domain.SeatID) *df.ScreenState {
	out := &df.ScreenState{
		Version:       view.Version,
		Phase:         string(view.Path),
		SpotlightSeat: strconv.Itoa(int(view.Spotlight)),
		Paused:        view.Paused,
	}
	switch kind {
	case df.ClientKind_CLIENT_KIND_PHONE:
		out.View = &df.ScreenState_Phone{Phone: projectPhone(view, seat)}
	case df.ClientKind_CLIENT_KIND_HOST:
		out.View = &df.ScreenState_Host{Host: projectHost(view)}
	default:
		out.View = &df.ScreenState_Dm{Dm: projectDM(view)}
	}
	return out
}

// ProjectDM converts a domain view to the DM protobuf view.
func ProjectDM(view domain.View) *df.DMView { return projectDM(view) }

// LobbyProjection carries room metadata that is owned by the composition
// root and is not part of a game phase. It is optional so older callers can
// continue projecting a domain view while the shared View contract catches
// up with the room-level lobby state.
type LobbyProjection struct {
	RoomCode string
	JoinURL  string
	QRURL    string
}

const (
	lobbyRoomCodeArg = "__lobby_room_code"
	lobbyJoinURLArg  = "__lobby_join_url"
	lobbyQRURLArg    = "__lobby_qr_url"
)

// AttachLobbyMetadata carries composition-root lobby data through the current
// domain view until the shared View contract grows a room-level Lobby field.
// The reserved notice arguments are not rendered as user-facing text.
func AttachLobbyMetadata(view domain.View, lobby LobbyProjection) domain.View {
	args := make(map[string]string, len(view.Notice.Args)+3)
	maps.Copy(args, view.Notice.Args)
	args[lobbyRoomCodeArg] = lobby.RoomCode
	args[lobbyJoinURLArg] = lobby.JoinURL
	args[lobbyQRURLArg] = lobby.QRURL
	view.Notice.Args = args
	return view
}

// ProjectDMWithLobby converts a domain view and room metadata to a DM view.
func ProjectDMWithLobby(view domain.View, lobby LobbyProjection) *df.DMView {
	return projectDM(view, lobby)
}

// ProjectPhone converts a domain view to the phone protobuf view for seat.
func ProjectPhone(view domain.View, seat domain.SeatID) *df.PhoneView {
	return projectPhone(view, seat)
}

// ProjectHost converts a domain view to the host protobuf view.
func ProjectHost(view domain.View) *df.HostView { return projectHost(view) }

func projectDM(view domain.View, lobby ...LobbyProjection) *df.DMView {
	if len(lobby) == 0 {
		if attached, ok := attachedLobby(view); ok {
			lobby = []LobbyProjection{attached}
		}
	}
	out := &df.DMView{
		BackgroundUrl:   view.Scene.BackgroundURL,
		Layers:          projectLayers(view.Scene.Layers),
		Narration:       projectNarration(view.Scene),
		Subtitle:        &df.Subtitle{Text: view.Scene.Subtitle},
		Callout:         view.Callout,
		BuildCards:      projectBuildCards(view.Seats),
		CreationChoices: projectCreationChoices(view.Seats),
		Preload:         append([]string(nil), view.Preload...),
		Music:           projectMusic(view.Music),
		Seats:           projectLobbySeats(view.Seats),
		KillCam:         projectKillcam(view.KillCam),
	}
	if len(lobby) > 0 {
		out.Lobby = projectLobby(lobby[0])
	}
	if view.Dice != nil {
		out.Dice = projectDice(*view.Dice)
	}
	if timer, ok := spotlightTimer(view); ok {
		out.TurnTimer = projectTimer(timer)
	}
	if view.Battlefield != nil {
		out.Battlefield = projectBattlefield(*view.Battlefield)
		out.Tokens = projectTokensForSeats(view.Battlefield.Tokens, view.Seats)
		out.Highlights = projectHighlights(view.Battlefield.Highlights)
		out.TurnOrder = projectTurnOrderForSeats(view.Battlefield.TurnOrder, view.Seats)
		out.Round = int32(view.Battlefield.Round)
		out.CombatBanner = view.Battlefield.Mode
		out.ContactInMs = view.Battlefield.Contact.RemainingMS
		out.Shake = projectShake(view.Battlefield.Shake)
	}
	if view.Combat != nil {
		if len(view.Combat.Tokens) > 0 {
			out.Tokens = projectTokensForSeats(view.Combat.Tokens, view.Seats)
		}
		if len(view.Combat.Highlights) > 0 {
			out.Highlights = projectHighlights(view.Combat.Highlights)
		}
		if len(view.Combat.TurnOrder) > 0 {
			out.TurnOrder = projectTurnOrderForSeats(view.Combat.TurnOrder, view.Seats)
		}
		out.Round = int32(view.Combat.Round)
		out.CombatBanner = view.Combat.Banner
		out.ContactInMs = view.Combat.Contact.RemainingMS
		if view.Combat.Shake.Seq != 0 {
			out.Shake = projectShake(view.Combat.Shake)
		}
	}
	return out
}

func attachedLobby(view domain.View) (LobbyProjection, bool) {
	args := view.Notice.Args
	if args == nil {
		return LobbyProjection{}, false
	}
	lobby := LobbyProjection{RoomCode: args[lobbyRoomCodeArg], JoinURL: args[lobbyJoinURLArg], QRURL: args[lobbyQRURLArg]}
	return lobby, lobby.RoomCode != "" || lobby.JoinURL != "" || lobby.QRURL != ""
}

func projectLobbySeats(seats []domain.SeatView) []*df.LobbySeat {
	out := make([]*df.LobbySeat, 0, len(seats))
	for _, seat := range seats {
		name := strings.TrimSpace(seat.PlayerName)
		if seat.Build != nil {
			if heroName := strings.TrimSpace(seat.Build.Name); heroName != "" {
				name = heroName
			}
		}
		if name == "" && seat.Character != nil {
			name = strings.TrimSpace(seat.Character.Name)
		}
		out = append(out, &df.LobbySeat{
			SeatId:       strconv.Itoa(int(seat.Seat)),
			PlayerNumber: int32(seat.PlayerNumber),
			Name:         name,
			Joined:       seat.Connected,
			Locale:       seat.Locale,
			Ready:        seatReady(seat),
		})
	}
	return out
}

func projectLobby(lobby LobbyProjection) *df.Lobby {
	if lobby.RoomCode == "" && lobby.JoinURL == "" && lobby.QRURL == "" {
		return nil
	}
	return &df.Lobby{RoomCode: lobby.RoomCode, JoinUrl: lobby.JoinURL, QrUrl: lobby.QRURL}
}

func projectPhone(view domain.View, seat domain.SeatID) *df.PhoneView {
	out := &df.PhoneView{Ptt: &df.PTT{State: df.PTTState_PTT_STATE_IDLE}, Narration: projectNarration(view.Scene)}
	out.Seats = projectLobbySeats(view.Seats)
	out.SceneImageUrl = view.Scene.BackgroundURL
	for _, item := range view.Seats {
		if item.Seat != seat {
			continue
		}
		out.PlayerNumber = int32(item.PlayerNumber)
		out.PlayerName = seatDisplayName(item)
		out.Character = projectCharacter(item.Character)
		if out.Character != nil {
			out.Character.Build = projectCharacterBuild(item.Character, item.Build)
			out.Character.Locked = characterLocked(item.Moves)
		}
		out.Moves = projectMoves(item.Moves)
		out.TurnTimer = projectTimer(item.TurnTimer)
		out.StatusText = phoneStatus(view, item)
		break
	}
	if view.Combat != nil {
		out.Combat = projectPhoneCombat(*view.Combat, view.Seats)
		out.Combat = projectPhoneCombatMap(out.Combat, view, seat)
		out.TurnOrder = projectTurnOrderForSeats(view.Combat.TurnOrder, view.Seats)
	}
	if view.Dice != nil {
		// INT-009: the phone needs the same check/attack roll the TV shows
		// (modifier, DC, kept face, and outcome), not just the move preview.
		out.Dice = projectDice(*view.Dice)
	}
	return out
}

// seatDisplayName returns the name a player has chosen for their seat,
// falling back to the rolled character's name when Join set no name.
func seatDisplayName(seat domain.SeatView) string {
	if name := strings.TrimSpace(seat.PlayerName); name != "" {
		return name
	}
	if seat.Character != nil && strings.TrimSpace(seat.Character.Name) != "" {
		return seat.Character.Name
	}
	return ""
}

func projectNarration(scene domain.SceneView) *df.Narration {
	return &df.Narration{
		Speaker:   scene.NarrationSpeaker,
		TextSoFar: scene.Narration,
		Done:      scene.NarrationDone,
		LineId:    string(scene.NarrationLineID),
	}
}

func projectHost(view domain.View) *df.HostView {
	return &df.HostView{
		Dm:                   projectDM(view),
		RunMode:              string(view.RunMode),
		NextD20:              int32(view.NextD20),
		TurnTimersEnabled:    view.TurnTimersEnabled,
		SplatEnabled:         view.SplatEnabled,
		SplatAvailable:       view.SplatAvailable,
		CombatCapRemainingMs: timerRemaining(view.Combat),
		AssetSlots:           projectSlots(view.Slots),
	}
}

func projectCharacter(character *domain.Character) *df.Character {
	if character == nil {
		return nil
	}
	return &df.Character{
		Name:               character.Name,
		ClassName:          character.Class,
		PersuasionModifier: int32(character.PersuasionModifier),
		PortraitUrl:        heroPortrait(string(character.Portrait), character.Species, character.Gender),
		HookText:           character.Hook,
		Species:            character.Species,
		Gender:             character.Gender,
		Flavor:             &df.CharacterFlavor{Name: character.Name, Hook: character.Hook},
		Build: &df.CharacterBuild{
			Hp:    int32(character.HP),
			HpMax: int32(character.MaxHP),
			Ac:    int32(character.AC),
		},
	}
}

func projectCharacterBuild(character *domain.Character, card *domain.BuildCard) *df.CharacterBuild {
	build := &df.CharacterBuild{}
	if character != nil {
		build.Hp, build.HpMax, build.Ac = int32(character.HP), int32(character.MaxHP), int32(character.AC)
	}
	if card == nil || card.Stats == nil {
		return build
	}
	stats := card.Stats
	build.Abilities = make([]int32, len(stats.Abilities))
	for index, score := range stats.Abilities {
		build.Abilities[index] = int32(score)
	}
	build.SaveProfs = append([]string(nil), stats.SaveProficiencies...)
	build.SkillProfs = make(map[string]string, len(stats.SkillProficiencies))
	maps.Copy(build.SkillProfs, stats.SkillProficiencies)
	build.Hp, build.HpMax, build.Ac = int32(stats.HP), int32(stats.MaxHP), int32(stats.AC)
	build.AttackName, build.AttackDice, build.AttackDamageType, build.AttackBonus = stats.AttackName, stats.AttackDice, stats.AttackDamageType, int32(stats.AttackBonus)
	build.Equipment = projectEquipment(stats.Equipment)
	return build
}

func projectEquipment(items []domain.EquipmentItem) []*df.EquipmentItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]*df.EquipmentItem, 0, len(items))
	for _, item := range items {
		out = append(out, &df.EquipmentItem{Name: item.Name, Description: item.Description, Slot: item.Slot, Worn: item.Worn})
	}
	return out
}

func characterLocked(moves []domain.MoveView) bool {
	for _, move := range moves {
		if move.ID == vocab.MoveReady {
			return !move.Enabled && move.Reason == "Your hero is already ready"
		}
	}
	return false
}

func phoneStatus(view domain.View, seat domain.SeatView) string {
	if seat.StatusText != "" {
		return seat.StatusText
	}
	if seat.Character != nil && characterLocked(seat.Moves) {
		return "Your hero is ready"
	}
	switch view.Path {
	case vocab.StateLobby:
		return "Waiting for the host"
	case vocab.StateCreation:
		if seat.Character == nil {
			return "Choose a species, gender, and class"
		}
		return "Your hero is ready to lock in"
	case vocab.StateOpening:
		return "The story is beginning"
	case vocab.StateExploration:
		return "Choose your next move"
	case vocab.StateConversation:
		return "Talk to Mother Vell"
	case vocab.StateCheck:
		return "The engine is rolling your Persuasion"
	case vocab.StateResolution:
		return "Mother Vell is deciding"
	case vocab.StateHookEvent:
		return "A stranger arrives"
	case vocab.StateCombat:
		return "Choose your combat move"
	case vocab.StateCliffhanger:
		return "The bell remembers"
	case vocab.StateEnd:
		return "The night ends here"
	default:
		return ""
	}
}

func projectBuildCards(seats []domain.SeatView) []*df.BuildCard {
	out := make([]*df.BuildCard, 0, len(seats))
	for _, seat := range seats {
		if seat.Build == nil {
			continue
		}
		species, gender := "", ""
		character := projectCharacter(seat.Character)
		if seat.Character != nil {
			species, gender = seat.Character.Species, seat.Character.Gender
			character.Build = projectCharacterBuild(seat.Character, seat.Build)
			character.Locked = seatReady(seat)
		}
		out = append(out, &df.BuildCard{PlayerNumber: int32(seat.Build.PlayerNumber), Name: seat.Build.Name,
			ClassName: seat.Build.Class, PortraitUrl: heroPortrait(string(seat.Build.Portrait), species, gender), Character: character, Ready: seatReady(seat)})
	}
	return out
}

// projectCreationChoices preserves both players' live picks, including a
// partially selected seat that has no BuildCard yet.
func projectCreationChoices(seats []domain.SeatView) []*df.CreationChoice {
	out := make([]*df.CreationChoice, 0, len(seats))
	for _, seat := range seats {
		choice := seat.Creation
		if choice == nil {
			choice = &domain.CreationChoiceView{PlayerNumber: seat.PlayerNumber}
		}
		out = append(out, &df.CreationChoice{
			PlayerNumber: int32(choice.PlayerNumber), PlayerName: choice.PlayerName,
			Name: choice.Name, Species: choice.Species, Gender: choice.Gender,
			ClassName: choice.Class, PortraitUrl: heroPortrait(string(choice.Portrait), choice.Species, choice.Gender),
			Rolled: choice.Rolled, Ready: choice.Ready,
		})
	}
	return out
}

func projectMoves(moves []domain.MoveView) []*df.Move {
	out := make([]*df.Move, 0, len(moves))
	for _, move := range moves {
		item := &df.Move{MoveId: string(move.ID), Label: move.Label, Enabled: move.Enabled, Reason: move.Reason,
			TargetId: string(move.TargetID), Cell: projectCell(move.Cell), Options: projectOptions(move.Options)}
		if move.Preview != nil {
			item.Preview = &df.MovePreview{Modifier: int32(move.Preview.Modifier), Vs: int32(move.Preview.VS), PSuccess: move.Preview.PSuccess,
				Damage: &df.DamagePreview{Dice: move.Preview.Damage.Dice, Bonus: int32(move.Preview.Damage.Bonus)}}
		}
		out = append(out, item)
	}
	return out
}

func projectOptions(options []domain.OptionView) []*df.Option {
	out := make([]*df.Option, 0, len(options))
	for _, option := range options {
		out = append(out, &df.Option{Id: option.ID, Label: option.Label})
	}
	return out
}

func projectTimer(timer domain.TimerView) *df.Timer {
	return &df.Timer{RemainingMs: timer.RemainingMS, TotalMs: timer.TotalMS, Frozen: timer.Frozen}
}

func spotlightTimer(view domain.View) (domain.TimerView, bool) {
	for _, seat := range view.Seats {
		if seat.Seat == view.Spotlight {
			return seat.TurnTimer, true
		}
	}
	return domain.TimerView{}, false
}

func projectDice(dice domain.DiceView) *df.Dice {
	out := &df.Dice{State: diceState(dice.State), D20: int32(dice.D20), Modifier: int32(dice.Modifier), Dc: int32(dice.DC),
		Outcome: dice.Outcome, Kind: diceKind(dice.Kind), VsLabel: dice.VSLLabel, Crit: dice.Crit}
	if dice.Damage != nil {
		out.Damage = &df.Damage{Dice: dice.Damage.Dice, Faces: ints32(dice.Damage.Faces), Bonus: int32(dice.Damage.Bonus), Total: int32(dice.Damage.Total), Type: dice.Damage.Type}
	}
	return out
}

func diceState(state string) df.DiceState {
	switch state {
	case "offered":
		return df.DiceState_DICE_STATE_OFFERED
	case "rolling":
		return df.DiceState_DICE_STATE_ROLLING
	case "resolved":
		return df.DiceState_DICE_STATE_RESOLVED
	default:
		return df.DiceState_DICE_STATE_UNSPECIFIED
	}
}

func diceKind(kind string) df.DiceKind {
	switch kind {
	case "check":
		return df.DiceKind_DICE_KIND_CHECK
	case "attack":
		return df.DiceKind_DICE_KIND_ATTACK
	default:
		return df.DiceKind_DICE_KIND_UNSPECIFIED
	}
}

func projectMusic(music domain.MusicView) *df.Music {
	if music.TrackID == "" && music.URL == "" && music.Cue == "" {
		return nil
	}
	return &df.Music{TrackId: music.TrackID, Url: music.URL, LoopStartMs: int64(music.LoopStartMS), LoopEndMs: int64(music.LoopEndMS),
		Bpm: int32(music.BPM), Level: float32(music.Level), Duck: float32(music.Duck), Cue: music.Cue}
}

func projectBattlefield(field domain.BattlefieldView) *df.Battlefield {
	out := &df.Battlefield{Mode: field.Mode, Visible: field.Visible, SceneUrl: field.SceneURL, LiteUrl: field.LiteURL,
		Transform: jsonString(field.Transform), Grid: projectGrid(field.Grid), Flat: projectFlat(field.Flat), Camera: projectCamera(field.Camera),
		ContactInMs: field.Contact.RemainingMS, Shake: projectShake(field.Shake)}
	keys := make([]string, 0, len(field.Cameras))
	for key := range field.Cameras {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out.Cameras = append(out.Cameras, &df.Camera{Preset: key})
	}
	return out
}

func projectGrid(grid domain.Grid) *df.Grid {
	out := &df.Grid{Origin: projectCell(domain.Cell{C: int(math.Round(grid.Origin[0])), R: int(math.Round(grid.Origin[1]))}), CellM: float32(grid.CellM), Cols: int32(grid.Cols), Rows: int32(grid.Rows)}
	for index, walkable := range grid.Walkable {
		if walkable && grid.Cols > 0 {
			out.Walkable = append(out.Walkable, projectCell(domain.Cell{C: index % grid.Cols, R: index / grid.Cols}))
		}
	}
	return out
}

func projectFlat(flat domain.FlatBattlefield) *df.FlatBattlefield {
	values := make([]float32, 0, 8)
	for _, point := range flat.FloorQuadPX {
		values = append(values, float32(point[0]), float32(point[1]))
	}
	return &df.FlatBattlefield{ImageUrl: flat.ImageURL, FloorQuadPx: values}
}

func projectCamera(camera domain.CameraView) *df.Camera {
	return &df.Camera{Preset: camera.Preset, FocusTokenId: string(camera.FocusTokenID), Seq: camera.Seq, Follow: camera.Follow, DurationMs: camera.DurationMS}
}

func projectShake(shake domain.ShakeView) *df.Shake {
	if shake.Seq == 0 && shake.AmplitudePX == 0 && shake.DurationMS == 0 {
		return nil
	}
	return &df.Shake{AmplitudePx: float32(shake.AmplitudePX), DurationMs: shake.DurationMS, Seq: shake.Seq}
}

func projectTokens(tokens []domain.TokenView) []*df.Token {
	out := make([]*df.Token, 0, len(tokens))
	for _, token := range tokens {
		statuses := []string(nil)
		if token.Status != "" {
			statuses = []string{token.Status}
		}
		out = append(out, &df.Token{TokenId: string(token.ID), Name: token.Name, PortraitUrl: string(token.Portrait), Cell: projectCell(token.Cell), Hp: int32(token.HP), HpMax: int32(token.HPMax), Active: token.Active, Statuses: statuses,
			Kind: token.Kind, Path: projectCells(token.Path), Anim: token.Anim, AnimSeq: token.AnimSeq, Clips: projectClips(token.Clips), StepMs: int32(token.StepMS)})
	}
	return out
}

// projectTokensForSeats preserves the same portrait selector used by the
// party cards when combat tokens are rendered. Combat token views carry only
// the generic species selector, while the seat character also carries the
// chosen gender and any generated portrait asset.
func projectTokensForSeats(tokens []domain.TokenView, seats []domain.SeatView) []*df.Token {
	out := projectTokens(tokens)
	for index := range out {
		seat, ok := combatSeatForToken(tokens[index].ID, seats)
		if !ok || seat.Character == nil {
			continue
		}
		out[index].PortraitUrl = heroPortrait(string(seat.Character.Portrait), seat.Character.Species, seat.Character.Gender)
	}
	return out
}

func combatSeatForToken(id domain.TokenID, seats []domain.SeatView) (domain.SeatView, bool) {
	for _, seat := range seats {
		if "pc-"+strconv.Itoa(int(seat.Seat)) == string(id) {
			return seat, true
		}
	}
	return domain.SeatView{}, false
}

func projectTurnOrderForSeats(entries []domain.TurnEntry, seats []domain.SeatView) []*df.TurnOrderEntry {
	out := projectTurnOrder(entries)
	for index := range out {
		seat, ok := combatSeatForToken(entries[index].TokenID, seats)
		if !ok || seat.Character == nil {
			continue
		}
		out[index].PortraitUrl = heroPortrait(string(seat.Character.Portrait), seat.Character.Species, seat.Character.Gender)
	}
	return out
}

func projectHighlights(highlights []domain.HighlightView) []*df.Highlight {
	var out []*df.Highlight
	for _, highlight := range highlights {
		cells := projectCells(highlight.Cells)
		item := &df.Highlight{Kind: highlight.Kind, Cells: cells}
		if len(cells) > 0 {
			item.Cell = cells[0]
		}
		out = append(out, item)
	}
	return out
}

func projectCells(cells []domain.Cell) []*df.Cell {
	if len(cells) == 0 {
		return nil
	}
	out := make([]*df.Cell, len(cells))
	for index, cell := range cells {
		out[index] = projectCell(cell)
	}
	return out
}

func projectClips(clips map[string]domain.AssetID) map[string]string {
	if len(clips) == 0 {
		return nil
	}
	out := make(map[string]string, len(clips))
	for name, asset := range clips {
		out[name] = string(asset)
	}
	return out
}

func projectTurnOrder(entries []domain.TurnEntry) []*df.TurnOrderEntry {
	out := make([]*df.TurnOrderEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, &df.TurnOrderEntry{TokenId: string(entry.TokenID), Name: entry.Name, PortraitUrl: string(entry.Portrait), Hp: int32(entry.HP), HpMax: int32(entry.HPMax), Active: entry.Active, Done: entry.Done})
	}
	return out
}

func projectPhoneCombat(combat domain.CombatView, seats []domain.SeatView) *df.CombatView {
	out := &df.CombatView{ContactInMs: combat.Contact.RemainingMS}
	if len(combat.Tokens) > 0 {
		token := projectTokensForSeats(combat.Tokens, seats)[0]
		out.TokenId, out.Hp, out.HpMax, out.Statuses, out.MyTurn = token.TokenId, token.Hp, token.HpMax, token.Statuses, token.Active
	}
	return out
}

func projectCell(cell domain.Cell) *df.Cell { return &df.Cell{C: int32(cell.C), R: int32(cell.R)} }

func projectLayers(layers []string) []*df.Layer {
	out := make([]*df.Layer, 0, len(layers))
	for index, layer := range layers {
		out = append(out, &df.Layer{Id: strconv.Itoa(index), Url: layer})
	}
	return out
}

func projectSlots(slots []domain.SlotView) []*df.AssetSlot {
	out := make([]*df.AssetSlot, 0, len(slots))
	for _, slot := range slots {
		out = append(out, &df.AssetSlot{Name: slot.Name, State: slot.State})
	}
	return out
}

func timerRemaining(combat *domain.CombatView) int64 {
	if combat == nil {
		return 0
	}
	return combat.Cap.RemainingMS
}

func jsonString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func ints32(values []int) []int32 {
	out := make([]int32, len(values))
	for index, value := range values {
		out[index] = int32(value)
	}
	return out
}

// heroPortrait is the portrait clients show for a hero: the generated one when
// it exists, otherwise a stand-in for the rolled species and gender (OPS-028
// registers ui/species_<species>_<gender> for every species x gender the
// creation phase offers). The server picks it so the TV and the player's
// phone always show the same stand-in, and so a paladin and a rogue of
// different genders no longer collapse onto one shared male portrait.
func heroPortrait(portrait, species, gender string) string {
	species = strings.ToLower(strings.TrimSpace(species))
	if strings.TrimSpace(portrait) != "" && portrait != "ui/species_"+species {
		return portrait
	}
	if species == "" {
		return ""
	}
	if gender = strings.ToLower(strings.TrimSpace(gender)); gender != "" {
		return "ui/species_" + species + "_" + gender
	}
	return "ui/species_" + species
}
