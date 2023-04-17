/*
 * Copyright 2022.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controllers

import (
	"context"
	"strings"

	rocketmqv1alpha1 "erda.cloud/rocketmq/api/v1alpha1"
	"erda.cloud/rocketmq/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *RocketMQReconciler) reconcileConsole(ctx context.Context, rocketMQ *rocketmqv1alpha1.RocketMQ) error {
	logger := log.FromContext(ctx)

	console := rocketMQ.Spec.ConsoleSpec
	found := &appsv1.Deployment{}
	dep := r.deploymentForConsole(rocketMQ)
	err := r.Get(ctx, types.NamespacedName{Name: console.Name, Namespace: rocketMQ.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			logger.Error(err, "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name, "Error", err)
			return err
		}
		return nil
	} else if err != nil {
		logger.Error(err, "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name, "Error", err)
		return err
	}

	svc := r.serviceForConsole(rocketMQ)
	err = r.Client.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: rocketMQ.Namespace}, svc)
	if err != nil && errors.IsNotFound(err) {
		err = r.Client.Create(ctx, svc)
		if err != nil {
			logger.Error(err, "Failed to create new Service for Console", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return err
		}
	} else if err != nil {
		return err
	}

	status := r.getConsoleStatus(found)
	if status != rocketMQ.Status.ConsoleStatus.Status {
		rocketMQ.Status.ConsoleStatus.Status = status
		err = r.Status().Update(ctx, rocketMQ)
		logger.Info("Update RocketMQ console status")
		if err != nil {
			logger.Error(err, "Failed to update RocketMQ status")
			return err
		}
	}
	return nil
}

func (r *RocketMQReconciler) getConsoleStatus(dep *appsv1.Deployment) rocketmqv1alpha1.ConditionStatus {
	if dep.Status.ReadyReplicas == dep.Status.Replicas {
		return rocketmqv1alpha1.ConditionReady
	}
	return rocketmqv1alpha1.ConditionInitializing
}

func (r *RocketMQReconciler) serviceForConsole(rocketMQ *rocketmqv1alpha1.RocketMQ) *corev1.Service {
	console := rocketMQ.Spec.ConsoleSpec
	ls := labelsForConsole(console.Name)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: rocketMQ.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       constants.ConsoleContainerPort,
					TargetPort: intstr.FromInt(constants.ConsoleContainerPort),
					Name:       constants.ConsoleContainerPortName,
				},
			},
			Selector: ls,
		},
	}

	return svc
}

func (r *RocketMQReconciler) deploymentForConsole(rocketMQ *rocketmqv1alpha1.RocketMQ) *appsv1.Deployment {
	console := rocketMQ.Spec.ConsoleSpec
	ls := labelsForConsole(console.Name)
	if console.Labels == nil {
		console.Labels = make(map[string]string)
	}
	labels := console.Labels
	for k, v := range ls {
		labels[k] = v
	}

	nameServers := getNameServers(rocketMQ.Spec.NameServiceSpec.Name, rocketMQ.Namespace, rocketMQ.Spec.NameServiceSpec.Size)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      console.Name,
			Namespace: rocketMQ.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &[]int32{1}[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Affinity:         console.Affinity,
					NodeSelector:     console.NodeSelector,
					ImagePullSecrets: console.ImagePullSecrets,
					Containers: []corev1.Container{
						{
							Resources:       console.Resources,
							Image:           console.Image,
							Name:            console.Name,
							ImagePullPolicy: console.ImagePullPolicy,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: constants.ConsoleContainerPort,
									Name:          constants.ConsoleContainerPortName,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  constants.EnvNameServiceAddress,
									Value: strings.Join(nameServers, ";"),
								},
							},
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(rocketMQ, deploy, r.Scheme)

	return deploy
}

func labelsForConsole(name string) map[string]string {
	return map[string]string{"app": name, "name_service_cr": name}
}
