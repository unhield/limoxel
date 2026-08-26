package cli

import (
	"fmt"
	"strconv"
	"strings"
)

// Flags holds parsed CLI options and positional arguments.
type Flags struct {
	values map[string]string
	bools  map[string]bool
	args   []string
}

// NewFlags creates an empty Flags instance.
func NewFlags() *Flags {
	return &Flags{
		values: make(map[string]string),
		bools:  make(map[string]bool),
		args:   make([]string, 0),
	}
}

// ParseFlags parses standard POSIX-style command arguments into a Flags structure.
func ParseFlags(rawArgs []string) (*Flags, error) {
	flags := NewFlags()
	for i := 0; i < len(rawArgs); i++ {
		arg := rawArgs[i]

		// End of flags delimiter
		if arg == "--" {
			flags.args = append(flags.args, rawArgs[i+1:]...)
			break
		}

		if strings.HasPrefix(arg, "--") {
			flagBody := arg[2:]
			if flagBody == "" {
				continue
			}

			if idx := strings.IndexByte(flagBody, '='); idx != -1 {
				name := flagBody[:idx]
				val := flagBody[idx+1:]
				flags.values[strings.ToLower(name)] = val
			} else {
				name := strings.ToLower(flagBody)
				// Check if next arg is value or another flag
				if i+1 < len(rawArgs) && !strings.HasPrefix(rawArgs[i+1], "-") {
					if isKnownBoolFlag(name) {
						flags.bools[name] = true
					} else {
						flags.values[name] = rawArgs[i+1]
						i++
					}
				} else {
					flags.bools[name] = true
				}
			}
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			shortFlag := arg[1:]
			if idx := strings.IndexByte(shortFlag, '='); idx != -1 {
				name := shortFlag[:idx]
				val := shortFlag[idx+1:]
				flags.values[strings.ToLower(name)] = val
			} else {
				name := strings.ToLower(shortFlag)
				canonical := expandShortFlag(name)
				if i+1 < len(rawArgs) && !strings.HasPrefix(rawArgs[i+1], "-") && !isKnownBoolFlag(canonical) {
					flags.values[canonical] = rawArgs[i+1]
					i++
				} else {
					flags.bools[canonical] = true
				}
			}
		} else {
			flags.args = append(flags.args, arg)
		}
	}

	return flags, nil
}

func isKnownBoolFlag(name string) bool {
	switch strings.ToLower(name) {
	case "help", "h", "verbose", "v", "debug", "trace", "interactive", "i", "all", "a", "json", "yaml", "direct", "strict", "overwrite", "redact", "force":
		return true
	default:
		return false
	}
}

func expandShortFlag(name string) string {
	switch name {
	case "h":
		return "help"
	case "v":
		return "verbose"
	case "i":
		return "interactive"
	case "c":
		return "config"
	case "f":
		return "format"
	case "o":
		return "output"
	case "r":
		return "repo"
	case "d":
		return "depth"
	case "l":
		return "limit"
	case "k":
		return "kind"
	case "p":
		return "package"
	case "t":
		return "target"
	default:
		return name
	}
}

// String returns the string value of a flag, or defaultVal if absent.
func (f *Flags) String(name string, defaultVal string) string {
	if f == nil {
		return defaultVal
	}
	clean := strings.ToLower(strings.TrimSpace(name))
	if val, ok := f.values[clean]; ok {
		return val
	}
	return defaultVal
}

// Bool returns the boolean value of a flag.
func (f *Flags) Bool(name string) bool {
	if f == nil {
		return false
	}
	clean := strings.ToLower(strings.TrimSpace(name))
	if val, ok := f.bools[clean]; ok {
		return val
	}
	if strVal, ok := f.values[clean]; ok {
		b, err := strconv.ParseBool(strVal)
		return err == nil && b
	}
	return false
}

// Int returns the integer value of a flag, or defaultVal if absent or invalid.
func (f *Flags) Int(name string, defaultVal int) int {
	if f == nil {
		return defaultVal
	}
	clean := strings.ToLower(strings.TrimSpace(name))
	if strVal, ok := f.values[clean]; ok {
		i, err := strconv.Atoi(strVal)
		if err == nil {
			return i
		}
	}
	return defaultVal
}

// Args returns positional arguments.
func (f *Flags) Args() []string {
	if f == nil {
		return nil
	}
	res := make([]string, len(f.args))
	copy(res, f.args)
	return res
}

// Arg returns the i-th positional argument, or empty string if out of range.
func (f *Flags) Arg(i int) string {
	if f == nil || i < 0 || i >= len(f.args) {
		return ""
	}
	return f.args[i]
}

// NArg returns the number of positional arguments.
func (f *Flags) NArg() int {
	if f == nil {
		return 0
	}
	return len(f.args)
}

// Format returns the requested OutputFormat based on --format, --json, or format flags.
func (f *Flags) Format() OutputFormat {
	if f == nil {
		return FormatText
	}
	if f.Bool("json") {
		return FormatJSON
	}
	if f.Bool("yaml") {
		return FormatYAML
	}
	fmtStr := strings.ToLower(f.String("format", "text"))
	switch fmtStr {
	case "json":
		return FormatJSON
	case "yaml", "yml":
		return FormatYAML
	case "toml":
		return FormatTOML
	case "xml":
		return FormatXML
	case "csv":
		return FormatCSV
	case "markdown", "md":
		return FormatMarkdown
	case "html", "htm":
		return FormatHTML
	case "pdf":
		return FormatPDF
	case "mermaid", "mmd":
		return FormatMermaid
	case "graphviz", "dot", "gv":
		return FormatGraphviz
	case "svg":
		return FormatSVG
	case "png":
		return FormatPNG
	case "interactive", "web":
		return FormatInteractive
	default:
		return FormatText
	}
}

// OutputFile returns the destination file path specified by --output or -o.
func (f *Flags) OutputFile() string {
	if f == nil {
		return ""
	}
	return f.String("output", "")
}

// RepoRoot returns the specified repository path or default "." if unspecified.
func (f *Flags) RepoRoot() string {
	if f == nil {
		return "."
	}
	return f.String("repo", ".")
}

// ConfigPath returns the specified configuration file path or empty string if unspecified.
func (f *Flags) ConfigPath() string {
	if f == nil {
		return ""
	}
	return f.String("config", "")
}

// Profile returns the specified active profile name or empty string if unspecified.
func (f *Flags) Profile() string {
	if f == nil {
		return ""
	}
	return f.String("profile", "")
}

// LogLevel returns the specified log level flag or empty string.
func (f *Flags) LogLevel() string {
	if f == nil {
		return ""
	}
	return f.String("log-level", "")
}

// LogFormat returns the specified log format flag or empty string.
func (f *Flags) LogFormat() string {
	if f == nil {
		return ""
	}
	return f.String("log-format", "")
}

// LogFile returns the specified log file path or empty string.
func (f *Flags) LogFile() string {
	if f == nil {
		return ""
	}
	return f.String("log-file", "")
}

// IsDebug returns true if --debug flag is set.
func (f *Flags) IsDebug() bool {
	if f == nil {
		return false
	}
	return f.Bool("debug")
}

// IsTrace returns true if --trace flag is set.
func (f *Flags) IsTrace() bool {
	if f == nil {
		return false
	}
	return f.Bool("trace")
}

// ProfileCPU returns the path specified for --profile-cpu or empty string.
func (f *Flags) ProfileCPU() string {
	if f == nil {
		return ""
	}
	return f.String("profile-cpu", "")
}

// ProfileMem returns the path specified for --profile-mem or empty string.
func (f *Flags) ProfileMem() string {
	if f == nil {
		return ""
	}
	return f.String("profile-mem", "")
}

// StringMap returns all key-value flags.
func (f *Flags) StringMap() map[string]string {
	if f == nil {
		return nil
	}
	res := make(map[string]string, len(f.values))
	for k, v := range f.values {
		res[k] = v
	}
	return res
}

// Set explicitly sets a flag value.
func (f *Flags) Set(name, val string) {
	if f == nil {
		return
	}
	f.values[strings.ToLower(name)] = val
}

// SetBool explicitly sets a boolean flag.
func (f *Flags) SetBool(name string, val bool) {
	if f == nil {
		return
	}
	f.bools[strings.ToLower(name)] = val
}

// DebugString returns formatted debug representation.
func (f *Flags) DebugString() string {
	if f == nil {
		return "Flags(nil)"
	}
	return fmt.Sprintf("Flags(args=%v, values=%v, bools=%v)", f.args, f.values, f.bools)
}
