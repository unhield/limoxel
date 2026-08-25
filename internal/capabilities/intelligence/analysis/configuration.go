package analysis

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/language"
)

// RawConfigEntry represents an in-memory or discovered key-value configuration item.
type RawConfigEntry struct {
	FilePath string
	Key      string
	Value    string
	Line     int
}

// ConfigurationAnalyzer executes Task 4: Configuration Analysis.
type ConfigurationAnalyzer struct {
	langModel *language.StructureModel
	entries   []*RawConfigEntry
}

// NewConfigurationAnalyzer constructs a ConfigurationAnalyzer.
func NewConfigurationAnalyzer(langModel *language.StructureModel, entries []*RawConfigEntry) *ConfigurationAnalyzer {
	eList := make([]*RawConfigEntry, len(entries))
	copy(eList, entries)
	return &ConfigurationAnalyzer{
		langModel: langModel,
		entries:   eList,
	}
}

// Analyze executes all configuration rules and returns an AnalyzerResult.
func (a *ConfigurationAnalyzer) Analyze() *AnalyzerResult {
	ruleResults := make(map[RuleID]*AnalysisRuleResult)

	ruleResults[RuleInvalidConfiguration] = a.analyzeInvalidConfiguration()
	ruleResults[RuleDuplicateConfiguration] = a.analyzeDuplicateConfiguration()
	ruleResults[RuleMissingConfiguration] = a.analyzeMissingConfiguration()
	ruleResults[RuleDeprecatedConfiguration] = a.analyzeDeprecatedConfiguration()
	ruleResults[RuleConfigurationConflicts] = a.analyzeConfigurationConflicts()

	return NewAnalyzerResult("configuration", ruleResults)
}

// RedactSecret ensures secret strings (passwords, tokens, keys) are never exposed in finding text.
var secretKeyPattern = regexp.MustCompile(`(?i)(password|secret|token|api_key|apikey|auth|private_key)`)

func RedactSecret(key, value string) string {
	if secretKeyPattern.MatchString(key) {
		return "[REDACTED]"
	}
	return value
}

// 11.1 Invalid Configuration
func (a *ConfigurationAnalyzer) analyzeInvalidConfiguration() *AnalysisRuleResult {
	var findings []*Finding

	for _, entry := range a.entries {
		if entry == nil {
			continue
		}
		cleanVal := strings.TrimSpace(entry.Value)
		// Check invalid empty values for mandatory keys
		if cleanVal == "" && (strings.EqualFold(entry.Key, "host") || strings.EqualFold(entry.Key, "database")) {
			finding := NewFinding(
				"configuration",
				RuleInvalidConfiguration,
				CategoryConfiguration,
				SeverityHigh,
				ConfidenceDefinite,
				fmt.Sprintf("Invalid empty configuration: %s", entry.Key),
				fmt.Sprintf("Mandatory configuration parameter '%s' in file %s is empty.", entry.Key, entry.FilePath),
				"",
				"",
				"",
				entry.FilePath,
				entry.Key,
				nil,
				fmt.Sprintf("empty value for mandatory key %s", entry.Key),
				nil,
				"Provide a valid non-empty configuration value.",
				"config_analyzer",
			)
			findings = append(findings, finding)
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleInvalidConfiguration, status, findings, fmt.Sprintf("invalid configuration analysis evaluated %d findings", len(findings)))
}

// 11.2 Duplicate Configuration
func (a *ConfigurationAnalyzer) analyzeDuplicateConfiguration() *AnalysisRuleResult {
	var findings []*Finding

	fileKeys := make(map[string]map[string]int)
	for _, entry := range a.entries {
		if entry == nil {
			continue
		}
		if fileKeys[entry.FilePath] == nil {
			fileKeys[entry.FilePath] = make(map[string]int)
		}
		fileKeys[entry.FilePath][entry.Key]++
	}

	var sortedFiles []string
	for file := range fileKeys {
		sortedFiles = append(sortedFiles, file)
	}
	sort.Strings(sortedFiles)

	for _, file := range sortedFiles {
		keys := fileKeys[file]
		var sortedKeys []string
		for key := range keys {
			sortedKeys = append(sortedKeys, key)
		}
		sort.Strings(sortedKeys)

		for _, key := range sortedKeys {
			count := keys[key]
			if count > 1 {
				finding := NewFinding(
					"configuration",
					RuleDuplicateConfiguration,
					CategoryConfiguration,
					SeverityMedium,
					ConfidenceDefinite,
					fmt.Sprintf("Duplicate configuration key: %s in %s", key, file),
					fmt.Sprintf("Configuration key '%s' is defined %d times in file %s.", key, count, file),
					"",
					"",
					"",
					file,
					key,
					nil,
					fmt.Sprintf("duplicate key definition count %d", count),
					nil,
					"Remove duplicate configuration key declaration.",
					"config_analyzer",
				)
				findings = append(findings, finding)
			}
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleDuplicateConfiguration, status, findings, fmt.Sprintf("duplicate configuration analysis evaluated %d findings", len(findings)))
}

// 11.3 Missing Configuration
func (a *ConfigurationAnalyzer) analyzeMissingConfiguration() *AnalysisRuleResult {
	var findings []*Finding

	if a.langModel != nil && a.langModel.BuildGraph() != nil {
		if a.langModel.BuildGraph().Count() == 0 {
			finding := NewFinding(
				"configuration",
				RuleMissingConfiguration,
				CategoryConfiguration,
				SeverityMedium,
				ConfidenceLikely,
				"Missing build configuration",
				"No build or module configuration files (such as go.mod, package.json, or Makefile) were discovered in the repository.",
				"",
				"",
				"",
				"",
				"",
				nil,
				"build graph config count == 0",
				nil,
				"Initialize standard module or build configuration in repository root.",
				"language_model",
			)
			findings = append(findings, finding)
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleMissingConfiguration, status, findings, fmt.Sprintf("missing configuration analysis evaluated %d findings", len(findings)))
}

// 11.4 Deprecated Configuration
func (a *ConfigurationAnalyzer) analyzeDeprecatedConfiguration() *AnalysisRuleResult {
	var findings []*Finding

	deprecatedKeys := map[string]string{
		"legacy_mode":      "use feature_flags instead",
		"enable_v1":        "v1 endpoints are deprecated",
		"deprecated_token": "migrate to oauth2 credentials",
	}

	for _, entry := range a.entries {
		if entry == nil {
			continue
		}
		if hint, deprecated := deprecatedKeys[strings.ToLower(entry.Key)]; deprecated {
			finding := NewFinding(
				"configuration",
				RuleDeprecatedConfiguration,
				CategoryConfiguration,
				SeverityLow,
				ConfidenceDefinite,
				fmt.Sprintf("Deprecated configuration key: %s", entry.Key),
				fmt.Sprintf("Configuration parameter '%s' in file %s is deprecated.", entry.Key, entry.FilePath),
				"",
				"",
				"",
				entry.FilePath,
				entry.Key,
				nil,
				fmt.Sprintf("deprecated key matching %s", entry.Key),
				nil,
				fmt.Sprintf("Migrate from deprecated setting: %s.", hint),
				"config_analyzer",
			)
			findings = append(findings, finding)
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleDeprecatedConfiguration, status, findings, fmt.Sprintf("deprecated configuration analysis evaluated %d findings", len(findings)))
}

// 11.5 Configuration Conflicts
func (a *ConfigurationAnalyzer) analyzeConfigurationConflicts() *AnalysisRuleResult {
	var findings []*Finding

	// Identify conflicting values for the same key across files in the same scope
	keyValues := make(map[string]map[string]string)
	for _, entry := range a.entries {
		if entry == nil {
			continue
		}
		if keyValues[entry.Key] == nil {
			keyValues[entry.Key] = make(map[string]string)
		}
		keyValues[entry.Key][entry.FilePath] = entry.Value
	}

	var sortedKeys []string
	for k := range keyValues {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		fileVals := keyValues[key]
		if len(fileVals) > 1 {
			// Compare values
			var distinctVals []string
			valSeen := make(map[string]bool)
			for _, val := range fileVals {
				if !valSeen[val] {
					valSeen[val] = true
					distinctVals = append(distinctVals, val)
				}
			}
			if len(distinctVals) > 1 {
				sort.Strings(distinctVals)
				// Conflict across multiple config files
				var files []string
				for f := range fileVals {
					files = append(files, f)
				}
				sort.Strings(files)

				redactedVal1 := RedactSecret(key, distinctVals[0])
				redactedVal2 := RedactSecret(key, distinctVals[1])

				finding := NewFinding(
					"configuration",
					RuleConfigurationConflicts,
					CategoryConfiguration,
					SeverityMedium,
					ConfidenceLikely,
					fmt.Sprintf("Configuration conflict: %s", key),
					fmt.Sprintf("Conflicting values for key '%s' discovered across files %s (%s vs %s).", key, strings.Join(files, ", "), redactedVal1, redactedVal2),
					"",
					"",
					"",
					files[0],
					key,
					nil,
					fmt.Sprintf("conflicting values across %d files for key %s", len(files), key),
					files,
					"Align configuration settings across overlapping files to prevent runtime ambiguity.",
					"config_analyzer",
				)
				findings = append(findings, finding)
			}
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleConfigurationConflicts, status, findings, fmt.Sprintf("configuration conflicts analysis evaluated %d findings", len(findings)))
}
