package seeder

import "testing"

var defaults = []seedUser{
	{Email: "admin@example.com", Password: "Admin123!", EnvEmail: "ADMIN_EMAIL", EnvPass: "ADMIN_PASSWORD", RoleSlug: "admin"},
	{Email: "user@example.com", Password: "User1234!", EnvEmail: "USER_EMAIL", EnvPass: "USER_PASSWORD", RoleSlug: "user"},
}

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestAccountsToSeed(t *testing.T) {
	t.Run("development seeds every demo account", func(t *testing.T) {
		got := accountsToSeed(defaults, env(map[string]string{"ENV": "development"}))
		if len(got) != 2 || got[0].Password != "Admin123!" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("production never seeds the public demo credentials", func(t *testing.T) {
		if got := accountsToSeed(defaults, env(map[string]string{"ENV": "production"})); len(got) != 0 {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("production seeds the admin given by the environment only", func(t *testing.T) {
		got := accountsToSeed(defaults, env(map[string]string{
			"ENV": "production", "ADMIN_EMAIL": "ops@goodfood.fr", "ADMIN_PASSWORD": "a-long-secret-1",
		}))
		if len(got) != 1 || got[0].Email != "ops@goodfood.fr" || got[0].Password != "a-long-secret-1" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("a demo server opts in", func(t *testing.T) {
		got := accountsToSeed(defaults, env(map[string]string{"ENV": "production", "SEED_DEMO_USERS": "true"}))
		if len(got) != 2 {
			t.Fatalf("got %+v", got)
		}
	})
}
