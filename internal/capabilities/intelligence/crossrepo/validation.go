package crossrepo

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ValidationFinding represents an individual finding discovered during cross-boundary validation.
type ValidationFinding struct {
	id           string
	kind         ValidationFindingKind
	severity     ValidationSeverity
	boundary     BoundaryKind
	sourceEntity string
	targetEntity string
	filePath     string
	line         int
	message      string
	evidence     string
}

// NewValidationFinding creates an immutable ValidationFinding.
func NewValidationFinding(
	id string,
	kind ValidationFindingKind,
	sev ValidationSeverity,
	boundary BoundaryKind,
	source, target, filePath string,
	line int,
	msg, evidence string,
) *ValidationFinding {
	cleanFile := filepath.ToSlash(filepath.Clean(filePath))
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		cleanID = "finding:" + cleanFile + ":" + string(kind)
	}

	return &ValidationFinding{
		id:           cleanID,
		kind:         kind,
		severity:     sev,
		boundary:     boundary,
		sourceEntity: strings.TrimSpace(source),
		targetEntity: strings.TrimSpace(target),
		filePath:     cleanFile,
		line:         line,
		message:      strings.TrimSpace(msg),
		evidence:     strings.TrimSpace(evidence),
	}
}

func (f *ValidationFinding) ID() string                   { return f.id }
func (f *ValidationFinding) Kind() ValidationFindingKind  { return f.kind }
func (f *ValidationFinding) Severity() ValidationSeverity { return f.severity }
func (f *ValidationFinding) Boundary() BoundaryKind       { return f.boundary }
func (f *ValidationFinding) SourceEntity() string         { return f.sourceEntity }
func (f *ValidationFinding) TargetEntity() string         { return f.targetEntity }
func (f *ValidationFinding) FilePath() string             { return f.filePath }
func (f *ValidationFinding) Line() int                    { return f.line }
func (f *ValidationFinding) Message() string              { return f.message }
func (f *ValidationFinding) Evidence() string             { return f.evidence }

// ValidationReport aggregates findings from cross-repository validation.
type ValidationReport struct {
	id            string
	findings      []*ValidationFinding
	totalErrors   int
	totalWarnings int
	totalInfos    int
	status        ValidationStatus
	analyzedAt    time.Time
}

// NewValidationReport creates an immutable ValidationReport with sorted findings.
func NewValidationReport(findings []*ValidationFinding, analyzedAt time.Time) *ValidationReport {
	fList := make([]*ValidationFinding, len(findings))
	copy(fList, findings)

	// Deterministic sorting
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

	status := StatusValid
	if errCount > 0 {
		status = StatusInvalid
	} else if warnCount > 0 {
		status = StatusUnresolved
	}

	return &ValidationReport{
		id:            "valrep:crossrepo",
		findings:      fList,
		totalErrors:   errCount,
		totalWarnings: warnCount,
		totalInfos:    infoCount,
		status:        status,
		analyzedAt:    analyzedAt,
	}
}

func (r *ValidationReport) ID() string               { return r.id }
func (r *ValidationReport) TotalErrors() int         { return r.totalErrors }
func (r *ValidationReport) TotalWarnings() int       { return r.totalWarnings }
func (r *ValidationReport) TotalInfos() int          { return r.totalInfos }
func (r *ValidationReport) Status() ValidationStatus { return r.status }
func (r *ValidationReport) AnalyzedAt() time.Time    { return r.analyzedAt }

func (r *ValidationReport) Findings() []*ValidationFinding {
	if r == nil || r.findings == nil {
		return nil
	}
	res := make([]*ValidationFinding, len(r.findings))
	copy(res, r.findings)
	return res
}

// CrossRepoValidator performs cross-boundary integrity validation.
type CrossRepoValidator struct{}

// NewCrossRepoValidator creates a new validator.
func NewCrossRepoValidator() *CrossRepoValidator {
	return &CrossRepoValidator{}
}

// Validate executes validation checks across all cross-boundary dimensions.
func (v *CrossRepoValidator) Validate(
	fileRels []*FileRelationship,
	symbolProps []*SymbolPropagation,
	crossFileDeps []*CrossFileDependency,
	sharedConfigs []*SharedConfig,
	pkgComms []*PackageCommunication,
	modRels []*ModuleRelationship,
	versionCompats []*VersionCompatibility,
	ws *WorkspaceModel,
) *ValidationReport {
	var findings []*ValidationFinding

	// 1. Cross-File Relationship Validation
	knownFiles := make(map[string]bool)
	for _, rel := range fileRels {
		knownFiles[rel.SourceFile()] = true
		knownFiles[rel.TargetFile()] = true
	}

	for _, dep := range crossFileDeps {
		if dep.TargetFile() == "" || dep.TargetFile() == "." {
			findings = append(findings, NewValidationFinding(
				"val:filedep:missing_target:"+dep.SourceFile(),
				FindingMissingCrossTarget,
				SeverityError,
				BoundaryFile,
				dep.SourceFile(),
				dep.TargetFile(),
				dep.SourceFile(),
				1,
				"Cross-file dependency target is missing or unresolvable",
				dep.ID(),
			))
		}
	}

	// 2. Symbol Propagation Validation
	for _, prop := range symbolProps {
		if prop.ExportingPackage() != "" && len(prop.ConsumingPackages()) > 0 && len(prop.ReferencingFiles()) == 0 {
			findings = append(findings, NewValidationFinding(
				"val:prop:no_ref_files:"+prop.SymbolID(),
				FindingUnresolvedCrossReference,
				SeverityWarning,
				BoundaryPackage,
				prop.ExportingPackage(),
				strings.Join(prop.ConsumingPackages(), ","),
				prop.DeclaringFile(),
				1,
				"Exported symbol is listed with consuming packages but has no recorded referencing files",
				prop.SymbolName(),
			))
		}
	}

	// 3. Shared Configuration Validation
	for _, cfg := range sharedConfigs {
		if len(cfg.AffectedFiles()) == 0 && len(cfg.AffectedPackages()) == 0 && len(cfg.AffectedRepositories()) == 0 {
			findings = append(findings, NewValidationFinding(
				"val:cfg:unreferenced:"+cfg.ConfigPath(),
				FindingConflictingCrossConfig,
				SeverityWarning,
				BoundaryFile,
				cfg.ConfigPath(),
				"",
				cfg.ConfigPath(),
				1,
				"Shared configuration file is defined but affects zero engineering entities",
				cfg.ConfigFormat(),
			))
		}
	}

	// 4. Version Compatibility Validation
	for _, vc := range versionCompats {
		if vc.State() == CompatIncompatible {
			findings = append(findings, NewValidationFinding(
				"val:compat:incompatible:"+vc.ModulePath(),
				FindingInvalidCrossDependency,
				SeverityError,
				BoundaryModule,
				vc.ModulePath(),
				vc.ResolvedVersion(),
				vc.ModulePath(),
				1,
				"Module version constraint is incompatible: required "+vc.RequiredVersion()+" but resolved "+vc.ResolvedVersion(),
				vc.Details(),
			))
		}
	}

	// 5. Workspace Inter-Repository Validation
	if ws != nil {
		knownRepos := make(map[string]bool)
		for _, repo := range ws.Repositories() {
			knownRepos[repo.ID()] = true
		}
		for _, rel := range ws.Relationships() {
			if !knownRepos[rel.TargetRepoID()] {
				findings = append(findings, NewValidationFinding(
					"val:wsrel:missing_repo:"+rel.TargetRepoID(),
					FindingMissingCrossTarget,
					SeverityError,
					BoundaryWorkspace,
					rel.SourceRepoID(),
					rel.TargetRepoID(),
					ws.WorkspaceRoot(),
					1,
					"Workspace relationship references repository that does not exist in workspace",
					rel.Evidence(),
				))
			}
		}
	}

	return NewValidationReport(findings, time.Now().UTC())
}
