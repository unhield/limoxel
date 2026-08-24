package symbol

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// SourcePosition represents a deterministic location within a source file.
type SourcePosition struct {
	file   string
	line   int
	column int
	offset int
}

// NewSourcePosition constructs an immutable SourcePosition record.
func NewSourcePosition(file string, line, column, offset int) *SourcePosition {
	return &SourcePosition{
		file:   filepath.ToSlash(filepath.Clean(file)),
		line:   line,
		column: column,
		offset: offset,
	}
}

// File returns the repository-relative file path.
func (sp *SourcePosition) File() string {
	if sp == nil {
		return ""
	}
	return sp.file
}

// Line returns the 1-based line number.
func (sp *SourcePosition) Line() int {
	if sp == nil {
		return 0
	}
	return sp.line
}

// Column returns the 1-based column offset.
func (sp *SourcePosition) Column() int {
	if sp == nil {
		return 0
	}
	return sp.column
}

// Offset returns the byte offset from the start of the file.
func (sp *SourcePosition) Offset() int {
	if sp == nil {
		return 0
	}
	return sp.offset
}

// String returns a canonical representation of the source position (file:line:column).
func (sp *SourcePosition) String() string {
	if sp == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d:%d", sp.file, sp.line, sp.column)
}

// DocEntry represents an extracted documentation or comment metadata entry.
type DocEntry struct {
	id             string
	targetSymbolID string
	kind           DocKind
	content        string
	rawText        string
	pos            *SourcePosition
}

// NewDocEntry constructs an immutable DocEntry record.
func NewDocEntry(id, targetSymbolID string, kind DocKind, content, rawText string, pos *SourcePosition) *DocEntry {
	cleanContent := strings.TrimSpace(content)
	cleanRaw := strings.TrimSpace(rawText)
	cleanID := strings.TrimSpace(id)
	if cleanID == "" && pos != nil {
		cleanID = fmt.Sprintf("doc:%s:%d:%d", pos.File(), pos.Line(), pos.Column())
	}

	return &DocEntry{
		id:             cleanID,
		targetSymbolID: strings.TrimSpace(targetSymbolID),
		kind:           kind,
		content:        cleanContent,
		rawText:        cleanRaw,
		pos:            pos,
	}
}

// ID returns the stable identifier of the documentation entry.
func (de *DocEntry) ID() string {
	if de == nil {
		return ""
	}
	return de.id
}

// TargetSymbolID returns the identifier of the associated symbol if attached.
func (de *DocEntry) TargetSymbolID() string {
	if de == nil {
		return ""
	}
	return de.targetSymbolID
}

// Kind returns the category of documentation entry.
func (de *DocEntry) Kind() DocKind {
	if de == nil {
		return DocKindGeneral
	}
	return de.kind
}

// Content returns the clean, normalized documentation text.
func (de *DocEntry) Content() string {
	if de == nil {
		return ""
	}
	return de.content
}

// RawText returns the original verbatim comment text.
func (de *DocEntry) RawText() string {
	if de == nil {
		return ""
	}
	return de.rawText
}

// Position returns the source position where the comment begins.
func (de *DocEntry) Position() *SourcePosition {
	if de == nil {
		return nil
	}
	return de.pos
}

// Symbol represents an extracted code symbol with location, signature, and metadata.
type Symbol struct {
	id                string
	kind              SymbolKind
	name              string
	packageName       string
	packagePath       string
	filePath          string
	isExported        bool
	receiverType      string
	isPointerReceiver bool
	signature         string
	typeDefinition    string
	isAlias           bool
	generics          []string
	fields            []string
	pos               *SourcePosition
	doc               *DocEntry
}

// NewSymbol constructs an immutable Symbol record.
func NewSymbol(
	id string,
	kind SymbolKind,
	name string,
	pkgName string,
	pkgPath string,
	filePath string,
	receiverType string,
	isPointerRecv bool,
	signature string,
	typeDef string,
	isAlias bool,
	generics []string,
	fields []string,
	pos *SourcePosition,
	doc *DocEntry,
) *Symbol {
	cleanName := strings.TrimSpace(name)
	cleanPkgName := strings.TrimSpace(pkgName)
	cleanPkgPath := filepath.ToSlash(filepath.Clean(pkgPath))
	cleanFilePath := filepath.ToSlash(filepath.Clean(filePath))

	// Compute export status from name
	isExported := false
	if len(cleanName) > 0 {
		isExported = unicode.IsUpper([]rune(cleanName)[0])
	}

	// Compute deterministic symbol ID if not provided
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		if receiverType != "" {
			cleanID = fmt.Sprintf("%s.(%s).%s", cleanPkgPath, receiverType, cleanName)
		} else if cleanPkgPath != "" && cleanPkgPath != "." {
			cleanID = fmt.Sprintf("%s.%s", cleanPkgPath, cleanName)
		} else {
			cleanID = cleanName
		}
	}

	cleanGenerics := make([]string, len(generics))
	copy(cleanGenerics, generics)
	sort.Strings(cleanGenerics)

	cleanFields := make([]string, len(fields))
	copy(cleanFields, fields)
	sort.Strings(cleanFields)

	return &Symbol{
		id:                cleanID,
		kind:              kind,
		name:              cleanName,
		packageName:       cleanPkgName,
		packagePath:       cleanPkgPath,
		filePath:          cleanFilePath,
		isExported:        isExported,
		receiverType:      strings.TrimSpace(receiverType),
		isPointerReceiver: isPointerRecv,
		signature:         strings.TrimSpace(signature),
		typeDefinition:    strings.TrimSpace(typeDef),
		isAlias:           isAlias,
		generics:          cleanGenerics,
		fields:            cleanFields,
		pos:               pos,
		doc:               doc,
	}
}

// ID returns the unique deterministic identifier of the symbol.
func (s *Symbol) ID() string {
	if s == nil {
		return ""
	}
	return s.id
}

// Kind returns the symbol kind.
func (s *Symbol) Kind() SymbolKind {
	if s == nil {
		return SymbolKindUnknown
	}
	return s.kind
}

// Name returns the unqualified symbol name.
func (s *Symbol) Name() string {
	if s == nil {
		return ""
	}
	return s.name
}

// PackageName returns the declaring package name.
func (s *Symbol) PackageName() string {
	if s == nil {
		return ""
	}
	return s.packageName
}

// PackagePath returns the repository-relative directory path of the declaring package.
func (s *Symbol) PackagePath() string {
	if s == nil {
		return ""
	}
	return s.packagePath
}

// FilePath returns the repository-relative file path containing the declaration.
func (s *Symbol) FilePath() string {
	if s == nil {
		return ""
	}
	return s.filePath
}

// IsExported reports whether the symbol is exported (public).
func (s *Symbol) IsExported() bool {
	if s == nil {
		return false
	}
	return s.isExported
}

// ReceiverType returns the receiver type for methods (empty for standalone functions).
func (s *Symbol) ReceiverType() string {
	if s == nil {
		return ""
	}
	return s.receiverType
}

// IsPointerReceiver reports whether a method receiver is a pointer receiver.
func (s *Symbol) IsPointerReceiver() bool {
	if s == nil {
		return false
	}
	return s.isPointerReceiver
}

// Signature returns the type or function signature representation.
func (s *Symbol) Signature() string {
	if s == nil {
		return ""
	}
	return s.signature
}

// TypeDefinition returns the underlying type definition string.
func (s *Symbol) TypeDefinition() string {
	if s == nil {
		return ""
	}
	return s.typeDefinition
}

// IsAlias reports whether the symbol is a type alias (type A = B).
func (s *Symbol) IsAlias() bool {
	if s == nil {
		return false
	}
	return s.isAlias
}

// Generics returns a defensive copy of generic type parameter definitions.
func (s *Symbol) Generics() []string {
	if s == nil || len(s.generics) == 0 {
		return nil
	}
	cloned := make([]string, len(s.generics))
	copy(cloned, s.generics)
	return cloned
}

// Fields returns a defensive copy of struct fields or interface methods.
func (s *Symbol) Fields() []string {
	if s == nil || len(s.fields) == 0 {
		return nil
	}
	cloned := make([]string, len(s.fields))
	copy(cloned, s.fields)
	return cloned
}

// Position returns the source position of the symbol declaration.
func (s *Symbol) Position() *SourcePosition {
	if s == nil {
		return nil
	}
	return s.pos
}

// Doc returns the attached documentation entry, if any.
func (s *Symbol) Doc() *DocEntry {
	if s == nil {
		return nil
	}
	return s.doc
}

// String returns a human-readable representation of the symbol.
func (s *Symbol) String() string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("Symbol<%s, kind=%s, pkg=%s>", s.id, s.kind, s.packageName)
}

// SymbolRelationship represents a directed relationship between symbols.
type SymbolRelationship struct {
	sourceID string
	targetID string
	kind     RelationshipKind
	evidence string
	pos      *SourcePosition
}

// NewSymbolRelationship constructs an immutable SymbolRelationship record.
func NewSymbolRelationship(sourceID, targetID string, kind RelationshipKind, evidence string, pos *SourcePosition) *SymbolRelationship {
	return &SymbolRelationship{
		sourceID: strings.TrimSpace(sourceID),
		targetID: strings.TrimSpace(targetID),
		kind:     kind,
		evidence: strings.TrimSpace(evidence),
		pos:      pos,
	}
}

// SourceID returns the source symbol ID.
func (sr *SymbolRelationship) SourceID() string {
	if sr == nil {
		return ""
	}
	return sr.sourceID
}

// TargetID returns the target symbol ID.
func (sr *SymbolRelationship) TargetID() string {
	if sr == nil {
		return ""
	}
	return sr.targetID
}

// Kind returns the relationship kind classification.
func (sr *SymbolRelationship) Kind() RelationshipKind {
	if sr == nil {
		return RelUnknown
	}
	return sr.kind
}

// Evidence returns the structural rationale or evidence string.
func (sr *SymbolRelationship) Evidence() string {
	if sr == nil {
		return ""
	}
	return sr.evidence
}

// Position returns the source location associated with this relationship.
func (sr *SymbolRelationship) Position() *SourcePosition {
	if sr == nil {
		return nil
	}
	return sr.pos
}

// SymbolDatabase represents the organized, queryable database of extracted symbols.
type SymbolDatabase struct {
	symbols []*Symbol
	byId    map[string]*Symbol
}

// NewSymbolDatabase constructs an immutable SymbolDatabase with fast lookups.
func NewSymbolDatabase(symbols []*Symbol) *SymbolDatabase {
	symList := make([]*Symbol, len(symbols))
	copy(symList, symbols)
	sort.Slice(symList, func(i, j int) bool {
		if symList[i].packagePath != symList[j].packagePath {
			return symList[i].packagePath < symList[j].packagePath
		}
		return symList[i].id < symList[j].id
	})

	byId := make(map[string]*Symbol, len(symList))
	for _, s := range symList {
		byId[s.id] = s
	}

	return &SymbolDatabase{
		symbols: symList,
		byId:    byId,
	}
}

// SymbolByID retrieves a symbol by its exact symbol ID in $O(1)$ time.
func (sdb *SymbolDatabase) SymbolByID(id string) *Symbol {
	if sdb == nil || len(sdb.byId) == 0 {
		return nil
	}
	return sdb.byId[strings.TrimSpace(id)]
}

// SymbolsByPackage returns all symbols declared within the given package path.
func (sdb *SymbolDatabase) SymbolsByPackage(pkgPath string) []*Symbol {
	if sdb == nil || len(sdb.symbols) == 0 {
		return nil
	}
	clean := filepath.ToSlash(filepath.Clean(pkgPath))
	var res []*Symbol
	for _, s := range sdb.symbols {
		if s.packagePath == clean {
			res = append(res, s)
		}
	}
	return res
}

// SymbolsByFile returns all symbols declared within the given file path.
func (sdb *SymbolDatabase) SymbolsByFile(filePath string) []*Symbol {
	if sdb == nil || len(sdb.symbols) == 0 {
		return nil
	}
	clean := filepath.ToSlash(filepath.Clean(filePath))
	var res []*Symbol
	for _, s := range sdb.symbols {
		if s.filePath == clean {
			res = append(res, s)
		}
	}
	return res
}

// SymbolsByKind returns all symbols matching the given symbol kind.
func (sdb *SymbolDatabase) SymbolsByKind(kind SymbolKind) []*Symbol {
	if sdb == nil || len(sdb.symbols) == 0 {
		return nil
	}
	var res []*Symbol
	for _, s := range sdb.symbols {
		if s.kind == kind {
			res = append(res, s)
		}
	}
	return res
}

// SymbolsByName returns all symbols matching the unqualified name across all packages.
func (sdb *SymbolDatabase) SymbolsByName(name string) []*Symbol {
	if sdb == nil || len(sdb.symbols) == 0 {
		return nil
	}
	clean := strings.TrimSpace(name)
	var res []*Symbol
	for _, s := range sdb.symbols {
		if s.name == clean {
			res = append(res, s)
		}
	}
	return res
}

// AllSymbols returns a defensive copy of all symbols in deterministic order.
func (sdb *SymbolDatabase) AllSymbols() []*Symbol {
	if sdb == nil || len(sdb.symbols) == 0 {
		return nil
	}
	cloned := make([]*Symbol, len(sdb.symbols))
	copy(cloned, sdb.symbols)
	return cloned
}

// TotalCount returns the total number of symbols in the database.
func (sdb *SymbolDatabase) TotalCount() int {
	if sdb == nil {
		return 0
	}
	return len(sdb.symbols)
}

// DocumentationDatabase represents the queryable collection of documentation and comment metadata.
type DocumentationDatabase struct {
	docs []*DocEntry
}

// NewDocumentationDatabase constructs an immutable DocumentationDatabase.
func NewDocumentationDatabase(docs []*DocEntry) *DocumentationDatabase {
	docList := make([]*DocEntry, len(docs))
	copy(docList, docs)
	sort.Slice(docList, func(i, j int) bool {
		if docList[i].pos != nil && docList[j].pos != nil {
			if docList[i].pos.file != docList[j].pos.file {
				return docList[i].pos.file < docList[j].pos.file
			}
			return docList[i].pos.line < docList[j].pos.line
		}
		return docList[i].id < docList[j].id
	})

	return &DocumentationDatabase{docs: docList}
}

// DocsForSymbol returns all documentation entries attached to a given symbol ID.
func (ddb *DocumentationDatabase) DocsForSymbol(symbolID string) []*DocEntry {
	if ddb == nil || len(ddb.docs) == 0 {
		return nil
	}
	clean := strings.TrimSpace(symbolID)
	var res []*DocEntry
	for _, d := range ddb.docs {
		if d.targetSymbolID == clean {
			res = append(res, d)
		}
	}
	return res
}

// TODOs returns all extracted TODO comment entries in deterministic order.
func (ddb *DocumentationDatabase) TODOs() []*DocEntry {
	if ddb == nil || len(ddb.docs) == 0 {
		return nil
	}
	var res []*DocEntry
	for _, d := range ddb.docs {
		if d.kind == DocKindTODO {
			res = append(res, d)
		}
	}
	return res
}

// FIXMEs returns all extracted FIXME comment entries in deterministic order.
func (ddb *DocumentationDatabase) FIXMEs() []*DocEntry {
	if ddb == nil || len(ddb.docs) == 0 {
		return nil
	}
	var res []*DocEntry
	for _, d := range ddb.docs {
		if d.kind == DocKindFIXME {
			res = append(res, d)
		}
	}
	return res
}

// AllDocs returns a defensive copy of all documentation entries.
func (ddb *DocumentationDatabase) AllDocs() []*DocEntry {
	if ddb == nil || len(ddb.docs) == 0 {
		return nil
	}
	cloned := make([]*DocEntry, len(ddb.docs))
	copy(cloned, ddb.docs)
	return cloned
}

// TotalCount returns the count of documentation records.
func (ddb *DocumentationDatabase) TotalCount() int {
	if ddb == nil {
		return 0
	}
	return len(ddb.docs)
}

// SymbolRelationshipGraph represents the directed graph of symbol relationships.
type SymbolRelationshipGraph struct {
	relationships []*SymbolRelationship
}

// NewSymbolRelationshipGraph constructs an immutable SymbolRelationshipGraph.
func NewSymbolRelationshipGraph(relationships []*SymbolRelationship) *SymbolRelationshipGraph {
	relList := make([]*SymbolRelationship, len(relationships))
	copy(relList, relationships)
	sort.Slice(relList, func(i, j int) bool {
		if relList[i].sourceID != relList[j].sourceID {
			return relList[i].sourceID < relList[j].sourceID
		}
		if relList[i].targetID != relList[j].targetID {
			return relList[i].targetID < relList[j].targetID
		}
		return relList[i].kind < relList[j].kind
	})

	return &SymbolRelationshipGraph{relationships: relList}
}

// RelationshipsForSource returns all relationships originating from the specified source symbol ID.
func (srg *SymbolRelationshipGraph) RelationshipsForSource(sourceID string) []*SymbolRelationship {
	if srg == nil || len(srg.relationships) == 0 {
		return nil
	}
	clean := strings.TrimSpace(sourceID)
	var res []*SymbolRelationship
	for _, r := range srg.relationships {
		if r.sourceID == clean {
			res = append(res, r)
		}
	}
	return res
}

// RelationshipsForTarget returns all relationships pointing to the specified target symbol ID.
func (srg *SymbolRelationshipGraph) RelationshipsForTarget(targetID string) []*SymbolRelationship {
	if srg == nil || len(srg.relationships) == 0 {
		return nil
	}
	clean := strings.TrimSpace(targetID)
	var res []*SymbolRelationship
	for _, r := range srg.relationships {
		if r.targetID == clean {
			res = append(res, r)
		}
	}
	return res
}

// RelationshipsByKind returns all relationships of the specified kind.
func (srg *SymbolRelationshipGraph) RelationshipsByKind(kind RelationshipKind) []*SymbolRelationship {
	if srg == nil || len(srg.relationships) == 0 {
		return nil
	}
	var res []*SymbolRelationship
	for _, r := range srg.relationships {
		if r.kind == kind {
			res = append(res, r)
		}
	}
	return res
}

// AllRelationships returns a defensive copy of all symbol relationships in deterministic order.
func (srg *SymbolRelationshipGraph) AllRelationships() []*SymbolRelationship {
	if srg == nil || len(srg.relationships) == 0 {
		return nil
	}
	cloned := make([]*SymbolRelationship, len(srg.relationships))
	copy(cloned, srg.relationships)
	return cloned
}

// TotalCount returns the count of symbol relationships.
func (srg *SymbolRelationshipGraph) TotalCount() int {
	if srg == nil {
		return 0
	}
	return len(srg.relationships)
}
