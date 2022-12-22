/*
Copyright 2022.

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
	"time"

	"erda.cloud/rocketmq/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	rocketmqv1alpha1 "erda.cloud/rocketmq/api/v1alpha1"
)

// RocketMQReconciler reconciles a RocketMQ object
type RocketMQReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	ExporterImage string
}

//+kubebuilder:rbac:groups=addons.erda.cloud,resources=rocketmqs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=addons.erda.cloud,resources=rocketmqs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=addons.erda.cloud,resources=rocketmqs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the RocketMQ object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *RocketMQReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling RocketMQ")

	// Fetch the RocketMQ instance
	rocketMQ := &rocketmqv1alpha1.RocketMQ{}
	err := r.Client.Get(ctx, req.NamespacedName, rocketMQ)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("RocketMQ resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get RocketMQ")
		return ctrl.Result{}, err
	}

	// set default status condition
	if rocketMQ.Status.ConditionStatus == "" {
		rocketMQ.Status.ConditionStatus = rocketmqv1alpha1.ConditionInitializing
	}

	if err := r.reconcileNameService(ctx, rocketMQ); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileBroker(ctx, rocketMQ); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileConsole(ctx, rocketMQ); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updateStatus(ctx, rocketMQ); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: time.Second * constants.RequeueIntervalInSecond,
	}, nil
}

func getStatusCondition(rocketMQ *rocketmqv1alpha1.RocketMQ) rocketmqv1alpha1.ConditionStatus {
	if rocketMQ.Status.BrokerStatus.Status == rocketmqv1alpha1.ConditionFailed ||
		rocketMQ.Status.NameServiceStatus.Status == rocketmqv1alpha1.ConditionFailed ||
		rocketMQ.Status.ConsoleStatus.Status == rocketmqv1alpha1.ConditionFailed {
		return rocketmqv1alpha1.ConditionFailed
	}
	if rocketMQ.Status.BrokerStatus.Status == rocketmqv1alpha1.ConditionReady &&
		rocketMQ.Status.NameServiceStatus.Status == rocketmqv1alpha1.ConditionReady &&
		rocketMQ.Status.ConsoleStatus.Status == rocketmqv1alpha1.ConditionReady {
		return rocketmqv1alpha1.ConditionReady
	}
	return rocketmqv1alpha1.ConditionInitializing
}

func (r *RocketMQReconciler) updateStatus(ctx context.Context, rocketMQ *rocketmqv1alpha1.RocketMQ) error {
	status := getStatusCondition(rocketMQ)
	if status != rocketMQ.Status.ConditionStatus {
		rocketMQ.Status.ConditionStatus = status
		if err := r.Client.Status().Update(ctx, rocketMQ); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RocketMQReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rocketmqv1alpha1.RocketMQ{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
