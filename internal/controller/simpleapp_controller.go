package controller

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1alpha1 "github.com/yeongki0944/simpleapp-operator/api/v1"
)

// SimpleAppReconciler reconciles a SimpleApp object
type SimpleAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.yeongki.dev,resources=simpleapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.yeongki.dev,resources=simpleapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.yeongki.dev,resources=simpleapps/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

func (r *SimpleAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// 1. SimpleApp 리소스 가져오기
	var simpleApp appsv1alpha1.SimpleApp
	if err := r.Get(ctx, req.NamespacedName, &simpleApp); err != nil {
		if errors.IsNotFound(err) {
			log.Info("SimpleApp resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get SimpleApp")
		return ctrl.Result{}, err
	}

	// 2. ConfigMap 생성/업데이트
	if err := r.reconcileConfigMap(ctx, &simpleApp); err != nil {
		return ctrl.Result{}, err
	}

	// 3. Deployment 생성/업데이트
	if err := r.reconcileDeployment(ctx, &simpleApp); err != nil {
		return ctrl.Result{}, err
	}

	// 4. Service 생성/업데이트
	if err := r.reconcileService(ctx, &simpleApp); err != nil {
		return ctrl.Result{}, err
	}

	// 5. 상태 업데이트
	if err := r.updateStatus(ctx, &simpleApp); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SimpleAppReconciler) reconcileConfigMap(ctx context.Context, simpleApp *appsv1alpha1.SimpleApp) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      simpleApp.Name + "-config",
			Namespace: simpleApp.Namespace,
		},
		Data: map[string]string{
			"MESSAGE": simpleApp.Spec.Message,
		},
	}

	// Owner Reference 설정
	if err := ctrl.SetControllerReference(simpleApp, configMap, r.Scheme); err != nil {
		return err
	}

	// ConfigMap 생성 또는 업데이트
	found := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, configMap)
	} else if err != nil {
		return err
	}

	// 데이터가 다르면 업데이트
	if !reflect.DeepEqual(found.Data, configMap.Data) {
		found.Data = configMap.Data
		return r.Update(ctx, found)
	}

	return nil
}

func (r *SimpleAppReconciler) reconcileDeployment(ctx context.Context, simpleApp *appsv1alpha1.SimpleApp) error {
	// 기본값 설정
	replicas := int32(1)
	if simpleApp.Spec.Replicas != nil {
		replicas = *simpleApp.Spec.Replicas
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      simpleApp.Name,
			Namespace: simpleApp.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": simpleApp.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": simpleApp.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: simpleApp.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: simpleApp.Spec.Port,
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: simpleApp.Name + "-config",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Owner Reference 설정
	if err := ctrl.SetControllerReference(simpleApp, deployment, r.Scheme); err != nil {
		return err
	}

	// Deployment 생성 또는 업데이트
	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, deployment)
	} else if err != nil {
		return err
	}

	// 스펙이 다르면 업데이트
	if !reflect.DeepEqual(found.Spec, deployment.Spec) {
		found.Spec = deployment.Spec
		return r.Update(ctx, found)
	}

	return nil
}

func (r *SimpleAppReconciler) reconcileService(ctx context.Context, simpleApp *appsv1alpha1.SimpleApp) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      simpleApp.Name + "-service",
			Namespace: simpleApp.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": simpleApp.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       simpleApp.Spec.Port,
					TargetPort: intstr.FromInt(int(simpleApp.Spec.Port)),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	// Owner Reference 설정
	if err := ctrl.SetControllerReference(simpleApp, service, r.Scheme); err != nil {
		return err
	}

	// Service 생성 또는 업데이트
	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, service)
	} else if err != nil {
		return err
	}

	// 포트가 다르면 업데이트
	if len(found.Spec.Ports) == 0 || found.Spec.Ports[0].Port != service.Spec.Ports[0].Port {
		found.Spec.Ports = service.Spec.Ports
		return r.Update(ctx, found)
	}

	return nil
}

func (r *SimpleAppReconciler) updateStatus(ctx context.Context, simpleApp *appsv1alpha1.SimpleApp) error {
	// Deployment 상태 확인
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: simpleApp.Name, Namespace: simpleApp.Namespace}, deployment)
	if err != nil {
		return err
	}

	// 상태 업데이트
	simpleApp.Status.AvailableReplicas = deployment.Status.AvailableReplicas

	// Condition 업데이트
	condition := metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  "DeploymentNotReady",
		Message: fmt.Sprintf("Deployment %s is not ready", deployment.Name),
	}

	if deployment.Status.ReadyReplicas == deployment.Status.Replicas && deployment.Status.Replicas > 0 {
		condition.Status = metav1.ConditionTrue
		condition.Reason = "DeploymentReady"
		condition.Message = fmt.Sprintf("Deployment %s is ready", deployment.Name)
	}

	// 기존 condition 업데이트 또는 새로 추가
	found := false
	for i := range simpleApp.Status.Conditions {
		if simpleApp.Status.Conditions[i].Type == condition.Type {
			simpleApp.Status.Conditions[i] = condition
			found = true
			break
		}
	}
	if !found {
		simpleApp.Status.Conditions = append(simpleApp.Status.Conditions, condition)
	}

	return r.Status().Update(ctx, simpleApp)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SimpleAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.SimpleApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
