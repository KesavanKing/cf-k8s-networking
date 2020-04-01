package resourcebuilders

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	networkingv1alpha1 "github.com/cf-k8s-networking/routecontroller/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ServiceBuilder", func() {
	It("returns a Service resource for each route destination", func() {
		routes := networkingv1alpha1.RouteList{Items: []networkingv1alpha1.Route{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "route-guid-0",
					Namespace: "workload-namespace",
					Labels: map[string]string{
						"cloudfoundry.org/space_guid": "space-guid-0",
						"cloudfoundry.org/org_guid":   "org-guid-0",
					},
				},
				Spec: networkingv1alpha1.RouteSpec{
					Host: "test0",
					Path: "/path0",
					Url:  "test0.domain0.example.com/path0",
					Domain: networkingv1alpha1.RouteDomain{
						Name:     "domain0.example.com",
						Internal: false,
					},
					Destinations: []networkingv1alpha1.RouteDestination{
						networkingv1alpha1.RouteDestination{
							Guid:   "route-0-destination-guid-0",
							Port:   intPtr(9000),
							Weight: intPtr(91),
							App: networkingv1alpha1.DestinationApp{
								Guid:    "app-guid-0",
								Process: networkingv1alpha1.AppProcess{Type: "process-type-1"},
							},
							Selector: networkingv1alpha1.DestinationSelector{
								MatchLabels: map[string]string{
									"cloudfoundry.org/app_guid":     "app-guid-0",
									"cloudfoundry.org/process_type": "process-type-1",
								},
							},
						},
						networkingv1alpha1.RouteDestination{
							Guid:   "route-0-destination-guid-1",
							Port:   intPtr(9001),
							Weight: intPtr(9),
							App: networkingv1alpha1.DestinationApp{
								Guid:    "app-guid-1",
								Process: networkingv1alpha1.AppProcess{Type: "process-type-1"},
							},
							Selector: networkingv1alpha1.DestinationSelector{
								MatchLabels: map[string]string{
									"cloudfoundry.org/app_guid":     "app-guid-1",
									"cloudfoundry.org/process_type": "process-type-1",
								},
							},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "route-guid-1",
					Namespace: "workload-namespace",
					Labels: map[string]string{
						"cloudfoundry.org/space_guid": "space-guid-1",
						"cloudfoundry.org/org_guid":   "org-guid-1",
					},
				},
				Spec: networkingv1alpha1.RouteSpec{
					Host: "test1",
					Path: "",
					Url:  "test1.domain1.apps.internal",
					Domain: networkingv1alpha1.RouteDomain{
						Name:     "domain1.apps.internal",
						Internal: true,
					},
					Destinations: []networkingv1alpha1.RouteDestination{
						networkingv1alpha1.RouteDestination{
							Guid:   "route-1-destination-guid-0",
							Port:   intPtr(8080),
							Weight: intPtr(100),
							App: networkingv1alpha1.DestinationApp{
								Guid:    "app-guid-2",
								Process: networkingv1alpha1.AppProcess{Type: "process-type-2"},
							},
							Selector: networkingv1alpha1.DestinationSelector{
								MatchLabels: map[string]string{
									"cloudfoundry.org/app_guid":     "app-guid-2",
									"cloudfoundry.org/process_type": "process-type-2",
								},
							},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "route-guid-2",
					Namespace: "workload-namespace",
					Labels: map[string]string{
						"cloudfoundry.org/space_guid": "space-guid-1",
						"cloudfoundry.org/org_guid":   "org-guid-1",
					},
				},
				Spec: networkingv1alpha1.RouteSpec{
					Host: "test0",
					Path: "/some-path",
					Url:  "test0.domain1.example.com/some-path",
					Domain: networkingv1alpha1.RouteDomain{
						Name:     "domain0.example.com",
						Internal: false,
					},
					Destinations: []networkingv1alpha1.RouteDestination{
						networkingv1alpha1.RouteDestination{
							Guid:   "route-2-destination-guid-0",
							Port:   intPtr(8080),
							Weight: intPtr(100),
							App: networkingv1alpha1.DestinationApp{
								Guid:    "app-guid-1",
								Process: networkingv1alpha1.AppProcess{Type: "process-type-1"},
							},
							Selector: networkingv1alpha1.DestinationSelector{
								MatchLabels: map[string]string{
									"cloudfoundry.org/app_guid":     "app-guid-1",
									"cloudfoundry.org/process_type": "process-type-1",
								},
							},
						},
					},
				},
			},
		},
		}

		// networkingv1alpha1.RouteDestination{
		// 	Guid:   "route-0-destination-guid-0",
		// 	Port:   intPtr(9000),
		// 	Weight: intPtr(91),
		// 	// TODO: Selector:
		// 	App: networkingv1alpha1.DestinationApp{
		// 		Guid:    "app-guid-0",
		// 		Process: networkingv1alpha1.AppProcess{Type: "process-type-1"},
		// 	},
		// },
		expectedServices := []K8sResource{
			Service{
				ApiVersion: "v1",
				Kind:       "Service",
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s-route-0-destination-guid-0",
					Namespace: "workload-namespace",
					Labels: map[string]string{
						"cloudfoundry.org/route_guid":   "route-guid-0",
						"cloudfoundry.org/app_guid":     "app-guid-0",
						"cloudfoundry.org/process_type": "process-type-1",
					},
					Annotations: map[string]string{
						"cloudfoundry.org/route-fqdn": "test0.domain0.example.com",
					},
				},
				Spec: ServiceSpec{
					Selector: map[string]string{
						"cloudfoundry.org/app_guid":     "app-guid-0",
						"cloudfoundry.org/process_type": "process-type-1",
					},

					Ports: []ServicePort{
						ServicePort{
							Port: 9000,
							Name: "http",
						},
					},
				},
			},
			Service{
				ApiVersion: "v1",
				Kind:       "Service",
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s-route-0-destination-guid-1",
					Namespace: "workload-namespace",
					Labels: map[string]string{
						"cloudfoundry.org/route_guid":   "route-guid-0",
						"cloudfoundry.org/app_guid":     "app-guid-1",
						"cloudfoundry.org/process_type": "process-type-1",
					},
					Annotations: map[string]string{
						"cloudfoundry.org/route-fqdn": "test0.domain0.example.com",
					},
				},
				Spec: ServiceSpec{
					Selector: map[string]string{
						"cloudfoundry.org/app_guid":     "app-guid-1",
						"cloudfoundry.org/process_type": "process-type-1",
					},

					Ports: []ServicePort{
						ServicePort{
							Port: 9001,
							Name: "http",
						},
					},
				},
			},
			Service{
				ApiVersion: "v1",
				Kind:       "Service",
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s-route-1-destination-guid-0",
					Namespace: "workload-namespace",
					Labels: map[string]string{
						"cloudfoundry.org/route_guid":   "route-guid-1",
						"cloudfoundry.org/app_guid":     "app-guid-2",
						"cloudfoundry.org/process_type": "process-type-2",
					},
					Annotations: map[string]string{
						"cloudfoundry.org/route-fqdn": "test1.domain1.apps.internal",
					},
				},
				Spec: ServiceSpec{
					Selector: map[string]string{
						"cloudfoundry.org/app_guid":     "app-guid-2",
						"cloudfoundry.org/process_type": "process-type-2",
					},

					Ports: []ServicePort{
						ServicePort{
							Port: 8080,
							Name: "http",
						},
					},
				},
			},
			Service{
				ApiVersion: "v1",
				Kind:       "Service",
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s-route-2-destination-guid-0",
					Namespace: "workload-namespace",
					Labels: map[string]string{
						"cloudfoundry.org/route_guid":   "route-guid-2",
						"cloudfoundry.org/app_guid":     "app-guid-1",
						"cloudfoundry.org/process_type": "process-type-1",
					},
					Annotations: map[string]string{
						"cloudfoundry.org/route-fqdn": "test0.domain0.example.com",
					},
				},
				Spec: ServiceSpec{
					Selector: map[string]string{
						"cloudfoundry.org/app_guid":     "app-guid-1",
						"cloudfoundry.org/process_type": "process-type-1",
					},

					Ports: []ServicePort{
						ServicePort{
							Port: 8080,
							Name: "http",
						},
					},
				},
			},
		}

		builder := ServiceBuilder{}

		Expect(builder.Build(&routes)).To(Equal(expectedServices))
	})

	Context("when a route has no destinations", func() {
		It("does not create a Service", func() {
			routes := networkingv1alpha1.RouteList{Items: []networkingv1alpha1.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "route-guid-0",
						Namespace: "workload-namespace",
						Labels: map[string]string{
							"cloudfoundry.org/space_guid": "space-guid-0",
							"cloudfoundry.org/org_guid":   "org-guid-0",
						},
					},
					Spec: networkingv1alpha1.RouteSpec{
						Host: "test0",
						Path: "/path0",
						Url:  "test0.domain0.example.com/path0",
						Domain: networkingv1alpha1.RouteDomain{
							Name:     "domain0.example.com",
							Internal: false,
						},
						Destinations: []networkingv1alpha1.RouteDestination{},
					},
				},
			},
			}

			builder := ServiceBuilder{}
			Expect(builder.Build(&routes)).To(Equal([]K8sResource{}))
		})
	})
})
