package api

import (
	"reflect"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// modulePath prefixes every package belonging to this service.
const modulePath = "github.com/softika/gopherizer/"

// schemaNamer names OpenAPI component schemas.
//
// huma keeps one global schema registry keyed by type name and its default
// namer ignores the package, so health.Response and profile.Response collide.
// Types declared in this module are therefore qualified by their package —
// HealthResponse, ProfileResponse — while foreign types (huma's own error
// model, for one) keep their conventional names.
func schemaNamer(t reflect.Type, hint string) string {
	name := huma.DefaultSchemaNamer(t, hint)

	elem := t
	for elem.Kind() == reflect.Pointer || elem.Kind() == reflect.Slice {
		elem = elem.Elem()
	}

	pkg := elem.PkgPath()
	if !strings.HasPrefix(pkg, modulePath) {
		return name
	}

	short := pkg[strings.LastIndex(pkg, "/")+1:]
	if short == "" {
		return name
	}

	return strings.ToUpper(short[:1]) + short[1:] + name
}

// newSchemaRegistry builds the registry used for the generated document.
func newSchemaRegistry() huma.Registry {
	return huma.NewMapRegistry("#/components/schemas/", schemaNamer)
}
