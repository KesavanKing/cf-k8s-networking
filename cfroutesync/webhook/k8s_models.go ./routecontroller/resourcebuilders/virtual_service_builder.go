package resourcebuilder

import (
	appsv1alpha1 "github.com/cf-k8s-networking/routecontroller/api/v1alpha1"
)

type VirtualServiceBuilder struct {
	IstioGateways []string
}

func (b *VirtualServiceBuilder) Build(routes appsv1alpha1.RouteList) []K8sResource {
	resources := []K8sResource{}

	routesForFQDN := groupByFQDN(routes)
	sortedFQDNs := sortFQDNs(routesForFQDN)

	for _, fqdn := range sortedFQDNs {
		destinations := destinationsForFQDN(fqdn, routesForFQDN)
		if len(destinations) != 0 {
			virtualService, err := b.fqdnToVirtualService(fqdn, routesForFQDN[fqdn], template)
			if err == nil {
				resources = append(resources, virtualService)
			} else {
				log.WithError(err).Errorf("unable to create VirtualService for fqdn '%s'", fqdn)
			}
		}
	}

	return resources
}


func groupByFQDN(routes appsv1alpha1.RouteList) map[string][]appsv1alpha1.Route {
	fqdns := make(map[string][]appsv1alpha1.Route)
	for _, route := range routes.Items {
		n := route.FQDN()
		fqdns[n] = append(fqdns[n], route)
	}
	return fqdns
}

func sortFQDNs(fqdns map[string][]appsv1alpha1.Route) []string {
	var fqdnSlice []string
	for fqdn, _ := range fqdns {
		fqdnSlice = append(fqdnSlice, fqdn)
	}
	// Sorting so that the results are stable
	sort.Strings(fqdnSlice)
	return fqdnSlice
}

func destinationsForFQDN(fqdn string, routesByFQDN map[string][]appsv1alpha1.Route) []appsv1alpha1.Destination {
	destinations := make([]appsv1alpha1.RouteDestination, 0)
	routes := routesByFQDN[fqdn]
	for _, route := range routes {
		destinations = append(destinations, route.RouteSpec.Destinations...)
	}
	return destinations
}

func (b *VirtualServiceBuilder) fqdnToVirtualService(fqdn string, routes []appsv1alpha1.Route) (VirtualService, error) {
	vs := VirtualService{
		ApiVersion: "networking.istio.io/v1alpha3",
		Kind:       "VirtualService",
		ObjectMeta: metav1.ObjectMeta{
			Name:   VirtualServiceName(fqdn),
			Labels: cloneLabels(template.ObjectMeta.Labels),
			Annotations: map[string]string{
				"cloudfoundry.org/fqdn": fqdn,
			},
		},
		Spec: VirtualServiceSpec{Hosts: []string{fqdn}},
	}

	err := validateRoutesForFQDN(routes)
	if err != nil {
		return VirtualService{}, err
	}

	if routes[0].Domain.Internal {
		vs.Spec.Gateways = []string{MeshInternalGateway}
	} else {
		vs.Spec.Gateways = b.IstioGateways
	}

	sortRoutes(routes)

	for _, route := range routes {
		if len(route.Destinations) != 0 {
			istioDestinations, err := destinationsToHttpRouteDestinations(route, route.Destinations)
			if err != nil {
				return VirtualService{}, err
			}

			istioRoute := HTTPRoute{
				Route: istioDestinations,
			}
			if route.Path != "" {
				istioRoute.Match = []HTTPMatchRequest{{Uri: HTTPPrefixMatch{Prefix: route.Path}}}
			}
			vs.Spec.Http = append(vs.Spec.Http, istioRoute)
		}
	}

	return vs, nil
}
