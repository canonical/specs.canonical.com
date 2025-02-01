package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
	"unicode"
)

type Config struct {
	AppPort      string `env:"required"`
	AppSecretKey string `env:"required"`
	Host         string `env:"default:localhost"`

	BaseURL  string `env:"required"`
	AppEnv   string `env:"required,enums:development;production"`
	LogLevel string `env:"default:debug,enums:debug;info;warn;error"`

	PostgresqlDbConnectString string `env:"required"`

	GooglePrivateKeyID string `env:"required"`
	GooglePrivateKey   string `env:"required"`

	GoogleOAuthClientID     string `env:"required"`
	GoogleOAuthClientSecret string `env:"required"`

	SyncInterval string `env:"default:1h"`
}

func (c *Config) IsProduction() bool {
	return strings.ToLower(c.AppEnv) == "production"
}

func (c *Config) GetHost() string {
	return fmt.Sprintf("%s:%s", c.Host, c.AppPort)
}

func (c *Config) GetDBDSN() string {
	return c.PostgresqlDbConnectString
}

func (c *Config) GetSyncInterval() time.Duration {
	d, err := time.ParseDuration(c.SyncInterval)
	if err != nil {
		panic(err)
	}
	return d
}

func MustLoadConfig() *Config {
	config, err := LoadConfig()
	if err != nil {
		panic(err)
	}
	return config
}

func LoadConfig() (*Config, error) {
	config := &Config{}
	return config, parseConfig(config)
}

func parseConfig(config interface{}) error {
	configValue := reflect.ValueOf(config).Elem()
	configType := configValue.Type()

	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)
		envTag := field.Tag.Get("env")
		fieldValue := configValue.Field(i)

		envKey := field.Name
		envValue := GetEnv(field.Name)
		flags := strings.Split(envTag, ",")
		var required bool
		var defaultVal string
		var enums []string

		for _, flag := range flags {
			switch {
			case flag == "required":
				required = true
			case strings.HasPrefix(flag, "default:"):
				defaultVal = strings.TrimPrefix(flag, "default:")
			case strings.HasPrefix(flag, "enums:"):
				enums = strings.Split(strings.TrimPrefix(flag, "enums:"), ";")
			}
		}

		if envValue == "" {
			if required {
				return fmt.Errorf("required environment variable %s is not set", formatEnvKey(envKey))
			}
			envValue = defaultVal
		}

		if len(enums) > 0 {
			var valid bool
			for _, e := range enums {
				if envValue == e {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("environment variable %s must be one of %v", formatEnvKey(envKey), enums)
			}
		}

		fieldValue.SetString(envValue)
	}

	return nil
}

func GetEnv(key string) string {
	variations := []string{
		key,                  // original
		strings.ToUpper(key), // all uppercase
		strings.ToLower(key), // all lowercase
		// snake case variations
		toSnakeCase(key),
		strings.ToUpper(toSnakeCase(key)),
		strings.ToLower(toSnakeCase(key)),
	}

	for _, key := range variations {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return ""

}

func formatEnvKey(key string) string {
	return strings.ToUpper(toSnakeCase(key))
}

func toSnakeCase(str string) string {
	var result []rune
	runes := []rune(str)
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 {
			if !unicode.IsUpper(runes[i-1]) && result[len(result)-1] != '_' {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}
