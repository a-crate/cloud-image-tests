// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package imagetest

import (
	"fmt"
	"net/http"
	"sort"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-image-tests/utils"
	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-daisy/compute"
	"google.golang.org/api/compute/v1"
)

// Return an empty test workflow.
func NewTestWorkflowForUnitTest(name, image, timeout string) *TestWorkflow {
	t := &TestWorkflow{}
	t.Name = name
	t.Image = &compute.Image{}
	t.ImageURL = image
	t.MachineType = &compute.MachineType{}
	t.Project = &compute.Project{}
	t.Zone = &compute.Zone{}
	t.wf = daisy.New()
	t.wf.DefaultTimeout = timeout
	return t
}

func TestAddStartStep(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	if twf.wf == nil {
		t.Fatal("test workflow is malformed")
	}
	step, err := twf.addStartStep("stepname", "vmname")
	if err != nil {
		t.Errorf("failed to add start step to test workflow: %v", err)
	}
	if step.StartInstances == nil {
		t.Fatal("StartInstances step is missing")
	}
	if len(step.StartInstances.Instances) != 1 {
		t.Error("StartInstances step is malformed")
	}
	if step.StartInstances.Instances[0] != "vmname" {
		t.Error("StartInstances step is malformed")
	}
	if stepFromWF, ok := twf.wf.Steps["start-stepname"]; !ok || step != stepFromWF {
		t.Error("Step was not correctly added to workflow")
	}
}

func TestAddStopStep(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	if twf.wf == nil {
		t.Fatal("test workflow is malformed")
	}
	step, err := twf.addStopStep("stepname", "vmname")
	if err != nil {
		t.Errorf("failed to add stop step to test workflow: %v", err)
	}
	if step.StopInstances == nil {
		t.Fatal("StopInstances step is missing")
	}
	if len(step.StopInstances.Instances) != 1 {
		t.Error("StopInstances step is malformed")
	}
	if step.StopInstances.Instances[0] != "vmname" {
		t.Error("StopInstances step is malformed")
	}
	if stepFromWF, ok := twf.wf.Steps["stop-stepname"]; !ok || step != stepFromWF {
		t.Error("step was not correctly added to workflow")
	}
}

func TestCleanTestWorkflow(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	twf.wf.Project = "test-project"
	_, daisyFake, err := daisycompute.NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/projects/%s/aggregated/instances?alt=json&pageToken=&prettyPrint=false", "test-project") {
			fmt.Fprint(w, `{"Items":{"Instances":{"instances":[{"SelfLink": "projects/test-project/zones/test-zone/instances/test-instance-`+twf.wf.ID()+`", "Zone":"test-zone", "Name": "test-instance-`+twf.wf.ID()+`", "Description": "created by Daisy in workflow \"`+twf.wf.ID()+`\""}]}}}`)
		} else if r.Method == "DELETE" && r.URL.String() == fmt.Sprintf("/projects/%s/zones/test-zone/instances/test-instance-"+twf.wf.ID()+"?alt=json&prettyPrint=false", "test-project") {
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"DONE"}`))
		} else if r.Method == "POST" && r.URL.String() == fmt.Sprintf("/projects/%s/zones/test-zone/operations//wait?alt=json&prettyPrint=false", "test-project") {
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"DONE"}`))
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/projects/%s/regions/%s/forwardingRules?alt=json&pageToken=&prettyPrint=false", "test-project", "test-region") {
			fmt.Fprint(w, `{"items":[{"SelfLink": "projects/test-project/regions/test-region/forwardingRules/test-forwarding-rule", "Name": "test-forwarding-rule", "Network": "projects/test-project/global/networks/test-network-`+twf.wf.ID()+`"}]}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/projects/%s/aggregated/disks?alt=json&pageToken=&prettyPrint=false", "test-project") {
			fmt.Fprint(w, `{"items":{"zones/test-zone":{"disks":[{"SelfLink": "projects/test-project/zones/test-zone/disk/test-disk-`+twf.wf.ID()+`", "Zone":"test-zone", "Name": "test-disk-`+twf.wf.ID()+`", "Description": "created by Daisy in workflow \"`+twf.wf.ID()+`\""}]}}}`)
		} else if r.Method == "DELETE" && r.URL.String() == fmt.Sprintf("/projects/%s/zones/test-zone/disks/test-disk-"+twf.wf.ID()+"?alt=json&prettyPrint=false", "test-project") {
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"DONE"}`))
		} else if r.Method == "POST" && r.URL.String() == fmt.Sprintf("/projects/%s/zones/test-zone/operations//wait?alt=json&prettyPrint=false", "test-project") {
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"DONE"}`))
		} else if r.Method == "DELETE" && r.URL.String() == fmt.Sprintf("/projects/%s/regions/test-region/forwardingRules/test-forwarding-rule?alt=json&prettyPrint=false", "test-project") {
			fmt.Fprint(w, `{"Status":"DONE"}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/projects/%s/global/networks?alt=json&pageToken=&prettyPrint=false", "test-project") {
			fmt.Fprint(w, `{"items":[{"SelfLink": "projects/test-project/global/networks/test-network-`+twf.wf.ID()+`", "Name": "test-network-`+twf.wf.ID()+`", "Description": "created by Daisy in workflow \"`+twf.wf.ID()+`\""}]}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/projects/%s/global/firewalls?alt=json&pageToken=&prettyPrint=false", "test-project") {
			fmt.Fprint(w, `{"items":[{"SelfLink": "projects/test-project/global/firewalls/test-firewall", "Network": "projects/test-project/global/networks/test-network-`+twf.wf.ID()+`", "Name": "test-firewall", "Description": "created by Daisy in workflow \"`+twf.wf.ID()+`\""}]}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/projects/%s/aggregated/subnetworks?alt=json&pageToken=&prettyPrint=false", "test-project") {
			fmt.Fprint(w, `{"items":{"regions/test-region":{"subnetworks":[{"Network": "projects/test-project/global/networks/test-network-`+twf.wf.ID()+`","SelfLink": "projects/test-project/regions/test-region/subnetworks/test-subnetwork", "Name": "test-subnetwork", "Region": "test-region", "Description": "created by Daisy in workflow \"`+twf.wf.ID()+`\""}]}}}`)
		} else if r.Method == "DELETE" && r.URL.String() == fmt.Sprintf("/projects/%s/global/firewalls/test-firewall?alt=json&prettyPrint=false", "test-project") {
			fmt.Fprint(w, `{"Status":"DONE"}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/projects/%s/regions/%s/backendServices?alt=json&pageToken=&prettyPrint=false", "test-project", "test-region") {
			fmt.Fprint(w, `{"items":[{"SelfLink": "projects/test-project/regions/testRegion/backendServices/test-backend-service", "Name": "test-backend-service", "Network": "projects/test-project/global/networks/test-network-`+twf.wf.ID()+`"}]}`)
		} else if r.Method == "DELETE" && r.URL.String() == fmt.Sprintf("/projects/%s/regions/test-region/backendServices/test-backend-service?alt=json&prettyPrint=false", "test-project") {
			fmt.Fprint(w, `{"Status":"DONE"}`)
		} else if r.Method == "DELETE" && r.URL.String() == fmt.Sprintf("/projects/%s/global/networks/test-network-"+twf.wf.ID()+"?alt=json&prettyPrint=false", "test-project") {
			fmt.Fprint(w, `{"Status":"DONE"}`)
		} else if r.Method == "DELETE" && r.URL.String() == fmt.Sprintf("/projects/%s/regions/test-region/subnetworks/test-subnetwork?alt=json&prettyPrint=false", "test-project") {
			fmt.Fprint(w, `{"Status":"DONE"}`)
		} else if r.Method == "POST" && r.URL.String() == fmt.Sprintf("/projects/%s/global/operations//wait?alt=json&prettyPrint=false", "test-project") {
			fmt.Fprint(w, `{"Status":"DONE"}`)
		} else if r.Method == "POST" && r.URL.String() == fmt.Sprintf("/projects/%s/regions/test-region/operations//wait?alt=json&prettyPrint=false", "test-project") {
			fmt.Fprint(w, `{"Status":"DONE"}`)
		} else {
			w.WriteHeader(555)
			fmt.Fprint(w, "URL and Method not recognized:", r.Method, r.URL)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	twf.Client = daisyFake
	expect := []string{"projects/test-project/regions/test-region/backendServices/test-backend-service", "projects/test-project/regions/test-region/forwardingRules/test-forwarding-rule", "projects/test-project/global/firewalls/test-firewall", "projects/test-project/global/networks/test-network-" + twf.wf.ID(), "projects/test-project/regions/test-region/subnetworks/test-subnetwork", "projects/test-project/zones/test-zone/disks/test-disk-" + twf.wf.ID(), "projects/test-project/zones/test-zone/instances/test-instance-" + twf.wf.ID()}
	cleaned, errs := cleanTestWorkflow(twf)
	for _, err := range errs {
		t.Errorf("got error from cleanTestWorkflow: %v", err)
	}
	sort.Strings(cleaned)
	sort.Strings(expect)
	if len(cleaned) != len(expect) {
		t.Errorf("unexpected number of cleaned resources, want %d but got %d", len(expect), len(cleaned))
	}
	for i := range cleaned {
		if cleaned[i] != expect[i] {
			t.Errorf("unexpected cleaned resource at position %d, want %s but got %s", i, expect[i], cleaned[i])
		}
	}
}

func TestAddWaitStep(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	if twf.wf == nil {
		t.Fatal("test workflow is malformed")
	}
	step, err := twf.addWaitStep("stepname", "vmname")
	if err != nil {
		t.Errorf("failed to add wait step to test workflow: %v", err)
	}
	if step.WaitForInstancesSignal == nil {
		t.Fatal("WaitForInstancesSignal step is missing")
	}
	instancesSignal := []*daisy.InstanceSignal(*step.WaitForInstancesSignal)
	if len(instancesSignal) != 1 {
		t.Error("waitInstances step is malformed")
	}
	if instancesSignal[0].Name != "vmname" {
		t.Error("waitInstances step is malformed")
	}
	if instancesSignal[0].SerialOutput.SuccessMatch != successMatch {
		t.Error("waitInstances step is malformed")
	}
	if instancesSignal[0].Stopped {
		t.Error("waitInstances step is malformed")
	}
	guestAttributeSignal := instancesSignal[0].GuestAttribute
	if guestAttributeSignal == nil {
		t.Error("no guest attribute wait field was set for wait step")
	}
	if guestAttributeSignal.Namespace != utils.GuestAttributeTestNamespace {
		t.Errorf("wrong guest attribute namespace: got %s, expected %s", guestAttributeSignal.Namespace, utils.GuestAttributeTestNamespace)
	}
	if guestAttributeSignal.KeyName != utils.GuestAttributeTestKey {
		t.Errorf("wrong guest attribute namespace: got %s, expected %s", guestAttributeSignal.KeyName, utils.GuestAttributeTestKey)
	}
	if stepFromWF, ok := twf.wf.Steps["wait-stepname"]; !ok || step != stepFromWF {
		t.Error("step was not correctly added to workflow")
	}
}

// This tests that in the special case where the test reboots and we need results
// from the second boot, the instance signal for the step is correct.
func TestAddWaitRebootGAStep(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	if twf.wf == nil {
		t.Fatal("test workflow is malformed")
	}
	step, err := twf.addWaitRebootGAStep("stepname", "vmname")
	if err != nil {
		t.Errorf("failed to add wait step to test workflow: %v", err)
	}
	if step.WaitForInstancesSignal == nil {
		t.Fatal("WaitForInstancesSignal step is missing")
	}
	instancesSignal := []*daisy.InstanceSignal(*step.WaitForInstancesSignal)
	if len(instancesSignal) != 1 {
		t.Error("waitInstances step is malformed")
	}
	if instancesSignal[0].Name != "vmname" {
		t.Error("waitInstances step is malformed")
	}
	if instancesSignal[0].SerialOutput.SuccessMatch != successMatch {
		t.Error("waitInstances step is malformed")
	}
	if instancesSignal[0].Stopped {
		t.Error("waitInstances step is malformed")
	}
	guestAttributeSignal := instancesSignal[0].GuestAttribute
	if guestAttributeSignal == nil {
		t.Error("no guest attribute wait field was set for wait step")
	}
	if guestAttributeSignal.Namespace != utils.GuestAttributeTestNamespace {
		t.Errorf("wrong guest attribute namespace: got %s, expected %s", guestAttributeSignal.Namespace, utils.GuestAttributeTestNamespace)
	}
	if guestAttributeSignal.KeyName != utils.FirstBootGAKey {
		t.Errorf("wrong guest attribute namespace: got %s, expected %s", guestAttributeSignal.KeyName, utils.FirstBootGAKey)
	}
	if stepFromWF, ok := twf.wf.Steps["wait-stepname"]; !ok || step != stepFromWF {
		t.Error("step was not correctly added to workflow")
	}
}

func TestAddWaitStoppedStep(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	if twf.wf == nil {
		t.Fatal("test workflow is malformed")
	}
	step, err := twf.addWaitStoppedStep("stepname", "vmname")
	if err != nil {
		t.Errorf("failed to add wait step to test workflow: %v", err)
	}
	if step.WaitForInstancesSignal == nil {
		t.Fatal("WaitForInstancesSignal step is missing")
	}
	instancesSignal := []*daisy.InstanceSignal(*step.WaitForInstancesSignal)
	if len(instancesSignal) != 1 {
		t.Error("waitInstances step is malformed")
	}
	if instancesSignal[0].Name != "vmname" {
		t.Error("waitInstances step is malformed")
	}
	if instancesSignal[0].SerialOutput != nil {
		t.Error("waitInstances step is malformed")
	}
	if !instancesSignal[0].Stopped {
		t.Error("waitInstances step is malformed")
	}
	if stepFromWF, ok := twf.wf.Steps["wait-stopped-stepname"]; !ok || step != stepFromWF {
		t.Error("step was not correctly added to workflow")
	}
}

func TestAppendCreateDisksStep(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	if twf.wf == nil {
		t.Fatal("test workflow is malformed")
	}
	step, err := twf.appendCreateDisksStep(&compute.Disk{Name: "diskname"})
	if err != nil {
		t.Errorf("failed to add wait step to test workflow: %v", err)
	}
	if step.CreateDisks == nil {
		t.Fatal("CreateDisks step is missing")
	}
	disks := []*daisy.Disk(*step.CreateDisks)
	if len(disks) != 1 {
		t.Error("CreateDisks step is malformed")
	}
	if disks[0].Name != "diskname" {
		t.Error("CreateDisks step is malformed")
	}
	if disks[0].SourceImage != "image" {
		t.Error("CreateDisks step is malformed")
	}
	stepFromWF, ok := twf.wf.Steps["create-disks"]
	if !ok || step != stepFromWF {
		t.Error("step was not correctly added to workflow")
	}
	step2, err := twf.appendCreateMountDisksStep(&compute.Disk{Name: "diskname2", Type: HyperdiskExtreme, SizeGb: 100})
	if err != nil {
		t.Fatalf("failed to add wait step to test workflow: %v", err)
	}
	if step2 != stepFromWF {
		t.Fatal("CreateDisks step was not appended")
	}
	disks = []*daisy.Disk(*step2.CreateDisks)
	if len(disks) != 2 {
		t.Fatal("CreateDisks step was not appended")
	}
	if disks[1].Name != "diskname2" {
		t.Error("CreateDisks step is malformed")
	}
}

func TestAppendCreateVMStep(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	if twf.wf == nil {
		t.Fatal("test workflow is malformed")
	}
	if _, ok := twf.wf.Steps["create-disks"]; ok {
		t.Fatal("create-disks step already exists")
	}
	daisyInst := &daisy.Instance{}
	daisyInst.Hostname = ""
	step, _, err := twf.appendCreateVMStep([]*compute.Disk{{Name: "vmname"}}, daisyInst)
	if err != nil {
		t.Errorf("failed to add wait step to test workflow: %v", err)
	}
	if step.CreateInstances == nil {
		t.Fatal("CreateDisks step is missing")
	}
	instances := step.CreateInstances.Instances
	if len(instances) != 1 {
		t.Error("CreateInstances step is malformed")
	}
	if instances[0].Name != "vmname" {
		t.Error("CreateInstances step is malformed")
	}
	stepFromWF, ok := twf.wf.Steps["create-vms"]
	if !ok || step != stepFromWF {
		t.Error("step was not correctly added to workflow")
	}
	daisyInst2 := &daisy.Instance{}
	daisyInst2.Hostname = ""
	step2, _, err := twf.appendCreateVMStep([]*compute.Disk{{Name: "vmname2"}}, daisyInst2)
	if err != nil {
		t.Fatalf("failed to add wait step to test workflow: %v", err)
	}
	if step2 != stepFromWF {
		t.Fatal("CreateDisks step was not appended")
	}
	instances = step.CreateInstances.Instances
	if len(instances) != 2 {
		t.Fatal("CreateDisks step was not appended")
	}
	if instances[1].Name != "vmname2" {
		t.Error("CreateInstances step is malformed")
	}
}

func TestAppendCreateVMStepBeta(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	if twf.wf == nil {
		t.Fatal("test workflow is malformed")
	}
	if _, ok := twf.wf.Steps["create-disks"]; ok {
		t.Fatal("create-disks step already exists")
	}
	daisyInst := &daisy.InstanceBeta{}
	daisyInst.Hostname = ""
	step, _, err := twf.appendCreateVMStepBeta([]*compute.Disk{{Name: "vmname"}}, daisyInst)
	if err != nil {
		t.Errorf("failed to add wait step to test workflow: %v", err)
	}
	if step.CreateInstances == nil {
		t.Fatal("CreateDisks step is missing")
	}
	instances := step.CreateInstances.InstancesBeta
	if len(instances) != 1 {
		t.Error("CreateInstances step is malformed")
	}
	if instances[0].Name != "vmname" {
		t.Error("CreateInstances step is malformed")
	}
	stepFromWF, ok := twf.wf.Steps["create-vms"]
	if !ok || step != stepFromWF {
		t.Error("step was not correctly added to workflow")
	}
	daisyInst2 := &daisy.InstanceBeta{}
	daisyInst2.Hostname = ""
	step2, _, err := twf.appendCreateVMStepBeta([]*compute.Disk{{Name: "vmname2"}}, daisyInst2)
	if err != nil {
		t.Fatalf("failed to add wait step to test workflow: %v", err)
	}
	if step2 != stepFromWF {
		t.Fatal("CreateDisks step was not appended")
	}
	instances = step.CreateInstances.InstancesBeta
	if len(instances) != 2 {
		t.Fatal("CreateDisks step was not appended")
	}
	if instances[1].Name != "vmname2" {
		t.Error("CreateInstances step is malformed")
	}
}

func TestAppendCreateVMStepMultipleDisks(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	if twf.wf == nil {
		t.Fatal("test workflow is malformed")
	}
	if _, ok := twf.wf.Steps["create-disks"]; ok {
		t.Fatal("create-disks step already exists")
	}
	daisyInst := &daisy.Instance{}
	daisyInst.Hostname = ""
	daisyInst.MachineType = "n1-standard-1"
	step, instanceFromStep, err := twf.appendCreateVMStep([]*compute.Disk{
		{Name: "vmname"}, {Name: "mountdiskname", Type: PdBalanced}}, daisyInst)
	if err != nil {
		t.Errorf("failed to add wait step to test workflow: %v", err)
	}
	if step.CreateInstances == nil {
		t.Fatal("CreateDisks step is missing")
	}
	instances := step.CreateInstances.Instances
	if len(instances) != 1 {
		t.Error("CreateInstances step is malformed")
	}
	if instances[0].Name != "vmname" {
		t.Error("CreateInstances step is malformed")
	}
	if len(instanceFromStep.Disks) != 2 {
		t.Error("CreateInstances step failed to create multiple disks properly")
	}
	stepFromWF, ok := twf.wf.Steps["create-vms"]
	if !ok || step != stepFromWF {
		t.Error("step was not correctly added to workflow")
	}
}

func TestAppendCreateVMStepCustomHostname(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	if twf.wf == nil {
		t.Fatal("test workflow is malformed")
	}
	if _, ok := twf.wf.Steps["create-disks"]; ok {
		t.Fatal("create-disks step already exists")
	}
	daisyInst := &daisy.Instance{}
	daisyInst.Hostname = "vmname.example.com"
	step, _, err := twf.appendCreateVMStep([]*compute.Disk{{Name: "vmname"}}, daisyInst)
	if err != nil {
		t.Errorf("failed to add wait step to test workflow: %v", err)
	}
	if step.CreateInstances == nil {
		t.Fatal("CreateDisks step is missing")
	}
	instances := step.CreateInstances.Instances
	if len(instances) != 1 {
		t.Error("CreateInstances step is malformed")
	}
	if instances[0].Name != "vmname" {
		t.Error("CreateInstances step is malformed")
	}
	if instances[0].Hostname != "vmname.example.com" {
		t.Error("CreateInstances step is malformed")
	}
}

func TestNewTestWorkflow(t *testing.T) {
	testcases := []struct {
		name                string
		arch                string
		image               string
		imagename           string
		project             string
		zone                string
		x86Shape            string
		arm64Shape          string
		timeout             string
		expectedMachineType string
	}{
		{
			name:                "arm",
			arch:                "ARM64",
			image:               "projects/fake-cloud/global/images/fakeos-v1",
			imagename:           "fakeos-v1",
			project:             "gcp-guest",
			zone:                "us-central1-a",
			x86Shape:            "n1-stanard-1",
			arm64Shape:          "t2a-standard-1",
			timeout:             "30m",
			expectedMachineType: "t2a-standard-1",
		},
		{
			name:                "x86",
			arch:                "X86_64",
			image:               "projects/fake-cloud/global/images/family/fakeos",
			imagename:           "fakeos",
			project:             "gcp-guest",
			zone:                "us-central1-a",
			x86Shape:            "n1-standard-1",
			arm64Shape:          "t2a-standard-1",
			timeout:             "20m",
			expectedMachineType: "n1-standard-1",
		},
		{
			name:                "unspecified arch",
			arch:                "",
			image:               "projects/fake-cloud/global/images/family/fakeos",
			imagename:           "fakeos",
			project:             "gcp-guest",
			zone:                "us-central1-a",
			x86Shape:            "n1-standard-1",
			arm64Shape:          "t2a-standard-1",
			timeout:             "20m",
			expectedMachineType: "n1-standard-1",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			srv, client, err := daisycompute.NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/projects/%s?alt=json&prettyPrint=false", tc.project) {
					fmt.Fprintf(w, `{"Name":"%s"}`, tc.project)
				} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/projects/%s/zones/%s?alt=json&prettyPrint=false", tc.project, tc.zone) {
					fmt.Fprintf(w, `{"Name":"%s"}`, tc.zone)
				} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/projects/%s/zones/%s/machineTypes/%s?alt=json&prettyPrint=false", tc.project, tc.zone, tc.x86Shape) {
					fmt.Fprintf(w, `{"Name":"%s"}`, tc.x86Shape)
				} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/projects/%s/zones/%s/machineTypes/%s?alt=json&prettyPrint=false", tc.project, tc.zone, tc.arm64Shape) {
					fmt.Fprintf(w, `{"Name":"%s"}`, tc.arm64Shape)
				} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s?alt=json&prettyPrint=false", tc.image) {
					fmt.Fprintf(w, `{"Name":"%s", "Architecture":"%s"}`, tc.imagename, tc.arch)
				} else {
					w.WriteHeader(500)
					fmt.Fprintln(w, "URL and Method not recognized:", r.Method, r.URL)
				}
			}))
			if err != nil {
				t.Fatal(err)
			}
			defer srv.Close()
			twf, err := NewTestWorkflow(client, "", tc.name, tc.image, tc.timeout, tc.project, tc.zone, tc.x86Shape, tc.arm64Shape)
			if err != nil {
				t.Fatalf("failed to create test workflow: %v", err)
			}
			if twf.Name != tc.name {
				t.Errorf("unexpected workflow name, want %s got %s", tc.name, twf.Name)
			}
			if twf.Image.Architecture != tc.arch {
				t.Errorf("unexpected image architecture, want %s got %s", tc.arch, twf.Image.Architecture)
			}
			if twf.Image.Name != tc.imagename {
				t.Errorf("unexpected image name, want %s got %s", tc.imagename, twf.Image.Name)
			}
			if twf.Project.Name != tc.project {
				t.Errorf("unexpected project name, want %s got %s", tc.project, twf.Project.Name)
			}
			if twf.Zone.Name != tc.zone {
				t.Errorf("unexpected zone name, want %s got %s", tc.zone, twf.Zone.Name)
			}
			if twf.MachineType.Name != tc.expectedMachineType {
				t.Errorf("unexpected machine type, want %q got %q", twf.MachineType.Name, tc.expectedMachineType)
			}
			if twf.wf.DefaultTimeout != tc.timeout {
				t.Errorf("unexpected workflow timeout, want %v got %v", tc.timeout, twf.wf.DefaultTimeout)
			}
			if len(twf.wf.Steps) > 0 {
				t.Errorf("workflow has initial steps: %v", twf.wf.Steps)
			}
		})
	}
}

func TestGetLastStepForVM(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	if _, err := twf.CreateTestVM("vm"); err != nil {
		t.Errorf("failed to create test vm: %v", err)
	}
	step, err := twf.getLastStepForVM("vm")
	if err != nil {
		t.Errorf("failed to get last step for vm: %v", err)
	}
	if step.WaitForInstancesSignal == nil {
		t.Error("not wait step")
	}
	if twf.wf.Steps["wait-vm"] != step {
		t.Error("not wait-vm step")
	}
}

func TestGetLastStepForVMWhenReboot(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	tvm, err := twf.CreateTestVM("vm")
	if err != nil {
		t.Errorf("failed to create test vm: %v", err)
	}
	if err := tvm.Reboot(); err != nil {
		t.Errorf("failed to reboot: %v", err)
	}
	step, err := twf.getLastStepForVM("vm")
	if err != nil {
		t.Errorf("failed to get last step for vm: %v", err)
	}
	if step.WaitForInstancesSignal == nil {
		t.Error("not wait step")
	}
	if twf.wf.Steps["wait-started-vm-1"] != step {
		t.Error("not wait-started-vm-1 step")
	}
}

func TestGetLastStepForVMWhenMultipleReboot(t *testing.T) {
	twf := NewTestWorkflowForUnitTest("name", "image", "30m")
	tvm, err := twf.CreateTestVM("vm")
	if err != nil {
		t.Errorf("failed to create test vm: %v", err)
	}
	if err := tvm.Reboot(); err != nil {
		t.Errorf("failed to reboot: %v", err)
	}
	if err := tvm.Reboot(); err != nil {
		t.Errorf("failed to reboot: %v", err)
	}
	step, err := twf.getLastStepForVM("vm")
	if err != nil {
		t.Errorf("failed to get last step for vm: %v", err)
	}
	if step.WaitForInstancesSignal == nil {
		t.Error("not wait step")
	}
	if twf.wf.Steps["wait-started-vm-2"] != step {
		t.Error("not wait-started-vm-2 step")
	}
}
