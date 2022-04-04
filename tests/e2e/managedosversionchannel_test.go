/*
Copyright © 2022 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	provv1 "github.com/rancher-sandbox/rancheros-operator/pkg/apis/rancheros.cattle.io/v1"
	fleet "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	http "github.com/rancher-sandbox/ele-testhelpers/http"
	kubectl "github.com/rancher-sandbox/ele-testhelpers/kubectl"

	"github.com/rancher-sandbox/rancheros-operator/tests/catalog"
)

var _ = Describe("ManagedOSVersionChannel e2e tests", func() {
	k := kubectl.New()
	Context("Create ManagedOSVersions from JSON", func() {

		BeforeEach(func() {
			By("Create a ManagedOSVersionChannel")
			ui := catalog.NewManagedOSVersionChannel(
				"testchannel",
				"json",
				map[string]interface{}{},
			)

			Eventually(func() error {
				return k.ApplyYAML("fleet-default", "testchannel", ui)
			}, 2*time.Minute, 2*time.Second).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			kubectl.New().Delete("managedosversionchannel", "-n", "fleet-default", "testchannel")
		})

		It("Creates a list of ManagedOSVersion", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			versions := []provv1.ManagedOSVersion{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v1"},
					Spec: provv1.ManagedOSVersionSpec{
						Version:    "v1",
						Type:       "container",
						MinVersion: "0.0.0",
						Metadata: &fleet.GenericMap{
							Data: map[string]interface{}{
								"upgradeImage": "registry.com/repository/image:v1",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v2"},
					Spec: provv1.ManagedOSVersionSpec{
						Version:    "v2",
						Type:       "container",
						MinVersion: "0.0.0",
						Metadata: &fleet.GenericMap{
							Data: map[string]interface{}{
								"upgradeImage": "registry.com/repository/image:v2",
							},
						},
					},
				},
			}

			b, err := json.Marshal(versions)
			Expect(err).ShouldNot(HaveOccurred())

			http.Server(ctx, bridgeIP+":9999", string(b))

			By("Create a ManagedOSVersionChannel")
			ui := catalog.NewManagedOSVersionChannel(
				"testchannel",
				"json",
				map[string]interface{}{"uri": "http://" + bridgeIP + ":9999"},
			)

			err = k.ApplyYAML("fleet-default", "testchannel", ui)
			Expect(err).ShouldNot(HaveOccurred())

			r, err := kubectl.GetData("fleet-default", "ManagedOSVersionChannel", "testchannel", `jsonpath={.spec.type}`)
			if err != nil {
				fmt.Println(err)
			}

			Expect(string(r)).To(Equal("json"))

			Eventually(func() string {
				r, err := kubectl.GetData("fleet-default", "ManagedOSVersion", "v1", `jsonpath={.spec.metadata.upgradeImage}`)
				if err != nil {
					fmt.Println(err)
				}

				return string(r)
			}, 1*time.Minute, 2*time.Second).Should(
				Equal("registry.com/repository/image:v1"),
			)

			Eventually(func() string {
				r, err := kubectl.GetData("fleet-default", "ManagedOSVersion", "v2", `jsonpath={.spec.metadata.upgradeImage}`)
				if err != nil {
					fmt.Println(err)
				}
				return string(r)
			}, 1*time.Minute, 2*time.Second).Should(
				Equal("registry.com/repository/image:v2"),
			)
		})
	})
})
