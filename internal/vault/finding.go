package vault

// Severity of a lint finding.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Finding is a single rule violation.
type Finding struct {
	Rule     int
	Severity Severity
	Path     string
	Message  string
}
