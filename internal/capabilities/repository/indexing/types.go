package indexing

// FileType represents the structural classification of a file.
type FileType string

const (
	// FileTypeSource represents standard source code files.
	FileTypeSource FileType = "source"

	// FileTypeTest represents test suite files.
	FileTypeTest FileType = "test"

	// FileTypeConfig represents configuration files.
	FileTypeConfig FileType = "config"

	// FileTypeDoc represents documentation files.
	FileTypeDoc FileType = "doc"

	// FileTypeBinary represents binary or non-text files.
	FileTypeBinary FileType = "binary"

	// FileTypeUnsupported represents recognized but unsupported file formats.
	FileTypeUnsupported FileType = "unsupported"

	// FileTypeUnknown represents an unclassified file format.
	FileTypeUnknown FileType = "unknown"
)

// GenerationStatus represents whether a file was machine-generated or handwritten.
type GenerationStatus string

const (
	// GenerationStatusGenerated indicates deterministic evidence of automated code generation.
	GenerationStatusGenerated GenerationStatus = "generated"

	// GenerationStatusHandwritten indicates standard handwritten source code.
	GenerationStatusHandwritten GenerationStatus = "handwritten"

	// GenerationStatusUnknown indicates generation status cannot be determined with certainty.
	GenerationStatusUnknown GenerationStatus = "unknown"
)

// EncodingType represents the detected character encoding of a file.
type EncodingType string

const (
	// EncodingUTF8 represents UTF-8 text encoding.
	EncodingUTF8 EncodingType = "UTF-8"

	// EncodingASCII represents pure 7-bit ASCII text.
	EncodingASCII EncodingType = "ASCII"

	// EncodingUTF16LE represents UTF-16 Little Endian encoding.
	EncodingUTF16LE EncodingType = "UTF-16LE"

	// EncodingUTF16BE represents UTF-16 Big Endian encoding.
	EncodingUTF16BE EncodingType = "UTF-16BE"

	// EncodingInvalid represents corrupted or invalid byte sequences.
	EncodingInvalid EncodingType = "Invalid"

	// EncodingUnknown represents unclassified or binary encoding.
	EncodingUnknown EncodingType = "Unknown"
)

// LineEndingType represents the newline format detected in a text file.
type LineEndingType string

const (
	// LineEndingLF represents standard Unix line endings (\n).
	LineEndingLF LineEndingType = "LF"

	// LineEndingCRLF represents Windows line endings (\r\n).
	LineEndingCRLF LineEndingType = "CRLF"

	// LineEndingCR represents classic Mac line endings (\r).
	LineEndingCR LineEndingType = "CR"

	// LineEndingMixed represents mixed line endings within the same file.
	LineEndingMixed LineEndingType = "Mixed"

	// LineEndingUnknown represents unknown or empty line endings.
	LineEndingUnknown LineEndingType = "Unknown"
)

// RelationshipType represents the category of relationship between repository artifacts.
type RelationshipType string

const (
	// RelImport represents a source file importing another package/module.
	RelImport RelationshipType = "import"

	// RelPackageMembership represents a source file belonging to a package.
	RelPackageMembership RelationshipType = "package_membership"

	// RelParentChild represents a structural directory/containment relationship.
	RelParentChild RelationshipType = "parent_child"

	// RelTestToSource represents a test file associated with a source file.
	RelTestToSource RelationshipType = "test_to_source"

	// RelConfigToSource represents a configuration file associated with a module/source.
	RelConfigToSource RelationshipType = "config_to_source"

	// RelDocToModule represents a documentation file associated with a module.
	RelDocToModule RelationshipType = "doc_to_module"

	// RelUnknown represents an unclassified relationship type.
	RelUnknown RelationshipType = "unknown"
)
