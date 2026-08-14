package hetrixtools

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	blacklistMonitorNamePattern   = regexp.MustCompile(`^[a-zA-Z0-9 .-]+$`)
	blacklistMonitorTargetPattern = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	hetrixToolsIDPattern          = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)
)

func (r PaginationRequest) appendQuery(values map[string]string) {
	setInt(values, "page", r.Page)
	setInt(values, "per_page", r.PerPage)
}

func setString(values map[string]string, key string, value string) {
	if value != "" {
		values[key] = value
	}
}

func setInt(values map[string]string, key string, value int) {
	if value != 0 {
		values[key] = strconv.Itoa(value)
	}
}

func setInt64(values map[string]string, key string, value int64) {
	if value != 0 {
		values[key] = strconv.FormatInt(value, 10)
	}
}

func setBool(values map[string]string, key string, value *bool) {
	if value != nil {
		values[key] = strconv.FormatBool(*value)
	}
}

func validateQuery(query any) error {
	if err := validateStruct(query); err != nil {
		return fmt.Errorf("invalid query: %w", err)
	}
	return nil
}

func validateRequest(request any) error {
	if err := validateStruct(request); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}
	return nil
}

type validationErrors []string

func (e validationErrors) Error() string { return strings.Join(e, "; ") }

func (e *validationErrors) add(field, tag, parameter string) {
	detail := fmt.Sprintf("field %s failed %s validation", field, tag)
	if parameter != "" {
		detail += " (" + parameter + ")"
	}
	*e = append(*e, detail)
}

// validateStruct implements the small, deliberately constrained subset of
// validator tags used by this package. Keeping validation tags on the request
// structs avoids duplicating field rules while avoiding a large dependency
// tree for required/enum/range/date checks.
func validateStruct(value any) error {
	var errs validationErrors
	validateValue(reflect.ValueOf(value), &errs)
	validateCrossFields(value, &errs)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateValue(value reflect.Value, errs *validationErrors) {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		errs.add("request", "struct", "")
		return
	}

	typeOfValue := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := typeOfValue.Field(i)
		fieldValue := value.Field(i)
		if fieldType.Anonymous {
			validateValue(fieldValue, errs)
		}
		if rules := fieldType.Tag.Get("validate"); rules != "" {
			validateField(fieldType.Name, fieldValue, strings.Split(rules, ","), errs)
		}
	}
}

func validateField(name string, value reflect.Value, rules []string, errs *validationErrors) {
	for i, rule := range rules {
		if rule == "omitempty" {
			if value.IsZero() {
				return
			}
			continue
		}
		if rule == "dive" {
			for j := 0; j < value.Len(); j++ {
				validateField(name+"["+strconv.Itoa(j)+"]", value.Index(j), rules[i+1:], errs)
			}
			return
		}

		tag, parameter, _ := strings.Cut(rule, "=")
		if !passesRule(value, tag, parameter) {
			errs.add(name, tag, parameter)
		}
	}
}

func passesRule(value reflect.Value, tag, parameter string) bool {
	switch tag {
	case "required":
		return !value.IsZero()
	case "oneof":
		return slicesContain(strings.Fields(parameter), value.String())
	case "min", "max":
		limit, err := strconv.ParseInt(parameter, 10, 64)
		if err != nil {
			return false
		}
		n := value.Int()
		if tag == "min" {
			return n >= limit
		}
		return n <= limit
	case "datetime":
		_, err := time.Parse(parameter, value.String())
		return err == nil
	case "hetrixtools_id":
		return hetrixToolsIDPattern.MatchString(value.String())
	case "blacklist_monitor_name":
		return blacklistMonitorNamePattern.MatchString(value.String())
	case "blacklist_monitor_target":
		return blacklistMonitorTargetPattern.MatchString(value.String())
	case "uptime_location":
		_, ok := uptimeLocationCode(value.String())
		return ok
	default:
		return false
	}
}

func slicesContain(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func validateCrossFields(value any, errs *validationErrors) {
	switch request := value.(type) {
	case ListContactListsRequest:
		validatePerPage(request.PerPage, 200, errs)
	case ListBlacklistMonitorsRequest:
		validatePerPage(request.PerPage, 1024, errs)
	case ListUptimeMonitorsRequest:
		validatePerPage(request.PerPage, 200, errs)
	case ListUptimeMonitorDowntimesRequest:
		validatePerPage(request.PerPage, 200, errs)
	case ListStatusPagesRequest:
		validatePerPage(request.PerPage, 100, errs)
	case ListScheduledMaintenancesRequest:
		validatePerPage(request.PerPage, 200, errs)
	case UptimeMonitorRequest:
		validateUptimeMonitorRequest(request, errs)
	}
}

func validatePerPage(perPage, max int, errs *validationErrors) {
	if perPage > max {
		errs.add("PerPage", "max", strconv.Itoa(max))
	}
}

func validateUptimeMonitorRequest(request UptimeMonitorRequest, errs *validationErrors) {
	monitorType := request.Type
	if monitorType == "" {
		return
	}

	if monitorType != "http" {
		if request.HTTPMethod != "" {
			errs.add("HTTPMethod", "excluded_unless", "type http")
		}
		if request.MaxRedirects != 0 {
			errs.add("MaxRedirects", "excluded_unless", "type http")
		}
		if request.Keyword != "" {
			errs.add("Keyword", "excluded_unless", "type http")
		}
		if len(request.HTTPCodes) > 0 {
			errs.add("HTTPCodes", "excluded_unless", "type http")
		}
	}

	if monitorType != "smtp" {
		if request.Port != 0 {
			errs.add("Port", "excluded_unless", "type smtp")
		}
		if request.SMTPUser != "" {
			errs.add("SMTPUser", "excluded_unless", "type smtp")
		}
		if request.SMTPPass != "" {
			errs.add("SMTPPass", "excluded_unless", "type smtp")
		}
	}
	if monitorType == "smtp" && request.Port == 0 {
		errs.add("Port", "required", "")
	}
	if (request.SMTPUser == "") != (request.SMTPPass == "") {
		errs.add("SMTPUser", "required_with", "smtp_password")
		errs.add("SMTPPass", "required_with", "smtp_user")
	}
	if (monitorType == "http" || monitorType == "ping" || monitorType == "smtp") && request.Target == "" {
		errs.add("Target", "required", "")
	}

	if monitorType != "heartbeat" && (request.Grace != 0 || request.INFOPub != nil || request.CPUPub != nil || request.RAMPub != nil || request.DISKPub != nil || request.NETPub != nil) {
		errs.add("Type", "heartbeat_fields", "")
	}
	if monitorType == "heartbeat" {
		if request.Target != "" {
			errs.add("Target", "excluded_if", "type heartbeat")
		}
		if len(request.Locations) > 0 {
			errs.add("Locations", "excluded_if", "type heartbeat")
		}
		if request.FailedLocations != 0 {
			errs.add("FailedLocations", "excluded_if", "type heartbeat")
		}
	}

	if monitorType != "http" && monitorType != "smtp" {
		if request.VerSSLCert != nil {
			errs.add("VerSSLCert", "excluded_unless", "type http smtp")
		}
		if request.VerSSLHost != nil {
			errs.add("VerSSLHost", "excluded_unless", "type http smtp")
		}
	}
}
