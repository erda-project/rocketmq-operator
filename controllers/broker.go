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
	"fmt"
	"strings"

	rocketmqv1alpha1 "erda.cloud/rocketmq/api/v1alpha1"
	"erda.cloud/rocketmq/pkg/constants"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *RocketMQReconciler) reconcileBroker(ctx context.Context, rocketMQ *rocketmqv1alpha1.RocketMQ) error {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling RocketMQ broker")

	found := &appsv1.StatefulSet{}
	sts := r.getBrokerStatefulSet(rocketMQ)
	err := r.Get(ctx, types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
		err = r.Create(ctx, sts)
		if err != nil {
			logger.Error(err, "Failed to create new StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
			return err
		}
		return nil
	} else if err != nil {
		logger.Error(err, "Failed to get StatefulSet")
		return err
	}

	svc := r.serviceForBroker(rocketMQ)
	err = r.Client.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: rocketMQ.Namespace}, svc)
	if err != nil && errors.IsNotFound(err) {
		err = r.Client.Create(ctx, svc)
		if err != nil {
			logger.Error(err, "Failed to create new Service for Broker", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	size := rocketMQ.Spec.BrokerSpec.Size
	nameServers := getNameServers(rocketMQ.Spec.NameServiceSpec.Name, rocketMQ.Namespace, rocketMQ.Spec.NameServiceSpec.Size)
	nameServiceStr := getNameServiceStr(found)
	resourceDiff := cmp.Diff(found.Spec.Template.Spec.Containers[0].Resources, sts.Spec.Template.Spec.Containers[0].Resources)
	if *found.Spec.Replicas != size || nameServiceStr != strings.Join(nameServers, ";") || resourceDiff != "" {
		sts.Spec.VolumeClaimTemplates = found.Spec.VolumeClaimTemplates
		for i := range sts.Spec.Template.Spec.Containers {
			for j := range sts.Spec.Template.Spec.Containers[i].VolumeMounts {
				sts.Spec.Template.Spec.Containers[i].VolumeMounts[j].Name = found.Spec.Template.Spec.Containers[i].VolumeMounts[j].Name
			}
		}
		logger.Info("Updating StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
		if _, err := r.KubeClientSet.AppsV1().StatefulSets(sts.Namespace).Update(ctx, sts, metav1.UpdateOptions{}); err != nil {
			logger.Error(err, "Failed to update StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
			return err
		}
	}

	return r.updateBrokerStatus(ctx, rocketMQ, found)
}

func (r *RocketMQReconciler) updateBrokerStatus(ctx context.Context, rocketMQ *rocketmqv1alpha1.RocketMQ, sts *appsv1.StatefulSet) error {
	logger := log.FromContext(ctx)
	broker := rocketMQ.Spec.BrokerSpec

	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(labelsForBroker(broker.Name))
	listOps := &client.ListOptions{Namespace: rocketMQ.Namespace, LabelSelector: labelSelector}
	err := r.Client.List(ctx, podList, listOps)
	if err != nil {
		logger.Error(err, "Failed to list pods", "RocketMQ.Namespace", rocketMQ.Namespace, "RocketMQ.Name", rocketMQ.Name)
		return err
	}

	runningBrokers := getRunningNameServersNum(podList.Items)
	status := getStatusFromSts(sts)
	if runningBrokers != rocketMQ.Status.BrokerStatus.Running ||
		status != rocketMQ.Status.BrokerStatus.Status {
		rocketMQ.Status.BrokerStatus.Running = runningBrokers
		rocketMQ.Status.BrokerStatus.Status = status
		err = r.Client.Status().Update(ctx, rocketMQ)
		logger.Info("Updating RocketMQ broker status")
		if err != nil {
			logger.Error(err, "Failed to update RocketMQ broker status")
			return err
		}
	}
	return nil
}

func (r *RocketMQReconciler) serviceForBroker(rocketMQ *rocketmqv1alpha1.RocketMQ) *corev1.Service {
	labels := labelsForBroker(rocketMQ.Spec.BrokerSpec.Name)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rocketMQ.Spec.BrokerSpec.Name,
			Namespace: rocketMQ.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       constants.BrokerVipContainerPortName,
					Port:       constants.BrokerVipContainerPort,
					TargetPort: intstr.FromInt(constants.BrokerVipContainerPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       constants.BrokerMainContainerPortName,
					Port:       constants.BrokerMainContainerPort,
					TargetPort: intstr.FromInt(constants.BrokerMainContainerPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       constants.BrokerHighAvailabilityContainerPortName,
					Port:       constants.BrokerHighAvailabilityContainerPort,
					TargetPort: intstr.FromInt(constants.BrokerHighAvailabilityContainerPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: labels,
		},
	}
	ctrl.SetControllerReference(rocketMQ, svc, r.Scheme)
	return svc
}

func (r *RocketMQReconciler) getBrokerStatefulSet(rocketMQ *rocketmqv1alpha1.RocketMQ) *appsv1.StatefulSet {
	broker := rocketMQ.Spec.BrokerSpec
	ls := labelsForBroker(broker.Name)
	if broker.Labels == nil {
		broker.Labels = make(map[string]string)
	}
	labels := broker.Labels
	for k, v := range ls {
		labels[k] = v
	}

	if strings.EqualFold(broker.VolumeClaimTemplates[0].Name, "") {
		broker.VolumeClaimTemplates[0].Name = uuid.New().String()
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      broker.Name,
			Namespace: rocketMQ.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &broker.Size,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: broker.ServiceAccountName,
					HostNetwork:        broker.HostNetwork,
					Affinity:           broker.Affinity,
					Tolerations:        broker.Tolerations,
					NodeSelector:       broker.NodeSelector,
					PriorityClassName:  broker.PriorityClassName,
					ImagePullSecrets:   broker.ImagePullSecrets,
					Containers: []corev1.Container{
						{
							Resources:       broker.Resources,
							Image:           broker.Image,
							Name:            broker.Name,
							SecurityContext: getBrokerContainerSecurityContext(&broker),
							ImagePullPolicy: broker.ImagePullPolicy,
							Env:             getBrokerEnv(rocketMQ),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: constants.BrokerVipContainerPort,
									Name:          constants.BrokerVipContainerPortName,
								},
								{
									ContainerPort: constants.BrokerMainContainerPort,
									Name:          constants.BrokerMainContainerPortName,
								},
								{
									ContainerPort: constants.BrokerHighAvailabilityContainerPort,
									Name:          constants.BrokerHighAvailabilityContainerPortName,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: constants.LogMountPath,
									Name:      broker.VolumeClaimTemplates[0].Name,
								},
								{
									MountPath: constants.StoreMountPath,
									Name:      broker.VolumeClaimTemplates[0].Name,
								},
							},
						},
					},
					Volumes:         getVolumes(&broker),
					SecurityContext: getPodSecurityContext(&broker),
				},
			},
			VolumeClaimTemplates: getBrokerVolumeClaimTemplates(&broker),
		},
	}
	ctrl.SetControllerReference(rocketMQ, sts, r.Scheme)

	return sts
}

func getNameServiceStr(sts *appsv1.StatefulSet) string {
	envs := sts.Spec.Template.Spec.Containers[0].Env
	var nameServiceStr string
	for _, env := range envs {
		if env.Name == constants.EnvNameServiceAddress {
			nameServiceStr = env.Value
			break
		}
	}
	return nameServiceStr
}

func getVolumes(broker *rocketmqv1alpha1.BrokerSpec) []corev1.Volume {
	switch broker.StorageMode {
	case constants.StorageModeStorageClass:
		return broker.Volumes
	case constants.StorageModeEmptyDir:
		volumes := broker.Volumes
		volumes = append(volumes, corev1.Volume{
			Name: broker.VolumeClaimTemplates[0].Name,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{}},
		})
		return volumes
	case constants.StorageModeHostPath:
		fallthrough
	default:
		volumes := broker.Volumes
		volumes = append(volumes, corev1.Volume{
			Name: broker.VolumeClaimTemplates[0].Name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: broker.HostPath,
				}},
		})
		return volumes
	}
}

func getStatusFromSts(sts *appsv1.StatefulSet) rocketmqv1alpha1.ConditionStatus {
	if sts.Status.ReadyReplicas == *sts.Spec.Replicas {
		return rocketmqv1alpha1.ConditionReady
	}
	return rocketmqv1alpha1.ConditionInitializing
}

func getBrokerEnv(rocketMQ *rocketmqv1alpha1.RocketMQ) []corev1.EnvVar {
	nameServers := getNameServers(rocketMQ.Spec.NameServiceSpec.Name, rocketMQ.Namespace, rocketMQ.Spec.NameServiceSpec.Size)
	broker := rocketMQ.Spec.BrokerSpec
	envMap := make(map[string]string)
	for _, env := range broker.Env {
		envMap[env.Name] = env.Value
	}
	envMap[constants.EnvNameServiceAddress] = strings.Join(nameServers, ";")
	envs := make([]corev1.EnvVar, 0)
	for k, v := range envMap {
		envs = append(envs, corev1.EnvVar{Name: k, Value: v})
	}
	switch {
	case broker.Size == 1:
		envs = append(envs, corev1.EnvVar{
			Name:  "BROKER_NAME",
			Value: "broker-0",
		}, corev1.EnvVar{
			Name:  "CLUSTER_NAME",
			Value: "cluster-0",
		}, corev1.EnvVar{
			Name:  "BROKER_ID",
			Value: "0",
		}, corev1.EnvVar{
			Name:  "BROKER_ROLE",
			Value: "ASYNC_MASTER",
		})
	default:
		clusterNum := int(broker.Size / 2)
		for i := 0; i < clusterNum; i++ {
			for j := 0; j < 2; j++ {
				envPrefix := fmt.Sprintf("N%d", 2*i+j)
				brokerRole := "ASYNC_MASTER"
				if j%2 == 1 {
					brokerRole = "SLAVE"
				}
				envs = append(envs, corev1.EnvVar{Name: fmt.Sprintf("%s_BROKER_NAME", envPrefix), Value: fmt.Sprintf("broker-%d", 2*i+j)})
				envs = append(envs, corev1.EnvVar{Name: fmt.Sprintf("%s_CLUSTER_NAME", envPrefix), Value: fmt.Sprintf("cluster-%d", i)})
				envs = append(envs, corev1.EnvVar{Name: fmt.Sprintf("%s_BROKER_ID", envPrefix), Value: fmt.Sprintf("%d", j%2)})
				envs = append(envs, corev1.EnvVar{Name: fmt.Sprintf("%s_BROKER_ROLE", envPrefix), Value: brokerRole})
			}
		}
	}
	return envs
}

func getBrokerVolumeClaimTemplates(broker *rocketmqv1alpha1.BrokerSpec) []corev1.PersistentVolumeClaim {
	switch broker.StorageMode {
	case constants.StorageModeStorageClass:
		return broker.VolumeClaimTemplates
	case constants.StorageModeEmptyDir, constants.StorageModeHostPath:
		fallthrough
	default:
		return nil
	}
}

func getPodSecurityContext(broker *rocketmqv1alpha1.BrokerSpec) *corev1.PodSecurityContext {
	var securityContext = corev1.PodSecurityContext{}
	if broker.PodSecurityContext != nil {
		securityContext = *broker.PodSecurityContext
	}
	return &securityContext
}

func getBrokerContainerSecurityContext(broker *rocketmqv1alpha1.BrokerSpec) *corev1.SecurityContext {
	var securityContext = corev1.SecurityContext{}
	if broker.ContainerSecurityContext != nil {
		securityContext = *broker.ContainerSecurityContext
	}
	return &securityContext
}

func labelsForBroker(name string) map[string]string {
	return map[string]string{"app": name, "component": "broker"}
}
