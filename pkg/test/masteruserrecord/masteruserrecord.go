package masteruserrecord

import (
	"fmt"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/gofrs/uuid"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type MurModifier func(mur *toolchainv1alpha1.MasterUserRecord) error
type UaInMurModifier func(targetCluster string, mur *toolchainv1alpha1.MasterUserRecord)

const DefaultUserTierName = "deactivate30"

// DefaultUserTier the default UserTier used to initialize the MasterUserRecord
func DefaultUserTier() toolchainv1alpha1.UserTier {
	return toolchainv1alpha1.UserTier{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.HostOperatorNs,
			Name:      DefaultUserTierName,
		},
		Spec: toolchainv1alpha1.UserTierSpec{
			DeactivationTimeoutDays: 30,
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
			TierName:     "deactivate30",
			UserID:       userID,
			UserAccounts: []toolchainv1alpha1.UserAccountEmbedded{newEmbeddedUa(test.MemberClusterName)},
			PropagatedClaims: toolchainv1alpha1.PropagatedClaims{
				Sub:         "44332211",
				UserID:      "135246",
				AccountID:   "357468",
				OriginalSub: "11223344",
				Email:       "joe@redhat.com",
			},
		},
	}
	err := Modify(mur, modifiers...)
	require.NoError(t, err)
	return mur
}

func newEmbeddedUa(targetCluster string) toolchainv1alpha1.UserAccountEmbedded {
	return toolchainv1alpha1.UserAccountEmbedded{
		TargetCluster: targetCluster,
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
func Account(cluster string, modifiers ...UaInMurModifier) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		mur.Spec.UserAccounts = []toolchainv1alpha1.UserAccountEmbedded{}
		return AdditionalAccount(cluster, modifiers...)(mur)
	}
}

// AdditionalAccount sets an additional account on the MasterUserRecord
func AdditionalAccount(cluster string, modifiers ...UaInMurModifier) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		ua := toolchainv1alpha1.UserAccountEmbedded{
			TargetCluster: cluster,
		}
		// set the user account
		mur.Spec.UserAccounts = append(mur.Spec.UserAccounts, ua)
		for _, modify := range modifiers {
			modify(cluster, mur)
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
