package traffic

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/kuadrant/kcp-glbc/pkg/_internal/metadata"
	v1 "github.com/kuadrant/kcp-glbc/pkg/apis/kuadrant/v1"
)

type hostResult struct {
	Status   ReconcileStatus
	Err      error
	Accessor Interface
}

func TestReconcileHost(t *testing.T) {
	generatedHost := "123.test.com"
	accessor := func(rules []networkingv1.IngressRule, tls []networkingv1.IngressTLS) Interface {
		i := &networkingv1.Ingress{
			Spec: networkingv1.IngressSpec{
				Rules: rules,
			},
		}
		i.Spec.TLS = tls

		return &Ingress{
			Ingress:       i,
			generatedHost: generatedHost,
		}
	}

	var buildResult = func(r Reconciler, a Interface) hostResult {
		status, err := r.Reconcile(context.TODO(), a)
		return hostResult{
			Status:   status,
			Err:      err,
			Accessor: a,
		}
	}

	var commonValidation = func(hr hostResult, expectedStatus ReconcileStatus) error {
		if hr.Status != expectedStatus {
			return fmt.Errorf("unexpected status ")
		}
		if hr.Err != nil {
			return fmt.Errorf("unexpected error from Reconcile : %s", hr.Err)
		}
		hostAnnotation := hr.Accessor.GetHCGHost()
		if hostAnnotation == "" {
			return fmt.Errorf("host annotation is missing from dns record")
		}
		if hostAnnotation == "" {
			return fmt.Errorf("expected annotation %s to be set", ANNOTATION_HCG_HOST)
		}
		for _, host := range hr.Accessor.GetHosts() {
			if host != hostAnnotation {
				return fmt.Errorf("expected the host to be set to %s, but got %s", generatedHost, host)
			}
		}
		return nil
	}

	cases := []struct {
		Name     string
		Accessor func() Interface
		Validate func(hr hostResult) error
	}{
		{
			Name: "test custom host replaced with generated managed host",
			Accessor: func() Interface {
				a := accessor([]networkingv1.IngressRule{{
					Host: "api.example.com",
				}}, []networkingv1.IngressTLS{})
				return a
			},
			Validate: func(hr hostResult) error {
				err := commonValidation(hr, ReconcileStatusContinue)
				if err != nil {
					return err
				}
				if !metadata.HasAnnotation(hr.Accessor, ANNOTATION_HCG_CUSTOM_HOST_REPLACED) {
					return fmt.Errorf("expected the custom host annotation to be present")
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler := &HostReconciler{
				GetDomainVerifications: func(ctx context.Context, accessor Interface) (*v1.DomainVerificationList, error) {
					return &v1.DomainVerificationList{}, nil
				},
			}

			if err := tc.Validate(buildResult(reconciler, tc.Accessor())); err != nil {
				t.Fatalf("fail: %s", err)
			}

		})
	}
}

func TestProcessCustomHostValidation(t *testing.T) {
	generatedHost := "generated.host.net"

	testCases := []struct {
		name                string
		accessor            Interface
		domainVerifications *v1.DomainVerificationList
		expectedRules       []networkingv1.IngressRule
		expectedTLS         []networkingv1.IngressTLS
	}{
		{
			name: "Empty host",
			accessor: &Ingress{
				generatedHost: generatedHost,
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ingress",
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path: "/",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			domainVerifications: &v1.DomainVerificationList{},
			expectedRules: []networkingv1.IngressRule{
				{
					Host: "",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/",
								},
							},
						},
					},
				},
				{
					Host: "generated.host.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Custom host verified",
			accessor: &Ingress{
				generatedHost: generatedHost,
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ingress",
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "test.pb-custom.hcpapps.net",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path: "/path",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			domainVerifications: &v1.DomainVerificationList{
				Items: []v1.DomainVerification{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pb-custom.hcpapps.net",
						},
						Spec: v1.DomainVerificationSpec{
							Domain: "pb-custom.hcpapps.net",
						},
						Status: v1.DomainVerificationStatus{
							Verified: true,
						},
					},
				},
			},
			expectedRules: []networkingv1.IngressRule{
				{
					Host: "test.pb-custom.hcpapps.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/path",
								},
							},
						},
					},
				},
				{
					Host: "generated.host.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/path",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "subdomain of verifiied custom host",
			accessor: &Ingress{
				generatedHost: generatedHost,
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ingress",
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "sub.test.pb-custom.hcpapps.net",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path: "/path",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			domainVerifications: &v1.DomainVerificationList{
				Items: []v1.DomainVerification{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pb-custom.hcpapps.net",
						},
						Spec: v1.DomainVerificationSpec{
							Domain: "pb-custom.hcpapps.net",
						},
						Status: v1.DomainVerificationStatus{
							Verified: true,
						},
					},
				},
			},
			expectedRules: []networkingv1.IngressRule{
				{
					Host: "sub.test.pb-custom.hcpapps.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/path",
								},
							},
						},
					},
				},
				{
					Host: "generated.host.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/path",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Custom host unverified",
			accessor: &Ingress{
				generatedHost: generatedHost,
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ingress",
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "test.pb-custom.hcpapps.net",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path: "/path",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			domainVerifications: &v1.DomainVerificationList{
				Items: []v1.DomainVerification{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pb-custom.hcpapps.net",
						},
						Spec: v1.DomainVerificationSpec{
							Domain: "pb-custom.hcpapps.net",
						},
						Status: v1.DomainVerificationStatus{
							Verified: false,
						},
					},
				},
			},
			expectedRules: []networkingv1.IngressRule{
				{
					Host: "generated.host.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/path",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Unverfied host TLS section is removed",
			accessor: &Ingress{
				generatedHost: generatedHost,
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ingress",
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{
							{
								Hosts: []string{
									"test.pb-custom.hcpapps.net",
								},
								SecretName: "tls-secret",
							},
							{
								Hosts: []string{
									"generated.host.net",
								},
								SecretName: "tls-secret",
							},
						},
						Rules: []networkingv1.IngressRule{
							{
								Host: "test.pb-custom.hcpapps.net",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path: "/path",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			domainVerifications: &v1.DomainVerificationList{
				Items: []v1.DomainVerification{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pb-custom.hcpapps.net",
						},
						Spec: v1.DomainVerificationSpec{
							Domain: "pb-custom.hcpapps.net",
						},
						Status: v1.DomainVerificationStatus{
							Verified: false,
						},
					},
				},
			},
			expectedRules: []networkingv1.IngressRule{
				{
					Host: "generated.host.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/path",
								},
							},
						},
					},
				},
			},
			expectedTLS: []networkingv1.IngressTLS{
				{
					Hosts: []string{
						"generated.host.net",
					},
					SecretName: "tls-secret",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ingressAccessor := testCase.accessor.(*Ingress)
			if err := testCase.accessor.ProcessCustomHosts(
				context.TODO(),
				testCase.domainVerifications,
				func(ctx context.Context, i Interface) error {
					return nil
				},
				func(ctx context.Context, i Interface) error {
					return nil
				},
			); err != nil {
				t.Fatal(err)
			}

			// Assert the reconciled rules match the expected rules
			for _, expectedRule := range testCase.expectedRules {
				foundExpectedRule := false
				for _, rule := range ingressAccessor.Spec.Rules {
					if equality.Semantic.DeepEqual(expectedRule, rule) {
						foundExpectedRule = true
						break
					}
				}
				if !foundExpectedRule {
					t.Fatalf("Expected rule not found: %+v", expectedRule)
				}
			}

			for _, expectedTLS := range testCase.expectedTLS {
				foundExpectedTLS := false
				for _, tls := range ingressAccessor.Spec.TLS {
					if equality.Semantic.DeepEqual(expectedTLS, tls) {
						foundExpectedTLS = true
						break
					}
				}

				if !foundExpectedTLS {
					t.Fatalf("Expected TLS not found: %+v", expectedTLS)
				}
			}
		})
	}
}

func TestAddHostAnnotations(t *testing.T) {

	type args struct {
		record metav1.Object
		host   string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test host generated",
			args: args{
				record: &v1.DNSRecord{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddHostAnnotations(tt.args.record, tt.args.host)
		})
		if !metadata.HasAnnotation(tt.args.record, ANNOTATION_HCG_HOST) {
			t.Fatalf("generated host annotation wasn't added")
		}
	}
}
