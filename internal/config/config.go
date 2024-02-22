package config

type Config struct {
	Verbose                bool   `name:"verbose" help:"Enable verbose logging"`
	ListenAddr             string `name:"listen-addr" default:"localhost:8137"`
	AdminUsername          string `name:"admin-username" help:"Username for admin user"`
	AdminPassword          string `name:"admin-password" help:"Password for admin user (run through bcrypt)"`
	CSRFKey                string `name:"csrf-key" help:"Key material for CSRF protection"`
	SessionHashKey         string `name:"session-hash-key" help:"Key material for session hash"`
	SessionBlockKey        string `name:"session-block-key" help:"Key material for session block"`
	SessionInsecureAllowed bool   `name:"session-insecure-allowed" help:"Allow session cookie to be transferred on non-HTTPS connections" default:"false"`
	DBConnString           string `name:"db-conn-string" help:"Connection string for database"`
}
