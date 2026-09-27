package authz

import (
	"backend/pkg/log"
	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// RESTful 模型：subject 为角色，object 为路由（支持 :param），action 为 HTTP 方法。
const modelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`

func NewEnforcer(db *gorm.DB, logger *log.Logger) (*casbin.Enforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}
	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, err
	}
	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, err
	}
	if err := seed(e); err != nil {
		return nil, err
	}
	logger.Info("casbin enforcer ready")
	return e, nil
}

func seed(e *casbin.Enforcer) error {
	policies := [][]string{
		{"admin", "/api/v1/*", "*"},
		{"user", "/api/v1/credentials", "GET"},
		{"user", "/api/v1/credentials", "POST"},
		{"user", "/api/v1/credentials/:id", "PUT"},
		{"user", "/api/v1/credentials/:id", "DELETE"},
		{"user", "/api/v1/credentials/access-logs", "GET"},
		{"user", "/api/v1/tasks", "GET"},
		{"user", "/api/v1/tasks", "POST"},
		{"user", "/api/v1/tasks/:id", "GET"},
		{"user", "/api/v1/tasks/:id", "DELETE"},
		{"user", "/api/v1/tasks/:id/cancel", "POST"},
		{"user", "/api/v1/tasks/:id/retry", "POST"},
		{"user", "/api/v1/questions", "GET"},
		{"user", "/api/v1/questions/lookup", "GET"},
		{"user", "/api/v1/me", "GET"},
		{"user", "/api/v1/platforms", "GET"},
		{"user", "/api/v1/platforms/:id/items", "GET"},
		{"user", "/api/v1/platforms/:id/discover", "POST"},
	}
	for _, p := range policies {
		if _, err := e.AddPolicy(p[0], p[1], p[2]); err != nil {
			return err
		}
	}
	return nil
}
