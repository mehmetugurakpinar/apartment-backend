package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig
	DB        DBConfig
	Redis     RedisConfig
	JWT       JWTConfig
	MinIO     MinIOConfig
	SMTP      SMTPConfig
	CORS      CORSConfig
	RateLimit RateLimitConfig
}

type AppConfig struct {
	Env  string
	Host string
	Port string
}

type DBConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type CORSConfig struct {
	Origins string
}

type RateLimitConfig struct {
	Max    int
	Window time.Duration
}

func Load() (*Config, error) {
	// Viper yapılandırması
	viper.SetConfigName(".env") // Dosya adı
	viper.SetConfigType("env")  // Dosya tipi
	viper.AddConfigPath(".")    // Çalışma dizininde ara
	viper.AutomaticEnv()        // Sistemdeki environment variable'ları otomatik oku

	// Varsayılan Değerler
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_HOST", "0.0.0.0")
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("JWT_ACCESS_EXPIRY", "15m")
	viper.SetDefault("JWT_REFRESH_EXPIRY", "720h")
	viper.SetDefault("MINIO_USE_SSL", false)
	viper.SetDefault("RATE_LIMIT_MAX", 100)
	viper.SetDefault("RATE_LIMIT_WINDOW", "1m")

	// Yapılandırma dosyasını oku
	if err := viper.ReadInConfig(); err != nil {
		// Eğer hata "dosya bulunamadı" hatasıysa bunu görmezden geliyoruz.
		// Çünkü Docker Compose environment variable'ları sisteme zaten yüklüyor.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// Süreleri (Duration) ayrıştır
	accessExpiry, err := time.ParseDuration(viper.GetString("JWT_ACCESS_EXPIRY"))
	if err != nil {
		accessExpiry = 15 * time.Minute
	}

	refreshExpiry, err := time.ParseDuration(viper.GetString("JWT_REFRESH_EXPIRY"))
	if err != nil {
		refreshExpiry = 30 * 24 * time.Hour
	}

	rateLimitWindow, err := time.ParseDuration(viper.GetString("RATE_LIMIT_WINDOW"))
	if err != nil {
		rateLimitWindow = time.Minute
	}

	// Config nesnesini oluştur ve döndür
	cfg := &Config{
		App: AppConfig{
			Env:  viper.GetString("APP_ENV"),
			Host: viper.GetString("APP_HOST"),
			Port: viper.GetString("APP_PORT"),
		},
		DB: DBConfig{
			Host:         viper.GetString("DB_HOST"),
			Port:         viper.GetString("DB_PORT"),
			User:         viper.GetString("DB_USER"),
			Password:     viper.GetString("DB_PASSWORD"),
			Name:         viper.GetString("DB_NAME"),
			SSLMode:      viper.GetString("DB_SSLMODE"),
			MaxOpenConns: viper.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns: viper.GetInt("DB_MAX_IDLE_CONNS"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			AccessSecret:  viper.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),
			AccessExpiry:  accessExpiry,
			RefreshExpiry: refreshExpiry,
		},
		MinIO: MinIOConfig{
			Endpoint:  viper.GetString("MINIO_ENDPOINT"),
			AccessKey: viper.GetString("MINIO_ACCESS_KEY"),
			SecretKey: viper.GetString("MINIO_SECRET_KEY"),
			Bucket:    viper.GetString("MINIO_BUCKET"),
			UseSSL:    viper.GetBool("MINIO_USE_SSL"),
		},
		SMTP: SMTPConfig{
			Host:     viper.GetString("SMTP_HOST"),
			Port:     viper.GetInt("SMTP_PORT"),
			User:     viper.GetString("SMTP_USER"),
			Password: viper.GetString("SMTP_PASSWORD"),
			From:     viper.GetString("SMTP_FROM"),
		},
		CORS: CORSConfig{
			Origins: viper.GetString("CORS_ORIGINS"),
		},
		RateLimit: RateLimitConfig{
			Max:    viper.GetInt("RATE_LIMIT_MAX"),
			Window: rateLimitWindow,
		},
	}

	return cfg, nil
}

func (d DBConfig) DSN() string {
	return "postgres://" + d.User + ":" + d.Password + "@" + d.Host + ":" + d.Port + "/" + d.Name + "?sslmode=" + d.SSLMode
}
