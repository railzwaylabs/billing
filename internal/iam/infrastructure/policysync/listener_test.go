package policysync

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/railzwaylabs/billing/internal/iam/domain"
)

type fakeVersions struct{ values map[uuid.UUID]int64 }

func (f fakeVersions) PolicyVersions(context.Context) (map[uuid.UUID]int64, error) {
	return f.values, nil
}

type fakeEvaluator struct{ versions map[uuid.UUID]int64 }

func (f *fakeEvaluator) Enforce(context.Context, domain.Principal, uuid.UUID, string, domain.PermissionName) (bool, error) {
	return false, nil
}
func (f *fakeEvaluator) Reload(context.Context) error  { return nil }
func (f *fakeEvaluator) Versions() map[uuid.UUID]int64 { return f.versions }

func TestVersionChanged(t *testing.T) {
	organizationID := uuid.New()
	tests := []struct {
		name           string
		stored, loaded map[uuid.UUID]int64
		want           bool
	}{
		{"same", map[uuid.UUID]int64{organizationID: 2}, map[uuid.UUID]int64{organizationID: 2}, false},
		{"version advanced", map[uuid.UUID]int64{organizationID: 3}, map[uuid.UUID]int64{organizationID: 2}, true},
		{"organization added", map[uuid.UUID]int64{organizationID: 1}, map[uuid.UUID]int64{}, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			listener := Listener{source: fakeVersions{values: test.stored}, evaluator: &fakeEvaluator{versions: test.loaded}}
			got, err := listener.versionChanged(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("changed = %t, want %t", got, test.want)
			}
		})
	}
}
