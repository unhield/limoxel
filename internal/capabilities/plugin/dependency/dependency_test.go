package dependency_test

import (
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/dependency"
	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

type mockPlugin struct {
	id   model.Identity
	ver  version.SemVer
	deps []model.Dependency
}

func (m *mockPlugin) GetID() model.Identity               { return m.id }
func (m *mockPlugin) GetVersion() version.SemVer          { return m.ver }
func (m *mockPlugin) GetDependencies() []model.Dependency { return m.deps }

func TestResolver_SuccessTopological(t *testing.T) {
	// A depends on B, B depends on C -> Order: C, B, A
	v1, _ := version.ParseSemVer("1.0.0")

	pC := &mockPlugin{id: model.MustParseIdentity("c.plugin"), ver: v1}
	pB := &mockPlugin{id: model.MustParseIdentity("b.plugin"), ver: v1, deps: []model.Dependency{
		{ID: pC.id, Constraint: ">= 1.0.0"},
	}}
	pA := &mockPlugin{id: model.MustParseIdentity("a.plugin"), ver: v1, deps: []model.Dependency{
		{ID: pB.id, Constraint: "^1.0.0"},
	}}

	res := dependency.NewResolver()
	plan, err := res.Resolve([]dependency.Resolvable{pA, pB, pC})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(plan.ActivationOrder) != 3 {
		t.Fatalf("expected 3 items in activation order, got %d", len(plan.ActivationOrder))
	}

	// C must be before B, B must be before A
	idxC, idxB, idxA := -1, -1, -1
	for i, id := range plan.ActivationOrder {
		switch id {
		case pC.id:
			idxC = i
		case pB.id:
			idxB = i
		case pA.id:
			idxA = i
		}
	}

	if !(idxC < idxB && idxB < idxA) {
		t.Errorf("expected order C < B < A, got C=%d, B=%d, A=%d", idxC, idxB, idxA)
	}

	// Deactivation must be exact reverse
	if plan.DeactivationOrder[0] != plan.ActivationOrder[2] ||
		plan.DeactivationOrder[1] != plan.ActivationOrder[1] ||
		plan.DeactivationOrder[2] != plan.ActivationOrder[0] {
		t.Errorf("deactivation order mismatch: %v vs %v", plan.DeactivationOrder, plan.ActivationOrder)
	}
}

func TestResolver_CycleDetection(t *testing.T) {
	// A -> B -> C -> A
	v1, _ := version.ParseSemVer("1.0.0")

	idA := model.MustParseIdentity("a.plugin")
	idB := model.MustParseIdentity("b.plugin")
	idC := model.MustParseIdentity("c.plugin")

	pA := &mockPlugin{id: idA, ver: v1, deps: []model.Dependency{{ID: idB}}}
	pB := &mockPlugin{id: idB, ver: v1, deps: []model.Dependency{{ID: idC}}}
	pC := &mockPlugin{id: idC, ver: v1, deps: []model.Dependency{{ID: idA}}}

	res := dependency.NewResolver()
	_, err := res.Resolve([]dependency.Resolvable{pA, pB, pC})
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}

	pErr, ok := err.(*pkgerr.PluginError)
	if !ok || pErr.Code() != pkgerr.CodeDependencyCycle {
		t.Errorf("expected CodeDependencyCycle, got %v", err)
	}
}

func TestResolver_MissingMandatoryDependency(t *testing.T) {
	v1, _ := version.ParseSemVer("1.0.0")
	pA := &mockPlugin{
		id:  model.MustParseIdentity("a.plugin"),
		ver: v1,
		deps: []model.Dependency{
			{ID: model.MustParseIdentity("missing.dep"), Optional: false},
		},
	}

	res := dependency.NewResolver()
	_, err := res.Resolve([]dependency.Resolvable{pA})
	if err == nil {
		t.Fatal("expected missing dependency error, got nil")
	}

	pErr, ok := err.(*pkgerr.PluginError)
	if !ok || pErr.Code() != pkgerr.CodeDependencyMissing {
		t.Errorf("expected CodeDependencyMissing, got %v", err)
	}
}

func TestResolver_MissingOptionalDependency(t *testing.T) {
	v1, _ := version.ParseSemVer("1.0.0")
	pA := &mockPlugin{
		id:  model.MustParseIdentity("a.plugin"),
		ver: v1,
		deps: []model.Dependency{
			{ID: model.MustParseIdentity("optional.dep"), Optional: true},
		},
	}

	res := dependency.NewResolver()
	plan, err := res.Resolve([]dependency.Resolvable{pA})
	if err != nil {
		t.Fatalf("expected optional missing dependency to succeed, got error: %v", err)
	}
	if len(plan.ActivationOrder) != 1 {
		t.Errorf("expected 1 plugin in activation order, got %d", len(plan.ActivationOrder))
	}
}

func TestResolver_IncompatibleVersion(t *testing.T) {
	v1, _ := version.ParseSemVer("1.0.0")
	v2, _ := version.ParseSemVer("2.0.0")

	pB := &mockPlugin{id: model.MustParseIdentity("b.plugin"), ver: v2}
	pA := &mockPlugin{
		id:  model.MustParseIdentity("a.plugin"),
		ver: v1,
		deps: []model.Dependency{
			{ID: pB.id, Constraint: "< 2.0.0"}, // requires < 2.0.0 but B is 2.0.0
		},
	}

	res := dependency.NewResolver()
	_, err := res.Resolve([]dependency.Resolvable{pA, pB})
	if err == nil {
		t.Fatal("expected incompatible version error, got nil")
	}

	pErr, ok := err.(*pkgerr.PluginError)
	if !ok || pErr.Code() != pkgerr.CodeIncompatibleVersion {
		t.Errorf("expected CodeIncompatibleVersion, got %v", err)
	}
}
