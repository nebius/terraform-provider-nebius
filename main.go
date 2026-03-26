package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	provider "github.com/nebius/terraform-provider-nebius/provider/impl"
	"github.com/nebius/terraform-provider-nebius/provider/version"
)

const (
	testFlagEnv = "NEBIUS_TERRAFORM_PROVIDER_TEST"
)

var versionFlag = flag.Bool("version", false, "print version and exit")

func main() {
	flag.Parse()
	if *versionFlag {
		v, err := version.BuildVersion()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(v)
		return
	}

	isTesting := os.Getenv(testFlagEnv) != ""
	opts := providerserver.ServeOpts{
		Address: provider.Address,
		Debug:   isTesting,
	}

	err := providerserver.Serve(context.Background(), provider.New(), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
