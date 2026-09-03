package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*computeSnapshotResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeSnapshotResource)(nil)
	_ resource.ResourceWithImportState = (*computeSnapshotResource)(nil)
)

type computeSnapshotResourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	Size      types.Int64  `tfsdk:"size"`
	CreatedAt types.String `tfsdk:"created_at"`

	Name     types.String `tfsdk:"name"`
	VolumeID types.Int64  `tfsdk:"volume_id"`
}

func (d *computeSnapshotResourceData) FromEntity(snapshot compute.Snapshot) {
	d.ID = types.Int64Value(int64(snapshot.ID))
	d.Size = types.Int64Value(int64(snapshot.Size))
	d.CreatedAt = types.StringValue(snapshot.CreatedAt.String())

	d.Name = types.StringValue(snapshot.Name)
	d.VolumeID = types.Int64Value(int64(snapshot.Volume.ID))
}

func (t computeSnapshotResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the snapshot",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "size of the snapshot in GiB",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "date and time when the snapshot was created",
				Computed:            true,
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "name of the snapshot",
				Required:            true,
			},
			"volume_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the volume",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func newComputeSnapshotResource() resource.Resource {
	return &computeSnapshotResource{}
}

func (r *computeSnapshotResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_snapshot"
}

func (r *computeSnapshotResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	r.snapshotService = compute.NewSnapshotService(client)
}

type computeSnapshotResource struct {
	snapshotService compute.SnapshotService
}

func (r computeSnapshotResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeSnapshotResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.SnapshotCreate{
		Name:     config.Name.ValueString(),
		VolumeID: int(config.VolumeID.ValueInt64()),
	}

	var snapshot compute.Snapshot
	err := retryCreate(ctx, "create snapshot", func() (err error) {
		snapshot, err = r.snapshotService.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create snapshot: %s", err))
		return
	}

	tflog.Trace(ctx, "created snapshot", map[string]interface{}{
		"id":   snapshot.ID,
		"data": snapshot,
	})

	available, err := r.waitForSnapshotAvailable(ctx, snapshot.ID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for snapshot to be available: %s", err))
	}
	if available.ID != 0 {
		snapshot = available
	}

	var state computeSnapshotResourceData
	state.FromEntity(snapshot)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeSnapshotResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeSnapshotResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	snapshot, err := r.snapshotService.Get(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("snapshot %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get snapshot: %s", err))
		return
	}

	state.FromEntity(snapshot)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeSnapshotResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state computeSnapshotResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeSnapshotResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	if config.Name.Equal(state.Name) {
		return
	}

	tflog.Debug(ctx, "snapshot name has changed: updating snapshot", map[string]interface{}{
		"snapshot_id":    state.ID,
		"previous_name":  state.Name,
		"requested_name": config.Name,
	})

	update := compute.SnapshotUpdate{
		Name: config.Name.ValueString(),
	}

	var snapshot compute.Snapshot
	err := retry(ctx, "update snapshot", func() (err error) {
		snapshot, err = r.snapshotService.Update(ctx, int(state.ID.ValueInt64()), update)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update snapshot: %s", err))
		return
	}

	state.FromEntity(snapshot)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeSnapshotResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeSnapshotResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete snapshot", func() error {
		return r.snapshotService.Delete(ctx, int(state.ID.ValueInt64()))
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete snapshot: %s", err))
		return
	}
}

func (r computeSnapshotResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}

// the snapshot stays in the creating state while the data is copied —
// restoring or deleting it in that window is refused
func (r computeSnapshotResource) waitForSnapshotAvailable(ctx context.Context, snapshotID int) (snapshot compute.Snapshot, err error) {
	err = waitFor(ctx, snapshotTimeout, defaultWaitInterval, fmt.Sprintf("snapshot %d to be available", snapshotID), func(ctx context.Context) (bool, error) {
		got, err := r.snapshotService.Get(ctx, snapshotID)
		if err != nil {
			return false, err
		}
		snapshot = got

		switch snapshot.Status.ID {
		case compute.SnapshotStatusAvailable:
			return true, nil
		case compute.SnapshotStatusError:
			return false, fmt.Errorf("snapshot %d is in error state", snapshotID)
		default:
			return false, nil
		}
	})

	return snapshot, err
}
