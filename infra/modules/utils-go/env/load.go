package env

// Represents the environment configuration for the application.

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

func LoadDBConfig() DBConfig {
	return DBConfig{
		Host:     GetEnv("DATABASE_HOST", EnvWarn),
		Port:     GetEnv("DATABASE_PORT", EnvWarn),
		User:     GetEnv("DATABASE_USER", EnvWarn),
		Password: GetEnv("DATABASE_PASSWORD", EnvWarn),
		Database: GetEnv("DATABASE_NAME", EnvWarn),
		SSLMode:  GetEnv("DATABASE_SSL_MODE", EnvWarn),
	}
}

type AuthConfig struct {
	JWT_ACCESS_SECRET  string
	JWT_REFRESH_SECRET string
}

func LoadAuthConfig() AuthConfig {
	return AuthConfig{
		JWT_ACCESS_SECRET:  GetEnv("JWT_ACCESS_SECRET", EnvWarn),
		JWT_REFRESH_SECRET: GetEnv("JWT_REFRESH_SECRET", EnvWarn),
	}
}

type GatewayConfig struct {
	WS_SERVICE_URL string
}

func LoadGatewayConfig() GatewayConfig {
	return GatewayConfig{
		WS_SERVICE_URL: GetEnv("WS_SERVICE_URL", EnvDefault, "http://ws-service:8081"),
	}
}

type Runtime string

const (
	PROD Runtime = "PROD"
	TEST Runtime = "TEST"
	DEV  Runtime = "DEV"
)

type ServerConfig struct {
	SERVER_PORT string
	RUNNING_IN  string
}

func LoadServerConfig() ServerConfig {
	return ServerConfig{
		SERVER_PORT: GetEnv("SERVER_PORT", EnvWarn),
		RUNNING_IN:  GetEnv("RUNNING_IN", EnvRequiredEnum, string(PROD), string(DEV), string(TEST)),
	}
}

type NatsConfig struct {
	URL string
}

func LoadNatsConfig() NatsConfig {
	return NatsConfig{
		URL: GetEnv("NATS_URL", EnvWarn),
	}
}

// type GatewayConfig struct {
// 	AUTH_SERVICE_URL    string
// 	API_SERVICE_URL     string
// 	LANDING_SERVICE_URL string
// 	LOGIN_SERVICE_URL   string
// 	USER_SERVICE_URL    string
// 	ADMIN_SERVICE_URL   string
// }

// type NatsConfig struct {
// 	URL string
// }

// type Config struct {
// 	DBConfig
// 	ServerConfig
// 	AuthConfig
// 	GatewayConfig
// 	NatsConfig
// }

// func Load() Config {
// 	return Config{
// 		DBConfig: DBConfig{
// 			Host:     GetEnv("DATABASE_HOST", EnvWarn),
// 			Port:     GetEnv("DATABASE_PORT", EnvWarn),
// 			User:     GetEnv("DATABASE_USER", EnvWarn),
// 			Password: GetEnv("DATABASE_PASSWORD", EnvWarn),
// 			Database: GetEnv("DATABASE_NAME", EnvWarn),
// 			SSLMode:  GetEnv("DATABASE_SSL_MODE", EnvWarn),
// 		},
// 		ServerConfig: ServerConfig{
// 			PORT:       GetEnv("SERVER_PORT", EnvWarn),
// 			RUNNING_IN: GetEnv("RUNNING_IN", EnvRequiredEnum, string(PROD), string(DEV), string(TEST)),
// 		},
// 		AuthConfig: AuthConfig{
// 			JWT_ACCESS_SECRET:  GetEnv("JWT_ACCESS_SECRET", EnvWarn),
// 			JWT_REFRESH_SECRET: GetEnv("JWT_REFRESH_SECRET", EnvWarn),
// 		},
// 		GatewayConfig: GatewayConfig{
// 			AUTH_SERVICE_URL:    GetEnv("AUTH_SERVICE_URL", EnvDefault, "http://localhost:8081"),
// 			API_SERVICE_URL:     GetEnv("API_SERVICE_URL", EnvDefault, "http://localhost:8082"),
// 			LANDING_SERVICE_URL: GetEnv("LANDING_SERVICE_URL", EnvDefault, "http://localhost:3000"),
// 			LOGIN_SERVICE_URL:   GetEnv("LOGIN_SERVICE_URL", EnvDefault, "http://localhost:3001"),
// 			USER_SERVICE_URL:    GetEnv("USER_SERVICE_URL", EnvDefault, "http://localhost:3002"),
// 			ADMIN_SERVICE_URL:   GetEnv("ADMIN_SERVICE_URL", EnvDefault, "http://localhost:3003"),
// 		},
// 		NatsConfig: NatsConfig{
// 			URL: GetEnv("NATS_URL", EnvDefault, "nats://localhost:4222"),
// 		},
// 	}
// }
