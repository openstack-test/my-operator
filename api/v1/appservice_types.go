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

package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AppServiceSpec定义了AppService的期望状态
type AppServiceSpec struct {
	Replicas  *int32                      `json:"replicas"`            // 副本数
	Image     string                      `json:"image"`               // 镜像
	Resources corev1.ResourceRequirements `json:"resources,omitempty"` // 资源限制
	Envs      []corev1.EnvVar             `json:"envs,omitempty"`      // 环境变量
	Ports     []corev1.ServicePort        `json:"ports,omitempty"`     // 服务端口
}

// AppServiceStatus定义了AppService的观察状态
type AppServiceStatus struct {
	appsv1.DeploymentStatus `json:",inline"` // 直接引用DeploymentStatus
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AppService是appservices API的Schema
type AppService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppServiceSpec   `json:"spec,omitempty"`
	Status AppServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AppServiceList包含一个AppService列表
type AppServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppService{}, &AppServiceList{})
}
