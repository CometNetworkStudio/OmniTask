package repository

import (
	"context"

	"backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository struct{ r *Repository }

func NewUserRepository(r *Repository) *UserRepository { return &UserRepository{r: r} }

func (repo *UserRepository) Create(ctx context.Context, user *model.User) error {
	return repo.r.DB(ctx).Create(user).Error
}

func (repo *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := repo.r.DB(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (repo *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	if err := repo.r.DB(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

type CredentialRepository struct{ r *Repository }

func NewCredentialRepository(r *Repository) *CredentialRepository {
	return &CredentialRepository{r: r}
}

func (repo *CredentialRepository) Upsert(ctx context.Context, cred *model.Credential) error {
	return repo.r.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "platform"}},
		DoUpdates: clause.AssignmentColumns([]string{"cipher", "nonce", "updated_at"}),
	}).Create(cred).Error
}

func (repo *CredentialRepository) Get(ctx context.Context, userID, platform string) (*model.Credential, error) {
	var cred model.Credential
	if err := repo.r.DB(ctx).Where("user_id = ? AND platform = ?", userID, platform).First(&cred).Error; err != nil {
		return nil, err
	}
	return &cred, nil
}

func (repo *CredentialRepository) ListByUser(ctx context.Context, userID string) ([]model.Credential, error) {
	var creds []model.Credential
	if err := repo.r.DB(ctx).Where("user_id = ?", userID).Order("platform").Find(&creds).Error; err != nil {
		return nil, err
	}
	return creds, nil
}

type TaskRepository struct{ r *Repository }

func NewTaskRepository(r *Repository) *TaskRepository { return &TaskRepository{r: r} }

func (repo *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	return repo.r.DB(ctx).Create(task).Error
}

func (repo *TaskRepository) Get(ctx context.Context, userID, id string) (*model.Task, error) {
	var task model.Task
	if err := repo.r.DB(ctx).Where("id = ? AND user_id = ?", id, userID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (repo *TaskRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]model.Task, int64, error) {
	var (
		tasks []model.Task
		total int64
	)
	q := repo.r.DB(ctx).Model(&model.Task{}).Where("user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (repo *TaskRepository) UpdateStatus(ctx context.Context, id string, fields map[string]interface{}) error {
	return repo.r.DB(ctx).Model(&model.Task{}).Where("id = ?", id).Updates(fields).Error
}

func (repo *TaskRepository) ListByStatus(ctx context.Context, status string, limit int) ([]model.Task, error) {
	var tasks []model.Task
	if err := repo.r.DB(ctx).
		Where("status = ?", status).
		Order("created_at ASC").
		Limit(limit).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (repo *TaskRepository) AppendEvent(ctx context.Context, event *model.TaskEvent) error {
	return repo.r.DB(ctx).Create(event).Error
}

// Claim 原子地把任务从 from 状态改为 to 状态；返回是否抢到。
func (repo *TaskRepository) Claim(ctx context.Context, id, from, to string) (bool, error) {
	res := repo.r.DB(ctx).Model(&model.Task{}).
		Where("id = ? AND status = ?", id, from).
		Update("status", to)
	return res.RowsAffected > 0, res.Error
}

type QuestionRepository struct{ r *Repository }

func NewQuestionRepository(r *Repository) *QuestionRepository { return &QuestionRepository{r: r} }

func (repo *QuestionRepository) Lookup(ctx context.Context, platform, questionKey string) (*model.Question, error) {
	var q model.Question
	if err := repo.r.DB(ctx).Where("platform = ? AND question_key = ?", platform, questionKey).First(&q).Error; err != nil {
		return nil, err
	}
	_ = repo.r.DB(ctx).Model(&model.Question{}).
		Where("platform = ? AND question_key = ?", platform, questionKey).
		UpdateColumn("hit_count", gorm.Expr("hit_count + 1")).Error
	q.HitCount++
	return &q, nil
}

func (repo *QuestionRepository) Save(ctx context.Context, q *model.Question, overwrite bool) (bool, error) {
	var res *gorm.DB
	if overwrite {
		res = repo.r.DB(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "platform"}, {Name: "question_key"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"stem", "options", "answer", "source", "confidence", "updated_at",
			}),
		}).Create(q)
	} else {
		res = repo.r.DB(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(q)
	}
	return res.RowsAffected > 0, res.Error
}

func (repo *QuestionRepository) List(ctx context.Context, platform string, limit, offset int) ([]model.Question, int64, error) {
	var (
		questions []model.Question
		total     int64
	)
	q := repo.r.DB(ctx).Model(&model.Question{})
	if platform != "" {
		q = q.Where("platform = ?", platform)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&questions).Error; err != nil {
		return nil, 0, err
	}
	return questions, total, nil
}
