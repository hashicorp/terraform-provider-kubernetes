// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	batchapi "k8s.io/api/batch/v1"
	coreapi "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

// This is a local HTTP/protocol fixture, not a Kubernetes conformance server.
// It uses client-go's wire decoder (including protobuf) and the minimal defaults
// below, and deliberately implements no scheduling, admission, or controller.
type batchCLIAPI struct {
	server        *httptest.Server
	mu            sync.Mutex
	jobs          map[string]*batchapi.Job
	crons         map[string]*batchapi.CronJob
	calls         map[string]int
	serial        int
	rv            int
	denied        string
	jobFails      bool
	expireZeroTTL bool
	admitComputed bool
}

func newBatchCLIAPI(t *testing.T) *batchCLIAPI {
	t.Helper()
	api := &batchCLIAPI{
		jobs:  make(map[string]*batchapi.Job),
		crons: make(map[string]*batchapi.CronJob),
		calls: make(map[string]int),
	}
	api.server = httptest.NewServer(http.HandlerFunc(api.serveHTTP))
	t.Cleanup(func() {
		api.server.Close()
		api.mu.Lock()
		defer api.mu.Unlock()
		t.Logf("local API calls: %v; final objects: jobs=%d cronjobs=%d", api.calls, len(api.jobs), len(api.crons))
		if len(api.jobs)+len(api.crons) != 0 {
			t.Errorf("local API leaked objects: jobs=%d cronjobs=%d", len(api.jobs), len(api.crons))
		}
	})
	return api
}

func (api *batchCLIAPI) serveHTTP(w http.ResponseWriter, r *http.Request) {
	api.mu.Lock()
	defer api.mu.Unlock()
	const prefix = "/apis/batch/v1/namespaces/default/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		api.calls["unexpected "+r.Method+" "+r.URL.Path]++
		api.status(w, http.StatusNotFound, metav1.StatusReasonNotFound, "unsupported local API endpoint")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, prefix), "/")
	collection := parts[0]
	if (collection != "jobs" && collection != "cronjobs") || len(parts) > 2 {
		api.status(w, http.StatusNotFound, metav1.StatusReasonNotFound, "unsupported local API resource")
		return
	}
	name := ""
	if len(parts) == 2 {
		name = parts[1]
	}
	api.calls[r.Method+" "+collection]++
	if api.denied == r.Method+" "+collection {
		api.status(w, http.StatusForbidden, metav1.StatusReasonForbidden, collection+" is forbidden: local QA permission denied")
		return
	}
	existing := api.object(collection, name)
	switch r.Method {
	case http.MethodGet:
		if name == "" {
			api.list(w, collection)
			return
		}
		if existing == nil {
			api.status(w, http.StatusNotFound, metav1.StatusReasonNotFound, collection+" "+name+" not found")
			return
		}
		if job, ok := existing.(*batchapi.Job); ok && api.expireZeroTTL &&
			job.Spec.TTLSecondsAfterFinished != nil && *job.Spec.TTLSecondsAfterFinished == 0 {
			delete(api.jobs, name)
			api.calls["TTL jobs"]++
			api.status(w, http.StatusNotFound, metav1.StatusReasonNotFound, "completed Job expired before the waiter observed it")
			return
		}
		api.respond(w, http.StatusOK, existing)
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		if r.Method != http.MethodPost && existing == nil {
			api.status(w, http.StatusNotFound, metav1.StatusReasonNotFound, collection+" "+name+" not found")
			return
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			api.status(w, http.StatusBadRequest, metav1.StatusReasonBadRequest, err.Error())
			return
		}
		if r.Method == http.MethodPatch {
			data, err = api.patch(existing, r.Header.Get("Content-Type"), data)
			if err != nil {
				api.status(w, http.StatusBadRequest, metav1.StatusReasonBadRequest, err.Error())
				return
			}
		}
		var object runtime.Object = &batchapi.Job{}
		if collection == "cronjobs" {
			object = &batchapi.CronJob{}
		}
		if _, _, err := clientgoscheme.Codecs.UniversalDeserializer().Decode(data, nil, object); err != nil {
			api.status(w, http.StatusBadRequest, metav1.StatusReasonBadRequest, err.Error())
			return
		}
		metadata := batchCLIMetadata(object)
		if r.Method == http.MethodPost {
			api.serial++
			if metadata.Name == "" {
				if metadata.GenerateName == "" {
					api.status(w, http.StatusUnprocessableEntity, metav1.StatusReasonInvalid, "name or generateName is required")
					return
				}
				metadata.Name = metadata.GenerateName + fmt.Sprintf("%05d", api.serial)
			}
			if api.object(collection, metadata.Name) != nil {
				api.status(w, http.StatusConflict, metav1.StatusReasonAlreadyExists, "object already exists")
				return
			}
			metadata.UID = types.UID(fmt.Sprintf("batch-qa-uid-%06d", api.serial))
			metadata.CreationTimestamp = metav1.NewTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
			metadata.Generation = 1
		} else {
			old := batchCLIMetadata(existing)
			if metadata.Name != name {
				api.status(w, http.StatusBadRequest, metav1.StatusReasonBadRequest, "update cannot change metadata.name")
				return
			}
			if r.Method == http.MethodPut {
				// Both pinned Kubernetes batch strategies allow unconditional
				// updates. Record them so local-provider tests can still require
				// the safer fresh-GET resourceVersion while exercising releases.
				if metadata.ResourceVersion == "" {
					api.calls["unversioned PUT "+collection]++
				} else if metadata.ResourceVersion != old.ResourceVersion {
					api.status(w, http.StatusConflict, metav1.StatusReasonConflict, "resourceVersion must match")
					return
				}
			}
			metadata.UID = old.UID
			metadata.CreationTimestamp = old.CreationTimestamp
			metadata.Generation = old.Generation + 1
		}
		metadata.Namespace = "default"
		api.rv++
		metadata.ResourceVersion = strconv.Itoa(api.rv)
		switch object := object.(type) {
		case *batchapi.Job:
			object.TypeMeta = metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"}
			batchCLIDefaultJob(object)
			api.admitPod(&object.Spec.Template.Spec)
			// The job strategy, not the API defaulter, generates these labels.
			if !*object.Spec.ManualSelector {
				if object.Spec.Selector == nil {
					object.Spec.Selector = &metav1.LabelSelector{}
				}
				if object.Spec.Selector.MatchLabels == nil {
					object.Spec.Selector.MatchLabels = make(map[string]string)
				}
				object.Spec.Selector.MatchLabels["batch.kubernetes.io/controller-uid"] = string(object.UID)
				if object.Spec.Template.Labels == nil {
					object.Spec.Template.Labels = make(map[string]string)
				}
				for key, value := range map[string]string{
					"controller-uid": string(object.UID), "job-name": object.Name,
					"batch.kubernetes.io/controller-uid": string(object.UID), "batch.kubernetes.io/job-name": object.Name,
				} {
					object.Spec.Template.Labels[key] = value
				}
			}
			// A completed object makes the real provider waiter deterministic.
			object.Status = batchapi.JobStatus{
				Succeeded: 1,
				Conditions: []batchapi.JobCondition{{
					Type: batchapi.JobComplete, Status: coreapi.ConditionTrue,
				}},
			}
			if api.jobFails {
				object.Status.Succeeded = 0
				object.Status.Failed = 1
				object.Status.Conditions[0].Type = batchapi.JobFailed
			}
			api.jobs[object.Name] = object.DeepCopy()
		case *batchapi.CronJob:
			object.TypeMeta = metav1.TypeMeta{APIVersion: "batch/v1", Kind: "CronJob"}
			batchCLIDefaultCronJob(object)
			api.admitPod(&object.Spec.JobTemplate.Spec.Template.Spec)
			api.crons[object.Name] = object.DeepCopy()
		}
		code := http.StatusOK
		if r.Method == http.MethodPost {
			code = http.StatusCreated
		}
		api.respond(w, code, object)
	case http.MethodDelete:
		if existing == nil {
			api.status(w, http.StatusNotFound, metav1.StatusReasonNotFound, collection+" "+name+" not found")
			return
		}
		if collection == "jobs" {
			delete(api.jobs, name)
		} else {
			delete(api.crons, name)
		}
		api.respond(w, http.StatusOK, &metav1.Status{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
			Status:   metav1.StatusSuccess, Code: http.StatusOK,
		})
	default:
		api.status(w, http.StatusMethodNotAllowed, metav1.StatusReasonMethodNotAllowed, "unsupported local API verb")
	}
}

func (api *batchCLIAPI) patch(existing runtime.Object, contentType string, patch []byte) ([]byte, error) {
	before, err := json.Marshal(existing)
	if err != nil {
		return nil, err
	}
	switch contentType {
	case string(types.JSONPatchType):
		operations, err := jsonpatch.DecodePatch(patch)
		if err != nil {
			return nil, err
		}
		return operations.Apply(before)
	case string(types.MergePatchType):
		return jsonpatch.MergePatch(before, patch)
	case string(types.StrategicMergePatchType):
		return strategicpatch.StrategicMergePatch(before, patch, existing)
	default:
		return nil, fmt.Errorf("unsupported patch content type %q", contentType)
	}
}

func (api *batchCLIAPI) object(collection, name string) runtime.Object {
	if collection == "jobs" {
		if job := api.jobs[name]; job != nil {
			return job.DeepCopy()
		}
	} else if cron := api.crons[name]; cron != nil {
		return cron.DeepCopy()
	}
	return nil
}

func (api *batchCLIAPI) list(w http.ResponseWriter, collection string) {
	names := make([]string, 0)
	if collection == "jobs" {
		for name := range api.jobs {
			names = append(names, name)
		}
		sort.Strings(names)
		list := &batchapi.JobList{TypeMeta: metav1.TypeMeta{APIVersion: "batch/v1", Kind: "JobList"}}
		for _, name := range names {
			list.Items = append(list.Items, *api.jobs[name].DeepCopy())
		}
		api.respond(w, http.StatusOK, list)
		return
	}
	for name := range api.crons {
		names = append(names, name)
	}
	sort.Strings(names)
	list := &batchapi.CronJobList{TypeMeta: metav1.TypeMeta{APIVersion: "batch/v1", Kind: "CronJobList"}}
	for _, name := range names {
		list.Items = append(list.Items, *api.crons[name].DeepCopy())
	}
	api.respond(w, http.StatusOK, list)
}

func (*batchCLIAPI) respond(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func (api *batchCLIAPI) status(w http.ResponseWriter, code int, reason metav1.StatusReason, message string) {
	api.respond(w, code, &metav1.Status{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
		Status:   metav1.StatusFailure, Code: int32(code), Reason: reason, Message: message,
	})
}

func batchCLIMetadata(object runtime.Object) *metav1.ObjectMeta {
	switch object := object.(type) {
	case *batchapi.Job:
		return &object.ObjectMeta
	case *batchapi.CronJob:
		return &object.ObjectMeta
	default:
		panic(fmt.Sprintf("unsupported batch test object %T", object))
	}
}

// These fixture defaults match the exercised fields in the pinned Kubernetes
// v1.33.6 pkg/apis/{batch,core}/v1/{defaults,zz_generated.defaults}.go.
// Importing those defaulters requires additional production go.mod dependencies.
// CronJob defaulting intentionally does NOT default its embedded JobSpec.
func batchCLIDefaultJob(job *batchapi.Job) {
	if job.Spec.Completions == nil && job.Spec.Parallelism == nil {
		job.Spec.Completions = ptr.To[int32](1)
		job.Spec.Parallelism = ptr.To[int32](1)
	}
	if job.Spec.Parallelism == nil {
		job.Spec.Parallelism = ptr.To[int32](1)
	}
	if job.Spec.BackoffLimit == nil {
		job.Spec.BackoffLimit = ptr.To[int32](6)
		if job.Spec.BackoffLimitPerIndex != nil {
			job.Spec.BackoffLimit = ptr.To[int32](2147483647)
		}
	}
	if len(job.Labels) == 0 && job.Spec.Template.Labels != nil {
		job.Labels = job.Spec.Template.DeepCopy().Labels
	}
	if job.Spec.CompletionMode == nil {
		job.Spec.CompletionMode = ptr.To(batchapi.NonIndexedCompletion)
	}
	if job.Spec.Suspend == nil {
		job.Spec.Suspend = ptr.To(false)
	}
	if job.Spec.ManualSelector == nil {
		job.Spec.ManualSelector = ptr.To(false)
	}
	batchCLIDefaultPod(&job.Spec.Template.Spec)
}

func batchCLIDefaultCronJob(cron *batchapi.CronJob) {
	if cron.Spec.ConcurrencyPolicy == "" {
		cron.Spec.ConcurrencyPolicy = batchapi.AllowConcurrent
	}
	if cron.Spec.Suspend == nil {
		cron.Spec.Suspend = ptr.To(false)
	}
	if cron.Spec.SuccessfulJobsHistoryLimit == nil {
		cron.Spec.SuccessfulJobsHistoryLimit = ptr.To[int32](3)
	}
	if cron.Spec.FailedJobsHistoryLimit == nil {
		cron.Spec.FailedJobsHistoryLimit = ptr.To[int32](1)
	}
	batchCLIDefaultPod(&cron.Spec.JobTemplate.Spec.Template.Spec)
}

func batchCLIDefaultPod(pod *coreapi.PodSpec) {
	if pod.DNSPolicy == "" {
		pod.DNSPolicy = coreapi.DNSClusterFirst
	}
	if pod.RestartPolicy == "" {
		pod.RestartPolicy = coreapi.RestartPolicyAlways
	}
	if pod.SecurityContext == nil {
		pod.SecurityContext = &coreapi.PodSecurityContext{}
	}
	if pod.TerminationGracePeriodSeconds == nil {
		pod.TerminationGracePeriodSeconds = ptr.To[int64](30)
	}
	if pod.SchedulerName == "" {
		pod.SchedulerName = coreapi.DefaultSchedulerName
	}
	for _, containers := range [][]coreapi.Container{pod.Containers, pod.InitContainers} {
		for index := range containers {
			container := &containers[index]
			if container.ImagePullPolicy == "" {
				container.ImagePullPolicy = coreapi.PullIfNotPresent
				image, _, digest := strings.Cut(container.Image, "@")
				last := image[strings.LastIndex(image, "/")+1:]
				if !digest && (!strings.Contains(last, ":") || strings.HasSuffix(last, ":latest")) {
					container.ImagePullPolicy = coreapi.PullAlways
				}
			}
			if container.TerminationMessagePath == "" {
				container.TerminationMessagePath = coreapi.TerminationMessagePathDefault
			}
			if container.TerminationMessagePolicy == "" {
				container.TerminationMessagePolicy = coreapi.TerminationMessageReadFile
			}
		}
	}
}

func (api *batchCLIAPI) admitPod(pod *coreapi.PodSpec) {
	if !api.admitComputed {
		return
	}
	// Explicit admission-style mutations, not Kubernetes template defaults:
	// limits imply requests for Pods, but not Job/CronJob PodTemplates.
	if len(pod.ImagePullSecrets) == 0 {
		pod.ImagePullSecrets = []coreapi.LocalObjectReference{{Name: "admission-registry"}}
	}
	if len(pod.ReadinessGates) == 0 {
		pod.ReadinessGates = []coreapi.PodReadinessGate{{ConditionType: "example.com/admitted"}}
	}
	for _, containers := range [][]coreapi.Container{pod.Containers, pod.InitContainers} {
		for index := range containers {
			if len(containers[index].Resources.Requests) == 0 {
				containers[index].Resources.Requests = coreapi.ResourceList{
					coreapi.ResourceCPU:    apiresource.MustParse("50m"),
					coreapi.ResourceMemory: apiresource.MustParse("8Mi"),
				}
			}
		}
	}
}

func TestBatchCLI_HTTPFixture(t *testing.T) {
	api := newBatchCLIAPI(t)
	client, err := kubernetes.NewForConfig(&rest.Config{Host: api.server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	jobs := client.BatchV1().Jobs("default")
	job, err := jobs.Create(ctx, &batchapi.Job{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "fixture-"},
		Spec: batchapi.JobSpec{Template: coreapi.PodTemplateSpec{Spec: coreapi.PodSpec{
			RestartPolicy: coreapi.RestartPolicyNever,
			Containers:    []coreapi.Container{{Name: "task", Image: "busybox:1.36"}},
		}}},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if job.UID == "" || job.Spec.Selector == nil || job.Spec.Template.Spec.Containers[0].ImagePullPolicy != coreapi.PullIfNotPresent {
		t.Fatalf("Job metadata/defaults were not applied: %#v", job)
	}
	if _, err := jobs.Get(ctx, job.Name, metav1.GetOptions{}); err != nil {
		t.Fatal(err)
	}
	list, err := jobs.List(ctx, metav1.ListOptions{})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("Job list: %#v, %v", list, err)
	}
	job, err = jobs.Patch(ctx, job.Name, types.JSONPatchType,
		[]byte(`[{"op":"replace","path":"/spec/parallelism","value":2}]`), metav1.PatchOptions{})
	if err != nil || *job.Spec.Parallelism != 2 {
		t.Fatalf("Job patch: %#v, %v", job, err)
	}
	uid := job.UID
	*job.Spec.Parallelism = 3
	job, err = jobs.Update(ctx, job, metav1.UpdateOptions{})
	if err != nil || job.UID != uid || *job.Spec.Parallelism != 3 {
		t.Fatalf("Job update: %#v, %v", job, err)
	}
	if err := jobs.Delete(ctx, job.Name, metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	crons := client.BatchV1().CronJobs("default")
	cron, err := crons.Create(ctx, &batchapi.CronJob{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "fixture-"},
		Spec: batchapi.CronJobSpec{
			Schedule: "0 * * * *",
			JobTemplate: batchapi.JobTemplateSpec{Spec: batchapi.JobSpec{
				Template: coreapi.PodTemplateSpec{Spec: coreapi.PodSpec{
					RestartPolicy: coreapi.RestartPolicyNever,
					Containers:    []coreapi.Container{{Name: "task", Image: "busybox:1.36"}},
				}},
			}},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if cron.UID == "" || cron.Spec.ConcurrencyPolicy != batchapi.AllowConcurrent {
		t.Fatalf("CronJob metadata/defaults were not applied: %#v", cron)
	}
	if _, err := crons.Get(ctx, cron.Name, metav1.GetOptions{}); err != nil {
		t.Fatal(err)
	}
	cronList, err := crons.List(ctx, metav1.ListOptions{})
	if err != nil || len(cronList.Items) != 1 {
		t.Fatalf("CronJob list: %#v, %v", cronList, err)
	}
	cron, err = crons.Patch(ctx, cron.Name, types.MergePatchType,
		[]byte(`{"spec":{"schedule":"15 * * * *"}}`), metav1.PatchOptions{})
	if err != nil || cron.Spec.Schedule != "15 * * * *" {
		t.Fatalf("CronJob patch: %#v, %v", cron, err)
	}
	uid = cron.UID
	cron.Spec.Schedule = "30 * * * *"
	cron, err = crons.Update(ctx, cron, metav1.UpdateOptions{})
	if err != nil || cron.UID != uid || cron.Spec.Schedule != "30 * * * *" {
		t.Fatalf("CronJob update: %#v, %v", cron, err)
	}
	if err := crons.Delete(ctx, cron.Name, metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
}
