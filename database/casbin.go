package database

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
)

func Casbin() *casbin.Enforcer {
	// Initialize casbin adapter
	adapter, err := gormadapter.NewAdapterByDB(Postgres)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize casbin adapter: %v", err))
	}

	// Load model configuration file and policy store adapter
	e, err := casbin.NewEnforcer("config/restful_rbac_model.conf", adapter)
	if err != nil {
		panic(fmt.Sprintf("failed to create casbin enforcer: %v", err))
	}

	// Add default policy
	if hasPolicy, _ := e.HasPolicy("admin", "/v1/admin*", "(GET)|(POST)|(PUT)|(DELETE)"); !hasPolicy {
		e.AddPolicy("admin", "/v1/admin*", "(GET)|(POST)|(PUT)|(DELETE)")
	}

	e.LoadPolicy()
	return e
}
