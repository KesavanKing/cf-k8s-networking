package resourcebuilders

import (
	networkingv1alpha1 "github.com/cf-k8s-networking/routecontroller/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceBuilder struct{}

func (b *ServiceBuilder) Build(routes *networkingv1alpha1.RouteList) []K8sResource {
	resources := []K8sResource{}
	for _, route := range routes.Items {
		for _, s := range routeToServices(route) {
			resources = append(resources, s)
		}
	}
	return resources
}

func routeToServices(route networkingv1alpha1.Route) []Service {
	const httpPortName = "http"
	services := []Service{}
	for _, dest := range route.Spec.Destinations {
		service := Service{
			ApiVersion: "v1",
			Kind:       "Service",
			ObjectMeta: metav1.ObjectMeta{
				Name:        serviceName(dest),
				Namespace:   route.ObjectMeta.Namespace,
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
			Spec: ServiceSpec{
				Selector: dest.Selector.MatchLabels,
				Ports: []ServicePort{
					{
						Port: *dest.Port,
						Name: httpPortName,
					}},
			},
		}
		service.ObjectMeta.Labels["cloudfoundry.org/app_guid"] = dest.App.Guid
		service.ObjectMeta.Labels["cloudfoundry.org/process_type"] = dest.App.Process.Type
		service.ObjectMeta.Labels["cloudfoundry.org/route_guid"] = route.ObjectMeta.Name
		service.ObjectMeta.Annotations["cloudfoundry.org/route-fqdn"] = route.FQDN()
		services = append(services, service)
	}
	return services
}

// TODO: This should probably be replaced with the code from the virtual service builder
// service names cannot start with numbers
// func serviceName(dest models.Destination) string {
// 	return fmt.Sprintf("s-%s", dest.Guid)
// }
