package casbin

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	casbinlib "github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/iam/domain"
)

const authorizationModel = `[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && r.dom == p.dom && resourceMatch(p.obj, r.obj) && r.act == p.act
`

type policySource interface {
	LoadCompiledPolicies(context.Context) ([]domain.CompiledPolicy, map[uuid.UUID]int64, error)
}

type snapshot struct {
	enforcer *casbinlib.SyncedEnforcer
	versions map[uuid.UUID]int64
}

type Evaluator struct {
	source policySource
	value  atomic.Pointer[snapshot]
	reload sync.Mutex
}

var _ domain.Evaluator = (*Evaluator)(nil)

func NewEvaluator(source domain.PolicyStore) *Evaluator {
	return newEvaluator(source)
}

func newEvaluator(source policySource) *Evaluator {
	return &Evaluator{source: source}
}

func (e *Evaluator) Reload(ctx context.Context) error {
	e.reload.Lock()
	defer e.reload.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	policies, versions, err := e.source.LoadCompiledPolicies(ctx)
	if err != nil {
		return fmt.Errorf("load compiled IAM policies: %w", err)
	}
	enforcer, err := newEnforcer(policies)
	if err != nil {
		return err
	}
	e.value.Store(&snapshot{enforcer: enforcer, versions: cloneVersions(versions)})
	return nil
}

func (e *Evaluator) Enforce(
	ctx context.Context,
	principal domain.Principal,
	organizationID uuid.UUID,
	resourceName string,
	permission domain.PermissionName,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	current := e.value.Load()
	if current == nil {
		return false, fmt.Errorf("IAM evaluator is not initialized")
	}

	allowed, err := current.enforcer.Enforce(
		principal.Key(),
		organizationID.String(),
		resourceName,
		string(permission),
	)
	if err != nil {
		return false, fmt.Errorf("casbin enforce: %w", err)
	}

	return allowed, nil
}

func (e *Evaluator) Versions() map[uuid.UUID]int64 {
	current := e.value.Load()
	if current == nil {
		return map[uuid.UUID]int64{}
	}

	return cloneVersions(current.versions)
}

func newEnforcer(policies []domain.CompiledPolicy) (*casbinlib.SyncedEnforcer, error) {
	m, err := model.NewModelFromString(authorizationModel)
	if err != nil {
		return nil, fmt.Errorf("parse Casbin model: %w", err)
	}

	enforcer, err := casbinlib.NewSyncedEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("create Casbin enforcer: %w", err)
	}

	enforcer.AddFunction("resourceMatch", func(arguments ...interface{}) (interface{}, error) {
		if len(arguments) != 2 {
			return false, fmt.Errorf("resourceMatch expects two arguments")
		}
		parent, parentOK := arguments[0].(string)
		target, targetOK := arguments[1].(string)
		if !parentOK || !targetOK {
			return false, nil
		}
		return domain.ResourceMatch(parent, target), nil
	})

	rules := make([][]string, 0, len(policies))
	for _, policy := range policies {
		rules = append(rules, []string{
			policy.Principal.Key(),
			policy.OrganizationID.String(),
			policy.ResourceName,
			string(policy.Permission),
		})
	}

	sort.Slice(rules, func(i, j int) bool {
		for column := range rules[i] {
			if rules[i][column] != rules[j][column] {
				return rules[i][column] < rules[j][column]
			}
		}
		return false
	})
	if len(rules) > 0 {
		if _, err := enforcer.AddPolicies(rules); err != nil {
			return nil, fmt.Errorf("install Casbin policies: %w", err)
		}
	}

	return enforcer, nil
}

func cloneVersions(source map[uuid.UUID]int64) map[uuid.UUID]int64 {
	result := make(map[uuid.UUID]int64, len(source))
	for organizationID, version := range source {
		result[organizationID] = version
	}
	return result
}
