package masteruserrecord

import (
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/redhat-cop/operator-utils/pkg/util"
	uuid "github.com/satori/go.uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MurModifier func(mur *toolchainv1alpha1.MasterUserRecord)
type UaInMurModifier func(targetCluster string, mur *toolchainv1alpha1.MasterUserRecord)

func NewMasterUserRecord(userName string, modifiers ...MurModifier) *toolchainv1alpha1.MasterUserRecord {
	userId := uuid.NewV4().String()
	mur := &toolchainv1alpha1.MasterUserRecord{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userName,
			Namespace: test.HostOperatorNs,
		},
		Spec: toolchainv1alpha1.MasterUserRecordSpec{
			UserID:       userId,
			UserAccounts: []toolchainv1alpha1.UserAccountEmbedded{newEmbeddedUa(test.MemberClusterName, userId)},
		},
	}
	Modify(mur, modifiers...)
	return mur
}

func newEmbeddedUa(targetCluster, userId string) toolchainv1alpha1.UserAccountEmbedded {
	return toolchainv1alpha1.UserAccountEmbedded{
		TargetCluster: targetCluster,
		SyncIndex:     "123abc",
		Spec: toolchainv1alpha1.UserAccountSpec{
			UserID:  userId,
			NSLimit: "basic",
			NSTemplateSet: toolchainv1alpha1.NSTemplateSetSpec{
				TierName: "basic",
				Namespaces: []toolchainv1alpha1.NSTemplateSetNamespace{
					{
						Type:     "dev",
						Revision: "123abc",
						Template: "",
					},
					{
						Type:     "code",
						Revision: "123abc",
						Template: "",
					},
					{
						Type:     "stage",
						Revision: "123abc",
						Template: "",
					},
				},
			},
		},
	}
}

func Modify(mur *toolchainv1alpha1.MasterUserRecord, modifiers ...MurModifier) {
	for _, modify := range modifiers {
		modify(mur)
	}
}

func ModifyUaInMur(mur *toolchainv1alpha1.MasterUserRecord, targetCluster string, modifiers ...UaInMurModifier) {
	for _, modify := range modifiers {
		modify(targetCluster, mur)
	}
}

func StatusCondition(con toolchainv1alpha1.Condition) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) {
		mur.Status.Conditions, _ = condition.AddOrUpdateStatusConditions(mur.Status.Conditions, con)
	}
}

func MetaNamespace(namespace string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) {
		mur.Namespace = namespace
	}
}

func TargetCluster(targetCluster string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) {
		for i := range mur.Spec.UserAccounts {
			mur.Spec.UserAccounts[i].TargetCluster = targetCluster
		}
	}
}

func AdditionalAccounts(clusters ...string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) {
		for _, cluster := range clusters {
			mur.Spec.UserAccounts = append(mur.Spec.UserAccounts, newEmbeddedUa(cluster, mur.Spec.UserID))
		}
	}
}

func NsLimit(limit string) UaInMurModifier {
	return func(targetCluster string, mur *toolchainv1alpha1.MasterUserRecord) {
		for i, ua := range mur.Spec.UserAccounts {
			if ua.TargetCluster == targetCluster {
				mur.Spec.UserAccounts[i].Spec.NSLimit = limit
				return
			}
		}
	}
}

func TierName(tierName string) UaInMurModifier {
	return func(targetCluster string, mur *toolchainv1alpha1.MasterUserRecord) {
		for i, ua := range mur.Spec.UserAccounts {
			if ua.TargetCluster == targetCluster {
				mur.Spec.UserAccounts[i].Spec.NSTemplateSet.TierName = tierName
				return
			}
		}
	}
}

func Namespace(nsType, revision string) UaInMurModifier {
	return func(targetCluster string, mur *toolchainv1alpha1.MasterUserRecord) {
		for uaIndex, ua := range mur.Spec.UserAccounts {
			if ua.TargetCluster == targetCluster {
				for nsIndex, ns := range mur.Spec.UserAccounts[uaIndex].Spec.NSTemplateSet.Namespaces {
					if ns.Type == nsType {
						mur.Spec.UserAccounts[uaIndex].Spec.NSTemplateSet.Namespaces[nsIndex].Revision = revision
					}
				}
				return
			}
		}
	}
}

func ToBeDeleted() MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) {
		util.AddFinalizer(mur, "finalizer.toolchain.dev.openshift.com")
		mur.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	}
}
