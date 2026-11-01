package diagnostic

// Severity identifies an error or warning.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Issue is a diagnostic tied to a file.
type Issue struct {
	Severity Severity
	File     string
	Message  string
}

// Report collects diagnostics. Its zero value is ready to use.
type Report struct {
	issues []Issue
}

// AddError records a problem that blocks the operation.
func (r *Report) AddError(file, message string) {
	r.issues = append(r.issues, Issue{Severity: SeverityError, File: file, Message: message})
}

// AddWarning records a problem that does not block the operation.
func (r *Report) AddWarning(file, message string) {
	r.issues = append(r.issues, Issue{Severity: SeverityWarning, File: file, Message: message})
}

// Issues returns diagnostics in the order they were found.
func (r *Report) Issues() []Issue {
	return append([]Issue(nil), r.issues...)
}

// Errors returns the errors.
func (r *Report) Errors() []Issue {
	return filterBySeverity(r.issues, SeverityError)
}

// Warnings returns the warnings.
func (r *Report) Warnings() []Issue {
	return filterBySeverity(r.issues, SeverityWarning)
}

// HasErrors reports whether any errors were recorded.
func (r *Report) HasErrors() bool {
	for _, issue := range r.issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

func filterBySeverity(issues []Issue, severity Severity) []Issue {
	filtered := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		if issue.Severity == severity {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}
