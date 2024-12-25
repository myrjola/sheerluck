package models

// Investigation holds the state of an ongoing investigation.
// It targets a person or a scene. It contains all the completions
// (AI-chat questions and answers) and relevant clues.
type Investigation struct {
	Target      InvestigationTarget
	Completions []Completion
}

type InvestigationTargetType string

const (
	InvestigationTargetTypePerson InvestigationTargetType = "person"
	InvestigationTargetTypeScene  InvestigationTargetType = "scene"
)

// InvestigationTarget is a person or a scene that is being investigated.
type InvestigationTarget struct {
	ID        string
	Name      string
	ShortName string
	Type      InvestigationTargetType
	ImagePath string
}

// Completion is a question and answer pair that is part of an investigation.
type Completion struct {
	ID       int64
	Order    int64
	Question string
	Answer   string
}
