package runtime

import "github.com/monstercameron/DungeonFlux/internal/domain"

func parentScope(scope domain.Scope) domain.Scope {
	if scope.Key == "" {
		return domain.Scope{}
	}
	return domain.Scope{Machine: scope.Machine, Epoch: scope.Epoch}
}
