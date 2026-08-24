package symbol

// SymbolKind represents the category or classification of an extracted code symbol.
type SymbolKind string

const (
	// SymbolKindPackage represents a package-level declaration.
	SymbolKindPackage SymbolKind = "package"

	// SymbolKindStruct represents a struct type declaration.
	SymbolKindStruct SymbolKind = "struct"

	// SymbolKindInterface represents an interface type declaration.
	SymbolKindInterface SymbolKind = "interface"

	// SymbolKindFunction represents a package-level function declaration.
	SymbolKindFunction SymbolKind = "function"

	// SymbolKindMethod represents a method attached to a receiver type.
	SymbolKindMethod SymbolKind = "method"

	// SymbolKindVariable represents a package or file level variable.
	SymbolKindVariable SymbolKind = "variable"

	// SymbolKindConstant represents a declared constant.
	SymbolKindConstant SymbolKind = "constant"

	// SymbolKindType represents a named defined type.
	SymbolKindType SymbolKind = "type"

	// SymbolKindGeneric represents a generic type parameter or constraint.
	SymbolKindGeneric SymbolKind = "generic"

	// SymbolKindAlias represents a type alias declaration (type A = B).
	SymbolKindAlias SymbolKind = "alias"

	// SymbolKindUnknown represents an unclassified symbol kind.
	SymbolKindUnknown SymbolKind = "unknown"
)

// RelationshipKind represents the structural relationship between symbols.
type RelationshipKind string

const (
	// RelFunctionOwnership represents a package owning a standalone function.
	RelFunctionOwnership RelationshipKind = "function_ownership"

	// RelMethodReceiver represents a method attached to a receiver struct or type.
	RelMethodReceiver RelationshipKind = "method_receiver"

	// RelInterfaceImplementation represents a struct implementing an interface.
	RelInterfaceImplementation RelationshipKind = "interface_implementation"

	// RelStructEmbedding represents a struct or interface embedding another type.
	RelStructEmbedding RelationshipKind = "struct_embedding"

	// RelTypeAlias represents a type alias pointing to its target type.
	RelTypeAlias RelationshipKind = "type_alias"

	// RelGenericConstraint represents a generic type parameter bounded by a constraint.
	RelGenericConstraint RelationshipKind = "generic_constraint"

	// RelUnknown represents an unclassified relationship kind.
	RelUnknown RelationshipKind = "unknown"
)

// DocKind represents the category of extracted documentation or comment metadata.
type DocKind string

const (
	// DocKindPackage represents documentation attached to a package declaration.
	DocKindPackage DocKind = "package_doc"

	// DocKindStruct represents documentation attached to a struct declaration.
	DocKindStruct DocKind = "struct_doc"

	// DocKindFunction represents documentation attached to a function declaration.
	DocKindFunction DocKind = "function_doc"

	// DocKindInterface represents documentation attached to an interface declaration.
	DocKindInterface DocKind = "interface_doc"

	// DocKindMethod represents documentation attached to a method declaration.
	DocKindMethod DocKind = "method_doc"

	// DocKindTODO represents an observational TODO comment found in source code.
	DocKindTODO DocKind = "todo"

	// DocKindFIXME represents an observational FIXME comment found in source code.
	DocKindFIXME DocKind = "fixme"

	// DocKindGeneral represents general standalone documentation comments.
	DocKindGeneral DocKind = "general_doc"
)
