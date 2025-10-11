/*
Copyright 2025 Keith McClellan

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

package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	platformv1 "github.com/aykay76/kidp/api/v1"
)

func TestResolveTenant_TenantRef(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = platformv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tenant := &platformv1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: "acme"}}
	team := &platformv1.Team{}
	team.Spec.TenantRef = &platformv1.ObjectReference{Name: "acme"}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, team).Build()

	got, err := ResolveTenant(context.Background(), cl, team)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "acme" {
		t.Fatalf("expected tenant acme got %s", got.Name)
	}
}

func TestResolveTenant_OwnerChain(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = platformv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tenant := &platformv1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: "acme"}}
	team := &platformv1.Team{ObjectMeta: metav1.ObjectMeta{Namespace: "dev", Name: "platform-team"}}
	team.Spec.TenantRef = &platformv1.ObjectReference{Name: "acme"}
	app := &platformv1.Application{ObjectMeta: metav1.ObjectMeta{Namespace: "dev", Name: "app1"}}
	app.Spec.Owner = platformv1.OwnerReference{Kind: "Team", Name: "platform-team"}
	db := &platformv1.Database{ObjectMeta: metav1.ObjectMeta{Namespace: "dev", Name: "db1"}}
	db.Spec.Owner = platformv1.OwnerReference{Kind: "Application", Name: "app1"}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, team, app, db).Build()

	got, err := ResolveTenant(context.Background(), cl, db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "acme" {
		t.Fatalf("expected tenant acme got %s", got.Name)
	}
}

func TestResolveTenant_NamespaceLabel(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = platformv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tenant := &platformv1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: "acme"}}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "dev", Labels: map[string]string{"platform.company.com/tenant": "acme"}}}
	db := &platformv1.Database{ObjectMeta: metav1.ObjectMeta{Namespace: "dev", Name: "db1"}}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, ns, db).Build()

	got, err := ResolveTenant(context.Background(), cl, db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "acme" {
		t.Fatalf("expected tenant acme got %s", got.Name)
	}
}
