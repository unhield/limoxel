package reporting

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// StructuredExporter coordinates deterministic export across JSON, YAML, TOML, XML, and CSV.
type StructuredExporter struct{}

// NewStructuredExporter creates a new StructuredExporter instance.
func NewStructuredExporter() *StructuredExporter {
	return &StructuredExporter{}
}

// Export serializes data into the requested format and writes it to w.
func (e *StructuredExporter) Export(format Format, data any, w io.Writer) error {
	if w == nil {
		return fmt.Errorf("reporting: output writer is nil")
	}
	if data == nil {
		return fmt.Errorf("reporting: export data payload is nil")
	}

	switch format {
	case FormatJSON:
		return e.ExportJSON(data, w)
	case FormatYAML:
		return e.ExportYAML(data, w)
	case FormatTOML:
		return e.ExportTOML(data, w)
	case FormatXML:
		return e.ExportXML(data, w)
	case FormatCSV:
		return e.ExportCSV(data, w)
	default:
		return fmt.Errorf("reporting: unsupported structured format %q", format)
	}
}

// ExportJSON writes deterministic, indented JSON to w.
func (e *StructuredExporter) ExportJSON(data any, w io.Writer) error {
	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("reporting: JSON serialization failed: %w", err)
	}
	_, err = fmt.Fprintln(w, string(buf))
	return err
}

// ExportXML writes XML with a standard XML header to w.
func (e *StructuredExporter) ExportXML(data any, w io.Writer) error {
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("reporting: XML serialization failed: %w", err)
	}
	_, err := fmt.Fprintln(w)
	return err
}

// ExportYAML writes deterministic YAML 1.2 representation to w.
func (e *StructuredExporter) ExportYAML(data any, w io.Writer) error {
	var buf bytes.Buffer
	if err := writeYAMLValue(&buf, reflect.ValueOf(data), 0, false); err != nil {
		return fmt.Errorf("reporting: YAML serialization failed: %w", err)
	}
	_, err := w.Write(buf.Bytes())
	return err
}

func writeYAMLValue(buf *bytes.Buffer, v reflect.Value, indent int, inList bool) error {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			buf.WriteString("null\n")
			return nil
		}
		v = v.Elem()
	}

	indentStr := strings.Repeat("  ", indent)

	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			tag := field.Tag.Get("yaml")
			if tag == "-" {
				continue
			}
			fieldName := tag
			if idx := strings.Index(fieldName, ","); idx != -1 {
				fieldName = fieldName[:idx]
			}
			if fieldName == "" {
				fieldName = strings.ToLower(field.Name)
			}

			fieldVal := v.Field(i)
			if i > 0 || !inList {
				buf.WriteString(indentStr)
			}
			buf.WriteString(fieldName)
			buf.WriteString(":")

			// Check if struct or slice to indent next line
			underlying := fieldVal
			for underlying.Kind() == reflect.Pointer || underlying.Kind() == reflect.Interface {
				if !underlying.IsNil() {
					underlying = underlying.Elem()
				} else {
					break
				}
			}

			if underlying.Kind() == reflect.Struct || underlying.Kind() == reflect.Map || underlying.Kind() == reflect.Slice {
				if underlying.Kind() == reflect.Slice && underlying.Len() == 0 {
					buf.WriteString(" []\n")
				} else if (underlying.Kind() == reflect.Struct || underlying.Kind() == reflect.Map) && underlying.IsZero() {
					buf.WriteString(" {}\n")
				} else {
					buf.WriteString("\n")
					if err := writeYAMLValue(buf, fieldVal, indent+1, false); err != nil {
						return err
					}
				}
			} else {
				buf.WriteString(" ")
				if err := writeYAMLValue(buf, fieldVal, indent+1, false); err != nil {
					return err
				}
			}
		}
		return nil

	case reflect.Map:
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool {
			return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
		})
		for i, k := range keys {
			val := v.MapIndex(k)
			if i > 0 || !inList {
				buf.WriteString(indentStr)
			}
			fmt.Fprintf(buf, "%s: ", k.Interface())
			if err := writeYAMLValue(buf, val, indent+1, false); err != nil {
				return err
			}
		}
		return nil

	case reflect.Slice, reflect.Array:
		if v.Len() == 0 {
			buf.WriteString("[]\n")
			return nil
		}
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			buf.WriteString(indentStr)
			buf.WriteString("- ")
			if err := writeYAMLValue(buf, elem, indent+1, true); err != nil {
				return err
			}
		}
		return nil

	case reflect.String:
		s := v.String()
		if strings.Contains(s, "\n") {
			buf.WriteString("|\n")
			lines := strings.Split(s, "\n")
			for _, l := range lines {
				buf.WriteString(indentStr)
				buf.WriteString("  ")
				buf.WriteString(l)
				buf.WriteString("\n")
			}
		} else if strings.ContainsAny(s, ":#[]{},&*!|>'\"%@`") || s == "" {
			buf.WriteString(strconv.Quote(s))
			buf.WriteString("\n")
		} else {
			buf.WriteString(s)
			buf.WriteString("\n")
		}
		return nil

	case reflect.Bool:
		buf.WriteString(strconv.FormatBool(v.Bool()))
		buf.WriteString("\n")
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		buf.WriteString(strconv.FormatInt(v.Int(), 10))
		buf.WriteString("\n")
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		buf.WriteString(strconv.FormatUint(v.Uint(), 10))
		buf.WriteString("\n")
		return nil

	case reflect.Float32, reflect.Float64:
		buf.WriteString(strconv.FormatFloat(v.Float(), 'f', -1, 64))
		buf.WriteString("\n")
		return nil

	default:
		fmt.Fprintf(buf, "%v\n", v.Interface())
		return nil
	}
}

// ExportTOML writes deterministic TOML representation to w.
func (e *StructuredExporter) ExportTOML(data any, w io.Writer) error {
	var buf bytes.Buffer
	v := reflect.ValueOf(data)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return fmt.Errorf("reporting: cannot export nil to TOML")
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct && v.Kind() != reflect.Map {
		return fmt.Errorf("reporting: TOML top-level must be struct or map, got %s", v.Kind())
	}

	if err := writeTOMLTable(&buf, "", v); err != nil {
		return fmt.Errorf("reporting: TOML serialization failed: %w", err)
	}

	_, err := w.Write(buf.Bytes())
	return err
}

func writeTOMLTable(buf *bytes.Buffer, sectionPrefix string, v reflect.Value) error {
	t := v.Type()

	var simpleFields []int
	var tableFields []int
	var arrayTableFields []int

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get("toml")
		if tag == "-" {
			continue
		}

		fieldVal := v.Field(i)
		for fieldVal.Kind() == reflect.Pointer || fieldVal.Kind() == reflect.Interface {
			if !fieldVal.IsNil() {
				fieldVal = fieldVal.Elem()
			} else {
				break
			}
		}

		if fieldVal.Kind() == reflect.Struct {
			tableFields = append(tableFields, i)
		} else if fieldVal.Kind() == reflect.Slice && fieldVal.Len() > 0 && fieldVal.Index(0).Kind() == reflect.Struct {
			arrayTableFields = append(arrayTableFields, i)
		} else {
			simpleFields = append(simpleFields, i)
		}
	}

	// 1. Write simple key-value fields first
	for _, idx := range simpleFields {
		field := t.Field(idx)
		fieldName := getFieldName(field, "toml")
		fieldVal := v.Field(idx)
		fmt.Fprintf(buf, "%s = %s\n", fieldName, formatTOMLScalar(fieldVal))
	}

	// 2. Write nested tables
	for _, idx := range tableFields {
		field := t.Field(idx)
		fieldName := getFieldName(field, "toml")
		secName := fieldName
		if sectionPrefix != "" {
			secName = sectionPrefix + "." + fieldName
		}
		fmt.Fprintf(buf, "\n[%s]\n", secName)
		if err := writeTOMLTable(buf, secName, v.Field(idx)); err != nil {
			return err
		}
	}

	// 3. Write array of tables
	for _, idx := range arrayTableFields {
		field := t.Field(idx)
		fieldName := getFieldName(field, "toml")
		secName := fieldName
		if sectionPrefix != "" {
			secName = sectionPrefix + "." + fieldName
		}
		sliceVal := v.Field(idx)
		for j := 0; j < sliceVal.Len(); j++ {
			fmt.Fprintf(buf, "\n[[%s]]\n", secName)
			if err := writeTOMLTable(buf, secName, sliceVal.Index(j)); err != nil {
				return err
			}
		}
	}

	return nil
}

func getFieldName(field reflect.StructField, tagKey string) string {
	tag := field.Tag.Get(tagKey)
	if tag != "" && tag != "-" {
		if idx := strings.Index(tag, ","); idx != -1 {
			tag = tag[:idx]
		}
		if tag != "" {
			return tag
		}
	}
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		if idx := strings.Index(jsonTag, ","); idx != -1 {
			jsonTag = jsonTag[:idx]
		}
		if jsonTag != "" {
			return jsonTag
		}
	}
	return strings.ToLower(field.Name)
}

func formatTOMLScalar(v reflect.Value) string {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return "\"\""
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		return strconv.Quote(v.String())
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Slice:
		var items []string
		for i := 0; i < v.Len(); i++ {
			items = append(items, formatTOMLScalar(v.Index(i)))
		}
		return "[" + strings.Join(items, ", ") + "]"
	default:
		return strconv.Quote(fmt.Sprint(v.Interface()))
	}
}

// ExportCSV writes tabular RFC 4180 CSV to w.
func (e *StructuredExporter) ExportCSV(data any, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	v := reflect.ValueOf(data)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return fmt.Errorf("reporting: CSV data payload is nil")
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Slice {
		if v.Len() == 0 {
			return nil
		}
		first := v.Index(0)
		for first.Kind() == reflect.Pointer || first.Kind() == reflect.Interface {
			first = first.Elem()
		}

		if first.Kind() == reflect.Struct {
			var headers []string
			t := first.Type()
			for i := 0; i < first.NumField(); i++ {
				f := t.Field(i)
				if f.IsExported() {
					headers = append(headers, getFieldName(f, "csv"))
				}
			}
			if err := writer.Write(headers); err != nil {
				return err
			}

			for i := 0; i < v.Len(); i++ {
				item := v.Index(i)
				for item.Kind() == reflect.Pointer || item.Kind() == reflect.Interface {
					item = item.Elem()
				}
				var row []string
				for j := 0; j < item.NumField(); j++ {
					if t.Field(j).IsExported() {
						row = append(row, fmt.Sprint(item.Field(j).Interface()))
					}
				}
				if err := writer.Write(row); err != nil {
					return err
				}
			}
			return nil
		}
	}

	// Single struct flattening
	if v.Kind() == reflect.Struct {
		t := v.Type()
		var headers []string
		var values []string
		for i := 0; i < v.NumField(); i++ {
			f := t.Field(i)
			if f.IsExported() {
				fieldVal := v.Field(i)
				if fieldVal.Kind() != reflect.Slice && fieldVal.Kind() != reflect.Struct && fieldVal.Kind() != reflect.Map {
					headers = append(headers, getFieldName(f, "csv"))
					values = append(values, fmt.Sprint(fieldVal.Interface()))
				}
			}
		}
		if err := writer.Write(headers); err != nil {
			return err
		}
		return writer.Write(values)
	}

	return fmt.Errorf("reporting: CSV export requires a slice or struct, got %s", v.Kind())
}
