package hetrixtools

import (
	"fmt"
	"regexp"
	"strconv"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	queryValidatorOnce sync.Once
	queryValidator     *validator.Validate

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
	ensureValidator()
	if err := validateStruct(query); err != nil {
		return fmt.Errorf("invalid query: %w", err)
	}
	return nil
}

func validateRequest(request any) error {
	ensureValidator()
	if err := validateStruct(request); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}
	return nil
}

func ensureValidator() {
	queryValidatorOnce.Do(func() {
		queryValidator = validator.New(validator.WithRequiredStructEnabled())
		_ = queryValidator.RegisterValidation("hetrixtools_id", validateHetrixToolsID)
		_ = queryValidator.RegisterValidation("blacklist_monitor_name", validateBlacklistMonitorName)
		_ = queryValidator.RegisterValidation("blacklist_monitor_target", validateBlacklistMonitorTarget)
		_ = queryValidator.RegisterValidation("uptime_location", validateUptimeLocation)
		queryValidator.RegisterStructValidation(validateContactListsRequest, ListContactListsRequest{})
		queryValidator.RegisterStructValidation(validateBlacklistMonitorsRequest, ListBlacklistMonitorsRequest{})
		queryValidator.RegisterStructValidation(validateUptimeMonitorsRequest, ListUptimeMonitorsRequest{})
		queryValidator.RegisterStructValidation(validateUptimeMonitorDowntimesRequest, ListUptimeMonitorDowntimesRequest{})
		queryValidator.RegisterStructValidation(validateStatusPagesRequest, ListStatusPagesRequest{})
		queryValidator.RegisterStructValidation(validateScheduledMaintenancesRequest, ListScheduledMaintenancesRequest{})
		queryValidator.RegisterStructValidation(validateUptimeMonitorRequest, UptimeMonitorRequest{})
	})
}

func validateStruct(value any) error {
	if err := queryValidator.Struct(value); err != nil {
		return err
	}
	return nil
}

func validateBlacklistMonitorName(level validator.FieldLevel) bool {
	return blacklistMonitorNamePattern.MatchString(level.Field().String())
}

func validateHetrixToolsID(level validator.FieldLevel) bool {
	return hetrixToolsIDPattern.MatchString(level.Field().String())
}

func validateBlacklistMonitorTarget(level validator.FieldLevel) bool {
	return blacklistMonitorTargetPattern.MatchString(level.Field().String())
}

func validateUptimeLocation(level validator.FieldLevel) bool {
	_, ok := uptimeLocationCode(level.Field().String())
	return ok
}

func validateContactListsRequest(level validator.StructLevel) {
	request := level.Current().Interface().(ListContactListsRequest)
	reportPerPageAboveMax(level, request.PerPage, 200)
}

func validateBlacklistMonitorsRequest(level validator.StructLevel) {
	request := level.Current().Interface().(ListBlacklistMonitorsRequest)
	reportPerPageAboveMax(level, request.PerPage, 1024)
}

func validateUptimeMonitorsRequest(level validator.StructLevel) {
	request := level.Current().Interface().(ListUptimeMonitorsRequest)
	reportPerPageAboveMax(level, request.PerPage, 200)
}

func validateUptimeMonitorDowntimesRequest(level validator.StructLevel) {
	request := level.Current().Interface().(ListUptimeMonitorDowntimesRequest)
	reportPerPageAboveMax(level, request.PerPage, 200)
}

func validateStatusPagesRequest(level validator.StructLevel) {
	request := level.Current().Interface().(ListStatusPagesRequest)
	reportPerPageAboveMax(level, request.PerPage, 100)
}

func validateScheduledMaintenancesRequest(level validator.StructLevel) {
	request := level.Current().Interface().(ListScheduledMaintenancesRequest)
	reportPerPageAboveMax(level, request.PerPage, 200)
}

func reportPerPageAboveMax(level validator.StructLevel, perPage int, max int) {
	if perPage > max {
		level.ReportError(perPage, "PerPage", "per_page", "max", strconv.Itoa(max))
	}
}

func validateUptimeMonitorRequest(level validator.StructLevel) {
	request := level.Current().Interface().(UptimeMonitorRequest)
	monitorType := request.Type
	if monitorType == "" {
		return
	}

	if monitorType != "http" {
		if request.HTTPMethod != "" {
			level.ReportError(request.HTTPMethod, "HTTPMethod", "http_method", "excluded_unless", "type http")
		}
		if request.MaxRedirects != 0 {
			level.ReportError(request.MaxRedirects, "MaxRedirects", "max_redirects", "excluded_unless", "type http")
		}
		if request.Keyword != "" {
			level.ReportError(request.Keyword, "Keyword", "keyword", "excluded_unless", "type http")
		}
		if len(request.HTTPCodes) > 0 {
			level.ReportError(request.HTTPCodes, "HTTPCodes", "accepted_http_codes", "excluded_unless", "type http")
		}
	}

	if monitorType != "smtp" {
		if request.Port != 0 {
			level.ReportError(request.Port, "Port", "port", "excluded_unless", "type smtp")
		}
		if request.SMTPUser != "" {
			level.ReportError(request.SMTPUser, "SMTPUser", "smtp_user", "excluded_unless", "type smtp")
		}
		if request.SMTPPass != "" {
			level.ReportError(request.SMTPPass, "SMTPPass", "smtp_password", "excluded_unless", "type smtp")
		}
	}
	if monitorType == "smtp" && request.Port == 0 {
		level.ReportError(request.Port, "Port", "port", "required", "")
	}
	if (request.SMTPUser == "") != (request.SMTPPass == "") {
		level.ReportError(request.SMTPUser, "SMTPUser", "smtp_user", "required_with", "smtp_password")
		level.ReportError(request.SMTPPass, "SMTPPass", "smtp_password", "required_with", "smtp_user")
	}
	if (monitorType == "http" || monitorType == "ping" || monitorType == "smtp") && request.Target == "" {
		level.ReportError(request.Target, "Target", "target", "required", "")
	}

	if monitorType != "heartbeat" {
		if request.Grace != 0 || request.INFOPub != nil || request.CPUPub != nil || request.RAMPub != nil || request.DISKPub != nil || request.NETPub != nil {
			level.ReportError(monitorType, "Type", "type", "heartbeat_fields", "")
		}
	}
	if monitorType == "heartbeat" {
		if request.Target != "" {
			level.ReportError(request.Target, "Target", "target", "excluded_if", "type heartbeat")
		}
		if len(request.Locations) > 0 {
			level.ReportError(request.Locations, "Locations", "locations", "excluded_if", "type heartbeat")
		}
		if request.FailedLocations != 0 {
			level.ReportError(request.FailedLocations, "FailedLocations", "failed_locations", "excluded_if", "type heartbeat")
		}
	}

	if monitorType != "http" && monitorType != "smtp" {
		if request.VerSSLCert != nil {
			level.ReportError(request.VerSSLCert, "VerSSLCert", "verify_ssl_certificate", "excluded_unless", "type http smtp")
		}
		if request.VerSSLHost != nil {
			level.ReportError(request.VerSSLHost, "VerSSLHost", "verify_ssl_host", "excluded_unless", "type http smtp")
		}
	}
}
