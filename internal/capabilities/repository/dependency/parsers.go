package dependency

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
)

func parseGoMod(absPath, relPath, moduleDir string, diagnostics *[]*discovery.Diagnostic) []*Dependency {
	f, err := os.Open(absPath)
	if err != nil {
		if diagnostics != nil {
			*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
				discovery.SeverityWarning,
				"GOMOD_OPEN_ERROR",
				err.Error(),
				relPath,
				false,
			))
		}
		return nil
	}
	defer f.Close()

	var deps []*Dependency
	var inRequireBlock bool
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if line == "require (" {
			inRequireBlock = true
			continue
		}
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		if inRequireBlock || strings.HasPrefix(line, "require ") {
			cleanLine := strings.TrimPrefix(line, "require ")
			cleanLine = strings.TrimSpace(cleanLine)

			isIndirect := strings.Contains(cleanLine, "// indirect")
			cleanLine = strings.TrimSpace(strings.Split(cleanLine, "//")[0])

			parts := strings.Fields(cleanLine)
			if len(parts) >= 2 {
				name := parts[0]
				ver := parts[1]

				deps = append(deps, NewDependency(
					name,
					ver,
					EcosystemGo,
					DependencyExternal,
					!isIndirect,
					isIndirect,
					false,
					true,
					relPath,
					moduleDir,
					NewLicenseInfo(LicenseUnknown, "", "go.mod", false),
					NewHealthInfo(HealthActive, false, false, false, true, 1.0, nil),
				))
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil && diagnostics != nil {
		*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
			discovery.SeverityWarning,
			"GOMOD_SCAN_ERROR",
			scanErr.Error(),
			relPath,
			false,
		))
	}

	return deps
}

func parsePackageJSON(absPath, relPath, moduleDir string, diagnostics *[]*discovery.Diagnostic) []*Dependency {
	bytes, err := os.ReadFile(absPath)
	if err != nil {
		if diagnostics != nil {
			*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
				discovery.SeverityWarning,
				"PACKAGE_JSON_READ_ERROR",
				err.Error(),
				relPath,
				false,
			))
		}
		return nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		if diagnostics != nil {
			*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
				discovery.SeverityWarning,
				"PACKAGE_JSON_PARSE_ERROR",
				err.Error(),
				relPath,
				false,
			))
		}
		return nil
	}

	var deps []*Dependency
	licStr, _ := raw["license"].(string)
	license := classifyLicense(licStr)

	extract := func(key string, isDev bool) {
		if section, ok := raw[key].(map[string]interface{}); ok {
			for name, v := range section {
				verStr, _ := v.(string)
				depType := DependencyRuntime
				if isDev {
					depType = DependencyDevelopment
				}

				deps = append(deps, NewDependency(
					name,
					verStr,
					EcosystemNpm,
					depType,
					true,
					false,
					false,
					true,
					relPath,
					moduleDir,
					license,
					NewHealthInfo(HealthActive, false, false, false, true, 1.0, nil),
				))
			}
		}
	}

	extract("dependencies", false)
	extract("devDependencies", true)
	extract("peerDependencies", false)

	return deps
}

func parseCargoToml(absPath, relPath, moduleDir string, diagnostics *[]*discovery.Diagnostic) []*Dependency {
	f, err := os.Open(absPath)
	if err != nil {
		if diagnostics != nil {
			*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
				discovery.SeverityWarning,
				"CARGO_TOML_OPEN_ERROR",
				err.Error(),
				relPath,
				false,
			))
		}
		return nil
	}
	defer f.Close()

	var deps []*Dependency
	var currentSection string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.Trim(line, "[] ")
			continue
		}

		isDepSection := currentSection == "dependencies" ||
			currentSection == "dev-dependencies" ||
			currentSection == "build-dependencies" ||
			strings.HasPrefix(currentSection, "dependencies.")

		if isDepSection {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				rawVal := strings.TrimSpace(parts[1])

				var ver string
				if strings.HasPrefix(rawVal, "\"") {
					ver = strings.Trim(rawVal, "\"")
				} else if strings.HasPrefix(rawVal, "{") {
					// Inline table e.g. { version = "1.0", features = [...] }
					if idx := strings.Index(rawVal, "version"); idx != -1 {
						sub := rawVal[idx:]
						if vParts := strings.SplitN(sub, "=", 2); len(vParts) == 2 {
							vRight := strings.TrimSpace(vParts[1])
							vRight = strings.TrimLeft(vRight, "\"")
							if endIdx := strings.Index(vRight, "\""); endIdx != -1 {
								ver = vRight[:endIdx]
							}
						}
					}
				}

				if name != "" {
					depType := DependencyRuntime
					if currentSection == "dev-dependencies" {
						depType = DependencyDevelopment
					} else if currentSection == "build-dependencies" {
						depType = DependencyBuild
					}

					deps = append(deps, NewDependency(
						name,
						ver,
						EcosystemCargo,
						depType,
						true,
						false,
						false,
						true,
						relPath,
						moduleDir,
						NewLicenseInfo(LicenseUnknown, "", "Cargo.toml", false),
						NewHealthInfo(HealthActive, false, false, false, true, 1.0, nil),
					))
				}
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil && diagnostics != nil {
		*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
			discovery.SeverityWarning,
			"CARGO_TOML_SCAN_ERROR",
			scanErr.Error(),
			relPath,
			false,
		))
	}

	return deps
}

func parsePomXML(absPath, relPath, moduleDir string, diagnostics *[]*discovery.Diagnostic) []*Dependency {
	f, err := os.Open(absPath)
	if err != nil {
		if diagnostics != nil {
			*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
				discovery.SeverityWarning,
				"POM_XML_OPEN_ERROR",
				err.Error(),
				relPath,
				false,
			))
		}
		return nil
	}
	defer f.Close()

	var deps []*Dependency
	var inDep bool
	var groupID, artifactID, version string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "<dependency>") {
			inDep = true
			groupID, artifactID, version = "", "", ""
			continue
		}
		if strings.Contains(line, "</dependency>") {
			if inDep && (groupID != "" || artifactID != "") {
				name := artifactID
				if groupID != "" {
					name = groupID + ":" + artifactID
				}
				deps = append(deps, NewDependency(
					name,
					version,
					EcosystemMaven,
					DependencyRuntime,
					true,
					false,
					false,
					true,
					relPath,
					moduleDir,
					NewLicenseInfo(LicenseUnknown, "", "pom.xml", false),
					NewHealthInfo(HealthActive, false, false, false, true, 1.0, nil),
				))
			}
			inDep = false
			continue
		}

		if inDep {
			if strings.Contains(line, "<groupId>") {
				groupID = extractXMLTag(line, "groupId")
			}
			if strings.Contains(line, "<artifactId>") {
				artifactID = extractXMLTag(line, "artifactId")
			}
			if strings.Contains(line, "<version>") {
				version = extractXMLTag(line, "version")
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil && diagnostics != nil {
		*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
			discovery.SeverityWarning,
			"POM_XML_SCAN_ERROR",
			scanErr.Error(),
			relPath,
			false,
		))
	}

	return deps
}

func parseGradle(absPath, relPath, moduleDir string, diagnostics *[]*discovery.Diagnostic) []*Dependency {
	f, err := os.Open(absPath)
	if err != nil {
		if diagnostics != nil {
			*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
				discovery.SeverityWarning,
				"GRADLE_OPEN_ERROR",
				err.Error(),
				relPath,
				false,
			))
		}
		return nil
	}
	defer f.Close()

	var deps []*Dependency
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}

		prefixes := []string{"implementation", "api", "testImplementation", "compileOnly", "runtimeOnly"}
		for _, prefix := range prefixes {
			if strings.HasPrefix(line, prefix) {
				sub := strings.TrimPrefix(line, prefix)
				sub = strings.Trim(sub, " ('\")")
				parts := strings.Split(sub, ":")
				if len(parts) >= 2 {
					name := parts[0] + ":" + parts[1]
					var ver string
					if len(parts) >= 3 {
						ver = parts[2]
					}
					deps = append(deps, NewDependency(
						name,
						ver,
						EcosystemGradle,
						DependencyRuntime,
						true,
						false,
						false,
						true,
						relPath,
						moduleDir,
						NewLicenseInfo(LicenseUnknown, "", filepath.Base(relPath), false),
						NewHealthInfo(HealthActive, false, false, false, true, 1.0, nil),
					))
				}
				break
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil && diagnostics != nil {
		*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
			discovery.SeverityWarning,
			"GRADLE_SCAN_ERROR",
			scanErr.Error(),
			relPath,
			false,
		))
	}

	return deps
}

func parseRequirementsTxt(absPath, relPath, moduleDir string, diagnostics *[]*discovery.Diagnostic) []*Dependency {
	f, err := os.Open(absPath)
	if err != nil {
		if diagnostics != nil {
			*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
				discovery.SeverityWarning,
				"REQUIREMENTS_TXT_OPEN_ERROR",
				err.Error(),
				relPath,
				false,
			))
		}
		return nil
	}
	defer f.Close()

	var deps []*Dependency
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		// Handle operators: ==, >=, <=, ~=, !=, >
		operators := []string{"==", ">=", "<=", "~=", "!=", ">", "<"}
		var name, ver string
		found := false

		for _, op := range operators {
			if idx := strings.Index(line, op); idx != -1 {
				name = strings.TrimSpace(line[:idx])
				ver = strings.TrimSpace(line[idx+len(op):])
				found = true
				break
			}
		}

		if !found {
			name = line
		}

		if name != "" {
			deps = append(deps, NewDependency(
				name,
				ver,
				EcosystemPython,
				DependencyRuntime,
				true,
				false,
				false,
				true,
				relPath,
				moduleDir,
				NewLicenseInfo(LicenseUnknown, "", "requirements.txt", false),
				NewHealthInfo(HealthActive, false, false, false, true, 1.0, nil),
			))
		}
	}

	if scanErr := scanner.Err(); scanErr != nil && diagnostics != nil {
		*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
			discovery.SeverityWarning,
			"REQUIREMENTS_TXT_SCAN_ERROR",
			scanErr.Error(),
			relPath,
			false,
		))
	}

	return deps
}

func parseComposerJSON(absPath, relPath, moduleDir string, diagnostics *[]*discovery.Diagnostic) []*Dependency {
	bytes, err := os.ReadFile(absPath)
	if err != nil {
		if diagnostics != nil {
			*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
				discovery.SeverityWarning,
				"COMPOSER_JSON_READ_ERROR",
				err.Error(),
				relPath,
				false,
			))
		}
		return nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		if diagnostics != nil {
			*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
				discovery.SeverityWarning,
				"COMPOSER_JSON_PARSE_ERROR",
				err.Error(),
				relPath,
				false,
			))
		}
		return nil
	}

	var deps []*Dependency
	licStr, _ := raw["license"].(string)
	license := classifyLicense(licStr)

	extract := func(key string, isDev bool) {
		if section, ok := raw[key].(map[string]interface{}); ok {
			for name, v := range section {
				verStr, _ := v.(string)
				depType := DependencyRuntime
				if isDev {
					depType = DependencyDevelopment
				}

				deps = append(deps, NewDependency(
					name,
					verStr,
					EcosystemComposer,
					depType,
					true,
					false,
					false,
					true,
					relPath,
					moduleDir,
					license,
					NewHealthInfo(HealthActive, false, false, false, true, 1.0, nil),
				))
			}
		}
	}

	extract("require", false)
	extract("require-dev", true)

	return deps
}

func parseSourceImports(
	absPath, relPath, langID, sourcePkg string,
	diagnostics *[]*discovery.Diagnostic,
) []*InternalImport {
	f, err := os.Open(absPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var imports []*InternalImport
	scanner := bufio.NewScanner(f)

	switch langID {
	case "go":
		var inImportBlock bool
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "import (" {
				inImportBlock = true
				continue
			}
			if inImportBlock && line == ")" {
				inImportBlock = false
				continue
			}

			if inImportBlock || strings.HasPrefix(line, "import ") {
				clean := strings.TrimPrefix(line, "import ")
				clean = strings.TrimSpace(clean)
				if idx := strings.Index(clean, "\""); idx != -1 {
					clean = clean[idx+1:]
					if endIdx := strings.Index(clean, "\""); endIdx != -1 {
						target := clean[:endIdx]
						imports = append(imports, NewInternalImport(sourcePkg, target, relPath, langID))
					}
				}
			}
		}
	case "javascript", "typescript":
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "import ") || strings.Contains(line, "require(") {
				if fromIdx := strings.Index(line, "from"); fromIdx != -1 {
					afterFrom := strings.TrimSpace(line[fromIdx+4:])
					afterFrom = strings.Trim(afterFrom, " '\";")
					if afterFrom != "" {
						imports = append(imports, NewInternalImport(sourcePkg, afterFrom, relPath, langID))
					}
				} else if reqIdx := strings.Index(line, "require("); reqIdx != -1 {
					sub := line[reqIdx+8:]
					if endIdx := strings.Index(sub, ")"); endIdx != -1 {
						target := strings.Trim(sub[:endIdx], " '\";")
						if target != "" {
							imports = append(imports, NewInternalImport(sourcePkg, target, relPath, langID))
						}
					}
				}
			}
		}
	case "python":
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "import ") {
				target := strings.TrimPrefix(line, "import ")
				target = strings.TrimSpace(strings.Split(target, " as ")[0])
				if target != "" {
					imports = append(imports, NewInternalImport(sourcePkg, target, relPath, langID))
				}
			} else if strings.HasPrefix(line, "from ") {
				target := strings.TrimPrefix(line, "from ")
				target = strings.TrimSpace(strings.Split(target, " import ")[0])
				if target != "" {
					imports = append(imports, NewInternalImport(sourcePkg, target, relPath, langID))
				}
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil && diagnostics != nil {
		*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
			discovery.SeverityWarning,
			"SOURCE_IMPORT_SCAN_ERROR",
			scanErr.Error(),
			relPath,
			false,
		))
	}

	return imports
}

func extractXMLTag(line, tag string) string {
	openTag := "<" + tag + ">"
	closeTag := "</" + tag + ">"
	start := strings.Index(line, openTag)
	end := strings.Index(line, closeTag)
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(line[start+len(openTag) : end])
	}
	return ""
}

func classifyLicense(raw string) *LicenseInfo {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return NewLicenseInfo(LicenseUnavailable, "", "", false)
	}

	upper := strings.ToUpper(clean)
	var lType LicenseType

	switch {
	case strings.Contains(upper, "MIT"):
		lType = LicenseMIT
	case strings.Contains(upper, "APACHE"):
		lType = LicenseApache2
	case strings.Contains(upper, "BSD"):
		lType = LicenseBSD
	case strings.Contains(upper, "LGPL"):
		lType = LicenseLGPL
	case strings.Contains(upper, "GPL"):
		lType = LicenseGPL
	case strings.Contains(upper, "MPL"):
		lType = LicenseMPL
	default:
		lType = LicenseUnknown
	}

	return NewLicenseInfo(lType, clean, "manifest", true)
}
