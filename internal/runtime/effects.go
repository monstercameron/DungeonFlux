package runtime

import (
	"reflect"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// runScopedEffects dispatches work under the scope context selected by the
// room loop. Control effects have already been applied synchronously.
func runScopedEffects(runner *Runner, effects []domain.Effect, scope domain.Scope, scopes *ScopeTree) {
	for _, effect := range effects {
		runScopedEffect(runner, effect, scope, scopes)
	}
}

func runScopedEffect(runner *Runner, effect domain.Effect, scope domain.Scope, scopes *ScopeTree) {
	if isControl(effect) {
		return
	}
	fn, ok := runner.registry[reflect.TypeOf(effect)]
	if !ok {
		runner.unregistered(effect)
		return
	}
	ctx := scopes.Context(scope, parentScope(scope))
	go runRecovered(ctx, runner.logger, effect, scope, runner.in, fn)
}

func parentScope(scope domain.Scope) domain.Scope {
	if scope.Key == "" {
		return domain.Scope{}
	}
	return domain.Scope{Machine: scope.Machine, Epoch: scope.Epoch}
}
