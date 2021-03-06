package internal

import (
	"os"

	"github.com/abergmeier/terraform-provider-exasol/internal/exaprovider"
	"github.com/grantstreetgroup/go-exasol-client"
)

func MustCreateTestConf() exasol.ConnConf {
	exaHost := os.Getenv("EXAHOST")
	if exaHost == "" {
		panic("Tests need EXAHOST to run")
	}

	exaUID := os.Getenv("EXAUID")
	if exaUID == "" {
		exaUID = "sys"
	}

	exaPWD := os.Getenv("EXAPWD")
	if exaPWD == "" {
		exaPWD = "exasol"
	}

	return exasol.ConnConf{
		Host:     exaHost,
		Port:     8563,
		Username: exaUID,
		Password: exaPWD,
		//LogLevel: "debug",
	}
}

func MustCreateTestClient() *exaprovider.Client {
	return exaprovider.NewClient(MustCreateTestConf())
}
