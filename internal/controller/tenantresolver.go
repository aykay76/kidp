package controller

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	// metav1 and runtime not required here
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1 "github.com/aykay76/kidp/api/v1"
)

// ResolveTenant tries to resolve the Tenant for a given object.
// Strategy:
// 1. If the object's spec contains a TenantRef (for supported types), return that Tenant.
// 2. Else, follow owner references defined in resource spec (for supported types) up to a depth limit.
// 3. Else, if namespaced, check namespace label "platform.company.com/tenant".
// Returns the Tenant object when found, or an error describing the failure.
func ResolveTenant(ctx context.Context, c client.Client, obj client.Object) (*platformv1.Tenant, error) {
	return resolveTenant(ctx, c, obj, 0)
}

func resolveTenant(ctx context.Context, c client.Client, obj client.Object, depth int) (*platformv1.Tenant, error) {
	const maxDepth = 6
	if depth > maxDepth {
		return nil, fmt.Errorf("tenant resolution exceeded max depth")
	}

	// 1) Type-specific: check spec.tenantRef where present
	switch o := obj.(type) {
	case *platformv1.Team:
		if o.Spec.TenantRef != nil && o.Spec.TenantRef.Name != "" {
			t := &platformv1.Tenant{}
			if err := c.Get(ctx, client.ObjectKey{Name: o.Spec.TenantRef.Name}, t); err != nil {
				return nil, fmt.Errorf("referenced tenant %s not found: %w", o.Spec.TenantRef.Name, err)
			}
			return t, nil
		}
	case *platformv1.Application:
		if o.Spec.Owner.Kind == "Tenant" {
			t := &platformv1.Tenant{}
			if err := c.Get(ctx, client.ObjectKey{Name: o.Spec.Owner.Name}, t); err != nil {
				return nil, fmt.Errorf("referenced tenant %s not found: %w", o.Spec.Owner.Name, err)
			}
			return t, nil
		}
	case *platformv1.Database:
		// Databases use OwnerReference struct
		if o.Spec.Owner.Kind == "Tenant" {
			t := &platformv1.Tenant{}
			if err := c.Get(ctx, client.ObjectKey{Name: o.Spec.Owner.Name}, t); err != nil {
				return nil, fmt.Errorf("referenced tenant %s not found: %w", o.Spec.Owner.Name, err)
			}
			return t, nil
		}
	}

	// 2) Follow owner chains where possible
	switch o := obj.(type) {
	case *platformv1.Database:
		if o.Spec.Owner.Kind == "Team" {
			// owner is a Team in the same namespace unless Namespace provided
			ns := o.Namespace
			if o.Spec.Owner.Namespace != "" {
				ns = o.Spec.Owner.Namespace
			}
			team := &platformv1.Team{}
			if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: o.Spec.Owner.Name}, team); err != nil {
				return nil, fmt.Errorf("owner team %s/%s not found: %w", ns, o.Spec.Owner.Name, err)
			}
			return resolveTenant(ctx, c, team, depth+1)
		}
		if o.Spec.Owner.Kind == "Application" {
			ns := o.Namespace
			if o.Spec.Owner.Namespace != "" {
				ns = o.Spec.Owner.Namespace
			}
			app := &platformv1.Application{}
			if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: o.Spec.Owner.Name}, app); err != nil {
				return nil, fmt.Errorf("owner application %s/%s not found: %w", ns, o.Spec.Owner.Name, err)
			}
			return resolveTenant(ctx, c, app, depth+1)
		}
	case *platformv1.Application:
		if o.Spec.Owner.Kind == "Team" {
			team := &platformv1.Team{}
			if err := c.Get(ctx, client.ObjectKey{Namespace: o.Namespace, Name: o.Spec.Owner.Name}, team); err != nil {
				return nil, fmt.Errorf("owner team %s/%s not found: %w", o.Namespace, o.Spec.Owner.Name, err)
			}
			return resolveTenant(ctx, c, team, depth+1)
		}
		if o.Spec.Owner.Kind == "Application" {
			parent := &platformv1.Application{}
			if err := c.Get(ctx, client.ObjectKey{Namespace: o.Namespace, Name: o.Spec.Owner.Name}, parent); err != nil {
				return nil, fmt.Errorf("owner application %s/%s not found: %w", o.Namespace, o.Spec.Owner.Name, err)
			}
			return resolveTenant(ctx, c, parent, depth+1)
		}
	case *platformv1.Team:
		// Team: if no TenantRef we may infer from namespace label below
	}

	// 3) Namespace label fallback (for namespaced objects)
	nsName := obj.GetNamespace()
	if nsName != "" {
		ns := &corev1.Namespace{}
		if err := c.Get(ctx, client.ObjectKey{Name: nsName}, ns); err == nil {
			if tn, ok := ns.Labels["platform.company.com/tenant"]; ok && tn != "" {
				t := &platformv1.Tenant{}
				if err := c.Get(ctx, client.ObjectKey{Name: tn}, t); err != nil {
					return nil, fmt.Errorf("tenant label %s on namespace %s points to missing tenant: %w", tn, nsName, err)
				}
				return t, nil
			}
		}
	}

	return nil, errors.New("tenant unresolved")
}
