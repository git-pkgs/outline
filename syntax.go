package outline

// ImportKind identifies the source-language form of an import.
type ImportKind string

const (
	// ImportNamed imports one or more named exports.
	ImportNamed ImportKind = "named"
	// ImportDefault imports a default export.
	ImportDefault ImportKind = "default"
	// ImportNamespace imports a module namespace.
	ImportNamespace ImportKind = "namespace"
	// ImportModule binds a module object using the language's module form.
	ImportModule ImportKind = "module"
	// ImportSideEffect loads a module without binding a local name.
	ImportSideEffect ImportKind = "side-effect"
	// ImportWildcard imports every exported name.
	ImportWildcard ImportKind = "wildcard"
)

// Name is one named import and its optional local alias. ImportDefault,
// ImportNamespace, and ImportModule use Alias for a local binding declared by
// the source syntax.
type Name struct {
	Name  string
	Alias string
}

// Import is one source import form. A statement containing more than one
// form, such as a JavaScript default plus named import, produces one Import
// value for each form. Names is empty for side-effect and wildcard imports,
// and when a local binding cannot be established from the statement alone,
// such as an unaliased Go import.
type Import struct {
	Module string
	Kind   ImportKind
	Names  []Name
	Line   int
}

// Ref is a direct member access on a caller-supplied receiver.
type Ref struct {
	Receiver string
	Member   string
	Line     int
}
