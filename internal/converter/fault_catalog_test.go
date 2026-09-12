package converter

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// faultTypeOneOf returns the `oneof=` values a struct field's validate tag
// admits for its type. The authoring layer has to restate the fault catalog as
// a literal so the generated JSON schema carries the enum, which means the
// literal can drift from the catalog it restates -- and it had: `poe_loss` was
// a supported interface fault that no authored surface would accept, so the one
// fault a PoE demo needs could be injected at runtime and never authored.
func faultTypeOneOf(t *testing.T, structType reflect.Type, field string) []string {
	t.Helper()
	found, ok := structType.FieldByName(field)
	if !ok {
		t.Fatalf("%s has no field %s", structType.Name(), field)
	}
	for rule := range strings.SplitSeq(found.Tag.Get("validate"), ",") {
		if after, cut := strings.CutPrefix(rule, "oneof="); cut {
			return strings.Fields(after)
		}
	}
	t.Fatalf("%s.%s has no oneof rule", structType.Name(), field)

	return nil
}

func TestAuthoredFaultTypesCoverTheRuntimeCatalog(t *testing.T) {
	supported := devicestate.AuthorableFaultTypes()
	authored := faultTypeOneOf(t, reflect.TypeFor[BehaviorFault](), "Type")
	slices.Sort(authored)

	for _, faultType := range supported {
		if !slices.Contains(authored, faultType) {
			t.Errorf("%q is a runtime fault no timeline can author", faultType)
		}
	}
	for _, faultType := range authored {
		if !slices.Contains(supported, faultType) {
			t.Errorf("%q is authorable but the runtime has no such fault", faultType)
		}
	}
}
