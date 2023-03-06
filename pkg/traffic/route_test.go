package traffic_test

import (
	"encoding/json"
	"fmt"
	"testing"

	workload "github.com/kcp-dev/kcp/pkg/apis/workload/v1alpha1"

	"github.com/kuadrant/kcp-glbc/pkg/dns"
	"github.com/kuadrant/kcp-glbc/pkg/traffic"
	testSupport "github.com/kuadrant/kcp-glbc/test/support/route"

	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplyTransformsRoute(t *testing.T) {
	cases := []struct {
		Name string
		// OriginalIngress is the ingress as the user created it
		OriginalRoute *routev1.Route
		// ReconciledIngress is the ingress after the controller has done its work and ready to save it
		ReconciledRoute *routev1.Route
		ExpectErr       bool
	}{{
		Name: "test origin spec not modified",
		OriginalRoute: &routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
				Annotations: map[string]string{
					"experimental.status.workload.kcp.io/c1": "",
				},
				Labels: map[string]string{
					"state.workload.kcp.io/c1": "Sync",
				},
			},
			Spec: routev1.RouteSpec{
				Host: "test.com",
				Path: "/",
				To: routev1.RouteTargetReference{
					Kind: "Service",
					Name: "test",
				},
				TLS: &routev1.TLSConfig{
					Termination:   routev1.TLSTerminationEdge,
					Certificate:   "xyz",
					Key:           "xyz",
					CACertificate: "xyz",
				},
			},
		},
		ReconciledRoute: &routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
				Annotations: map[string]string{
					"experimental.status.workload.kcp.io/c1": "",
				},
				Labels: map[string]string{
					"state.workload.kcp.io/c1": "Sync",
				},
			},
			Spec: routev1.RouteSpec{
				Host: "glbc.com",
				Path: "/",
				To: routev1.RouteTargetReference{
					Kind: "Service",
					Name: "test",
				},
				TLS: &routev1.TLSConfig{
					Termination:   routev1.TLSTerminationEdge,
					Certificate:   "abc",
					Key:           "abc",
					CACertificate: "abc",
				},
			},
		},
	}}

	for _, testCase := range cases {
		t.Run(testCase.Name, func(t *testing.T) {
			// take a copy before we apply transforms this will have all the expected changes to the spec
			transformedCopy := testCase.ReconciledRoute.DeepCopy()
			reconciled := traffic.NewRoute(testCase.ReconciledRoute)
			original := traffic.NewRoute(testCase.OriginalRoute)
			// Apply transforms, this will reset the spec to the original once done
			err := reconciled.Transform(original)
			// after the transform is done, we should have the specs of the original and transformed remain the same
			if !equality.Semantic.DeepEqual(testCase.OriginalRoute.Spec, testCase.ReconciledRoute.Spec) {
				t.Fatalf("expected the spec of the orignal and transformed to have remained the same. Expected %v Got %v", testCase.OriginalRoute.Spec, testCase.ReconciledRoute.Spec)
			}
			// we should now have annotations applying the transforms. Validate the transformed spec matches the transform annotations.
			if err := testSupport.ValidateTransformed(transformedCopy.Spec, reconciled); err != nil {
				t.Fatalf("transforms were invalid %s", err)
			}
			if testCase.ExpectErr {
				if err == nil {
					t.Fatalf("expected an error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("did not expect an error but got %v", err)
				}
			}
		})
	}
}

func TestGetDNSTargetsRoute(t *testing.T) {
	var (
		lbHostFmt = "lb%d.example.com"
		//lbIPFmt    = "53.23.2.%d"
		clusterFmt = "c%d"
	)

	var containsTarget = func(targets []dns.Target, target dns.Target) bool {
		for _, t := range targets {
			if equality.Semantic.DeepEqual(t, target) {
				return true
			}
		}
		return false
	}

	cases := []struct {
		Name      string
		Route     func() *routev1.Route
		ExpectErr bool
		Validate  func([]dns.Target) error
	}{{
		Name: "test single cluster host",
		Route: func() *routev1.Route {
			r := &routev1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			}
			status := routev1.RouteStatus{
				Ingress: []routev1.RouteIngress{
					{
						RouterCanonicalHostname: fmt.Sprintf(lbHostFmt, 0),
					},
				},
			}
			routeC1 := routev1.Route{Status: status}
			jsonRouteC1, _ := json.Marshal(routeC1)
			r.Annotations = map[string]string{}
			r.Annotations[workload.InternalSyncerViewAnnotationPrefix+fmt.Sprintf(clusterFmt, 0)] = string(jsonRouteC1)
			return r
		},
		Validate: func(t []dns.Target) error {
			if len(t) != 1 {
				return fmt.Errorf("expected a single dns target but got %d", len(t))
			}
			for i := range t {
				targetCluster := fmt.Sprintf(clusterFmt, i)
				targetHost := fmt.Sprintf(lbHostFmt, i)
				expectedTarget := dns.Target{
					Cluster:    targetCluster,
					TargetType: dns.TargetTypeHost,
					Value:      targetHost,
				}

				if !containsTarget(t, expectedTarget) {
					return fmt.Errorf("dns target %v not present", expectedTarget)
				}
			}
			return nil
		},
	},
		{
			Name: "test multiple cluster host",
			Route: func() *routev1.Route {
				r := &routev1.Route{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				}
				c1status := routev1.RouteStatus{
					Ingress: []routev1.RouteIngress{
						{
							RouterCanonicalHostname: fmt.Sprintf(lbHostFmt, 0),
						},
					},
				}
				c2status := routev1.RouteStatus{
					Ingress: []routev1.RouteIngress{
						{
							RouterCanonicalHostname: fmt.Sprintf(lbHostFmt, 1),
						},
					},
				}
				routeC1 := routev1.Route{Status: c1status}
				jsonRouteC1, _ := json.Marshal(routeC1)
				r.Annotations = map[string]string{}
				r.Annotations[workload.InternalSyncerViewAnnotationPrefix+fmt.Sprintf(clusterFmt, 0)] = string(jsonRouteC1)
				routeC2 := routev1.Route{Status: c2status}
				jsonRouteC2, _ := json.Marshal(routeC2)
				r.Annotations[workload.InternalSyncerViewAnnotationPrefix+fmt.Sprintf(clusterFmt, 1)] = string(jsonRouteC2)
				return r
			},
			Validate: func(t []dns.Target) error {
				if len(t) != 2 {
					return fmt.Errorf("expected a single dns target but got %d", len(t))
				}
				for i := range t {
					targetCluster := fmt.Sprintf(clusterFmt, i)
					targetHost := fmt.Sprintf(lbHostFmt, i)
					expectedTarget := dns.Target{
						Cluster:    targetCluster,
						TargetType: dns.TargetTypeHost,
						Value:      targetHost,
					}

					if !containsTarget(t, expectedTarget) {
						return fmt.Errorf("dns target %v not present", expectedTarget)
					}
				}
				return nil
			},
		}}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ti := traffic.NewRoute(tc.Route())
			targets, err := ti.GetDNSTargets()
			if tc.ExpectErr && err == nil {
				t.Fatalf("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatalf("did not expect an error but got %s ", err)
			}
			t.Log("targets", targets)
			if err := tc.Validate(targets); err != nil {
				t.Fatalf("unable to validate dns targets %s", err)
			}
		})
	}

}
