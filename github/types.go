package github

import "time"

// PRRequest represents a pull request creation request
type PRRequest struct {
	BranchName  string       `json:"branch_name"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Changes     []FileChange `json:"changes"`
}

// PRResult represents the result of creating a pull request
type PRResult struct {
	URL    string `json:"url"`
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// FileChange represents a file modification
type FileChange struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// PanicEvent represents a captured panic with context
type PanicEvent struct {
	ID          string     `json:"id"`
	Timestamp   time.Time  `json:"timestamp"`
	Error       string     `json:"error"`
	StackTrace  string     `json:"stack_trace"`
	SourceFile  string     `json:"source_file"`
	LineNumber  int        `json:"line_number"`
	Function    string     `json:"function"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	Status      string     `json:"status"` // "queued", "processing", "completed", "failed"
}

// FixResponse represents the AI's response with a proposed fix
type FixResponse struct {
	ProposedFix string  `json:"proposed_fix"`
	Explanation string  `json:"explanation"`
	Confidence  float64 `json:"confidence"`
	IsValid     bool    `json:"is_valid"`
}
