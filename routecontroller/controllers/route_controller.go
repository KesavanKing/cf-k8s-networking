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

	networkingv1alpha1 "github.com/cf-k8s-networking/routecontroller/api/v1alpha1"
	"github.com/cf-k8s-networking/routecontroller/resourcebuilders"
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
	_ = r.List(ctx, routes)
	vsb := resourcebuilders.VirtualServiceBuilder{IstioGateways: []string{"foo"}}
	kresources := vsb.Build(routes)

	fmt.Printf("\nPrinting Virtual Services!!!!!!!\n")
	fmt.Printf("%+v", kresources)
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
