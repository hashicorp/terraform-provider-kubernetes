// Package init contains the list of backends that can be initialized and
// basic helper functions for initializing those backends.
package init

import (
	"os"
	"sync"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/hashicorp/terraform/terraform"

	backendAtlas "github.com/hashicorp/terraform/backend/atlas"
	backendLegacy "github.com/hashicorp/terraform/backend/legacy"
	backendLocal "github.com/hashicorp/terraform/backend/local"
	backendRemote "github.com/hashicorp/terraform/backend/remote"
	backendAzure "github.com/hashicorp/terraform/backend/remote-state/azure"
	backendConsul "github.com/hashicorp/terraform/backend/remote-state/consul"
	backendEtcdv3 "github.com/hashicorp/terraform/backend/remote-state/etcdv3"
	backendGCS "github.com/hashicorp/terraform/backend/remote-state/gcs"
	backendInmem "github.com/hashicorp/terraform/backend/remote-state/inmem"
	backendManta "github.com/hashicorp/terraform/backend/remote-state/manta"
	backendS3 "github.com/hashicorp/terraform/backend/remote-state/s3"
	backendSwift "github.com/hashicorp/terraform/backend/remote-state/swift"
)

// backends is the list of available backends. This is a global variable
// because backends are currently hardcoded into Terraform and can't be
// modified without recompilation.
//
// To read an available backend, use the Backend function. This ensures
// safe concurrent read access to the list of built-in backends.
//
// Backends are hardcoded into Terraform because the API for backends uses
// complex structures and supporting that over the plugin system is currently
// prohibitively difficult. For those wanting to implement a custom backend,
// they can do so with recompilation.
var backends map[string]backend.InitFn
var backendsLock sync.Mutex

// Init initializes the backends map with all our hardcoded backends.
func Init(services *disco.Disco) {
	backendsLock.Lock()
	defer backendsLock.Unlock()

	backends = map[string]backend.InitFn{
		// Enhanced backends.
		"local": func() backend.Backend { return backendLocal.New() },
		"remote": func() backend.Backend {
			b := backendRemote.New(services)
			if os.Getenv("TF_FORCE_LOCAL_BACKEND") != "" {
				return backendLocal.NewWithBackend(b)
			}
			return b
		},

		// Remote State backends.
		"atlas":   func() backend.Backend { return backendAtlas.New() },
		"azurerm": func() backend.Backend { return backendAzure.New() },
		"consul":  func() backend.Backend { return backendConsul.New() },
		"etcdv3":  func() backend.Backend { return backendEtcdv3.New() },
		"gcs":     func() backend.Backend { return backendGCS.New() },
		"inmem":   func() backend.Backend { return backendInmem.New() },
		"manta":   func() backend.Backend { return backendManta.New() },
		"s3":      func() backend.Backend { return backendS3.New() },
		"swift":   func() backend.Backend { return backendSwift.New() },

		// Deprecated backends.
		"azure": deprecateBackend(backendAzure.New(),
			`Warning: "azure" name is deprecated, please use "azurerm"`),
	}

	// Add the legacy remote backends that haven't yet been converted to
	// the new backend API.
	backendLegacy.Init(backends)
}

// Backend returns the initialization factory for the given backend, or
// nil if none exists.
func Backend(name string) backend.InitFn {
	backendsLock.Lock()
	defer backendsLock.Unlock()
	return backends[name]
}

// Set sets a new backend in the list of backends. If f is nil then the
// backend will be removed from the map. If this backend already exists
// then it will be overwritten.
//
// This method sets this backend globally and care should be taken to do
// this only before Terraform is executing to prevent odd behavior of backends
// changing mid-execution.
func Set(name string, f backend.InitFn) {
	backendsLock.Lock()
	defer backendsLock.Unlock()

	if f == nil {
		delete(backends, name)
		return
	}

	backends[name] = f
}

// deprecatedBackendShim is used to wrap a backend and inject a deprecation
// warning into the Validate method.
type deprecatedBackendShim struct {
	backend.Backend
	Message string
}

// Validate the Backend then add the deprecation warning.
func (b deprecatedBackendShim) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	warns, errs := b.Backend.Validate(c)
	warns = append(warns, b.Message)
	return warns, errs
}

// DeprecateBackend can be used to wrap a backend to retrun a deprecation
// warning during validation.
func deprecateBackend(b backend.Backend, message string) backend.InitFn {
	// Since a Backend wrapped by deprecatedBackendShim can no longer be
	// asserted as an Enhanced or Local backend, disallow those types here
	// entirely.  If something other than a basic backend.Backend needs to be
	// deprecated, we can add that functionality to schema.Backend or the
	// backend itself.
	if _, ok := b.(backend.Enhanced); ok {
		panic("cannot use DeprecateBackend on an Enhanced Backend")
	}

	if _, ok := b.(backend.Local); ok {
		panic("cannot use DeprecateBackend on a Local Backend")
	}

	return func() backend.Backend {
		return deprecatedBackendShim{
			Backend: b,
			Message: message,
		}
	}
}
