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
	ID        string                  `db:"id"`
	Name      string                  `db:"name"`
	ShortName string                  `db:"short_name"`
	Type      InvestigationTargetType `db:"type"`
}

// Completion is a question and answer pair that is part of an investigation.
type Completion struct {
	ID                    []byte `db:"id"`
	Order                 int    `db:"order"`
	Question              string `db:"question"`
	Answer                string `db:"answer"`
	UserID                string `db:"user_id"`
	InvestigationTargetID string `db:"investigation_target_id"`
}
