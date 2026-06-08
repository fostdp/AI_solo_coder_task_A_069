package config

import "os"

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	MongoURI      string
	MongoDB       string
	MinioEndpoint string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket   string
	ServerPort    string
}

func Load() *Config {
	return &Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "postgres"),
		DBName:         getEnv("DB_NAME", "sim_scenario"),
		MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:        getEnv("MONGO_DB", "sim_scenario"),
		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    getEnv("MINIO_BUCKET", "sim-scenario"),
		ServerPort:     getEnv("SERVER_PORT", "8080"),
	}
}

func (c *Config) DSN() string {
	return "postgres://" + c.DBUser + ":" + c.DBPassword + "@" + c.DBHost + ":" + c.DBPort + "/" + c.DBName + "?sslmode=disable"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
