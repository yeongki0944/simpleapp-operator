package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SimpleAppSpec defines the desired state of SimpleApp
type SimpleAppSpec struct {
	// Image는 사용할 컨테이너 이미지입니다
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// Port는 서비스가 노출할 포트입니다
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=80
	Port int32 `json:"port,omitempty"`

	// Replicas는 생성할 파드의 개수입니다
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`

	// Message는 환경변수로 전달할 메시지입니다
	// +kubebuilder:default="Hello World"
	Message string `json:"message,omitempty"`
}

// SimpleAppStatus defines the observed state of SimpleApp
type SimpleAppStatus struct {
	// AvailableReplicas는 현재 사용 가능한 파드 수입니다
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// Conditions는 SimpleApp의 현재 상태를 나타냅니다
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Image",type="string",JSONPath=".spec.image"
//+kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas"
//+kubebuilder:printcolumn:name="Available",type="integer",JSONPath=".status.availableReplicas"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// SimpleApp is the Schema for the simpleapps API
type SimpleApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SimpleAppSpec   `json:"spec,omitempty"`
	Status SimpleAppStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SimpleAppList contains a list of SimpleApp
type SimpleAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SimpleApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SimpleApp{}, &SimpleAppList{})
}
