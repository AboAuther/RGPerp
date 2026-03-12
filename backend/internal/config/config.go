package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	DB          DBConfig          `mapstructure:"db"`
	Redis       RedisConfig       `mapstructure:"redis"`
	RabbitMQ    RabbitMQConfig    `mapstructure:"rabbitmq"`
	JWT         JWTConfig         `mapstructure:"jwt"`
	Blockchain  BlockchainConfig  `mapstructure:"blockchain"`
	Hyperliquid HyperliquidConfig `mapstructure:"hyperliquid"`
	Price       PriceConfig       `mapstructure:"price"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
		c.User, c.Password, c.Host, c.Port, c.Name)
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type RabbitMQConfig struct {
	URL string `mapstructure:"url"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type BlockchainConfig struct {
	RPCURL             string `mapstructure:"rpc_url"`
	ChainID            int64  `mapstructure:"chain_id"`
	VaultAddress       string `mapstructure:"vault_address"`
	USDCAddress        string `mapstructure:"usdc_address"`
	OperatorPrivateKey string `mapstructure:"operator_private_key"`
}

type HyperliquidConfig struct {
	APIURL        string `mapstructure:"api_url"`
	WalletAddress string `mapstructure:"wallet_address"`
	PrivateKey    string `mapstructure:"private_key"`
}

type PriceConfig struct {
	Source string `mapstructure:"source"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	bindEnvMappings(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	cfg := &Config{}
	cfg.Server.Port = v.GetString("SERVER_PORT")
	cfg.Server.Mode = v.GetString("GIN_MODE")
	cfg.DB.Host = v.GetString("DB_HOST")
	cfg.DB.Port = v.GetString("DB_PORT")
	cfg.DB.User = v.GetString("DB_USER")
	cfg.DB.Password = v.GetString("DB_PASSWORD")
	cfg.DB.Name = v.GetString("DB_NAME")
	cfg.Redis.Addr = v.GetString("REDIS_ADDR")
	cfg.Redis.Password = v.GetString("REDIS_PASSWORD")
	cfg.Redis.DB = v.GetInt("REDIS_DB")
	cfg.RabbitMQ.URL = v.GetString("RABBITMQ_URL")
	cfg.JWT.Secret = v.GetString("JWT_SECRET")
	cfg.JWT.ExpireHours = v.GetInt("JWT_EXPIRE_HOURS")
	cfg.Blockchain.RPCURL = v.GetString("RPC_URL")
	cfg.Blockchain.ChainID = v.GetInt64("CHAIN_ID")
	cfg.Blockchain.VaultAddress = v.GetString("VAULT_ADDRESS")
	cfg.Blockchain.USDCAddress = v.GetString("USDC_ADDRESS")
	cfg.Blockchain.OperatorPrivateKey = v.GetString("OPERATOR_PRIVATE_KEY")
	cfg.Hyperliquid.APIURL = v.GetString("HYPERLIQUID_API_URL")
	cfg.Hyperliquid.WalletAddress = v.GetString("HYPERLIQUID_WALLET_ADDRESS")
	cfg.Hyperliquid.PrivateKey = v.GetString("HYPERLIQUID_PRIVATE_KEY")
	cfg.Price.Source = v.GetString("PRICE_SOURCE")

	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}

	return cfg, nil
}

func bindEnvMappings(v *viper.Viper) {
	envKeys := []string{
		"SERVER_PORT", "GIN_MODE",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
		"RABBITMQ_URL",
		"JWT_SECRET", "JWT_EXPIRE_HOURS",
		"RPC_URL", "CHAIN_ID", "VAULT_ADDRESS", "USDC_ADDRESS", "OPERATOR_PRIVATE_KEY",
		"HYPERLIQUID_API_URL", "HYPERLIQUID_WALLET_ADDRESS", "HYPERLIQUID_PRIVATE_KEY",
		"PRICE_SOURCE",
	}
	for _, key := range envKeys {
		_ = v.BindEnv(key)
	}
}
