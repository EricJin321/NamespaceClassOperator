# PR: Update +kubebuilder:rbac Annotations

## Summary
This PR ensures that all necessary `+kubebuilder:rbac` annotations are present in the `controller` and `webhook` logic. The annotations have been updated to align with the code logic and resource access patterns.

## Changes
1. **Controller Updates**
   - File: `internal/controller/namespaceiteminstance_controller.go`
   - Added/verified the following annotations:
     ```go
     // +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceiteminstances,verbs=get;list;watch;create;update;patch;delete
     // +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceiteminstances/status,verbs=get;update;patch
     // +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceiteminstances/finalizers,verbs=update
     // +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclassitems,verbs=get;list;watch
     // +kubebuilder:rbac:groups=*,resources=*,verbs=get;list;watch;create;update;patch;delete,namespace=*
     ```

2. **Webhook Updates**
   - File: `internal/webhook/v1alpha/namespaceclassitem_webhook.go`
   - Added/verified the following annotations:
     ```go
     // +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclassitems,verbs=get;list;watch;create;update;patch;delete
     // +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclassitems/status,verbs=get;update;patch
     ```

## Testing
- Verified that the annotations align with the resource access patterns in the code.
- Ensured no redundant or missing annotations.

## Notes
Please review the changes and let me know if further adjustments are needed.