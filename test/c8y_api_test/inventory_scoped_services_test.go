package api

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/agents"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devicegroups"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/deviceprofiles"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/smartgroups"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
)

// scopedService is the shared view of the services which keep their resource as
// a managed object (agents, device groups, smart groups and device profiles).
type scopedService struct {
	resource string
	// create body without the name; the service adds its own type and fragments.
	create    func(ctx context.Context, name string) op.Result[jsonmodels.ManagedObject]
	resolveID func(ctx context.Context, ref string, meta map[string]any) (string, error)
	get       func(ctx context.Context, ref string) op.Result[jsonmodels.ManagedObject]
	update    func(ctx context.Context, ref string, body any) op.Result[jsonmodels.ManagedObject]
	delete    func(ctx context.Context, ref string) op.Result[core.NoContent]
	list      func(ctx context.Context) op.Result[jsonmodels.ManagedObject]
	listAll   func(ctx context.Context) ([]jsonmodels.ManagedObject, error)
	// queryRef is a "query:" reference which matches the resource named name.
	queryRef func(name string) string
}

func scopedServices(c *api.Client) []scopedService {
	return []scopedService{
		{
			resource: "agent",
			create: func(ctx context.Context, name string) op.Result[jsonmodels.ManagedObject] {
				return c.Agents.Create(ctx, map[string]any{"name": name})
			},
			resolveID: c.Agents.ResolveID,
			get: func(ctx context.Context, ref string) op.Result[jsonmodels.ManagedObject] {
				return c.Agents.Get(ctx, agents.AgentRef(ref), agents.GetOptions{})
			},
			update: func(ctx context.Context, ref string, body any) op.Result[jsonmodels.ManagedObject] {
				return c.Agents.Update(ctx, agents.AgentRef(ref), body)
			},
			delete: func(ctx context.Context, ref string) op.Result[core.NoContent] {
				return c.Agents.Delete(ctx, agents.AgentRef(ref), agents.DeleteOptions{})
			},
			list: func(ctx context.Context) op.Result[jsonmodels.ManagedObject] {
				return c.Agents.List(ctx, agents.ListOptions{Query: agents.ScopeToAgents("")})
			},
			listAll: func(ctx context.Context) ([]jsonmodels.ManagedObject, error) {
				return collect(c.Agents.ListAll(ctx, agents.ListOptions{Query: agents.ScopeToAgents("")}).Items())
			},
			queryRef: func(name string) string { return "query:name eq '" + name + "'" },
		},
		{
			resource: "device group",
			create: func(ctx context.Context, name string) op.Result[jsonmodels.ManagedObject] {
				return c.DeviceGroups.Create(ctx, map[string]any{"name": name})
			},
			resolveID: c.DeviceGroups.ResolveID,
			get: func(ctx context.Context, ref string) op.Result[jsonmodels.ManagedObject] {
				return c.DeviceGroups.Get(ctx, devicegroups.GroupRef(ref), devicegroups.GetOptions{})
			},
			update: func(ctx context.Context, ref string, body any) op.Result[jsonmodels.ManagedObject] {
				return c.DeviceGroups.Update(ctx, devicegroups.GroupRef(ref), body)
			},
			delete: func(ctx context.Context, ref string) op.Result[core.NoContent] {
				return c.DeviceGroups.Delete(ctx, devicegroups.GroupRef(ref), devicegroups.DeleteOptions{})
			},
			list: func(ctx context.Context) op.Result[jsonmodels.ManagedObject] {
				return c.DeviceGroups.List(ctx, devicegroups.ListOptions{Query: devicegroups.ScopeToGroups("")})
			},
			listAll: func(ctx context.Context) ([]jsonmodels.ManagedObject, error) {
				return collect(c.DeviceGroups.ListAll(ctx, devicegroups.ListOptions{Query: devicegroups.ScopeToGroups("")}).Items())
			},
			queryRef: func(name string) string { return "query:name eq '" + name + "'" },
		},
		{
			resource: "smart group",
			create: func(ctx context.Context, name string) op.Result[jsonmodels.ManagedObject] {
				return c.SmartGroups.Create(ctx, map[string]any{"name": name})
			},
			resolveID: c.SmartGroups.ResolveID,
			get: func(ctx context.Context, ref string) op.Result[jsonmodels.ManagedObject] {
				return c.SmartGroups.Get(ctx, smartgroups.SmartGroupRef(ref), smartgroups.GetOptions{})
			},
			update: func(ctx context.Context, ref string, body any) op.Result[jsonmodels.ManagedObject] {
				return c.SmartGroups.Update(ctx, smartgroups.SmartGroupRef(ref), body)
			},
			delete: func(ctx context.Context, ref string) op.Result[core.NoContent] {
				return c.SmartGroups.Delete(ctx, smartgroups.SmartGroupRef(ref), smartgroups.DeleteOptions{})
			},
			list: func(ctx context.Context) op.Result[jsonmodels.ManagedObject] {
				return c.SmartGroups.List(ctx, smartgroups.ListOptions{Query: smartgroups.ScopeToSmartGroups("")})
			},
			listAll: func(ctx context.Context) ([]jsonmodels.ManagedObject, error) {
				return collect(c.SmartGroups.ListAll(ctx, smartgroups.ListOptions{Query: smartgroups.ScopeToSmartGroups("")}).Items())
			},
			queryRef: func(name string) string { return "query:name eq '" + name + "'" },
		},
		{
			resource: "device profile",
			create: func(ctx context.Context, name string) op.Result[jsonmodels.ManagedObject] {
				return c.DeviceProfiles.Create(ctx, map[string]any{"name": name})
			},
			resolveID: c.DeviceProfiles.ResolveID,
			get: func(ctx context.Context, ref string) op.Result[jsonmodels.ManagedObject] {
				return c.DeviceProfiles.Get(ctx, deviceprofiles.DeviceProfileRef(ref), deviceprofiles.GetOptions{})
			},
			update: func(ctx context.Context, ref string, body any) op.Result[jsonmodels.ManagedObject] {
				return c.DeviceProfiles.Update(ctx, deviceprofiles.DeviceProfileRef(ref), body)
			},
			delete: func(ctx context.Context, ref string) op.Result[core.NoContent] {
				return c.DeviceProfiles.Delete(ctx, deviceprofiles.DeviceProfileRef(ref), deviceprofiles.DeleteOptions{})
			},
			list: func(ctx context.Context) op.Result[jsonmodels.ManagedObject] {
				return c.DeviceProfiles.List(ctx, deviceprofiles.ListOptions{Query: deviceprofiles.ScopeToDeviceProfiles("")})
			},
			listAll: func(ctx context.Context) ([]jsonmodels.ManagedObject, error) {
				return collect(c.DeviceProfiles.ListAll(ctx, deviceprofiles.ListOptions{Query: deviceprofiles.ScopeToDeviceProfiles("")}).Items())
			},
			queryRef: func(name string) string { return "query:name eq '" + name + "'" },
		},
	}
}

func collect[T any](seq func(yield func(T, error) bool)) ([]T, error) {
	var items []T
	for item, err := range seq {
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, nil
}

// TestScopedServices runs the verbs and the reference resolution of the four
// services which share their lookup code against the fake server.
func TestScopedServices(t *testing.T) {
	testcore.SkipLive(t)
	client := testcore.CreateTestClient(t)

	for _, svc := range scopedServices(client) {
		t.Run(svc.resource, func(t *testing.T) {
			ctx := context.Background()
			name := "scoped " + strings.ReplaceAll(svc.resource, " ", "-")

			created := svc.create(ctx, name)
			if created.Err != nil {
				t.Fatalf("Create: %v", created.Err)
			}
			id := created.Data.ID()
			if id == "" {
				t.Fatal("Create returned no id")
			}

			t.Run("list", func(t *testing.T) {
				res := svc.list(ctx)
				if res.Err != nil {
					t.Fatalf("List: %v", res.Err)
				}
				if !containsID(res.Seq(), id) {
					t.Errorf("List does not contain %s", id)
				}
				items, err := svc.listAll(ctx)
				if err != nil {
					t.Fatalf("ListAll: %v", err)
				}
				found := false
				for _, item := range items {
					found = found || item.ID() == id
				}
				if !found {
					t.Errorf("ListAll does not contain %s", id)
				}
			})

			t.Run("resolve", func(t *testing.T) {
				cases := []struct {
					ref        string
					wantSource string
				}{
					{id, ""},
					{"id:" + id, ""},
					{"name:" + name, "name"},
					{svc.queryRef(name), "query"},
				}
				for _, tc := range cases {
					meta := map[string]any{}
					got, err := svc.resolveID(ctx, tc.ref, meta)
					if err != nil || got != id {
						t.Errorf("ResolveID(%q) = %q, %v; want %q", tc.ref, got, err, id)
					}
					if source, _ := meta["source"].(string); source != tc.wantSource {
						t.Errorf("ResolveID(%q) meta source = %q, want %q", tc.ref, source, tc.wantSource)
					}
				}
			})

			t.Run("resolve errors", func(t *testing.T) {
				var lookupErr *core.LookupError

				_, err := svc.resolveID(ctx, "name:nosuch", nil)
				if _, _, ok := core.NameNotFound(err); !ok || !core.IsNotFound(err) || !errors.As(err, &lookupErr) {
					t.Errorf("name not found: got %v, want a lookup error which is a NameNotFoundError", err)
				}
				if want := svc.resource + " not found: nosuch"; err == nil || !strings.HasSuffix(err.Error(), want) {
					t.Errorf("name not found: message %v, want suffix %q", err, want)
				}

				_, err = svc.resolveID(ctx, "query:name eq 'nosuch'", nil)
				if !core.IsNotFound(err) || !errors.As(err, &lookupErr) {
					t.Errorf("query not found: got %v, want a lookup error which is a 404", err)
				}
				if _, _, ok := core.NameNotFound(err); ok {
					t.Errorf("query not found: %v must not be a NameNotFoundError", err)
				}
				if want := svc.resource + " not found: name eq 'nosuch'"; err == nil || err.Error() != want {
					t.Errorf("query not found: message %v, want %q", err, want)
				}

				_, err = svc.resolveID(ctx, "", nil)
				if want := "empty " + svc.resource + " reference"; err == nil || err.Error() != want {
					t.Errorf("empty reference: got %v, want %q", err, want)
				}

				_, err = svc.resolveID(ctx, "bogus:x", nil)
				if want := "unknown " + svc.resource + " resolver scheme: \"bogus\""; err == nil || err.Error() != want {
					t.Errorf("unknown scheme: got %v, want %q", err, want)
				}
			})

			t.Run("get update delete by name", func(t *testing.T) {
				ref := "name:" + name
				got := svc.get(ctx, ref)
				if got.Err != nil || got.Data.ID() != id {
					t.Fatalf("Get(%s) = %v, %v", ref, got.Data.ID(), got.Err)
				}
				updated := svc.update(ctx, ref, map[string]any{"description": "updated"})
				if updated.Err != nil || updated.Data.Get("description").String() != "updated" {
					t.Fatalf("Update(%s): %v", ref, updated.Err)
				}
				if res := svc.delete(ctx, ref); res.Err != nil {
					t.Fatalf("Delete(%s): %v", ref, res.Err)
				}
				if res := svc.get(ctx, id); !core.IsNotFound(res.Err) {
					t.Errorf("Get after Delete: got %v, want not found", res.Err)
				}
				if res := svc.get(ctx, ref); !core.IsNotFound(res.Err) {
					t.Errorf("Get by name after Delete: got %v, want not found", res.Err)
				}
			})
		})
	}
}

func containsID(seq func(yield func(jsonmodels.ManagedObject) bool), id string) bool {
	for item := range seq {
		if item.ID() == id {
			return true
		}
	}
	return false
}
