package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxPlayerSeats = 2

// SessionServer handles room joins and keeps the API's stable seat tokens.
// Seat identity belongs to the room and therefore survives a game reset.
type SessionServer struct {
	df.UnimplementedSessionServiceServer
	inbox      ports.Inbox
	roomCode   string
	hostToken  string
	dmToken    string
	engine     ports.Engine
	mu         sync.Mutex
	seats      map[string]seatSession
	nextSeat   int
	roomLocale string
	locales    map[string]string
}

type seatSession struct {
	id           domain.SeatID
	playerNumber int
	token        string
	locale       string
	name         string
}

// NewSessionServer creates a session service for one configured room.
func NewSessionServer(inbox ports.Inbox, roomCode, hostToken, dmToken string) (*SessionServer, error) {
	if inbox == nil {
		return nil, errors.New("api: session inbox is required")
	}
	if roomCode == "" {
		return nil, errors.New("api: session room code is required")
	}
	return &SessionServer{inbox: inbox, roomCode: roomCode, hostToken: hostToken,
		dmToken: dmToken, seats: make(map[string]seatSession), locales: make(map[string]string)}, nil
}

// SettleLocale resolves a client's requested locale against the room default
// and records it under key (a seat token, "dm", or "host").
func (s *SessionServer) SettleLocale(key, requested string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locales == nil {
		s.locales = make(map[string]string)
	}
	tag := settleAPILocale(requested, s.roomLocale)
	s.locales[key] = tag
	return tag
}

// LocaleFor returns the settled locale for a key or the room default.
func (s *SessionServer) LocaleFor(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tag, ok := s.locales[key]; ok && tag != "" {
		return tag
	}
	if s.roomLocale != "" {
		return s.roomLocale
	}
	return "en"
}

// SetRoomLocale updates the room default locale for new joins.
func (s *SessionServer) SetRoomLocale(locale string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roomLocale = settleAPILocale(locale, s.roomLocale)
	if s.roomLocale == "" {
		s.roomLocale = "en"
	}
	return s.roomLocale
}

func settleAPILocale(requested, fallback string) string {
	norm := func(tag string) string {
		out := make([]byte, 0, len(tag))
		for i := 0; i < len(tag); i++ {
			c := tag[i]
			if c == '-' || c == '_' || c == ' ' {
				break
			}
			if c >= 'A' && c <= 'Z' {
				c += 'a' - 'A'
			}
			out = append(out, c)
		}
		return string(out)
	}
	if norm(requested) == "en" || norm(requested) == "es" {
		return norm(requested)
	}
	if norm(fallback) == "en" || norm(fallback) == "es" {
		return norm(fallback)
	}
	if norm(fallback) != "" {
		return norm(fallback)
	}
	return "en"
}

// Join authenticates a client and allocates or restores a phone seat.
func (s *SessionServer) Join(ctx context.Context, request *df.JoinRequest) (*df.JoinResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "join request is required")
	}
	if request.GetRoomCode() != s.roomCode {
		return nil, status.Error(codes.PermissionDenied, "room code is invalid")
	}
	kind := request.GetKind()
	if kind == df.ClientKind_CLIENT_KIND_UNSPECIFIED {
		kind = s.kindForToken(request.GetSeatToken())
	}
	switch kind {
	case df.ClientKind_CLIENT_KIND_PHONE:
		return s.joinPhone(ctx, request)
	case df.ClientKind_CLIENT_KIND_DM:
		if request.GetDmToken() == "" || request.GetDmToken() != s.dmToken {
			return nil, status.Error(codes.PermissionDenied, "dm token is invalid")
		}
		return &df.JoinResponse{Locale: s.SettleLocale("dm", request.GetLocale())}, nil
	case df.ClientKind_CLIENT_KIND_HOST:
		if request.GetHostToken() == "" || request.GetHostToken() != s.hostToken {
			return nil, status.Error(codes.PermissionDenied, "host token is invalid")
		}
		return &df.JoinResponse{Locale: s.SettleLocale("host", request.GetLocale())}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "client kind is required")
	}
}

func (s *SessionServer) kindForToken(token string) df.ClientKind {
	if token == "" {
		return df.ClientKind_CLIENT_KIND_UNSPECIFIED
	}
	if _, ok := s.seatForToken(token); ok {
		return df.ClientKind_CLIENT_KIND_PHONE
	}
	if token == s.dmToken {
		return df.ClientKind_CLIENT_KIND_DM
	}
	if token == s.hostToken {
		return df.ClientKind_CLIENT_KIND_HOST
	}
	return df.ClientKind_CLIENT_KIND_PHONE
}

func (s *SessionServer) joinPhone(ctx context.Context, request *df.JoinRequest) (*df.JoinResponse, error) {
	s.mu.Lock()
	if request.GetSeatToken() != "" {
		if seat, ok := s.seats[request.GetSeatToken()]; ok {
			if name := normalizePlayerName(request.GetPlayerName()); name != "" {
				seat.name = name
				s.seats[request.GetSeatToken()] = seat
			}
			locale := seat.locale
			if locale == "" {
				locale = s.roomLocale
			}
			if locale == "" {
				locale = "en"
			}
			s.mu.Unlock()
			if !s.postPhoneJoin(ctx, seat.id, locale, seat.name) {
				return nil, status.Error(codes.ResourceExhausted, "room inbox is full")
			}
			response := joinResponse(seat)
			response.Locale = locale
			return response, nil
		}
		s.mu.Unlock()
		return nil, status.Error(codes.PermissionDenied, "seat token is invalid")
	}
	if s.nextSeat >= maxPlayerSeats {
		s.mu.Unlock()
		return nil, status.Error(codes.ResourceExhausted, "all player seats are occupied")
	}
	id := domain.SeatID(s.nextSeat + 1)
	s.nextSeat++
	token, err := newSeatToken()
	if err != nil {
		s.mu.Unlock()
		return nil, status.Errorf(codes.Internal, "create seat token: %v", err)
	}
	locale := settleAPILocale(request.GetLocale(), s.roomLocale)
	if locale == "" {
		locale = "en"
	}
	seat := seatSession{id: id, playerNumber: s.nextSeat, token: token, locale: locale, name: normalizePlayerName(request.GetPlayerName())}
	s.seats[token] = seat
	if s.locales == nil {
		s.locales = make(map[string]string)
	}
	s.locales[token] = locale
	s.mu.Unlock()
	if !s.postPhoneJoin(ctx, id, locale, seat.name) {
		s.mu.Lock()
		delete(s.seats, token)
		delete(s.locales, token)
		s.mu.Unlock()
		return nil, status.Error(codes.ResourceExhausted, "room inbox is full")
	}
	response := joinResponse(seat)
	response.Locale = locale
	return response, nil
}

func (s *SessionServer) postPhoneJoin(ctx context.Context, seat domain.SeatID, locale, name string) bool {
	event := domain.Join{Seat: seat, JoinKind: "phone", Locale: locale}
	setJoinName(&event, name)
	return s.inbox.Post(ctx, domain.Envelope{Event: event})
}

// setJoinName bridges the ENG-018 Name field while the shared domain contract
// remains source-compatible with older generated workers. Once Name is in the
// contract this simply sets that exported string field; older contracts ignore
// it and keep the rest of the join event unchanged.
func setJoinName(join *domain.Join, name string) {
	if join == nil || name == "" {
		return
	}
	field := reflect.ValueOf(join).Elem().FieldByName("Name")
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
		field.SetString(name)
	}
}

func normalizePlayerName(name string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
}

func joinResponse(seat seatSession) *df.JoinResponse {
	return &df.JoinResponse{SeatId: fmt.Sprint(seat.id), SeatToken: seat.token, PlayerNumber: int32(seat.playerNumber), Locale: seat.locale}
}

func newSeatToken() (string, error) {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func (s *SessionServer) seatForToken(token string) (seatSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seat, ok := s.seats[token]
	return seat, ok
}
