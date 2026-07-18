package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Acceptance tests hit a REAL Bbox router. They're gated by TF_ACC=1 (the
// terraform-plugin-testing convention) so `go test ./...` never triggers them.
//
// To run:
//
//	TF_ACC=1 \
//	BBOX_BASE_URL=https://mabbox.bytel.fr \
//	BBOX_PASSWORD_FILE=$HOME/.bbox-password \
//	go test ./internal/provider/... -run TestAcc -v -timeout 30m
//
// Assumes the current MAP-T range covers 40080. Adjust external_port to a free
// port inside your router's range if that isn't valid — the check is server-
// side and there's no way to know it offline.
//
// This test is a template. Copy it for other resources once you're ready to
// gate them on your live setup.
func TestAccNATRule_lifecycle(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test (real router required)")
	}
	const cfg = `
resource "bbox_nat_rule" "acc" {
  name          = "tf-provider-acc-test"
  external_port = 40080
  target_ip     = "192.168.1.254"
  protocol      = "tcp"
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{{
			Config: cfg,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet("bbox_nat_rule.acc", "id"),
				resource.TestCheckResourceAttr("bbox_nat_rule.acc", "protocol", "tcp"),
				resource.TestCheckResourceAttrWith("bbox_nat_rule.acc", "id", func(v string) error {
					if v == "" || v == "0" {
						return fmt.Errorf("expected non-zero router-assigned id, got %q", v)
					}
					return nil
				}),
			),
		}},
	})
}
