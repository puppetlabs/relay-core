/*
Copyright 2016 The Kubernetes Authors.

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

package kops

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var DefaultChannelBase = "https://raw.githubusercontent.com/kubernetes/kops/master/channels/"

const (
	DefaultChannel = "stable"
	AlphaChannel   = "alpha"
)

type Channel struct {
	v1.TypeMeta `json:",inline"`
	ObjectMeta  metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ChannelSpec `json:"spec,omitempty"`
}

type ChannelSpec struct {
	Images []*ChannelImageSpec `json:"images,omitempty"`

	Cluster *ClusterSpec `json:"cluster,omitempty"`

	// KopsVersions allows us to recommend/require kops versions
	KopsVersions []KopsVersionSpec `json:"kopsVersions,omitempty"`

	// KubernetesVersions allows us to recommend/requires kubernetes versions
	KubernetesVersions []KubernetesVersionSpec `json:"kubernetesVersions,omitempty"`
}

type KopsVersionSpec struct {
	Range string `json:"range,omitempty"`

	// RecommendedVersion is the recommended version of kops to use for this Range of kops versions
	RecommendedVersion string `json:"recommendedVersion,omitempty"`

	// RequiredVersion is the required version of kops to use for this Range of kops versions, forcing an upgrade
	RequiredVersion string `json:"requiredVersion,omitempty"`

	// KubernetesVersion is the default version of kubernetes to use with this kops version e.g. for new clusters
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
}

type KubernetesVersionSpec struct {
	Range string `json:"range,omitempty"`

	RecommendedVersion string `json:"recommendedVersion,omitempty"`
	RequiredVersion    string `json:"requiredVersion,omitempty"`
}

type ChannelImageSpec struct {
	Labels map[string]string `json:"labels,omitempty"`

	ProviderID string `json:"providerID,omitempty"`

	Name string `json:"name,omitempty"`

	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
}
