package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/nebius/gosdk/constants"
	"github.com/nebius/gosdk/proto/fieldmask/mask"
	fieldmaskprotobuf "github.com/nebius/gosdk/proto/fieldmask/protobuf"
	common "github.com/nebius/gosdk/proto/nebius/common/v1"
	"github.com/nebius/terraform-provider-nebius/conversion"
	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
	"github.com/nebius/terraform-provider-nebius/conversion/writeonly"
	"github.com/nebius/terraform-provider-nebius/service/requestcontext"
)

type PreflightCheckInterface interface {
	PreflightCheck(
		ctx context.Context,
		preflightContext *common.PreflightCheckContext,
		metadata *common.ResourceMetadata,
		spec proto.Message,
	) (*common.PreflightCheckResult, *requestcontext.Context, error)
}

type RecreateOnChangeInterface interface {
	RecreateOnChangeFields() (*mask.Mask, error)
}

func (r *commonResource) preflightPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
	preflight PreflightCheckInterface,
) (*requestcontext.Context, diag.Diagnostics) {
	var data types.Object
	resp.Diagnostics.Append(resp.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return nil, nil
	}

	spec := r.implementation.SpecMessage()
	metadata := &common.ResourceMetadata{}
	var planData *types.Object
	if !data.IsUnknown() {
		planData = &data
	}
	dataWithWriteOnly, diags := r.addWriteOnlyFields(ctx, req.Config, planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return nil, nil
	}
	configuredUnknowns, diags := r.unknownsFromConfig(ctx, req.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return nil, nil
	}
	configuredUnknowns = ctypes.AppendUnknownMask(
		configuredUnknowns,
		mask.NewFieldPath(constants.FieldSpec),
		dataWithWriteOnly.configuredUnknowns,
	)

	plannedUnknowns := mask.New()
	if planData != nil {
		metadata, plannedUnknowns, diags = requestMessagesFromTFWithUnknowns(
			ctx,
			*dataWithWriteOnly.value,
			spec,
			r.implementation.FieldNameMap(),
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return nil, nil
		}
		plannedUnknowns = ctypes.AppendUnknownMask(
			plannedUnknowns,
			mask.NewFieldPath(constants.FieldSpec),
			dataWithWriteOnly.planUnknowns,
		)
	}
	// The plan also marks computed values as unknown. Keep plan paths only
	// when they overlap unknown expressions from the configuration.
	unknowns, err := overlappingUnknownBranches(plannedUnknowns, configuredUnknowns)
	if err != nil {
		resp.Diagnostics = ErrorToDiag(
			resp.Diagnostics, err, "failed to filter planned unknown paths",
		)
		return nil, nil
	}

	unknownPaths, err := marshalUnknownPaths(unknowns)
	if err != nil {
		resp.Diagnostics = ErrorToDiag(
			resp.Diagnostics, err, "failed to encode unknown paths",
		)
		return nil, nil
	}
	action := common.PreflightCheckContext_CREATE
	pathsRequireRecreate := ""
	if !req.State.Raw.IsNull() {
		action = common.PreflightCheckContext_UPDATE
		pathsRequireRecreate, err = r.plannedRecreatePaths(
			ctx, req, resp, metadata, spec, unknowns, dataWithWriteOnly.supplied,
		)
		if err != nil {
			resp.Diagnostics = ErrorToDiag(
				resp.Diagnostics, err, "failed to identify planned resource recreation",
			)
			return nil, nil
		}
		if resp.Diagnostics.HasError() {
			return nil, nil
		}
		if pathsRequireRecreate != "" {
			action = common.PreflightCheckContext_RECREATE
			if replaceErr := r.appendPreflightReplacements(
				ctx, req.Plan, resp, pathsRequireRecreate,
			); replaceErr != nil {
				resp.Diagnostics = ErrorToDiag(
					resp.Diagnostics, replaceErr, "invalid planned recreation paths",
				)
				return nil, nil
			}
		}
	}
	checkContext := &common.PreflightCheckContext{
		Action:               action,
		Tool:                 "terraform",
		UnknownPaths:         unknownPaths,
		PathsRequireRecreate: pathsRequireRecreate,
	}

	result, reqCtx, err := preflight.PreflightCheck(ctx, checkContext, metadata, spec)
	if err != nil {
		resp.Diagnostics = ErrorToDiag(
			resp.Diagnostics, err, "resource preflight check failed",
		)
		return reqCtx, nil
	}
	if result == nil {
		resp.Diagnostics.AddError(
			"resource preflight check failed",
			"The server returned an empty preflight result.",
		)
		return reqCtx, nil
	}
	if action != common.PreflightCheckContext_UPDATE || result.GetPathsRequireRecreate() == "" {
		return reqCtx, r.preflightDiagnostics(result)
	}

	if replaceErr := r.appendPreflightReplacements(
		ctx, req.Plan, resp, result.GetPathsRequireRecreate(),
	); replaceErr != nil {
		resp.Diagnostics = ErrorToDiag(
			resp.Diagnostics, replaceErr, "invalid preflight recreation paths",
		)
		return reqCtx, nil
	}

	recreateContext := &common.PreflightCheckContext{
		Action:               common.PreflightCheckContext_RECREATE,
		Tool:                 "terraform",
		UnknownPaths:         unknownPaths,
		PathsRequireRecreate: result.GetPathsRequireRecreate(),
	}
	recreateResult, recreateReqCtx, err := preflight.PreflightCheck(
		ctx, recreateContext, metadata, spec,
	)
	reqCtx = requestcontext.MergeContexts(recreateReqCtx, reqCtx)
	if err != nil {
		resp.Diagnostics = ErrorToDiag(
			resp.Diagnostics, err, "resource recreate preflight check failed",
		)
		return reqCtx, nil
	}
	if recreateResult == nil {
		resp.Diagnostics.AddError(
			"resource recreate preflight check failed",
			"The server returned an empty preflight result.",
		)
		return reqCtx, nil
	}
	return reqCtx, r.preflightDiagnostics(recreateResult)
}

func (r *commonResource) unknownsFromConfig(
	ctx context.Context,
	config tfsdk.Config,
) (*mask.Mask, diag.Diagnostics) {
	var data types.Object
	diags := config.Get(ctx, &data)
	if diags.HasError() || data.IsNull() {
		return nil, diags
	}
	if data.IsUnknown() {
		return mask.New(), diags
	}

	nameMap := r.implementation.FieldNameMap()
	_, metadataUnknowns, innerDiags := metadataFromTFWithUnknowns(ctx, data, nameMap)
	diags.Append(innerDiags...)
	if innerDiags.HasError() {
		return nil, diags
	}
	specUnknowns, innerDiags := conversion.MessageFromTF(
		ctx, data, r.implementation.SpecMessage(), nameMap,
	)
	diags.Append(innerDiags...)
	unknowns := ctypes.AppendUnknownMask(
		nil, mask.NewFieldPath(constants.FieldMetadata), metadataUnknowns,
	)
	unknowns = ctypes.AppendUnknownMask(
		unknowns, mask.NewFieldPath(constants.FieldSpec), specUnknowns,
	)
	return unknowns, diags
}

func (r *commonResource) plannedRecreatePaths(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
	plannedMetadata *common.ResourceMetadata,
	plannedSpec proto.Message,
	plannedUnknowns *mask.Mask,
	suppliedWriteOnly *mask.Mask,
) (string, error) {
	wellKnownIDChanged := false
	if _, ok := r.resourceSchema.Attributes[constants.FieldWellKnownID]; ok {
		var planned, current attr.Value
		attributePath := path.Root(constants.FieldWellKnownID)
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, attributePath, &planned)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, attributePath, &current)...)
		if resp.Diagnostics.HasError() {
			return "", nil
		}
		wellKnownIDChanged = planned == nil || !planned.Equal(current)
	}
	if wellKnownIDChanged {
		// well_known_id is client-only and has no metadata/spec field path.
		return "*", nil
	}

	recreateOnChange, ok := r.implementation.(RecreateOnChangeInterface)
	if !ok {
		return "", nil
	}
	recreateFields, err := recreateOnChange.RecreateOnChangeFields()
	if err != nil || recreateFields == nil {
		return "", err
	}
	writeOnlyVersionChanged := false
	if _, hasWriteOnly := r.resourceSchema.Attributes[writeonly.FieldName]; hasWriteOnly {
		var plannedSensitive, stateSensitive types.Object
		sensitivePath := path.Root(writeonly.FieldName)
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, sensitivePath, &plannedSensitive)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, sensitivePath, &stateSensitive)...)
		if resp.Diagnostics.HasError() {
			return "", nil
		}
		plannedVersion, plannedVersionKnown := knownWriteOnlyVersion(plannedSensitive)
		stateVersion, stateVersionKnown := knownWriteOnlyVersion(stateSensitive)
		writeOnlyVersionChanged = !plannedVersionKnown ||
			!stateVersionKnown ||
			!plannedVersion.Equal(stateVersion)
		if !writeOnlyVersionChanged {
			var plannedData types.Object
			resp.Diagnostics.Append(req.Plan.Get(ctx, &plannedData)...)
			if resp.Diagnostics.HasError() {
				return "", nil
			}
			plannedSpec = r.implementation.SpecMessage()
			var diags diag.Diagnostics
			plannedMetadata, _, diags = requestMessagesFromTFWithUnknowns(
				ctx, plannedData, plannedSpec, r.implementation.FieldNameMap(),
			)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return "", nil
			}
		}
	}

	var stateData types.Object
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return "", nil
	}
	stateSpec := r.implementation.SpecMessage()
	stateMetadata, _, diags := requestMessagesFromTFWithUnknowns(
		ctx, stateData, stateSpec, r.implementation.FieldNameMap(),
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return "", nil
	}

	changed, err := changedRecreateFields(
		plannedMetadata, stateMetadata, plannedSpec, stateSpec,
		plannedUnknowns, recreateFields,
	)
	if err != nil {
		return "", err
	}
	if writeOnlyVersionChanged {
		changedWriteOnly, err := changedWriteOnlyRecreateFields(recreateFields, suppliedWriteOnly)
		if err != nil {
			return "", err
		}
		if changedWriteOnly != nil {
			if changed == nil {
				changed = mask.New()
			}
			if err := changed.Merge(changedWriteOnly); err != nil {
				return "", fmt.Errorf("merge changed write-only fields: %w", err)
			}
		}
	}
	if changed == nil {
		return "", nil
	}
	return changed.Marshal()
}

func knownWriteOnlyVersion(sensitive types.Object) (types.String, bool) {
	if sensitive.IsUnknown() {
		return types.StringUnknown(), false
	}
	if sensitive.IsNull() {
		return types.StringNull(), true
	}
	version, ok := sensitive.Attributes()[writeonly.VersionField].(types.String)
	return version, ok && !version.IsUnknown()
}

func changedWriteOnlyRecreateFields(
	recreateFields, writeOnlyFields *mask.Mask,
) (*mask.Mask, error) {
	if recreateFields == nil || writeOnlyFields == nil {
		return nil, nil
	}
	changed := mask.New()
	for _, root := range []mask.FieldKey{constants.FieldMetadata, constants.FieldSpec} {
		candidates, err := recreateFields.GetSubMask(root)
		if err != nil {
			return nil, fmt.Errorf("get recreate-on-change fields for %s: %w", root, err)
		}
		var changedRoot *mask.Mask
		for _, candidate := range terminalMasks(candidates) {
			if !masksOverlap(candidate, writeOnlyFields) {
				continue
			}
			if changedRoot == nil {
				changedRoot = mask.New()
			}
			if err := changedRoot.Merge(candidate); err != nil {
				return nil, fmt.Errorf("merge changed write-only fields for %s: %w", root, err)
			}
		}
		if changedRoot != nil {
			changed.FieldParts[root] = changedRoot
		}
	}
	if changed.IsEmpty() {
		return nil, nil
	}
	return changed, nil
}

func changedRecreateFields(
	plannedMetadata, stateMetadata proto.Message,
	plannedSpec, stateSpec proto.Message,
	plannedUnknowns, recreateFields *mask.Mask,
) (*mask.Mask, error) {
	changed := mask.New()
	for _, root := range []struct {
		name             mask.FieldKey
		planned, current proto.Message
	}{
		{name: constants.FieldMetadata, planned: plannedMetadata, current: stateMetadata},
		{name: constants.FieldSpec, planned: plannedSpec, current: stateSpec},
	} {
		candidates, err := recreateFields.GetSubMask(root.name)
		if err != nil {
			return nil, fmt.Errorf("get recreate-on-change fields for %s: %w", root.name, err)
		}
		if candidates == nil {
			continue
		}
		var unknowns *mask.Mask
		if plannedUnknowns != nil && plannedUnknowns.IsEmpty() {
			unknowns = mask.New()
		} else {
			unknowns, err = plannedUnknowns.GetSubMask(root.name)
			if err != nil {
				return nil, fmt.Errorf("get unknown fields for %s: %w", root.name, err)
			}
		}
		changedRoot, err := changedMessageFields(
			root.planned, root.current, unknowns, candidates,
		)
		if err != nil {
			return nil, fmt.Errorf("compare recreate-on-change fields for %s: %w", root.name, err)
		}
		if changedRoot != nil {
			changed.FieldParts[root.name] = changedRoot
		}
	}
	if changed.IsEmpty() {
		return nil, nil
	}
	return changed, nil
}

func changedMessageFields(
	planned, current proto.Message,
	unknowns, candidates *mask.Mask,
) (*mask.Mask, error) {
	changed := mask.New()
	for _, candidate := range terminalMasks(candidates) {
		hasUnknown := masksOverlap(candidate, unknowns)
		plannedFiltered := proto.Clone(planned)
		currentFiltered := proto.Clone(current)
		if err := fieldmaskprotobuf.FilterWithSelectMask(plannedFiltered, candidate); err != nil {
			return nil, fmt.Errorf("filter planned fields: %w", err)
		}
		if err := fieldmaskprotobuf.FilterWithSelectMask(currentFiltered, candidate); err != nil {
			return nil, fmt.Errorf("filter current fields: %w", err)
		}
		if err := clearEmptyMessageAncestors(plannedFiltered.ProtoReflect(), candidate); err != nil {
			return nil, fmt.Errorf("normalize planned fields: %w", err)
		}
		if err := clearEmptyMessageAncestors(currentFiltered.ProtoReflect(), candidate); err != nil {
			return nil, fmt.Errorf("normalize current fields: %w", err)
		}
		if !hasUnknown && proto.Equal(plannedFiltered, currentFiltered) {
			continue
		}
		if err := changed.Merge(candidate); err != nil {
			return nil, fmt.Errorf("merge changed fields: %w", err)
		}
	}
	if changed.IsEmpty() {
		return nil, nil
	}
	return changed, nil
}

func masksOverlap(left, right *mask.Mask) bool {
	if left == nil || right == nil {
		return false
	}
	if left.IsEmpty() || right.IsEmpty() {
		return true
	}
	for key, leftChild := range left.FieldParts {
		if masksOverlap(leftChild, right.FieldParts[key]) || masksOverlap(leftChild, right.Any) {
			return true
		}
	}
	if left.Any == nil {
		return false
	}
	for _, rightChild := range right.FieldParts {
		if masksOverlap(left.Any, rightChild) {
			return true
		}
	}
	return masksOverlap(left.Any, right.Any)
}

// overlappingUnknownBranches keeps source paths that overlap the other mask.
// It preserves source granularity.
func overlappingUnknownBranches(source, other *mask.Mask) (*mask.Mask, error) {
	if source == nil || other == nil {
		return nil, nil
	}
	if source.IsEmpty() {
		return source.Copy()
	}

	var result *mask.Mask
	for _, branch := range unknownMaskBranches(source) {
		if !masksOverlap(branch, other) {
			continue
		}
		if result == nil {
			var err error
			result, err = branch.Copy()
			if err != nil {
				return nil, err
			}
			continue
		}
		if err := result.Merge(branch); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func unknownMaskBranches(fields *mask.Mask) []*mask.Mask {
	if fields == nil {
		return nil
	}
	if fields.IsEmpty() {
		return []*mask.Mask{fields}
	}

	var result []*mask.Mask
	for _, branch := range unknownMaskBranches(fields.Any) {
		result = append(result, &mask.Mask{Any: branch})
	}
	for key, child := range fields.FieldParts {
		for _, branch := range unknownMaskBranches(child) {
			result = append(result, &mask.Mask{FieldParts: map[mask.FieldKey]*mask.Mask{
				key: branch,
			}})
		}
	}
	return result
}

// clearEmptyMessageAncestors removes presence differences introduced by empty
// parent messages. Terraform does not change a child when both child values are
// empty, even if one parent object is null and the other parent object exists.
func clearEmptyMessageAncestors(message protoreflect.Message, selected *mask.Mask) error {
	if selected == nil || selected.IsEmpty() {
		return nil
	}
	fields := message.Descriptor().Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		if !message.Has(field) {
			continue
		}
		childMask, err := selected.GetSubMask(mask.FieldKey(field.Name()))
		if err != nil {
			return fmt.Errorf("get selected fields for %s: %w", field.Name(), err)
		}
		if childMask == nil || childMask.IsEmpty() {
			continue
		}
		if field.IsList() {
			if field.Message() == nil {
				continue
			}
			list := message.Get(field).List()
			for index := range list.Len() {
				element := list.Get(index)
				elementMask, err := childMask.GetSubMask(mask.FieldKey(strconv.Itoa(index)))
				if err != nil {
					return fmt.Errorf("get selected fields for %s[%d]: %w", field.Name(), index, err)
				}
				if err := clearEmptyMessageAncestors(element.Message(), elementMask); err != nil {
					return fmt.Errorf("normalize %s[%d]: %w", field.Name(), index, err)
				}
			}
			length := list.Len()
			for length > 0 && isEmptyProtoMessage(list.Get(length-1).Message()) {
				length--
			}
			list.Truncate(length)
			continue
		}
		if field.IsMap() {
			if field.MapValue().Message() == nil {
				continue
			}
			var mapErr error
			var emptyKeys []protoreflect.MapKey
			message.Get(field).Map().Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
				entryMask, err := childMask.GetSubMask(mask.FieldKey(fmt.Sprint(key.Interface())))
				if err == nil {
					err = clearEmptyMessageAncestors(value.Message(), entryMask)
				}
				if err != nil {
					mapErr = fmt.Errorf("normalize %s[%s]: %w", field.Name(), key.String(), err)
				} else if isEmptyProtoMessage(value.Message()) {
					emptyKeys = append(emptyKeys, key)
				}
				return mapErr == nil
			})
			if mapErr != nil {
				return mapErr
			}
			for _, key := range emptyKeys {
				message.Get(field).Map().Clear(key)
			}
			continue
		}
		if field.Message() == nil {
			continue
		}
		child := message.Get(field).Message()
		if err := clearEmptyMessageAncestors(child, childMask); err != nil {
			return fmt.Errorf("normalize %s: %w", field.Name(), err)
		}
		if isEmptyProtoMessage(child) {
			message.Clear(field)
		}
	}
	return nil
}

func isEmptyProtoMessage(message protoreflect.Message) bool {
	if len(message.GetUnknown()) != 0 {
		return false
	}
	empty := true
	message.Range(func(protoreflect.FieldDescriptor, protoreflect.Value) bool {
		empty = false
		return false
	})
	return empty
}

func terminalMasks(fields *mask.Mask) []*mask.Mask {
	if fields == nil {
		return nil
	}
	if fields.IsEmpty() || fields.Any != nil {
		return []*mask.Mask{fields}
	}
	keys := make([]string, 0, len(fields.FieldParts))
	for key, child := range fields.FieldParts {
		if child != nil {
			keys = append(keys, string(key))
		}
	}
	sort.Strings(keys)
	var result []*mask.Mask
	for _, key := range keys {
		for _, terminal := range terminalMasks(fields.FieldParts[mask.FieldKey(key)]) {
			result = append(result, &mask.Mask{FieldParts: map[mask.FieldKey]*mask.Mask{
				mask.FieldKey(key): terminal,
			}})
		}
	}
	return result
}

func marshalUnknownPaths(unknowns *mask.Mask) (string, error) {
	if unknowns == nil {
		return "", nil
	}
	if unknowns.IsEmpty() {
		return "*", nil
	}
	return unknowns.Marshal()
}

func (r *commonResource) preflightDiagnostics(
	result *common.PreflightCheckResult,
) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	for _, preflightDiagnostic := range result.GetDiagnostics() {
		if preflightDiagnostic == nil {
			continue
		}
		attributePath, pathErr := requestcontext.FieldPathToTFPath(
			preflightDiagnostic.GetPath(),
			r.resourceSchema.Type(),
			r.implementation.FieldNameMap(),
		)
		hasAttributePath := preflightDiagnostic.GetPath() != "" && pathErr == nil
		switch preflightDiagnostic.GetSeverity() {
		case common.PreflightCheckDiagnostic_ERROR:
			if hasAttributePath {
				diagnostics.AddAttributeError(
					attributePath,
					preflightDiagnostic.GetSummary(),
					preflightDiagnostic.GetDetails(),
				)
			} else {
				diagnostics.AddError(
					preflightDiagnostic.GetSummary(),
					preflightDiagnostic.GetDetails(),
				)
			}
		case common.PreflightCheckDiagnostic_WARNING:
			if hasAttributePath {
				diagnostics.AddAttributeWarning(
					attributePath,
					preflightDiagnostic.GetSummary(),
					preflightDiagnostic.GetDetails(),
				)
			} else {
				diagnostics.AddWarning(
					preflightDiagnostic.GetSummary(),
					preflightDiagnostic.GetDetails(),
				)
			}
		case common.PreflightCheckDiagnostic_UNSPECIFIED:
			diagnostics.AddError(
				"invalid preflight diagnostic",
				fmt.Sprintf(
					"The server returned diagnostic %q with unspecified severity.",
					preflightDiagnostic.GetSummary(),
				),
			)
		default:
			diagnostics.AddError(
				"invalid preflight diagnostic",
				fmt.Sprintf(
					"The server returned diagnostic %q with unknown severity %d and details %q.",
					preflightDiagnostic.GetSummary(),
					preflightDiagnostic.GetSeverity(),
					preflightDiagnostic.GetDetails(),
				),
			)
		}
	}
	return diagnostics
}

func (r *commonResource) appendPreflightReplacements(
	ctx context.Context,
	plan tfsdk.Plan,
	resp *resource.ModifyPlanResponse,
	paths string,
) error {
	recreateMask, err := mask.Parse(paths)
	if err != nil {
		return fmt.Errorf("parse paths requiring recreation: %w", err)
	}
	fieldPaths := recreateFieldPaths(recreateMask, nil)
	for _, fieldPath := range fieldPaths {
		if len(fieldPath) > 0 &&
			fieldPath[0] != constants.FieldMetadata &&
			fieldPath[0] != constants.FieldSpec {
			encodedPath, encodeErr := fieldPath.Marshal()
			if encodeErr != nil {
				return fmt.Errorf("encode recreation path: %w", encodeErr)
			}
			return fmt.Errorf(
				"recreation path %q must be rooted at metadata or spec",
				encodedPath,
			)
		}
		if len(fieldPath) == 0 ||
			(len(fieldPath) == 1 &&
				(fieldPath[0] == constants.FieldMetadata || fieldPath[0] == constants.FieldSpec)) {
			resp.RequiresReplace = append(resp.RequiresReplace, path.Empty())
			continue
		}
		encodedPath, err := fieldPath.Marshal()
		if err != nil {
			return fmt.Errorf("encode recreation path: %w", err)
		}
		tfPath, err := requestcontext.FieldPathToTFPath(
			encodedPath,
			r.resourceSchema.Type(),
			r.implementation.FieldNameMap(),
		)
		if err != nil {
			return fmt.Errorf("map recreation path %q: %w", encodedPath, err)
		}
		resp.RequiresReplace = append(
			resp.RequiresReplace,
			nearestAddressablePath(ctx, plan, tfPath),
		)
	}
	return nil
}

func nearestAddressablePath(ctx context.Context, plan tfsdk.Plan, candidate path.Path) path.Path {
	for current := candidate; len(current.Steps()) > 0; current = current.ParentPath() {
		matches, diags := plan.PathMatches(ctx, current.Expression())
		if !diags.HasError() && len(matches) > 0 {
			return matches[0]
		}
	}
	return path.Empty()
}

func recreateFieldPaths(recreateMask *mask.Mask, prefix mask.FieldPath) []mask.FieldPath {
	if recreateMask == nil {
		return nil
	}
	if recreateMask.Any != nil {
		// Terraform paths cannot address wildcard children. Replacing the nearest
		// concrete parent conservatively covers every path selected by the wildcard.
		return []mask.FieldPath{prefix}
	}
	if recreateMask.IsEmpty() {
		return []mask.FieldPath{prefix}
	}
	keys := make([]string, 0, len(recreateMask.FieldParts))
	for key, child := range recreateMask.FieldParts {
		if child != nil {
			keys = append(keys, string(key))
		}
	}
	sort.Strings(keys)
	var result []mask.FieldPath
	for _, key := range keys {
		child := recreateMask.FieldParts[mask.FieldKey(key)]
		childPaths := recreateFieldPaths(
			child,
			prefix.Join(mask.FieldKey(key)),
		)
		result = append(result, childPaths...)
	}
	return result
}
