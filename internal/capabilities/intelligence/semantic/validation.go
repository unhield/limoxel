package semantic

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ValidationFinding represents a single semantic integrity issue discovered during validation.
type ValidationFinding struct {
	id       string
	kind     ValidationFindingKind
	severity ValidationSeverity
	entityID string
	filePath string
	line     int
	message  string
	status   ValidationStatus
}

// NewValidationFinding constructs an immutable ValidationFinding.
func NewValidationFinding(
	id string,
	kind ValidationFindingKind,
	severity ValidationSeverity,
	entityID, filePath string,
	line int,
	message string,
	status ValidationStatus,
) *ValidationFinding {
	cleanFile := filepath.ToSlash(filepath.Clean(filePath))
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		cleanID = fmt.Sprintf("val:%s:%s:%d", kind, cleanFile, line)
	}

	return &ValidationFinding{
		id:       cleanID,
		kind:     kind,
		severity: severity,
		entityID: strings.TrimSpace(entityID),
		filePath: cleanFile,
		line:     line,
		message:  strings.TrimSpace(message),
		status:   status,
	}
}

func (f *ValidationFinding) ID() string                   { return f.id }
func (f *ValidationFinding) Kind() ValidationFindingKind  { return f.kind }
func (f *ValidationFinding) Severity() ValidationSeverity { return f.severity }
func (f *ValidationFinding) EntityID() string             { return f.entityID }
func (f *ValidationFinding) FilePath() string             { return f.filePath }
func (f *ValidationFinding) Line() int                    { return f.line }
func (f *ValidationFinding) Message() string              { return f.message }
func (f *ValidationFinding) Status() ValidationStatus     { return f.status }

// ValidationReport contains the aggregated findings and overall status of semantic validation.
type ValidationReport struct {
	status        ValidationStatus
	findings      []*ValidationFinding
	totalErrors   int
	totalWarnings int
	totalInfo     int
}

// NewValidationReport constructs an immutable ValidationReport.
func NewValidationReport(findings []*ValidationFinding) *ValidationReport {
	fList := make([]*ValidationFinding, len(findings))
	copy(fList, findings)
	sort.Slice(fList, func(i, j int) bool {
		if fList[i].FilePath() != fList[j].FilePath() {
			return fList[i].FilePath() < fList[j].FilePath()
		}
		if fList[i].Line() != fList[j].Line() {
			return fList[i].Line() < fList[j].Line()
		}
		return fList[i].ID() < fList[j].ID()
	})

	errCount := 0
	warnCount := 0
	infoCount := 0

	for _, f := range fList {
		switch f.Severity() {
		case SeverityError:
			errCount++
		case SeverityWarning:
			warnCount++
		case SeverityInfo:
			infoCount++
		}
	}

	overallStatus := StatusValid
	if errCount > 0 {
		overallStatus = StatusInvalid
	} else if warnCount > 0 {
		overallStatus = StatusUnresolved
	}

	return &ValidationReport{
		status:        overallStatus,
		findings:      fList,
		totalErrors:   errCount,
		totalWarnings: warnCount,
		totalInfo:     infoCount,
	}
}

func (r *ValidationReport) Status() ValidationStatus { return r.status }
func (r *ValidationReport) TotalErrors() int         { return r.totalErrors }
func (r *ValidationReport) TotalWarnings() int       { return r.totalWarnings }
func (r *ValidationReport) TotalInfo() int           { return r.totalInfo }
func (r *ValidationReport) IsValid() bool            { return r.status == StatusValid }

func (r *ValidationReport) Findings() []*ValidationFinding {
	if r == nil || r.findings == nil {
		return nil
	}
	res := make([]*ValidationFinding, len(r.findings))
	copy(res, r.findings)
	return res
}

// SemanticValidator executes semantic integrity and consistency validation rules across the SemanticModel.
type SemanticValidator struct {
	typeResolver   *TypeResolver
	symbolResolver *SymbolResolver
}

// NewSemanticValidator creates a new SemanticValidator.
func NewSemanticValidator(typeResolver *TypeResolver, symbolResolver *SymbolResolver) *SemanticValidator {
	return &SemanticValidator{
		typeResolver:   typeResolver,
		symbolResolver: symbolResolver,
	}
}

// Validate inspects the model for missing symbols, invalid types, invalid references, duplicate definitions, and scope conflicts.
func (v *SemanticValidator) Validate(
	syms map[string]*SemanticSymbol,
	types map[string]*SemanticType,
	ifaces map[string]*SemanticInterface,
	funcs map[string]*SemanticFunction,
	vars map[string]*SemanticVariable,
	scopes map[string]*SemanticScope,
) *ValidationReport {
	var findings []*ValidationFinding

	// 1. Check for Duplicate Definitions within each scope
	for _, scope := range scopes {
		if scope == nil {
			continue
		}
		seenSymbols := make(map[string]string)
		for _, symID := range scope.DeclaredSymbolIDs() {
			sym := syms[symID]
			if sym == nil {
				continue
			}
			name := sym.Name()
			if origID, exists := seenSymbols[name]; exists && origID != sym.ID() {
				orig := syms[origID]
				findings = append(findings, NewValidationFinding(
					fmt.Sprintf("dup:sym:%s:%s", scope.ID(), name),
					FindingDuplicateDefinition,
					SeverityError,
					sym.ID(),
					sym.FilePath(),
					sym.Line(),
					fmt.Sprintf("duplicate symbol '%s' declared in scope %s (previous at line %d)", name, scope.ID(), orig.Line()),
					StatusInvalid,
				))
			} else {
				seenSymbols[name] = symID
			}
		}

		seenVars := make(map[string]*SemanticVariable)
		for _, vr := range scope.DeclaredVariables() {
			if vr == nil {
				continue
			}
			name := vr.Name()
			if orig, exists := seenVars[name]; exists && orig.ID() != vr.ID() {
				findings = append(findings, NewValidationFinding(
					fmt.Sprintf("dup:var:%s:%s", scope.ID(), name),
					FindingDuplicateDefinition,
					SeverityError,
					vr.ID(),
					vr.FilePath(),
					vr.Line(),
					fmt.Sprintf("duplicate variable '%s' declared in scope %s", name, scope.ID()),
					StatusInvalid,
				))
			} else {
				seenVars[name] = vr
			}
		}
	}

	// 2. Check Type Validity and Cyclic Aliases
	if v.typeResolver != nil {
		for _, t := range types {
			if t == nil {
				continue
			}
			if t.IsAlias() {
				_, isCyclic := v.typeResolver.ResolveAliasChain(t.ID())
				if isCyclic {
					findings = append(findings, NewValidationFinding(
						fmt.Sprintf("cyclic:alias:%s", t.ID()),
						FindingInvalidType,
						SeverityError,
						t.ID(),
						t.FilePath(),
						1,
						fmt.Sprintf("type alias '%s' participates in a cyclic alias chain", t.Name()),
						StatusInvalid,
					))
				}
			}
		}
	}

	// 3. Check Symbol References and Cross-Package Visibility
	if v.symbolResolver != nil {
		for _, sym := range syms {
			if sym == nil {
				continue
			}
			for _, refID := range sym.References() {
				targetSym := syms[refID]
				if targetSym == nil {
					// Check if it exists in symbols
					findings = append(findings, NewValidationFinding(
						fmt.Sprintf("missing:ref:%s->%s", sym.ID(), refID),
						FindingMissingSymbol,
						SeverityWarning,
						sym.ID(),
						sym.FilePath(),
						sym.Line(),
						fmt.Sprintf("referenced symbol '%s' cannot be resolved", refID),
						StatusUnresolved,
					))
				} else {
					// Check visibility across package boundaries
					if !v.symbolResolver.CheckVisibility(targetSym.ID(), sym.PackagePath()) {
						findings = append(findings, NewValidationFinding(
							fmt.Sprintf("invis:ref:%s->%s", sym.ID(), targetSym.ID()),
							FindingInvalidReference,
							SeverityError,
							sym.ID(),
							sym.FilePath(),
							sym.Line(),
							fmt.Sprintf("symbol '%s' in package '%s' cannot access private symbol '%s' in package '%s'",
								sym.Name(), sym.PackagePath(), targetSym.Name(), targetSym.PackagePath()),
							StatusInvalid,
						))
					}
				}
			}
		}
	}

	// 4. Check Interface Implementor Consistency
	if v.typeResolver != nil {
		for _, iface := range ifaces {
			if iface == nil {
				continue
			}
			for _, implID := range iface.Implementors() {
				if !v.typeResolver.CheckInterfaceSatisfaction(implID, iface.ID()) {
					findings = append(findings, NewValidationFinding(
						fmt.Sprintf("incons:impl:%s->%s", implID, iface.ID()),
						FindingInconsistentRelationship,
						SeverityError,
						implID,
						iface.FilePath(),
						1,
						fmt.Sprintf("type '%s' does not fully satisfy interface contract '%s'", implID, iface.Name()),
						StatusInvalid,
					))
				}
			}
		}
	}

	return NewValidationReport(findings)
}
