package provider_test

import (
	"encoding/base64"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	e2eConfigFileEnv = "NEBIUS_E2E_CONFIG_FILE"
	e2eConfigB64Env  = "NEBIUS_E2E_CONFIG_B64"
	e2eClientIDEnv   = "NEBIUS_E2E_CLIENT_ID"
)

func TestE2EStorageBucketLifecycle(t *testing.T) {
	cfgPath := maybeWriteE2EConfig(t)
	if cfgPath == "" {
		t.Skipf(
			"Skipping e2e: no %s or %s provided",
			e2eConfigFileEnv,
			e2eConfigB64Env,
		)
	}
	bucketName := testE2EBucketName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testE2EStorageBucketConfig(cfgPath, testE2EClientID(), bucketName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.nebius_parent_id.current", "parent_id"),
					resource.TestCheckResourceAttrSet("nebius_storage_v1_bucket.test", "id"),
					resource.TestCheckResourceAttr("nebius_storage_v1_bucket.test", "name", bucketName),
					resource.TestCheckResourceAttr("nebius_storage_v1_bucket.test", "versioning_policy", "DISABLED"),
					resource.TestCheckResourceAttr("nebius_storage_v1_bucket.test", "max_size_bytes", "4096"),
				),
			},
		},
	})
}

func maybeWriteE2EConfig(t *testing.T) string {
	t.Helper()

	if path := os.Getenv(e2eConfigFileEnv); path != "" {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}

	b64 := os.Getenv(e2eConfigB64Env)
	if b64 == "" {
		return ""
	}

	content, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Skipf("Skipping e2e: failed to decode %s: %v", e2eConfigB64Env, err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "nebius-e2e-config.yaml")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write e2e config: %v", err)
	}
	return path
}

func testE2EClientID() string {
	if clientID := os.Getenv(e2eClientIDEnv); clientID != "" {
		return clientID
	}
	return "tf-provider-e2e"
}

func testE2EBucketName() string {
	return fmt.Sprintf("tf-e2e-%08x", rand.Uint32())
}

func testE2EStorageBucketConfig(configPath, clientID, bucketName string) string {
	return fmt.Sprintf(`
provider "nebius" {
  profile = {
    config_file    = %q
    client_id       = %q
    no_browser_open = true
  }
}

data "nebius_parent_id" "current" {}

resource "nebius_storage_v1_bucket" "test" {
  parent_id         = data.nebius_parent_id.current.parent_id
  name              = %q
  versioning_policy = "DISABLED"
  max_size_bytes    = 4096
}
`, configPath, clientID, bucketName)
}
