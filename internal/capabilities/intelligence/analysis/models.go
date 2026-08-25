package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// Finding represents an individual, evidence-backed defect or quality observation.
type Finding struct {
	id                  string
	analyzer            string
	ruleID              RuleID
	category            FindingCategory
	severity            Severity
	confidence          Confidence
	title               string
	description         string
	repository          string
	module              string
	packagePath         string
	filePath            string
	symbolID            string
	location            *symbol.SourcePosition
	evidence            string
	supportingRelations []string
	remediationHint     string
	provenance          string
}

// GenerateFindingID constructs a deterministic finding identifier.
func GenerateFindingID(analyzer string, ruleID RuleID, target string, evidenceKey string) string {
	hasher := sha256.New()
	hasher.Write(fmt.Appendf(nil, "%s:%s:%s:%s", analyzer, ruleID, target, evidenceKey))
	hashSum := hex.EncodeToString(hasher.Sum(nil))[:12]
	cleanTarget := strings.ReplaceAll(target, "/", ".")
	cleanTarget = strings.ReplaceAll(cleanTarget, "\\", ".")
	cleanTarget = strings.ReplaceAll(cleanTarget, ":", ".")
	if len(cleanTarget) > 40 {
		cleanTarget = cleanTarget[:40]
	}
	return fmt.Sprintf("finding:%s:%s:%s:%s", analyzer, ruleID, cleanTarget, hashSum)
}

// NewFinding constructs an immutable Finding with deterministic ID generation and defensive copying.
func NewFinding(
	analyzer string,
	ruleID RuleID,
	category FindingCategory,
	severity Severity,
	confidence Confidence,
	title string,
	description string,
	repository string,
	module string,
	packagePath string,
	filePath string,
	symbolID string,
	location *symbol.SourcePosition,
	evidence string,
	supportingRelations []string,
	remediationHint string,
	provenance string,
) *Finding {
	target := symbolID
	if target == "" {
		target = filePath
	}
	if target == "" {
		target = packagePath
	}
	if target == "" {
		target = module
	}
	if target == "" {
		target = repository
	}
	id := GenerateFindingID(analyzer, ruleID, target, evidence)

	var rels []string
	if len(supportingRelations) > 0 {
		rels = make([]string, len(supportingRelations))
		copy(rels, supportingRelations)
		sort.Strings(rels)
	}

	return &Finding{
		id:                  id,
		analyzer:            strings.TrimSpace(analyzer),
		ruleID:              ruleID,
		category:            category,
		severity:            severity,
		confidence:          confidence,
		title:               strings.TrimSpace(title),
		description:         strings.TrimSpace(description),
		repository:          strings.TrimSpace(repository),
		module:              strings.TrimSpace(module),
		packagePath:         strings.TrimSpace(packagePath),
		filePath:            strings.TrimSpace(filePath),
		symbolID:            strings.TrimSpace(symbolID),
		location:            location,
		evidence:            strings.TrimSpace(evidence),
		supportingRelations: rels,
		remediationHint:     strings.TrimSpace(remediationHint),
		provenance:          strings.TrimSpace(provenance),
	}
}

// Getters for Finding
func (f *Finding) ID() string                       { return f.id }
func (f *Finding) Analyzer() string                 { return f.analyzer }
func (f *Finding) RuleID() RuleID                   { return f.ruleID }
func (f *Finding) Category() FindingCategory        { return f.category }
func (f *Finding) Severity() Severity               { return f.severity }
func (f *Finding) Confidence() Confidence           { return f.confidence }
func (f *Finding) Title() string                    { return f.title }
func (f *Finding) Description() string              { return f.description }
func (f *Finding) Repository() string               { return f.repository }
func (f *Finding) Module() string                   { return f.module }
func (f *Finding) PackagePath() string              { return f.packagePath }
func (f *Finding) FilePath() string                 { return f.filePath }
func (f *Finding) SymbolID() string                 { return f.symbolID }
func (f *Finding) Location() *symbol.SourcePosition { return f.location }
func (f *Finding) Evidence() string                 { return f.evidence }
func (f *Finding) RemediationHint() string          { return f.remediationHint }
func (f *Finding) Provenance() string               { return f.provenance }

func (f *Finding) SupportingRelations() []string {
	if f == nil || len(f.supportingRelations) == 0 {
		return nil
	}
	out := make([]string, len(f.supportingRelations))
	copy(out, f.supportingRelations)
	return out
}

// SortFindings deterministically orders findings by Severity (Critical first), Analyzer, Rule, FilePath, Line, and ID.
func SortFindings(findings []*Finding) []*Finding {
	if len(findings) == 0 {
		return nil
	}
	sorted := make([]*Finding, len(findings))
	copy(sorted, findings)

	sort.Slice(sorted, func(i, j int) bool {
		f1, f2 := sorted[i], sorted[j]
		if f1.severity.SeverityRank() != f2.severity.SeverityRank() {
			return f1.severity.SeverityRank() < f2.severity.SeverityRank()
		}
		if f1.analyzer != f2.analyzer {
			return f1.analyzer < f2.analyzer
		}
		if f1.ruleID != f2.ruleID {
			return f1.ruleID < f2.ruleID
		}
		if f1.filePath != f2.filePath {
			return f1.filePath < f2.filePath
		}
		l1, l2 := 0, 0
		if f1.location != nil {
			l1 = f1.location.Line()
		}
		if f2.location != nil {
			l2 = f2.location.Line()
		}
		if l1 != l2 {
			return l1 < l2
		}
		return f1.id < f2.id
	})
	return sorted
}

// DeduplicateFindings eliminates duplicate findings sharing the same canonical finding ID.
func DeduplicateFindings(findings []*Finding) []*Finding {
	if len(findings) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var deduped []*Finding
	for _, f := range findings {
		if f == nil {
			continue
		}
		if !seen[f.id] {
			seen[f.id] = true
			deduped = append(deduped, f)
		}
	}
	return SortFindings(deduped)
}

// AnalysisRuleResult records the evaluation outcome of an individual analysis rule.
type AnalysisRuleResult struct {
	ruleID   RuleID
	status   EvaluationStatus
	findings []*Finding
	message  string
}

// NewAnalysisRuleResult constructs an immutable AnalysisRuleResult.
func NewAnalysisRuleResult(ruleID RuleID, status EvaluationStatus, findings []*Finding, msg string) *AnalysisRuleResult {
	sorted := DeduplicateFindings(findings)
	return &AnalysisRuleResult{
		ruleID:   ruleID,
		status:   status,
		findings: sorted,
		message:  strings.TrimSpace(msg),
	}
}

func (r *AnalysisRuleResult) RuleID() RuleID           { return r.ruleID }
func (r *AnalysisRuleResult) Status() EvaluationStatus { return r.status }
func (r *AnalysisRuleResult) Message() string          { return r.message }
func (r *AnalysisRuleResult) FindingCount() int        { return len(r.findings) }

func (r *AnalysisRuleResult) Findings() []*Finding {
	if r == nil || len(r.findings) == 0 {
		return nil
	}
	out := make([]*Finding, len(r.findings))
	copy(out, r.findings)
	return out
}

// AnalyzerResult represents the aggregated output from an analyzer component.
type AnalyzerResult struct {
	analyzer    string
	ruleResults map[RuleID]*AnalysisRuleResult
	findings    []*Finding
}

// NewAnalyzerResult constructs an immutable AnalyzerResult.
func NewAnalyzerResult(analyzer string, ruleResults map[RuleID]*AnalysisRuleResult) *AnalyzerResult {
	rMap := make(map[RuleID]*AnalysisRuleResult)
	var allFindings []*Finding
	for rID, res := range ruleResults {
		if res != nil {
			rMap[rID] = res
			allFindings = append(allFindings, res.Findings()...)
		}
	}
	deduped := DeduplicateFindings(allFindings)

	return &AnalyzerResult{
		analyzer:    strings.TrimSpace(analyzer),
		ruleResults: rMap,
		findings:    deduped,
	}
}

func (ar *AnalyzerResult) Analyzer() string   { return ar.analyzer }
func (ar *AnalyzerResult) TotalFindings() int { return len(ar.findings) }

func (ar *AnalyzerResult) RuleResult(ruleID RuleID) *AnalysisRuleResult {
	if ar == nil || ar.ruleResults == nil {
		return nil
	}
	return ar.ruleResults[ruleID]
}

func (ar *AnalyzerResult) Findings() []*Finding {
	if ar == nil || len(ar.findings) == 0 {
		return nil
	}
	out := make([]*Finding, len(ar.findings))
	copy(out, ar.findings)
	return out
}

// ScoreDeduction records a transparent deduction applied to a health dimension score.
type ScoreDeduction struct {
	ruleID   RuleID
	severity Severity
	points   float64
	reason   string
}

// NewScoreDeduction constructs an immutable ScoreDeduction.
func NewScoreDeduction(ruleID RuleID, severity Severity, points float64, reason string) *ScoreDeduction {
	return &ScoreDeduction{
		ruleID:   ruleID,
		severity: severity,
		points:   points,
		reason:   strings.TrimSpace(reason),
	}
}

func (sd *ScoreDeduction) RuleID() RuleID     { return sd.ruleID }
func (sd *ScoreDeduction) Severity() Severity { return sd.severity }
func (sd *ScoreDeduction) Points() float64    { return sd.points }
func (sd *ScoreDeduction) Reason() string     { return sd.reason }

// HealthDimension represents a single scored dimension of repository health.
type HealthDimension struct {
	name       string
	score      float64
	confidence float64
	coverage   float64
	weight     float64
	deductions []*ScoreDeduction
	metrics    map[string]float64
}

// NewHealthDimension constructs an immutable HealthDimension.
func NewHealthDimension(
	name string,
	score float64,
	confidence float64,
	coverage float64,
	weight float64,
	deductions []*ScoreDeduction,
	metrics map[string]float64,
) *HealthDimension {
	// Normalize bounds
	if score < 0.0 {
		score = 0.0
	} else if score > 100.0 {
		score = 100.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	} else if confidence > 1.0 {
		confidence = 1.0
	}
	if coverage < 0.0 {
		coverage = 0.0
	} else if coverage > 1.0 {
		coverage = 1.0
	}

	dedList := make([]*ScoreDeduction, len(deductions))
	copy(dedList, deductions)

	mMap := make(map[string]float64)
	for k, v := range metrics {
		mMap[k] = v
	}

	return &HealthDimension{
		name:       strings.TrimSpace(name),
		score:      score,
		confidence: confidence,
		coverage:   coverage,
		weight:     weight,
		deductions: dedList,
		metrics:    mMap,
	}
}

func (hd *HealthDimension) Name() string        { return hd.name }
func (hd *HealthDimension) Score() float64      { return hd.score }
func (hd *HealthDimension) Confidence() float64 { return hd.confidence }
func (hd *HealthDimension) Coverage() float64   { return hd.coverage }
func (hd *HealthDimension) Weight() float64     { return hd.weight }

func (hd *HealthDimension) Deductions() []*ScoreDeduction {
	if hd == nil || len(hd.deductions) == 0 {
		return nil
	}
	out := make([]*ScoreDeduction, len(hd.deductions))
	copy(out, hd.deductions)
	return out
}

func (hd *HealthDimension) Metrics() map[string]float64 {
	if hd == nil || len(hd.metrics) == 0 {
		return nil
	}
	out := make(map[string]float64)
	for k, v := range hd.metrics {
		out[k] = v
	}
	return out
}

// RepositoryHealthReport consolidates all health dimension scores and grades into a unified report.
type RepositoryHealthReport struct {
	overallScore    float64
	grade           string
	engineering     *HealthDimension
	architecture    *HealthDimension
	documentation   *HealthDimension
	test            *HealthDimension
	maintainability *HealthDimension
	analyzedAt      time.Time
}

// CalculateGrade computes a letter grade from an overall numerical score.
func CalculateGrade(score float64) string {
	switch {
	case score >= 90.0:
		return "A"
	case score >= 80.0:
		return "B"
	case score >= 70.0:
		return "C"
	case score >= 60.0:
		return "D"
	default:
		return "F"
	}
}

// NewRepositoryHealthReport constructs an immutable RepositoryHealthReport.
func NewRepositoryHealthReport(
	engineering *HealthDimension,
	architecture *HealthDimension,
	documentation *HealthDimension,
	test *HealthDimension,
	maintainability *HealthDimension,
	analyzedAt time.Time,
) *RepositoryHealthReport {
	// Weighted combination formula:
	// Overall = 0.30*Eng + 0.25*Arch + 0.20*Maint + 0.15*Test + 0.10*Doc
	var engScore, archScore, docScore, testScore, maintScore float64
	if engineering != nil {
		engScore = engineering.Score()
	}
	if architecture != nil {
		archScore = architecture.Score()
	}
	if documentation != nil {
		docScore = documentation.Score()
	}
	if test != nil {
		testScore = test.Score()
	}
	if maintainability != nil {
		maintScore = maintainability.Score()
	}

	overall := (0.30 * engScore) + (0.25 * archScore) + (0.20 * maintScore) + (0.15 * testScore) + (0.10 * docScore)
	if overall < 0.0 {
		overall = 0.0
	} else if overall > 100.0 {
		overall = 100.0
	}

	grade := CalculateGrade(overall)
	if analyzedAt.IsZero() {
		analyzedAt = time.Now().UTC()
	}

	return &RepositoryHealthReport{
		overallScore:    overall,
		grade:           grade,
		engineering:     engineering,
		architecture:    architecture,
		documentation:   documentation,
		test:            test,
		maintainability: maintainability,
		analyzedAt:      analyzedAt,
	}
}

func (rhr *RepositoryHealthReport) OverallScore() float64           { return rhr.overallScore }
func (rhr *RepositoryHealthReport) Grade() string                   { return rhr.grade }
func (rhr *RepositoryHealthReport) Engineering() *HealthDimension   { return rhr.engineering }
func (rhr *RepositoryHealthReport) Architecture() *HealthDimension  { return rhr.architecture }
func (rhr *RepositoryHealthReport) Documentation() *HealthDimension { return rhr.documentation }
func (rhr *RepositoryHealthReport) Test() *HealthDimension          { return rhr.test }
func (rhr *RepositoryHealthReport) Maintainability() *HealthDimension {
	return rhr.maintainability
}
func (rhr *RepositoryHealthReport) AnalyzedAt() time.Time { return rhr.analyzedAt }

// AnalysisModel represents the consolidated, immutable snapshot of all engineering analysis operations.
type AnalysisModel struct {
	qualityResult       *AnalyzerResult
	dependencyResult    *AnalyzerResult
	architectureResult  *AnalyzerResult
	configurationResult *AnalyzerResult
	allFindings         []*Finding
	healthReport        *RepositoryHealthReport
}

// NewAnalysisModel constructs an immutable AnalysisModel with defensively copied and deduplicated collections.
func NewAnalysisModel(
	quality *AnalyzerResult,
	dep *AnalyzerResult,
	arch *AnalyzerResult,
	conf *AnalyzerResult,
	health *RepositoryHealthReport,
) *AnalysisModel {
	var all []*Finding
	if quality != nil {
		all = append(all, quality.Findings()...)
	}
	if dep != nil {
		all = append(all, dep.Findings()...)
	}
	if arch != nil {
		all = append(all, arch.Findings()...)
	}
	if conf != nil {
		all = append(all, conf.Findings()...)
	}
	deduped := DeduplicateFindings(all)

	return &AnalysisModel{
		qualityResult:       quality,
		dependencyResult:    dep,
		architectureResult:  arch,
		configurationResult: conf,
		allFindings:         deduped,
		healthReport:        health,
	}
}

func (m *AnalysisModel) QualityResult() *AnalyzerResult        { return m.qualityResult }
func (m *AnalysisModel) DependencyResult() *AnalyzerResult     { return m.dependencyResult }
func (m *AnalysisModel) ArchitectureResult() *AnalyzerResult   { return m.architectureResult }
func (m *AnalysisModel) ConfigurationResult() *AnalyzerResult  { return m.configurationResult }
func (m *AnalysisModel) HealthReport() *RepositoryHealthReport { return m.healthReport }
func (m *AnalysisModel) TotalFindings() int                    { return len(m.allFindings) }

func (m *AnalysisModel) AllFindings() []*Finding {
	if m == nil || len(m.allFindings) == 0 {
		return nil
	}
	out := make([]*Finding, len(m.allFindings))
	copy(out, m.allFindings)
	return out
}

func (m *AnalysisModel) FindingsBySeverity(sev Severity) []*Finding {
	if m == nil || len(m.allFindings) == 0 {
		return nil
	}
	var filtered []*Finding
	for _, f := range m.allFindings {
		if f.Severity() == sev {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func (m *AnalysisModel) FindingsByRule(ruleID RuleID) []*Finding {
	if m == nil || len(m.allFindings) == 0 {
		return nil
	}
	var filtered []*Finding
	for _, f := range m.allFindings {
		if f.RuleID() == ruleID {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
