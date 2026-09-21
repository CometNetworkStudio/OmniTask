package repository

import (
	"context"
	"time"

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

func (repo *UserRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := repo.r.DB(ctx).Model(&model.User{}).Count(&total).Error
	return total, err
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

type ScriptRepository struct{ r *Repository }

func NewScriptRepository(r *Repository) *ScriptRepository { return &ScriptRepository{r: r} }

func (repo *ScriptRepository) List(ctx context.Context) ([]model.Script, error) {
	var scripts []model.Script
	if err := repo.r.DB(ctx).Order("id").Find(&scripts).Error; err != nil {
		return nil, err
	}
	return scripts, nil
}

func (repo *ScriptRepository) Get(ctx context.Context, id string) (*model.Script, error) {
	var script model.Script
	if err := repo.r.DB(ctx).Where("id = ?", id).First(&script).Error; err != nil {
		return nil, err
	}
	return &script, nil
}

func (repo *ScriptRepository) Upsert(ctx context.Context, script *model.Script) error {
	return repo.r.DB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "version", "source_url", "manifest", "enabled", "status", "updated_at",
		}),
	}).Create(script).Error
}

func (repo *ScriptRepository) SetEnabled(ctx context.Context, id string, enabled bool) error {
	return repo.r.DB(ctx).Model(&model.Script{}).Where("id = ?", id).
		Updates(map[string]interface{}{"enabled": enabled, "updated_at": time.Now()}).Error
}

func (repo *ScriptRepository) Delete(ctx context.Context, id string) error {
	return repo.r.DB(ctx).Where("id = ?", id).Delete(&model.Script{}).Error
}

func (repo *ScriptRepository) EnabledIDs(ctx context.Context) ([]string, error) {
	var ids []string
	err := repo.r.DB(ctx).Model(&model.Script{}).Where("enabled = ?", true).Pluck("id", &ids).Error
	return ids, err
}

// MarkStaleInterrupted 把重启后残留的 running/queued 任务标记为 interrupted。
func (repo *TaskRepository) MarkStaleInterrupted(ctx context.Context) (int64, error) {
	res := repo.r.DB(ctx).Model(&model.Task{}).
		Where("status IN ?", []string{"running", "queued"}).
		Updates(map[string]interface{}{
			"status":      "interrupted",
			"error":       "服务重启导致中断",
			"finished_at": time.Now(),
		})
	return res.RowsAffected, res.Error
}

func (repo *TaskRepository) Delete(ctx context.Context, userID, id string) (bool, error) {
	res := repo.r.DB(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.Task{})
	return res.RowsAffected > 0, res.Error
}

func (repo *TaskRepository) DeleteEvents(ctx context.Context, taskID string) error {
	return repo.r.DB(ctx).Where("task_id = ?", taskID).Delete(&model.TaskEvent{}).Error
}

func (repo *TaskRepository) Reset(ctx context.Context, userID, id string) error {
	return repo.r.DB(ctx).Model(&model.Task{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"status":      "pending",
			"progress":    0,
			"result":      "{}",
			"error":       "",
			"started_at":  nil,
			"finished_at": nil,
		}).Error
}

func (repo *UserRepository) List(ctx context.Context, limit, offset int) ([]model.User, int64, error) {
	var (
		users []model.User
		total int64
	)
	q := repo.r.DB(ctx).Model(&model.User{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at ASC").Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (repo *UserRepository) UpdateRole(ctx context.Context, id, role string) error {
	return repo.r.DB(ctx).Model(&model.User{}).Where("id = ?", id).
		Updates(map[string]interface{}{"role": role, "updated_at": time.Now()}).Error
}

func (repo *UserRepository) SetEnabled(ctx context.Context, id string, enabled bool) error {
	return repo.r.DB(ctx).Model(&model.User{}).Where("id = ?", id).
		Updates(map[string]interface{}{"enabled": enabled, "updated_at": time.Now()}).Error
}
