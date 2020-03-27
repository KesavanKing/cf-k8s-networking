package resourcebuilders

import (
	"crypto/sha256"
	"errors"
	"fmt"

	appsv1alpha1 "github.com/cf-k8s-networking/routecontroller/api/v1alpha1"
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

func (b *VirtualServiceBuilder) Build(routes *appsv1alpha1.RouteList) []K8sResource {
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

func (b *VirtualServiceBuilder) fqdnToVirtualService(fqdn string, routes []appsv1alpha1.Route) (VirtualService, error) {
	vs := VirtualService{
		ApiVersion: "networking.istio.io/v1alpha3",
		Kind:       "VirtualService",
		ObjectMeta: metav1.ObjectMeta{
			Name:   VirtualServiceName(fqdn),
			Labels: map[string]string{}, // TODO FIXME
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

func validateRoutesForFQDN(routes []appsv1alpha1.Route) error {
	// We are assuming that internal and external routes cannot share an fqdn
	// Cloud Controller should validate and prevent this scenario
	for _, route := range routes {
		if routes[0].Spec.Domain.Internal != route.Spec.Domain.Internal {
			msg := fmt.Sprintf(
				"route guid %s and route guid %s disagree on whether or not the domain is internal",
				routes[0].Guid(),
				route.Guid())
			return errors.New(msg)
		}
	}

	return nil
}

func destinationsForFQDN(fqdn string, routesByFQDN map[string][]appsv1alpha1.Route) []appsv1alpha1.RouteDestination {
	destinations := make([]appsv1alpha1.RouteDestination, 0)
	routes := routesByFQDN[fqdn]
	for _, route := range routes {
		destinations = append(destinations, route.Spec.Destinations...)
	}
	return destinations
}

func groupByFQDN(routes *appsv1alpha1.RouteList) map[string][]appsv1alpha1.Route {
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

func sortRoutes(routes []appsv1alpha1.Route) {
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

func destinationsToHttpRouteDestinations(route appsv1alpha1.Route, destinations []appsv1alpha1.RouteDestination) ([]HTTPRouteDestination, error) {
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
					Set: map[string]string{}, // TODO FIX ME: set labels
				},
			},
		}
		if destination.Weight != nil {
			httpDestination.Weight = destination.Weight
		}
		httpDestinations = append(httpDestinations, httpDestination)
	}
	if len(destinations) > 1 && destinations[0].Weight == nil {
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

func validateWeights(route appsv1alpha1.Route, destinations []appsv1alpha1.RouteDestination) error {
	// Cloud Controller validates these scenarios
	//
	weightSum := 0
	for _, d := range destinations {
		if (d.Weight == nil) != (destinations[0].Weight == nil) {
			msg := fmt.Sprintf(
				"invalid destinations for route %s: weights must be set on all or none",
				route.Guid())
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
			route.Guid())
		return errors.New(msg)
	}
	return nil
}

func intPtr(x int) *int {
	return &x
}

// service names cannot start with numbers
func serviceName(dest appsv1alpha1.RouteDestination) string {
	return fmt.Sprintf("s-%s", dest.Guid())
}
