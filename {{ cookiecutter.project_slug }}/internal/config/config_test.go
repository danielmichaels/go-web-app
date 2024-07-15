package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/joeshaw/envdecode"
)

func TestConfig(t *testing.T) {
	os.Setenv("POSTGRES_DB", "db")
	os.Setenv("POSTGRES_PORT", "9999")

	cfg := AppConfig()

	if cfg.Db.Db != "db" {
		t.Errorf("expected %q, got %q", "db", os.Getenv("POSTGRES_DB"))
	}
	if cfg.Db.Port != 9999 {
		t.Errorf("expected %q, got %q", "9999", os.Getenv("POSTGRES_PORT"))
	}
}

func ExampleAppConfig() {
	type exampleStruct struct {
		String string `env:"STRING"`
	}
	os.Setenv("STRING", "an example string!")

	var e exampleStruct
	err := envdecode.StrictDecode(&e)
	if err != nil {
		panic(err)
	}

	// if STRING is set, e.String will contain its value
	fmt.Println(e.String)

	// Output:
	// an example string!
}

func TestAppConfigError(t *testing.T) {
	type exampleStruct struct {
		String string `env:"BADSTRING,required"`
	}
	var e exampleStruct
	err := envdecode.StrictDecode(&e)
	fmt.Println(err)

	// Output:
	// the environment variable "BADSTRING" is missing
	want := "the environment variable \"BADSTRING\" is missing"
	if err.Error() != want {
		t.Errorf("expected: %q, got %q", want, err.Error())
	}
}
