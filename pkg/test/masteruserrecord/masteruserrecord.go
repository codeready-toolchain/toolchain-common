package masteruserrecord

import (
	"fmt"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	testtier "github.com/codeready-toolchain/toolchain-common/pkg/test/tier"
	"github.com/gofrs/uuid"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type MurModifier func(mur *toolchainv1alpha1.MasterUserRecord) error
type UaInMurModifier func(targetCluster string, mur *toolchainv1alpha1.MasterUserRecord)

const DefaultNSTemplateTierName = "basic"

// DefaultNSTemplateTier the default NSTemplateTier used to initialize the MasterUserRecord
func DefaultNSTemplateTier() toolchainv1alpha1.NSTemplateTier {
	return toolchainv1alpha1.NSTemplateTier{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.HostOperatorNs,
			Name:      DefaultNSTemplateTierName,
		},
		Spec: toolchainv1alpha1.NSTemplateTierSpec{
			Namespaces: []toolchainv1alpha1.NSTemplateTierNamespace{
				{
					TemplateRef: "basic-dev-123abc",
				},
				{
					TemplateRef: "basic-code-123abc",
				},
				{
					TemplateRef: "basic-stage-123abc",
				},
			},
			ClusterResources: &toolchainv1alpha1.NSTemplateTierClusterResources{
				TemplateRef: "basic-clusterresources-654321a",
			},
		},
	}
}

// DefaultNSTemplateSet the default NSTemplateSet used to initialize the MasterUserRecord
func DefaultNSTemplateSet() *toolchainv1alpha1.NSTemplateSet {
	return &toolchainv1alpha1.NSTemplateSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.HostOperatorNs,
			Name:      DefaultNSTemplateTierName,
		},
		Spec: toolchainv1alpha1.NSTemplateSetSpec{
			TierName: DefaultNSTemplateTierName,
			Namespaces: []toolchainv1alpha1.NSTemplateSetNamespace{
				{
					TemplateRef: "basic-dev-123abc",
				},
				{
					TemplateRef: "basic-code-123abc",
				},
				{
					TemplateRef: "basic-stage-123abc",
				},
			},
			ClusterResources: &toolchainv1alpha1.NSTemplateSetClusterResources{
				TemplateRef: "basic-clusterresources-654321a",
			},
		},
	}
}

func NewMasterUserRecords(t *testing.T, size int, nameFmt string, modifiers ...MurModifier) []runtime.Object {
	murs := make([]runtime.Object, size)
	for i := 0; i < size; i++ {
		murs[i] = NewMasterUserRecord(t, fmt.Sprintf(nameFmt, i), modifiers...)
	}
	return murs
}

func NewMasterUserRecord(t *testing.T, userName string, modifiers ...MurModifier) *toolchainv1alpha1.MasterUserRecord {
	userID := uuid.Must(uuid.NewV4()).String()
	mur := &toolchainv1alpha1.MasterUserRecord{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.HostOperatorNs,
			Name:      userName,
			Labels:    map[string]string{},
			Annotations: map[string]string{
				toolchainv1alpha1.MasterUserRecordEmailAnnotationKey: "joe@redhat.com",
			},
		},
		Spec: toolchainv1alpha1.MasterUserRecordSpec{
			TierName:     DefaultNSTemplateSet().Spec.TierName,
			UserID:       userID,
			UserAccounts: []toolchainv1alpha1.UserAccountEmbedded{newEmbeddedUa(test.MemberClusterName)},
		},
	}
	err := Modify(mur, modifiers...)
	require.NoError(t, err)
	return mur
}

func newEmbeddedUa(targetCluster string) toolchainv1alpha1.UserAccountEmbedded {
	return toolchainv1alpha1.UserAccountEmbedded{
		TargetCluster: targetCluster,
		SyncIndex:     "123abc",
	}
}

func Modify(mur *toolchainv1alpha1.MasterUserRecord, modifiers ...MurModifier) error {
	for _, modify := range modifiers {
		if err := modify(mur); err != nil {
			return err
		}
	}
	return nil
}

func ModifyUaInMur(mur *toolchainv1alpha1.MasterUserRecord, targetCluster string, modifiers ...UaInMurModifier) {
	for _, modify := range modifiers {
		modify(targetCluster, mur)
	}
}

func UserID(userID string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		mur.Spec.UserID = userID
		return nil
	}
}

func StatusCondition(con toolchainv1alpha1.Condition) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		mur.Status.Conditions, _ = condition.AddOrUpdateStatusConditions(mur.Status.Conditions, con)
		return nil
	}
}

func MetaNamespace(namespace string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		mur.Namespace = namespace
		return nil
	}
}

func Finalizer(finalizer string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		mur.Finalizers = append(mur.Finalizers, finalizer)
		return nil
	}
}

func TargetCluster(targetCluster string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		for i := range mur.Spec.UserAccounts {
			mur.Spec.UserAccounts[i].TargetCluster = targetCluster
		}
		return nil
	}
}

// Account sets the first account on the MasterUserRecord
func Account(cluster string, tier toolchainv1alpha1.NSTemplateTier, modifiers ...UaInMurModifier) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		mur.Spec.UserAccounts = []toolchainv1alpha1.UserAccountEmbedded{}
		return AdditionalAccount(cluster, tier, modifiers...)(mur)
	}
}

// AdditionalAccount sets an additional account on the MasterUserRecord
func AdditionalAccount(cluster string, tier toolchainv1alpha1.NSTemplateTier, modifiers ...UaInMurModifier) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		ua := toolchainv1alpha1.UserAccountEmbedded{
			TargetCluster: cluster,
			SyncIndex:     "123abc", // default value
		}
		// set the user account
		mur.Spec.UserAccounts = append(mur.Spec.UserAccounts, ua)
		for _, modify := range modifiers {
			modify(cluster, mur)
		}
		// set the labels for the tier templates in use
		hash, err := testtier.ComputeTemplateRefsHash(&tier)
		if err != nil {
			return err
		}
		mur.ObjectMeta.Labels = map[string]string{
			toolchainv1alpha1.LabelKeyPrefix + tier.Name + "-tier-hash": hash,
		}
		return nil
	}
}

func AdditionalAccounts(clusters ...string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		for _, cluster := range clusters {
			mur.Spec.UserAccounts = append(mur.Spec.UserAccounts, newEmbeddedUa(cluster))
		}
		return nil
	}
}

func TierName(tierName string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		mur.Spec.TierName = tierName
		return nil
	}
}

func SyncIndex(index string) UaInMurModifier {
	return func(targetCluster string, mur *toolchainv1alpha1.MasterUserRecord) {
		for i, ua := range mur.Spec.UserAccounts {
			if ua.TargetCluster == targetCluster {
				mur.Spec.UserAccounts[i].SyncIndex = index
				return
			}
		}
	}
}

func ToBeDeleted() MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		util.AddFinalizer(mur, "finalizer.toolchain.dev.openshift.com")
		mur.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		return nil
	}
}

// DisabledMur creates a MurModifier to change the disabled spec value
func DisabledMur(disabled bool) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		mur.Spec.Disabled = disabled
		return nil
	}
}

// ProvisionedMur creates a MurModifier to change the provisioned status value
func ProvisionedMur(provisionedTime *metav1.Time) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		mur.Status.ProvisionedTime = provisionedTime
		return nil
	}
}

// UserIDFromUserSignup creates a MurModifier to change the userID value to match the provided usersignup
func UserIDFromUserSignup(userSignup *toolchainv1alpha1.UserSignup) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		mur.Spec.UserID = userSignup.Name
		return nil
	}
}

// WithAnnotation sets an annotation with the given key/value
func WithAnnotation(key, value string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		if mur.Annotations == nil {
			mur.Annotations = map[string]string{}
		}
		mur.Annotations[key] = value
		return nil
	}
}

// WithLabel sets a label with the given key/value
func WithLabel(key, value string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		if mur.Labels == nil {
			mur.Labels = map[string]string{}
		}
		mur.Labels[key] = value
		return nil
	}
}

func WithOwnerLabel(owner string) MurModifier {
	return WithLabel(toolchainv1alpha1.MasterUserRecordOwnerLabelKey, owner)
}
