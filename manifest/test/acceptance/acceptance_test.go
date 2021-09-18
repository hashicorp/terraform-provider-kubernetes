// +build acceptance

package acceptance

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-exec/tfexec"
	tftest "github.com/hashicorp/terraform-plugin-test/v2"
	"k8s.io/client-go/rest"

	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	kuberneteshelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
)

var tfhelper *tftest.Helper
var k8shelper *kuberneteshelper.Helper
var reattachInfo tfexec.ReattachInfo

func TestMain(m *testing.M) {
	var err error
	reattachInfo, err = provider.ServeTest(context.TODO(), hclog.Default())
	if err != nil {
		//lintignore:R009
		panic(err)
	}

	sourceDir, err := os.Getwd()
	if err != nil {
		//lintignore:R009
		panic(err)
	}

	os.Setenv("TF_X_KUBERNETES_MANIFEST_RESOURCE", "true")

	// disables client-go resource deprecation warnings - they polute the test log
	rest.SetDefaultWarningHandler(rest.NoWarnings{})

	tfhelper = tftest.AutoInitProviderHelper(sourceDir)
	defer tfhelper.Close()

	k8shelper = kuberneteshelper.NewHelper()

	flag.Parse()

	rand.Seed(time.Now().UTC().UnixNano())

	exitcode := m.Run()
	os.Exit(exitcode)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz")

// randName does exactly what it sounds like it should do
func randName() string {
	b := make([]rune, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return fmt.Sprintf("tf-acc-test-%s", string(b))
}

// randString does exactly what it sounds like it should do
func randString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// TFVARS is a convenience type for supplying vars to the loadTerraformConfig func
type TFVARS map[string]interface{}

// loadTerraformConfig will read the contents of a terraform config from the testdata directory
// and add the supplied tfvars as variable blocks to the top of the config
func loadTerraformConfig(t *testing.T, filename string, tfvars TFVARS) string {
	tfconfig, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s", filename))
	if err != nil {
		t.Fatal(err)
		return ""
	}

	// FIXME HACK this is something we could probably add to the binary test helper
	// and it can supply the -var flag instead of doing this
	vars := ""
	for name, value := range tfvars {
		// FIXME the %#v directive will only work for primitive types
		// if we want to supply maps and lists from the tests we need
		// to format them correctly here
		vars += fmt.Sprintf(`
variable %q {
	default = %#v
}
`, name, value)
	}

	return vars + string(tfconfig)
}
