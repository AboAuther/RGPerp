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
	Admin       AdminConfig       `mapstructure:"admin"`
	Blockchain  BlockchainConfig  `mapstructure:"blockchain"`
	Hyperliquid HyperliquidConfig `mapstructure:"hyperliquid"`
	Binance     BinanceConfig     `mapstructure:"binance"`
	Price       PriceConfig       `mapstructure:"price"`
	Oracle      OracleConfig      `mapstructure:"oracle"`
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

type AdminConfig struct {
	Wallets []string `mapstructure:"wallets"`
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

type BinanceConfig struct {
	FuturesAPIURL string `mapstructure:"futures_api_url"`
}

type PriceConfig struct {
	Source         string `mapstructure:"source"`
	PollIntervalMS int    `mapstructure:"poll_interval_ms"`
	RetentionHours int    `mapstructure:"retention_hours"`
}

type OracleConfig struct {
	BTCUSDFeed string `mapstructure:"btc_usd_feed"`
	ETHUSDFeed string `mapstructure:"eth_usd_feed"`
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
	cfg.Admin.Wallets = normalizeWallets(v.GetString("ADMIN_WALLETS"))
	cfg.Blockchain.RPCURL = v.GetString("RPC_URL")
	cfg.Blockchain.ChainID = v.GetInt64("CHAIN_ID")
	cfg.Blockchain.VaultAddress = v.GetString("VAULT_ADDRESS")
	cfg.Blockchain.USDCAddress = v.GetString("USDC_ADDRESS")
	cfg.Blockchain.OperatorPrivateKey = v.GetString("OPERATOR_PRIVATE_KEY")
	cfg.Hyperliquid.APIURL = v.GetString("HYPERLIQUID_API_URL")
	cfg.Hyperliquid.WalletAddress = v.GetString("HYPERLIQUID_WALLET_ADDRESS")
	cfg.Hyperliquid.PrivateKey = v.GetString("HYPERLIQUID_PRIVATE_KEY")
	cfg.Binance.FuturesAPIURL = v.GetString("BINANCE_FUTURES_API_URL")
	cfg.Price.Source = v.GetString("PRICE_SOURCE")
	cfg.Price.PollIntervalMS = v.GetInt("PRICE_POLL_INTERVAL_MS")
	cfg.Price.RetentionHours = v.GetInt("PRICE_RETENTION_HOURS")
	cfg.Oracle.BTCUSDFeed = v.GetString("ORACLE_BTC_USD_FEED")
	cfg.Oracle.ETHUSDFeed = v.GetString("ORACLE_ETH_USD_FEED")

	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	if cfg.Price.PollIntervalMS <= 0 {
		cfg.Price.PollIntervalMS = 2000
	}
	if cfg.Price.RetentionHours <= 0 {
		cfg.Price.RetentionHours = 72
	}
	if cfg.Binance.FuturesAPIURL == "" {
		cfg.Binance.FuturesAPIURL = "https://fapi.binance.com"
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
		"ADMIN_WALLETS",
		"RPC_URL", "CHAIN_ID", "VAULT_ADDRESS", "USDC_ADDRESS", "OPERATOR_PRIVATE_KEY",
		"HYPERLIQUID_API_URL", "HYPERLIQUID_WALLET_ADDRESS", "HYPERLIQUID_PRIVATE_KEY",
		"BINANCE_FUTURES_API_URL",
		"PRICE_SOURCE", "PRICE_POLL_INTERVAL_MS", "PRICE_RETENTION_HOURS",
		"ORACLE_BTC_USD_FEED", "ORACLE_ETH_USD_FEED",
	}
	for _, key := range envKeys {
		_ = v.BindEnv(key)
	}
}

func normalizeWallets(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		wallet := strings.ToLower(strings.TrimSpace(part))
		if wallet == "" {
			continue
		}
		out = append(out, wallet)
	}
	return out
}
