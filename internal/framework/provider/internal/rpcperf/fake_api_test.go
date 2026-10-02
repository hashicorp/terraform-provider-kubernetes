// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rpcperf

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	k8sscheme "k8s.io/client-go/kubernetes/scheme"
)

// fakeAPI is a minimal in-memory Kubernetes API server. It stores created
// objects by request path, returns them on GET, echoes them on PATCH/PUT and
// answers unknown collections with an empty list. It does just enough for the
// provider's create/read/plan paths of the pod-template resources.
type fakeAPI struct {
	mu    sync.Mutex
	store map[string]map[string]any
	seq   int
}

func newFakeAPI() *fakeAPI {
	return &fakeAPI{store: map[string]map[string]any{}}
}

func writeNotFound(w http.ResponseWriter, name string) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"NotFound","code":404,"message":%q}`, name+" not found")
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := r.URL.Path
	segs := strings.Split(strings.Trim(p, "/"), "/")
	ns := indexOf(segs, "namespaces")
	collection := ns < 0 || len(segs) == ns+3
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodPost:
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(r.Header.Get("Content-Type"), "protobuf") {
			if obj, _, err := k8sscheme.Codecs.UniversalDeserializer().Decode(body, nil, nil); err == nil {
				body, _ = json.Marshal(obj)
			}
		}
		var obj map[string]any
		if err := json.Unmarshal(body, &obj); err != nil || obj == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		md, _ := obj["metadata"].(map[string]any)
		if md == nil {
			md = map[string]any{}
			obj["metadata"] = md
		}
		name, _ := md["name"].(string)
		if name == "" {
			f.seq++
			name = fmt.Sprintf("%sx%d", md["generateName"], f.seq)
			md["name"] = name
		}
		if _, ok := md["namespace"]; !ok && ns >= 0 && len(segs) > ns+1 {
			md["namespace"] = segs[ns+1]
		}
		f.seq++
		md["uid"] = fmt.Sprintf("00000000-0000-0000-0000-%012d", f.seq)
		md["resourceVersion"] = fmt.Sprintf("%d", f.seq)
		md["generation"] = 1
		md["creationTimestamp"] = "2026-01-01T00:00:00Z"
		if strings.HasSuffix(p, "/jobs") {
			// The Job controller defaults the selector; emulate it.
			if spec, ok := obj["spec"].(map[string]any); ok && spec["selector"] == nil {
				spec["selector"] = map[string]any{"matchLabels": map[string]any{"batch.kubernetes.io/controller-uid": md["uid"]}}
			}
		}
		if strings.HasSuffix(p, "/pods") {
			obj["status"] = map[string]any{"phase": "Running"}
		}
		f.store[p+"/"+name] = obj
		out, _ := json.Marshal(obj)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(out)
	case http.MethodGet:
		obj, ok := f.store[p]
		if !ok {
			if collection {
				_, _ = w.Write([]byte(`{"kind":"List","apiVersion":"v1","metadata":{},"items":[]}`))
				return
			}
			writeNotFound(w, p)
			return
		}
		out, _ := json.Marshal(obj)
		_, _ = w.Write(out)
	case http.MethodPatch, http.MethodPut:
		obj, ok := f.store[p]
		if !ok {
			writeNotFound(w, p)
			return
		}
		out, _ := json.Marshal(obj)
		_, _ = w.Write(out)
	case http.MethodDelete:
		delete(f.store, p)
		_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Success"}`))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}
