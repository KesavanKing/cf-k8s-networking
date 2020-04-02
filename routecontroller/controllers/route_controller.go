/*

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

package controllers

import (
	"context"
	"fmt"

	networkingv1alpha1 "code.cloudfoundry.org/cf-k8s-networking/routecontroller/api/v1alpha1"
	"code.cloudfoundry.org/cf-k8s-networking/routecontroller/resourcebuilders"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RouteReconciler reconciles a Route object
type RouteReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.cloudfoundry.org,resources=routes,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.cloudfoundry.org,resources=routes/status,verbs=get;update;patch

func (r *RouteReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("route", req.NamespacedName)

	// your logic goes here
	routes := &networkingv1alpha1.RouteList{}

	// TODO: only act on changes to routes?

	// watch finds a new route or change to a route
	// find all routes that share that fqdn and reconcile the single Virtual Service for that fqdn
	// reconcile the many Services for the route that was created/changed

	_ = r.List(ctx, routes)
	vsb := resourcebuilders.VirtualServiceBuilder{IstioGateways: []string{"foo"}}
	sb := resourcebuilders.ServiceBuilder{}

	virtualservices := vsb.Build(routes)
	services := sb.Build(routes)

	fmt.Printf("\nPrinting Services!!!!!!!\n")
	fmt.Printf("%+v", services)
	fmt.Printf("\nPrinting Virtual Services!!!!!!!\n")
	fmt.Printf("%+v", virtualservices)
	fmt.Printf("\nPrinting Routes!!!!!!!\n")
	fmt.Printf("%+v", routes)
	fmt.Println("\nDONE LISTING ROUTES!!!!!!!")

	return ctrl.Result{}, nil

}

func (r *RouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.Route{}).
		Complete(r)
}
