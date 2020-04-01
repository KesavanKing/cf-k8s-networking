package resourcebuilders

import (
	"crypto/sha256"
	"errors"
	"fmt"

	networkingv1alpha1 "github.com/cf-k8s-networking/routecontroller/api/v1alpha1"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sort"
)

type K8sResource interface{}

// "mesh" is a special reserved word on Istio VirtualServices
// https://istio.io/docs/reference/config/networking/v1alpha3/virtual-service/#VirtualService
const MeshInternalGateway = "mesh"

// Istio destination weights are percentage based and must sum to 100%
// https://istio.io/docs/concepts/traffic-management/
const IstioExpectedWeight = int(100)

type VirtualServiceBuilder struct {
	IstioGateways []string
}

func (b *VirtualServiceBuilder) Build(routes *networkingv1alpha1.RouteList) []K8sResource {
	resources := []K8sResource{}

	routesForFQDN := groupByFQDN(routes)
	sortedFQDNs := sortFQDNs(routesForFQDN)

	for _, fqdn := range sortedFQDNs {
		destinations := destinationsForFQDN(fqdn, routesForFQDN)
		if len(destinations) != 0 {
			virtualService, err := b.fqdnToVirtualService(fqdn, routesForFQDN[fqdn])
			if err == nil {
				resources = append(resources, virtualService)
			} else {
				log.WithError(err).Errorf("unable to create VirtualService for fqdn '%s'", fqdn)
			}
		}
	}

	return resources
}

// virtual service names cannot contain special characters
func VirtualServiceName(fqdn string) string {
	sum := sha256.Sum256([]byte(fqdn))
	return fmt.Sprintf("vs-%x", sum)
}

func (b *VirtualServiceBuilder) fqdnToVirtualService(fqdn string, routes []networkingv1alpha1.Route) (VirtualService, error) {
	vs := VirtualService{
		ApiVersion: "networking.istio.io/v1alpha3",
		Kind:       "VirtualService",
		ObjectMeta: metav1.ObjectMeta{
			Name:      VirtualServiceName(fqdn),
			Namespace: routes[0].ObjectMeta.Namespace,
			Labels:    map[string]string{},
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

	if routes[0].Spec.Domain.Internal {
		vs.Spec.Gateways = []string{MeshInternalGateway}
	} else {
		vs.Spec.Gateways = b.IstioGateways
	}

	sortRoutes(routes)

	for _, route := range routes {
		if len(route.Spec.Destinations) != 0 {
			istioDestinations, err := destinationsToHttpRouteDestinations(route, route.Spec.Destinations)
			if err != nil {
				return VirtualService{}, err
			}

			istioRoute := HTTPRoute{
				Route: istioDestinations,
			}
			if route.Spec.Path != "" {
				istioRoute.Match = []HTTPMatchRequest{{Uri: HTTPPrefixMatch{Prefix: route.Spec.Path}}}
			}
			vs.Spec.Http = append(vs.Spec.Http, istioRoute)
		}
	}

	return vs, nil
}

func validateRoutesForFQDN(routes []networkingv1alpha1.Route) error {
	for _, route := range routes {
		// We are assuming that internal and external routes cannot share an fqdn
		// Cloud Controller should validate and prevent this scenario
		if routes[0].Spec.Domain.Internal != route.Spec.Domain.Internal {
			msg := fmt.Sprintf(
				"route guid %s and route guid %s disagree on whether or not the domain is internal",
				routes[0].ObjectMeta.Name,
				route.ObjectMeta.Name)
			return errors.New(msg)
		}

		// Guard against two Routes for the same fqdn belonging to different namespaces
		if routes[0].ObjectMeta.Namespace != route.ObjectMeta.Namespace {
			msg := fmt.Sprintf(
				"route guid %s and route guid %s share the same FQDN but have different namespaces",
				routes[0].ObjectMeta.Name,
				route.ObjectMeta.Name)
			return errors.New(msg)
		}
	}

	return nil
}

func destinationsForFQDN(fqdn string, routesByFQDN map[string][]networkingv1alpha1.Route) []networkingv1alpha1.RouteDestination {
	destinations := make([]networkingv1alpha1.RouteDestination, 0)
	routes := routesByFQDN[fqdn]
	for _, route := range routes {
		destinations = append(destinations, route.Spec.Destinations...)
	}
	return destinations
}

func groupByFQDN(routes *networkingv1alpha1.RouteList) map[string][]networkingv1alpha1.Route {
	fqdns := make(map[string][]networkingv1alpha1.Route)
	for _, route := range routes.Items {
		n := route.FQDN()
		fqdns[n] = append(fqdns[n], route)
	}
	return fqdns
}

func sortFQDNs(fqdns map[string][]networkingv1alpha1.Route) []string {
	var fqdnSlice []string
	for fqdn, _ := range fqdns {
		fqdnSlice = append(fqdnSlice, fqdn)
	}
	// Sorting so that the results are stable
	sort.Strings(fqdnSlice)
	return fqdnSlice
}

func sortRoutes(routes []networkingv1alpha1.Route) {
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Spec.Url > routes[j].Spec.Url
	})
}

func cloneLabels(template map[string]string) map[string]string {
	labels := make(map[string]string)
	for k, v := range template {
		labels[k] = v
	}
	return labels
}

func destinationsToHttpRouteDestinations(route networkingv1alpha1.Route, destinations []networkingv1alpha1.RouteDestination) ([]HTTPRouteDestination, error) {
	err := validateWeights(route, destinations)
	if err != nil {
		return nil, err
	}
	httpDestinations := make([]HTTPRouteDestination, 0)
	for _, destination := range destinations {
		httpDestination := HTTPRouteDestination{
			Destination: VirtualServiceDestination{
				Host: serviceName(destination), // comes from service_builder, will add later
			},
			Headers: VirtualServiceHeaders{
				Request: VirtualServiceHeaderOperations{
					Set: map[string]string{
						"CF-App-Id":           destination.App.Guid,
						"CF-App-Process-Type": destination.App.Process.Type,
						"CF-Space-Id":         route.ObjectMeta.Labels["cloudfoundry.org/space_guid"],
						"CF-Organization-Id":  route.ObjectMeta.Labels["cloudfoundry.org/org_guid"],
					},
				},
			},
		}
		if destination.Weight != nil {
			httpDestination.Weight = destination.Weight
		}
		httpDestinations = append(httpDestinations, httpDestination)
	}
	if destinations[0].Weight == nil {
		n := len(destinations)
		for i, _ := range httpDestinations {
			weight := int(IstioExpectedWeight / n)
			if i == 0 {
				// pad the first destination's weight to ensure all weights sum to 100
				remainder := IstioExpectedWeight - n*weight
				weight += remainder
			}
			httpDestinations[i].Weight = intPtr(weight)
		}
	}
	return httpDestinations, nil
}

func validateWeights(route networkingv1alpha1.Route, destinations []networkingv1alpha1.RouteDestination) error {
	// Cloud Controller validates these scenarios
	//
	weightSum := 0
	for _, d := range destinations {
		if (d.Weight == nil) != (destinations[0].Weight == nil) {
			msg := fmt.Sprintf(
				"invalid destinations for route %s: weights must be set on all or none",
				route.ObjectMeta.Name)
			return errors.New(msg)
		}

		if d.Weight != nil {
			weightSum += *d.Weight
		}
	}

	weightsHaveBeenSet := destinations[0].Weight != nil
	if weightsHaveBeenSet && weightSum != IstioExpectedWeight {
		msg := fmt.Sprintf(
			"invalid destinations for route %s: weights must sum up to 100",
			route.ObjectMeta.Name)
		return errors.New(msg)
	}
	return nil
}

func intPtr(x int) *int {
	return &x
}

// service names cannot start with numbers
func serviceName(dest networkingv1alpha1.RouteDestination) string {
	return fmt.Sprintf("s-%s", dest.Guid)
}
