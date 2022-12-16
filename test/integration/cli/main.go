package main

import (
	"fmt"
	"os"
	"time"

	"github.com/trustbloc/wallet-sdk/test/integration/pkg/setup/oidc4ci"
	"github.com/trustbloc/wallet-sdk/test/integration/pkg/testenv"
)

func main() {
	args := os.Args[1:]

	if len(args) == 1 && args[0] == "issuance" {
		initiatePreAuthorizedIssuance()
	}
}

func initiatePreAuthorizedIssuance() {
	err := testenv.SetupTestEnv("fixtures/keys/tls/ec-cacert.pem")
	if err != nil {
		panic(err)
	}

	oidc4ciSetup, err := oidc4ci.NewSetup(testenv.NewHttpRequest())
	if err != nil {
		panic(err)
	}

	for i := 0; i < 120; i++ {
		err = oidc4ciSetup.AuthorizeIssuerBypassAuth("test_org")
		if err == nil {
			break
		}
		println(err.Error())
		time.Sleep(10 * time.Second)
	}
	if err != nil {
		panic(err)
	}

	initiateIssuanceURL, err := oidc4ciSetup.InitiatePreAuthorizedIssuance("bank_issuer")
	if err != nil {
		panic(err)
	}

	fmt.Print(initiateIssuanceURL)
}