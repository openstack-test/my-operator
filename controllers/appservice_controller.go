/*
Copyright 2022 developer.

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
	"encoding/json"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	appv1 "my-operator/api/v1"
	"my-operator/resources"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var oldSpecAnnotation = "old/spec"

// AppServiceReconciler reconciles a AppService object
type AppServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *AppServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = context.Background()
	log := r.Log.WithValues("appservice", req.NamespacedName)

	// 获取appService crd资源
	appService := &appv1.AppService{}
	if err := r.Client.Get(ctx, req.NamespacedName, appService); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// crd 资源标记为删除
	if appService.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}
	log.Info("fetch appservice objects", "appservice", appService)

	// 如果不存在，则创建关联资源; 如果存在，判断是否需要更新
	// 如果需要更新，则直接更新; 如果不需要更新，则正常返回
	oldDeploy := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, req.NamespacedName, oldDeploy); err != nil {
		// deployment 不存在，创建
		if errors.IsNotFound(err) {
			// 创建deployment
			if err := r.Client.Create(ctx, resources.NewDeploy(appService)); err != nil {
				return ctrl.Result{}, err
			}

			// 创建service
			if err := r.Client.Create(ctx, resources.NewService(appService)); err != nil {
				return ctrl.Result{}, err
			}

			// 更新 crd 资源的 Annotations
			data, _ := json.Marshal(appService.Spec)
			if appService.Annotations != nil {
				appService.Annotations["spec"] = string(data)
			} else {
				appService.Annotations = map[string]string{"spec": string(data)}
			}
			if err := r.Client.Update(ctx, appService); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	} else {
		// deployment 存在，更新
		oldSpec := appv1.AppServiceSpec{}
		if err := json.Unmarshal([]byte(appService.Annotations["spec"]), &oldSpec); err != nil {
			return ctrl.Result{}, err
		}

		if !reflect.DeepEqual(appService.Spec, oldSpec) {
			// 更新deployment
			newDeploy := resources.NewDeploy(appService)
			oldDeploy.Spec = newDeploy.Spec
			if err := r.Client.Update(ctx, oldDeploy); err != nil {
				return ctrl.Result{}, err
			}

			// 更新service
			newService := resources.NewService(appService)
			oldService := &corev1.Service{}
			if err := r.Client.Get(ctx, req.NamespacedName, oldService); err != nil {
				return ctrl.Result{}, err
			}
			// 更新 service 必须设置老的 clusterIP
			clusterIP := oldService.Spec.ClusterIP
			oldService.Spec = newService.Spec
			oldService.Spec.ClusterIP = clusterIP
			if err := r.Client.Update(ctx, oldService); err != nil {
				return ctrl.Result{}, err
			}

			// 更新 crd 资源的 Annotations
			data, _ := json.Marshal(appService.Spec)
			if appService.Annotations != nil {
				appService.Annotations["spec"] = string(data)
			} else {
				appService.Annotations = map[string]string{"spec": string(data)}
			}
			if err := r.Client.Update(ctx, appService); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.AppService{}).
		Complete(r)
}
