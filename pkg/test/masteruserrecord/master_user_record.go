package masteruserrecord

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/condition"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/redhat-cop/operator-utils/pkg/util"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type MurModifier func(mur *toolchainv1alpha1.MasterUserRecord) error
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

func NewMasterUserRecords(t *testing.T, size int, nameFmt string, modifiers ...MurModifier) []runtime.Object {
	murs := make([]runtime.Object, size)
	for i := 0; i < size; i++ {
		murs[i] = NewMasterUserRecord(t, fmt.Sprintf(nameFmt, i), modifiers...)
	}
	return murs
}

func NewMasterUserRecord(t *testing.T, userName string, modifiers ...MurModifier) *toolchainv1alpha1.MasterUserRecord {
	userID := uuid.NewV4().String()
	hash, err := computeTemplateRefsHash(DefaultNSTemplateTier) // we can assume the JSON marshalling will always work
	require.NoError(t, err)
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
	err = Modify(mur, modifiers...)
	require.NoError(t, err)
	return mur
}

// templateTierHashLabel returns the label key to specify the version of the templates of the given tier
func templateTierHashLabelKey(tierName string) string {
	return toolchainv1alpha1.LabelKeyPrefix + tierName + "-tier-hash"
}

type templateRefs struct {
	Refs []string `json:"refs"`
}

// computeTemplateRefsHash computes the hash of the `.spec.namespaces[].templateRef` + `.spec.clusteResource.TemplateRef`
func computeTemplateRefsHash(tier toolchainv1alpha1.NSTemplateTier) (string, error) {
	refs := []string{}
	for _, ns := range tier.Spec.Namespaces {
		refs = append(refs, ns.TemplateRef)
	}
	if tier.Spec.ClusterResources != nil {
		refs = append(refs, tier.Spec.ClusterResources.TemplateRef)
	}
	sort.Strings(refs)
	m, err := json.Marshal(templateRefs{Refs: refs})
	if err != nil {
		return "", err
	}
	md5hash := md5.New()
	// Ignore the error, as this implementation cannot return one
	_, _ = md5hash.Write(m)
	hash := hex.EncodeToString(md5hash.Sum(nil))
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
		templates := nstemplateSetFromTier(tier)
		ua := toolchainv1alpha1.UserAccountEmbedded{
			TargetCluster: cluster,
			SyncIndex:     "123abc", // default value
			Spec: toolchainv1alpha1.UserAccountSpecEmbedded{
				UserAccountSpecBase: toolchainv1alpha1.UserAccountSpecBase{
					NSLimit:       tier.Name,
					NSTemplateSet: templates,
				},
			},
		}
		// set the user account
		mur.Spec.UserAccounts = append(mur.Spec.UserAccounts, ua)
		for _, modify := range modifiers {
			modify(cluster, mur)
		}
		// set the labels for the tier templates in use
		hash, err := computeTemplateRefsHash(tier)
		if err != nil {
			return err
		}
		mur.ObjectMeta.Labels = map[string]string{
			toolchainv1alpha1.LabelKeyPrefix + tier.Name + "-tier-hash": hash,
		}
		return nil
	}
}

func nstemplateSetFromTier(tier toolchainv1alpha1.NSTemplateTier) toolchainv1alpha1.NSTemplateSetSpec {
	s := toolchainv1alpha1.NSTemplateSetSpec{}
	s.TierName = tier.Name
	s.Namespaces = make([]toolchainv1alpha1.NSTemplateSetNamespace, len(tier.Spec.Namespaces))
	for i, ns := range tier.Spec.Namespaces {
		s.Namespaces[i].TemplateRef = ns.TemplateRef
	}
	s.ClusterResources = &toolchainv1alpha1.NSTemplateSetClusterResources{}
	s.ClusterResources.TemplateRef = tier.Spec.ClusterResources.TemplateRef
	return s
}

func AdditionalAccounts(clusters ...string) MurModifier {
	return func(mur *toolchainv1alpha1.MasterUserRecord) error {
		for _, cluster := range clusters {
			mur.Spec.UserAccounts = append(mur.Spec.UserAccounts, newEmbeddedUa(cluster))
		}
		return nil
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
