package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

type ConfigAgent struct {
	AddrServer    string
	SleepInterval int
}

func NewAgentConfig() *ConfigAgent {
	v := viper.New()
	//v.SetEnvPrefix("gophermart")
	v.AutomaticEnv()

	v.SetDefault("AddrServer", "127.0.0.1:8080")
	v.SetDefault("SleepInterval", 70)

	return &ConfigAgent{
		AddrServer:    v.GetString("AddrServer"),
		SleepInterval: v.GetInt("SleepInterval"),
	}
}

type ConfigServer struct {
	once    sync.Once
	vals    map[string]string
	valsInt map[string]int
}

func NewConfig() *ConfigServer {
	return &ConfigServer{}
}

func LoadConfigServer(cfg *ConfigServer) {

	v := viper.New()
	//v.SetEnvPrefix("gophermart")
	v.AutomaticEnv()

	v.SetDefault("AddrServer", "127.0.0.1:8090")
	v.SetDefault("DatabaseDSN", "postgres://postgres:qwerty@localhost:5432/exam1?sslmode=disable")
	v.SetDefault("SessionKey", GetSessionKey())
	v.SetDefault("Salt", GetSalt())
	v.SetDefault("UserTable", "users")
	v.SetDefault("OrderTable", "orders5")
	v.SetDefault("WithdrawTable", "WithdrawTable1")
	v.SetDefault("TokenTTL", 24)

	cfg.vals = make(map[string]string)
	cfg.vals["AddrServer"] = v.GetString("AddrServer")
	cfg.vals["DatabaseDSN"] = v.GetString("DatabaseDSN")
	cfg.vals["SessionKey"] = v.GetString("SessionKey")
	cfg.vals["Salt"] = v.GetString("Salt")
	cfg.vals["UserTable"] = v.GetString("UserTable")
	cfg.vals["OrderTable"] = v.GetString("OrderTable")
	cfg.vals["WithdrawTable"] = v.GetString("WithdrawTable")

	cfg.valsInt = make(map[string]int)
	cfg.valsInt["TokenTTL"] = v.GetInt("TokenTTL")

}

func (cfg *ConfigServer) Get(k string) string {
	cfg.once.Do(func() {
		LoadConfigServer(cfg)
	})

	v, ok := cfg.vals[k]

	if !ok {
		log.Fatal("Не найдена настройка конфига " + k)
	}
	return v
}

func (cfg *ConfigServer) GetInt(k string) int {
	cfg.once.Do(func() {
		LoadConfigServer(cfg)
	})

	v, ok := cfg.valsInt[k]

	if !ok {
		log.Fatal("Не найдена настройка конфига int64" + k)
	}
	return v
}

func GetSessionKey() string {
	return "fsadfsadfsadfsadfsdfsa4564"
}

func GetSalt() string {
	return "sdfsadff45468/7asfas54"
}
