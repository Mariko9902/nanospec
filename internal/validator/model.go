package validator

import "nanospec/internal/diagnostic"

// AcceptancePolicy lists the supported ways to accept a delivery.
type AcceptancePolicy string

const (
	AcceptanceHuman    AcceptancePolicy = "human"
	AcceptanceVerified AcceptancePolicy = "verified"
)

// VerificationMode records who reviewed the evidence.
type VerificationMode string

const (
	VerificationSelf        VerificationMode = "self"
	VerificationIndependent VerificationMode = "independent"
)

// Story holds the fields needed for validation and execution.
type Story struct {
	ID                  string
	Title               string
	Order               string
	File                string
	Directory           string
	DependsOn           []string
	RegressionOf        []string
	ImplementationBrief string
	AcceptancePolicy    AcceptancePolicy
	AcceptanceIDs       []string
	HasOutcome          bool
}

// WaveEntry keeps a Story and its coordination notes together.
type WaveEntry struct {
	ID    string
	Tail  string
	Notes []string
}

// DependencyWave groups Stories with the same dependency depth.
type DependencyWave struct {
	Number  int
	Entries []WaveEntry
}

// FieldBlock is a named entry in the execution state.
type FieldBlock struct {
	ID     string
	Fields map[string]string
}

// ViewUpdate holds a file change that has passed validation.
type ViewUpdate struct {
	File   string
	Before string
	After  string
}

// ValidationResult contains diagnostics and any proposed file changes.
type ValidationResult struct {
	InputError string
	Errors     []diagnostic.Issue
	Warnings   []diagnostic.Issue
	Updates    []ViewUpdate
}

// CycleSnapshot contains validated Cycle data and generated views.
type CycleSnapshot struct {
	ID           string
	Directory    string
	SpecRevision string
	SpecLanguage string
	CycleText    string
	StateText    string
	Stories      map[string]Story
	Waves        []DependencyWave
	Acceptance   []FieldBlock
	ActiveLanes  []FieldBlock
}

// DependencyStatus records a prerequisite and its acceptance state.
type DependencyStatus struct {
	StoryID    string `json:"story"`
	Acceptance string `json:"acceptance"`
}

// ReadyCandidate is a Story whose recorded dependencies allow further checks.
type ReadyCandidate struct {
	StoryID      string
	Wave         int
	Acceptance   string
	Dependencies []DependencyStatus
}

// ExecutionPackage contains the execution input for an explicitly selected Story.
type ExecutionPackage struct {
	CycleID               string             `json:"cycle"`
	StoryID               string             `json:"story"`
	SpecRevision          string             `json:"spec_revision"`
	SpecLanguage          string             `json:"spec_language"`
	Acceptance            string             `json:"acceptance"`
	Gate                  string             `json:"gate"`
	Dependencies          []DependencyStatus `json:"dependencies"`
	OpenQuestions         bool               `json:"open_questions"`
	RequiredConfirmations []string           `json:"required_confirmations"`
	ContextMarkdown       string             `json:"context_markdown"`
}
