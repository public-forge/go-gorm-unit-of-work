package postgres

// PgConfig holds the configuration settings required to connect to a PostgreSQL database.
type PgConfig struct {
	Host                    string // Host is the database server address (e.g., "localhost" or an IP).
	DBName                  string // DBName is the name of the specific database to connect to.
	Schema                  string // Schema specifies the schema within the database (often "public").
	User                    string // User is the username for authenticating to the database.
	Password                string // Password is the password for the specified User.
	MaxOpenConnections      int    // MaxOpenConnections defines the maximum number of open connections allowed to the database.
	ConnectionMaxLifetimeMS int    // ConnectionMaxLifetimeMS sets the maximum time (in milliseconds) a connection can be reused.
	LogMode                 bool   // LogMode enables or disables SQL query logging (true for enabled).
	SSLMode                 string // SSLMode enables or disables SSL connection (e.g., "disable").
}
