package provider_test

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcmetadata "google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonpb "github.com/nebius/gosdk/proto/nebius/common/v1"
	computepb "github.com/nebius/gosdk/proto/nebius/compute/v1"
	providerimpl "github.com/nebius/terraform-provider-nebius/provider/impl"
)

func TestAccComputeDiskDataSource_TokenFromEnvironment(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	t.Setenv("NEBIUS_IAM_TOKEN", "env-token")
	testAccPreCheck(t)

	server := newTestDiskServer(t)
	server.getFn = func(ctx context.Context, req *computepb.GetDiskRequest) (*computepb.Disk, error) {
		require.Equal(t, "disk-from-env", req.GetId())
		require.Equal(t, []string{"Bearer env-token"}, authorizationValues(ctx))

		return &computepb.Disk{
			Metadata: &commonpb.ResourceMetadata{
				Id:       "disk-from-env",
				ParentId: "parent-id",
				Name:     "disk-name",
			},
			Spec: &computepb.DiskSpec{
				BlockSizeBytes: 4096,
				Type:           computepb.DiskSpec_NETWORK_SSD,
			},
			Status: &computepb.DiskStatus{
				State: computepb.DiskStatus_READY,
			},
		}, nil
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDiskDataSourceConfig(server.address(), "", "disk-from-env"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.nebius_compute_v1_disk.test", "id", "disk-from-env"),
					resource.TestCheckResourceAttr("data.nebius_compute_v1_disk.test", "name", "disk-name"),
					resource.TestCheckResourceAttr("data.nebius_compute_v1_disk.test", "parent_id", "parent-id"),
					resource.TestCheckResourceAttr("data.nebius_compute_v1_disk.test", "type", "NETWORK_SSD"),
				),
			},
		},
	})
}

func TestAccComputeDiskDataSource_ProviderTokenOverridesEnvAndNoCredentials(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	t.Setenv("NEBIUS_IAM_TOKEN", "env-token")
	testAccPreCheck(t)

	server := newTestDiskServer(t)
	server.getFn = func(ctx context.Context, req *computepb.GetDiskRequest) (*computepb.Disk, error) {
		require.Equal(t, "disk-auth", req.GetId())

		return &computepb.Disk{
			Metadata: &commonpb.ResourceMetadata{
				Id:       "disk-auth",
				ParentId: "parent-id",
				Name:     "disk-name",
			},
			Spec: &computepb.DiskSpec{
				BlockSizeBytes: 4096,
				Type:           computepb.DiskSpec_NETWORK_SSD,
			},
			Status: &computepb.DiskStatus{
				State: computepb.DiskStatus_READY,
			},
		}, nil
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDiskDataSourceConfig(server.address(), `token = "config-token"`, "disk-auth"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.nebius_compute_v1_disk.test", "id", "disk-auth"),
					func(_ *terraform.State) error {
						require.Equal(t, []string{"Bearer config-token"}, server.lastAuthorization())
						return nil
					},
				),
			},
			{
				Config: testAccDiskDataSourceConfig(server.address(), "no_credentials = true", "disk-auth"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.nebius_compute_v1_disk.test", "id", "disk-auth"),
					func(_ *terraform.State) error {
						require.Empty(t, server.lastAuthorization())
						return nil
					},
				),
			},
		},
	})
}

func TestAccComputeDiskResource_BasicLifecycle(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	testAccPreCheck(t)

	server := newTestDiskServer(t)
	server.createFn = func(ctx context.Context, req *computepb.CreateDiskRequest) (*commonpb.Operation, error) {
		require.Empty(t, authorizationValues(ctx))
		require.Equal(t, "disk-under-test", req.GetMetadata().GetName())
		require.Equal(t, "parent-id", req.GetMetadata().GetParentId())
		require.Equal(t, int64(4096), req.GetSpec().GetBlockSizeBytes())
		require.Equal(t, computepb.DiskSpec_NETWORK_SSD, req.GetSpec().GetType())

		server.setDisk(&computepb.Disk{
			Metadata: &commonpb.ResourceMetadata{
				Id:       "disk-id-1",
				ParentId: "parent-id",
				Name:     "disk-under-test",
			},
			Spec: &computepb.DiskSpec{
				BlockSizeBytes: 4096,
				Type:           computepb.DiskSpec_NETWORK_SSD,
			},
			Status: &computepb.DiskStatus{
				State: computepb.DiskStatus_READY,
			},
		})

		return completedOperation("create-disk", "disk-id-1"), nil
	}
	server.getFn = func(_ context.Context, req *computepb.GetDiskRequest) (*computepb.Disk, error) {
		require.Equal(t, "disk-id-1", req.GetId())
		return server.disk(), nil
	}
	server.deleteFn = func(_ context.Context, req *computepb.DeleteDiskRequest) (*commonpb.Operation, error) {
		require.Equal(t, "disk-id-1", req.GetId())
		server.markDeleted(req.GetId())
		return completedOperation("delete-disk", req.GetId()), nil
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDiskResourceConfig(server.address(), ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nebius_compute_v1_disk.test", "id", "disk-id-1"),
					resource.TestCheckResourceAttr("nebius_compute_v1_disk.test", "name", "disk-under-test"),
					resource.TestCheckResourceAttr("nebius_compute_v1_disk.test", "parent_id", "parent-id"),
					resource.TestCheckResourceAttr("nebius_compute_v1_disk.test", "type", "NETWORK_SSD"),
					resource.TestCheckResourceAttr("nebius_compute_v1_disk.test", "status.state", "READY"),
				),
			},
		},
	})

	require.Equal(t, []string{"disk-id-1"}, server.deletedIDs())
}

func TestAccComputeDiskResource_CreateError(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	testAccPreCheck(t)

	server := newTestDiskServer(t)
	server.createFn = func(_ context.Context, req *computepb.CreateDiskRequest) (*commonpb.Operation, error) {
		require.Equal(t, "disk-under-test", req.GetMetadata().GetName())
		return nil, grpcstatus.Error(codes.Internal, "test error")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccDiskResourceConfig(server.address(), ""),
				ExpectError: regexp.MustCompile(`(?s)resource creation failed.*rpc error: code = Internal desc =\s*test error`),
			},
		},
	})
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
}

func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"nebius": func() (tfprotov6.ProviderServer, error) {
			return providerserver.NewProtocol6WithError(providerimpl.New()())()
		},
	}
}

func testAccDiskDataSourceConfig(addr, providerSettings, diskID string) string {
	return testAccProviderConfig(addr, providerSettings) + fmt.Sprintf(`
data "nebius_compute_v1_disk" "test" {
  id = %q
}
`, diskID)
}

func testAccDiskResourceConfig(addr, providerSettings string) string {
	return testAccProviderConfig(addr, providerSettings) + `
resource "nebius_compute_v1_disk" "test" {
  name             = "disk-under-test"
  parent_id        = "parent-id"
  block_size_bytes = 4096
  type             = "NETWORK_SSD"
}
`
}

func testAccProviderConfig(addr, providerSettings string) string {
	return fmt.Sprintf(`
provider "nebius" {
  address_options = {
    "*" = {
      insecure = true
    }
  }

  resolvers = {
    "nebius.compute.*" = %q
  }

  %s
}
`, addr, providerSettings)
}

func completedOperation(operationID, resourceID string) *commonpb.Operation {
	return &commonpb.Operation{
		Id:         operationID,
		ResourceId: resourceID,
		CreatedAt:  timestamppb.Now(),
		FinishedAt: timestamppb.Now(),
		Status: &status.Status{
			Code: int32(codes.OK),
		},
	}
}

type testDiskServer struct {
	t *testing.T

	listener   net.Listener
	grpcServer *grpc.Server

	mu            sync.Mutex
	lastAuth      []string
	diskValue     *computepb.Disk
	deletedDiskID []string

	getFn    func(context.Context, *computepb.GetDiskRequest) (*computepb.Disk, error)
	createFn func(context.Context, *computepb.CreateDiskRequest) (*commonpb.Operation, error)
	deleteFn func(context.Context, *computepb.DeleteDiskRequest) (*commonpb.Operation, error)

	computepb.UnimplementedDiskServiceServer
}

func newTestDiskServer(t *testing.T) *testDiskServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &testDiskServer{
		t:        t,
		listener: listener,
	}
	server.grpcServer = grpc.NewServer()
	computepb.RegisterDiskServiceServer(server.grpcServer, server)

	go func() {
		_ = server.grpcServer.Serve(listener)
	}()

	t.Cleanup(func() {
		server.grpcServer.Stop()
		_ = listener.Close()
	})

	return server
}

func (s *testDiskServer) address() string {
	return s.listener.Addr().String()
}

func (s *testDiskServer) lastAuthorization() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.lastAuth...)
}

func (s *testDiskServer) setDisk(disk *computepb.Disk) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.diskValue = disk
}

func (s *testDiskServer) disk() *computepb.Disk {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.diskValue
}

func (s *testDiskServer) markDeleted(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deletedDiskID = append(s.deletedDiskID, id)
}

func (s *testDiskServer) deletedIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.deletedDiskID...)
}

func (s *testDiskServer) Get(ctx context.Context, req *computepb.GetDiskRequest) (*computepb.Disk, error) {
	s.recordAuthorization(ctx)

	if s.getFn != nil {
		return s.getFn(ctx, req)
	}

	s.t.Fatalf("unexpected Get request: %#v", req)
	return nil, nil
}

func (s *testDiskServer) Create(ctx context.Context, req *computepb.CreateDiskRequest) (*commonpb.Operation, error) {
	s.recordAuthorization(ctx)

	if s.createFn != nil {
		return s.createFn(ctx, req)
	}

	s.t.Fatalf("unexpected Create request: %#v", req)
	return nil, nil
}

func (s *testDiskServer) Delete(ctx context.Context, req *computepb.DeleteDiskRequest) (*commonpb.Operation, error) {
	s.recordAuthorization(ctx)

	if s.deleteFn != nil {
		return s.deleteFn(ctx, req)
	}

	s.t.Fatalf("unexpected Delete request: %#v", req)
	return nil, nil
}

func (s *testDiskServer) recordAuthorization(ctx context.Context) {
	md, _ := grpcmetadata.FromIncomingContext(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastAuth = append([]string(nil), md.Get("authorization")...)
}

func authorizationValues(ctx context.Context) []string {
	md, _ := grpcmetadata.FromIncomingContext(ctx)
	return md.Get("authorization")
}
