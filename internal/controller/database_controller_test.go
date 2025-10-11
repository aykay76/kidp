package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	platformv1 "github.com/aykay76/kidp/api/v1"
)

func TestDatabaseReconciler_LabelFromNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = platformv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tenant := &platformv1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: "acme"}}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "dev", Labels: map[string]string{"platform.company.com/tenant": "acme"}}}
	db := &platformv1.Database{ObjectMeta: metav1.ObjectMeta{Namespace: "dev", Name: "db1"}}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, ns, db).Build()

	r := &DatabaseReconciler{Client: cl, Scheme: scheme, BrokerRegistry: nil, Recorder: record.NewFakeRecorder(10)}

	// First reconcile will add finalizer and requeue
	_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "dev", Name: "db1"}})
	if err != nil {
		t.Fatalf("first reconcile returned error: %v", err)
	}
	// Second reconcile should perform tenant resolution and labeling
	_, err = r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "dev", Name: "db1"}})
	if err != nil {
		t.Fatalf("second reconcile returned error: %v", err)
	}

	out := &platformv1.Database{}
	if err := cl.Get(context.Background(), client.ObjectKey{Namespace: "dev", Name: "db1"}, out); err != nil {
		t.Fatalf("failed to get db: %v", err)
	}

	if out.Labels == nil || out.Labels["platform.company.com/tenant"] != "acme" {
		t.Fatalf("expected tenant label acme on database, got: %v", out.Labels)
	}
}

func TestDatabaseReconciler_SuspendWhenNoTenant(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = platformv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	db := &platformv1.Database{ObjectMeta: metav1.ObjectMeta{Namespace: "dev", Name: "db2"}}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(db).Build()

	r := &DatabaseReconciler{Client: cl, Scheme: scheme, BrokerRegistry: nil, Recorder: record.NewFakeRecorder(10)}

	// First reconcile will add finalizer
	_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "dev", Name: "db2"}})
	if err != nil {
		t.Fatalf("first reconcile returned error: %v", err)
	}
	// Ensure object still exists after first reconcile
	out1 := &platformv1.Database{}
	if err := cl.Get(context.Background(), client.ObjectKey{Namespace: "dev", Name: "db2"}, out1); err != nil {
		t.Fatalf("expected database to exist after first reconcile, but get failed: %v", err)
	}

	// Second reconcile should detect missing tenant and suspend
	_, err = r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "dev", Name: "db2"}})
	if err != nil {
		// Diagnostic: list Database objects in fake client to see what happened
		list := &platformv1.DatabaseList{}
		if lerr := cl.List(context.Background(), list); lerr == nil {
			t.Logf("databases present after reconcile error: %v", list.Items)
		} else {
			t.Logf("failed to list databases after reconcile error: %v", lerr)
		}
		// Check if the error is a NotFound from the API machinery
		if apierrors.IsNotFound(err) {
			t.Logf("error is NotFound: %v", err)
		} else {
			t.Logf("error is not NotFound: %v", err)
		}
		t.Fatalf("second reconcile returned error: %v", err)
	}

	out := &platformv1.Database{}
	if err := cl.Get(context.Background(), client.ObjectKey{Namespace: "dev", Name: "db2"}, out); err != nil {
		t.Fatalf("failed to get db: %v", err)
	}

	if out.Status.Phase != "Suspended" {
		t.Fatalf("expected db to be Suspended when no tenant found, got phase=%s", out.Status.Phase)
	}
}
