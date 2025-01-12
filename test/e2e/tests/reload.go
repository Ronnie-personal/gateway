// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

//go:build e2e
// +build e2e

package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
)

func init() {
	ConformanceTests = append(ConformanceTests, ReloadTest)
}

var ReloadTest = suite.ConformanceTest{
	ShortName:   "Reload",
	Description: "Envoy gateway reload route order",
	Manifests:   []string{"testdata/reload-route-order.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		t.Run("Envoy gateway reload", func(t *testing.T) {
			// Step 1: Start with an initial configuration for the Envoy Proxy

			var namespace = "envoy-gateway-system"

			initialConfig, err := getConfigDump(t, suite.RestConfig, suite.Client, namespace)
			if err != nil {
				t.Log(err)
				t.Fatal(err) // Fail the test and log the error
			}
			var numReloads = 5
			var numCheckingEachReload = 3
			// Channel to communicate between Goroutines
			errCh := make(chan error)

			for i := 0; i < numReloads; i++ {
				// Step 2: Restart or reload the Envoy Gateway in a Goroutine
				go func() {
					err := restartEnvoyGateway(t, suite.Client, namespace)
					if err != nil {
						errCh <- err
						return
					}
					errCh <- nil // Notify that Step 2 is completed
				}()

				// Nested loop to check config dump multiple times during each reload
				for j := 0; j < numCheckingEachReload; j++ {

					// Step 3: Retrieve the `/config_dump` output from the Envoy Proxy admin interface
					newConfigDump, err := getConfigDump(t, suite.RestConfig, suite.Client, namespace)
					if err != nil {
						t.Log(err)
						t.Fatal(err)
					}

					// Step 4: Compare the obtained `/config_dump` output with the initial configuration
					// Define options for comparison
					/*
						sortOpts := cmpopts.SortSlices(func(a, b interface{}) bool {
							// Type assertion to get the map[string]interface{} from the interface{}
							mapA, okA := a.(map[string]interface{})
							mapB, okB := b.(map[string]interface{})

							// If either a or b is not a map or does not have the "endpoint_config" key, return false
							if !okA || !okB {
								return false
							}

							endpointA, okA := mapA["endpoint_config"].(map[string]interface{})
							endpointB, okB := mapB["endpoint_config"].(map[string]interface{})

							// If either a or b does not have the "endpoint_config" key or is not a map, return false
							if !okA || !okB {
								return false
							}

							// Extract the "cluster_name" field from the endpoint_config map and compare the values
							clusterA, okA := endpointA["cluster_name"].(string)
							clusterB, okB := endpointB["cluster_name"].(string)

							// If either clusterA or clusterB is not a string, return false
							if !okA || !okB {
								return false
							}

							return clusterA < clusterB
						}) */

					// Compare the two JSON structures with sorting
					require.Empty(t, cmp.Diff(initialConfig, newConfigDump))
				}

				// Wait for Step 2 to complete before moving to the next reload
				if err := <-errCh; err != nil {
					t.Log(err)
					t.Fatal(err)
				}
				// Step 5: Repeat the above steps for the desired number of reloads
			}
		})
	},
}

func getConfigDump(t *testing.T, config *rest.Config, c client.Client, namespace string) (responseMap map[string]interface{}, err error) {
	selectorLabels := map[string]string{
		"gateway.envoyproxy.io/owning-gateway-name":      "all-namespaces",
		"gateway.envoyproxy.io/owning-gateway-namespace": "gateway-conformance-infra",
	}

	// Create a new PodList to store the matching pods
	podList := &corev1.PodList{}

	// Build the ListOptions with the namespace and selectors
	labelSelector := labels.SelectorFromSet(labels.Set(selectorLabels))
	listOptions := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labelSelector,
	}

	// List the pods using the ListOptions
	err = c.List(context.TODO(), podList, listOptions)
	if err != nil {
		t.Log(err)
		return nil, err
	}

	podName := podList.Items[0].Name

	localPort := 19002
	remotePort := 19000

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		t.Log(err)
		return nil, err
	}

	portForwardURL := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	portForwardURL = fmt.Sprintf("%v%s", config.Host, portForwardURL)

	serverURL, _ := url.Parse(portForwardURL)
	ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, serverURL)
	stopCh := make(chan struct{}, 1)
	readyCh := make(chan struct{})

	// Create a port forwarder
	portForwarder, err := portforward.New(dialer, ports, stopCh, readyCh, os.Stdout, os.Stderr)
	if err != nil {
		t.Log(err)
		return nil, err
	}

	// Start port forwarding
	go func() {
		err := portForwarder.ForwardPorts()
		if err != nil {
			t.Log(err)
		}
	}()

	// Wait until port forwarding is ready
	<-readyCh

	// Output the local address for accessing the forwarded port
	fmt.Printf("Port forwarding started. Access the service locally at: localhost:%d\n", localPort)

	// Perform an HTTP GET request to the forwarded port, reaches to envoy proxy admin
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/config_dump?include_eds", localPort))
	if err != nil {
		t.Log(err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body as a string
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal the response body into a map[string]interface{}
	err = json.Unmarshal(responseBody, &responseMap)
	if err != nil {
		return nil, err
	}

	// Wait for termination signal
	// <-stopCh

	portForwarder.Close()
	return responseMap, nil
}

func restartEnvoyGateway(t *testing.T, c client.Client, namespace string) (err error) {
	// Get the pod with the selector '-selector=control-plane=envoy-gateway'
	podList := &corev1.PodList{}
	err = c.List(context.TODO(), podList, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(labels.Set{"control-plane": "envoy-gateway"}),
	})
	if err != nil {
		t.Log(err)
		return err
	}
	var previousGatewayPodName string
	// Delete the first pod from the list
	if len(podList.Items) > 0 {
		pod := podList.Items[0]
		previousGatewayPodName = pod.Name
		err = c.Delete(context.TODO(), &pod)
		if err != nil {
			t.Log(err)
			return err
		}

		fmt.Printf("Deleting envoy gateway pod: %s\n", previousGatewayPodName)
	} else {
		fmt.Println("No pods found with the selector 'control-plane=envoy-gateway'")
	}

	timeout := 3 * time.Minute //  set the timeout duration
	startTime := time.Now()    // Store the start time
	// Check if another pod with the same selector comes back up and running
	for {
		podList := &corev1.PodList{}
		err = c.List(context.TODO(), podList, &client.ListOptions{
			Namespace:     namespace,
			LabelSelector: labels.SelectorFromSet(labels.Set{"control-plane": "envoy-gateway"}),
		})
		if err != nil {
			t.Log(err)
			return err
		}

		if len(podList.Items) > 0 {
			pod := podList.Items[0]
			if pod.Status.Phase == corev1.PodRunning && pod.Name != previousGatewayPodName {
				fmt.Printf("Envoy gateway pod %s is deleted. New pod %s is running\n\r", previousGatewayPodName, pod.Name)
				break
			} else {
				fmt.Printf("Envoy gateway pod %s is being deleted......\n", previousGatewayPodName)
			}
		}

		time.Sleep(2 * time.Second)

		// Check if the timeout duration has exceeded
		if time.Since(startTime) >= timeout {
			fmt.Println("Timeout exceeded. Pod did not come upt with the specified time.")
			return errors.New("timeout in gateway restart")
		}
	}
	return nil
}
