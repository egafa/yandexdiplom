package config

import (
	"flag"
	"os"
)

type ConfigServer struct {
	AddrServer     string
	DatabaseDSN    string
	AccuralAddress string
	TokenTTL       int
	SleepInterval  int
	Salt           string
	SessionKey     string
	UserTable      string
	OrderTable     string
	WithdrawTable  string
}

func NewConfig() *ConfigServer {
	return &ConfigServer{}
}

func LoadConfig() (cfg *ConfigServer) {

	runAaddressENV := "RUN_ADDRESS"
	databaseURIENV := "DATABASE_URI"
	accuralAddressENV := "ACCRUAL_SYSTEM_ADDRESS"

	runAaddress := flag.String("a", "127.0.0.1:8090", "адрес сервера")

	dsn := "postgres://postgres:qwerty@localhost:5432/exam1?sslmode=disable"
	databaseURI := flag.String("d", dsn, "database URI")
	accuralAddress := flag.String("r", "127.0.0.1:8080", "ACCRUAL_SYSTEM_ADDRESS")

	flag.Parse()

	SetVal(runAaddressENV, runAaddress)
	SetVal(databaseURIENV, databaseURI)
	SetVal(accuralAddressENV, accuralAddress)

	return &ConfigServer{
		AddrServer:     *runAaddress,
		DatabaseDSN:    *databaseURI,
		AccuralAddress: *accuralAddress,
		TokenTTL:       12,
		SleepInterval:  10,
		SessionKey:     GetSessionKey(),
		Salt:           GetSalt(),
		UserTable:      "users",
		OrderTable:     "orders5",
		WithdrawTable:  "withdrawtable1",
	}
}

func SetVal(env string, val *string) {
	valEnv, ok := os.LookupEnv(env)
	if ok {
		*val = valEnv
	}
}

func GetSessionKey() string {
	return "fsadfsadfsadfsadfsdfsa4564"
}

func GetSalt() string {
	return "sdfsadff45468/7asfas54"
}
