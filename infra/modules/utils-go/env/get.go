package env

import (
	"log"
	"os"
	"slices"
)

type EnvPolicy int

const (
	EnvRequired EnvPolicy = iota
	EnvWarn
	EnvDefault
	EnvRequiredEnum
)

func GetEnv(name string, policy EnvPolicy, def ...string) string {
	v := os.Getenv(name)
	if v != "" && policy != EnvRequiredEnum {
		return v
	}

	switch policy {
	case EnvDefault:
		if len(def) == 0 {
			panic("EnvDefault requires a default value")
		}
		log.Printf(
			"!!! Warning: environment variable `%s` not found, defaulting to `%s`",
			name, def[0],
		)
		return def[0]

	case EnvRequired:
		log.Fatalf("!!! Missing required environment variable `%s`", name)

	case EnvRequiredEnum:
		if v == "" || !slices.Contains(def, v) {
			log.Fatalf("!!! Missing/Unmatching enum based required environment variable `%s` `%v`", name, def)
			return ""
		}
		return v

	case EnvWarn:
		log.Printf("!!! Warning: environment variable `%s` not found", name)
		return ""

	default:
		panic("unknown EnvPolicy")
	}

	return ""

}
