package masteruserrecord

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
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

// DefaultNSTemplateTier the default NSTemplateTier used to initialize the MasterUserRecord
var DefaultNSTemplateTier = toolchainv1alpha1.NSTemplateTier{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: test.HostOperatorNs,
		Name:      "basic",
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

// DefaultNSTemplateSet the default NSTemplateSet used to initialize the MasterUserRecord
var DefaultNSTemplateSet = toolchainv1alpha1.NSTemplateSet{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: test.HostOperatorNs,
		Name:      DefaultNSTemplateTier.Name,
	},
	Spec: toolchainv1alpha1.NSTemplateSetSpec{
		TierName: DefaultNSTemplateTier.Name,
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

func NewMasterUserRecord(userName string, modifiers ...MurModifier) *toolchainv1alpha1.MasterUserRecord {
	userID := uuid.NewV4().String()
	hash, _ := computeTemplateRefsHash(DefaultNSTemplateTier) // we can assume the JSON marshalling will always work
	mur := &toolchainv1alpha1.MasterUserRecord{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.HostOperatorNs,
			Name:      userName,
			Labels: map[string]string{
				templateTierHashLabelKey(DefaultNSTemplateTier.Name): hash,
			},
		},
		Spec: toolchainv1alpha1.MasterUserRecordSpec{
			UserID:       userID,
			UserAccounts: []toolchainv1alpha1.UserAccountEmbedded{newEmbeddedUa(test.MemberClusterName)},
		},
	}
	Modify(mur, modifiers...)
	return mur
}

// templateTierHashLabel returns the label key to specify the version of the templates of the given tier
func templateTierHashLabelKey(tierName string) string {
	return toolchainv1alpha1.LabelKeyPrefix + tierName + "-tier-hash"
}

type templateRefs struct {
	Namespaces       []string `json:"namespaces"`
	ClusterResources string   `json:"clusterresource,omitempty"`
}

// ComputeTemplateRefsHash computes the hash of the `.spec.namespaces[].templateRef` + `.spec.clusteResource.TemplateRef`
func computeTemplateRefsHash(tier toolchainv1alpha1.NSTemplateTier) (string, error) {
	refs := templateRefs{}
	for _, ns := range tier.Spec.Namespaces {
		refs.Namespaces = append(refs.Namespaces, ns.TemplateRef)
	}
	if tier.Spec.ClusterResources != nil {
		refs.ClusterResources = tier.Spec.ClusterResources.TemplateRef
	}
	m, err := json.Marshal(refs)
	if err != nil {
		return "", err
	}
	md5hash := md5.New()
	// Ignore the error, as this implementation cannot return one
	_, _ = md5hash.Write(m)
	hash := hex.EncodeToString(md5hash.Sum(nil))
	fmt.Printf("[test] computing hash for '%v' -> '%s' -> '%s'\n", refs, m, hash)
	return hash, nil
}

func newEmbeddedUa(targetCluster string) toolchainv1alpha1.UserAccountEmbedded {
	return toolchainv1alpha1.UserAccountEmbedded{
		TargetCluster: targetCluster,
		SyncIndex:     "123abc",
		Spec: toolchainv1alpha1.UserAccountSpecEmbedded{
			UserAccountSpecBase: toolchainv1alpha1.UserAccountSpecBase{
				NSLimit:       "basic",
				NSTemplateSet: DefaultNSTemplateSet.Spec,
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

func Finalizer(finalizer string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) {
		mur.Finalizers = append(mur.Finalizers, finalizer)
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
			mur.Spec.UserAccounts = append(mur.Spec.UserAccounts, newEmbeddedUa(cluster))
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
					if strings.Contains(ns.TemplateRef, nsType) {
						templateRef := strings.ToLower(fmt.Sprintf("%s-%s-%s", mur.Spec.UserAccounts[uaIndex].Spec.NSTemplateSet.TierName, nsType, revision))
						mur.Spec.UserAccounts[uaIndex].Spec.NSTemplateSet.Namespaces[nsIndex].TemplateRef = templateRef
						return
					}
				}
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

// DisabledMur creates a MurModifier to change the disabled spec value
func DisabledMur(disabled bool) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) {
		mur.Spec.Disabled = disabled
	}
}
