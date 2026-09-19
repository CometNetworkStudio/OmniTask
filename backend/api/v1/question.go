package v1

import "time"

type QuestionSaveRequest struct {
	Platform    string  `json:"platform" binding:"required"`
	QuestionKey string  `json:"question_key" binding:"required"`
	Stem        string  `json:"stem"`
	Options     string  `json:"options"`
	Answer      string  `json:"answer"`
	Source      string  `json:"source"`
	Confidence  float64 `json:"confidence"`
	Overwrite   bool    `json:"overwrite"`
}

type QuestionResponse struct {
	ID          string    `json:"id"`
	Platform    string    `json:"platform"`
	QuestionKey string    `json:"question_key"`
	Stem        string    `json:"stem"`
	Answer      string    `json:"answer"`
	Source      string    `json:"source"`
	Confidence  float64   `json:"confidence"`
	HitCount    int64     `json:"hit_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
