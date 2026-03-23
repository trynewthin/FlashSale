package catalog

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type perfFieldType string

const (
	perfFieldTypeNumber   perfFieldType = "number"
	perfFieldTypeDuration perfFieldType = "duration"
	perfFieldTypeSelect   perfFieldType = "select"
)

type perfFieldSpec struct {
	Flag     string
	Type     perfFieldType
	Required bool
	Min      int64
	Max      int64
	Options  map[string]struct{}
}

type perfTaskSpec struct {
	Fields []perfFieldSpec
}

const (
	maxPerfOpenRate    int64 = 100_000
	maxPerfConcurrency int64 = 100_000
	maxPerfRequests    int64 = 10_000_000
	maxPerfTempUsers   int64 = 1_000_000
)

var perfTaskSpecs = map[string]perfTaskSpec{
	"perf.purchase_open": {
		Fields: []perfFieldSpec{
			numberField("rate", true, 1, maxPerfOpenRate),
			durationField("open-duration", true),
			numberField("concurrency", true, 1, maxPerfConcurrency),
			numberField("temp-users", false, 1, maxPerfTempUsers),
			durationField("timeout", true),
			selectField("output", true, "json", "text"),
		},
	},
	"perf.purchase_stress": {
		Fields: []perfFieldSpec{
			numberField("concurrency", true, 1, maxPerfConcurrency),
			numberField("temp-users", false, 1, maxPerfTempUsers),
			numberField("requests", true, 1, maxPerfRequests),
			durationField("timeout", true),
			selectField("output", true, "json", "text"),
		},
	},
	"perf.track_open": {
		Fields: []perfFieldSpec{
			numberField("rate", true, 1, maxPerfOpenRate),
			durationField("open-duration", true),
			numberField("concurrency", true, 1, maxPerfConcurrency),
			durationField("timeout", true),
			selectField("event-type", true, "pv", "click", "purchase_attempt"),
			selectField("output", true, "json", "text"),
		},
	},
	"perf.track_stress": {
		Fields: []perfFieldSpec{
			numberField("concurrency", true, 1, maxPerfConcurrency),
			numberField("requests", true, 1, maxPerfRequests),
			durationField("timeout", true),
			selectField("event-type", true, "pv", "click", "purchase_attempt"),
			selectField("output", true, "json", "text"),
		},
	},
	"perf.idempotency": {
		Fields: []perfFieldSpec{
			numberField("concurrency", true, 1, maxPerfConcurrency),
			numberField("requests", true, 1, maxPerfRequests),
			numberField("expect-max-success", true, 1, 10),
			selectField("output", true, "json", "text"),
		},
	},
}

func numberField(flag string, required bool, min, max int64) perfFieldSpec {
	return perfFieldSpec{Flag: flag, Type: perfFieldTypeNumber, Required: required, Min: min, Max: max}
}

func durationField(flag string, required bool) perfFieldSpec {
	return perfFieldSpec{Flag: flag, Type: perfFieldTypeDuration, Required: required}
}

func selectField(flag string, required bool, options ...string) perfFieldSpec {
	allowed := make(map[string]struct{}, len(options))
	for _, option := range options {
		allowed[option] = struct{}{}
	}
	return perfFieldSpec{Flag: flag, Type: perfFieldTypeSelect, Required: required, Options: allowed}
}

// BuildPerfJobArgs 将结构化字段转换为 fs perf 可接受的参数数组。
func BuildPerfJobArgs(taskID string, fields map[string]string, advancedArgsText string) ([]string, error) {
	spec, ok := perfTaskSpecs[taskID]
	if !ok {
		return nil, fmt.Errorf("不支持的压测任务: %s", taskID)
	}
	normalizedFields := normalizePerfFields(fields)
	if err := validateUnknownPerfFields(spec, normalizedFields); err != nil {
		return nil, err
	}

	args := make([]string, 0, len(spec.Fields)*2+8)
	for _, fieldSpec := range spec.Fields {
		rawValue := strings.TrimSpace(normalizedFields[fieldSpec.Flag])
		if rawValue == "" {
			if fieldSpec.Required {
				return nil, fmt.Errorf("%s 不能为空", fieldSpec.Flag)
			}
			continue
		}
		if err := validatePerfFieldValue(fieldSpec, rawValue); err != nil {
			return nil, err
		}
		args = append(args, "--"+fieldSpec.Flag, rawValue)
	}
	advancedArgs := parseAdvancedArgsText(advancedArgsText)
	if len(advancedArgs) > 0 {
		args = append(args, advancedArgs...)
	}
	return args, nil
}

func normalizePerfFields(fields map[string]string) map[string]string {
	if len(fields) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(fields))
	for key, value := range fields {
		name := strings.TrimLeft(strings.TrimSpace(key), "-")
		if name == "" {
			continue
		}
		out[name] = strings.TrimSpace(value)
	}
	return out
}

func validateUnknownPerfFields(spec perfTaskSpec, fields map[string]string) error {
	allowed := make(map[string]struct{}, len(spec.Fields))
	for _, fieldSpec := range spec.Fields {
		allowed[fieldSpec.Flag] = struct{}{}
	}
	for key := range fields {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("不支持的参数字段: %s", key)
		}
	}
	return nil
}

func validatePerfFieldValue(fieldSpec perfFieldSpec, value string) error {
	switch fieldSpec.Type {
	case perfFieldTypeNumber:
		number, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%s 必须是数字", fieldSpec.Flag)
		}
		if fieldSpec.Min > 0 && number < fieldSpec.Min {
			return fmt.Errorf("%s 不能小于 %d", fieldSpec.Flag, fieldSpec.Min)
		}
		if fieldSpec.Max > 0 && number > fieldSpec.Max {
			return fmt.Errorf("%s 不能大于 %d", fieldSpec.Flag, fieldSpec.Max)
		}
		return nil
	case perfFieldTypeDuration:
		if _, err := time.ParseDuration(value); err != nil {
			return fmt.Errorf("%s 必须是合法时长（例如 30s）", fieldSpec.Flag)
		}
		return nil
	case perfFieldTypeSelect:
		if len(fieldSpec.Options) == 0 {
			return nil
		}
		if _, ok := fieldSpec.Options[value]; !ok {
			return fmt.Errorf("%s 取值非法: %s", fieldSpec.Flag, value)
		}
		return nil
	default:
		return fmt.Errorf("未知字段类型: %s", fieldSpec.Type)
	}
}

func parseAdvancedArgsText(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return strings.Fields(trimmed)
}
