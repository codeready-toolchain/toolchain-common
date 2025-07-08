package nstemplatetiers

import (
	"fmt"
	"sort"
	"strings"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	commonTemplate "github.com/codeready-toolchain/toolchain-common/pkg/template"
	templatev1 "github.com/openshift/api/template/v1"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("templates")

type EnsureObject func(toEnsure runtimeclient.Object, tierName string) error

type TierGenerator struct {
	ensureObject    EnsureObject
	namespace       string
	scheme          *runtime.Scheme
	templatesByTier map[string]*tierData
}

type tierData struct {
	name          string
	rawTemplates  *templates
	tierTemplates []*toolchainv1alpha1.TierTemplate
	objects       []runtimeclient.Object
	basedOnTier   *BasedOnTier
}

// templates: namespaces and other cluster-scoped resources belonging to a given tier ("advanced", "base", "team", etc.) and the NSTemplateTier that combines them
type templates struct {
	nsTemplateTier     *template           // NSTemplateTier resource with tier-scoped configuration and references to namespace and cluster templates in its spec, in a single template file
	clusterTemplate    *template           // other cluster-scoped resources, in a single template file
	namespaceTemplates map[string]template // namespace templates (including roles, limits, etc.) indexed by type ("dev", "stage")
	spaceroleTemplates map[string]template // spacerole templates (including rolebindings, etc.) indexed by role ("admin", "viewer", etc.)
	basedOnTier        *template           // a special config defining which tier should be reused and which parameters should be overridden
}

// template: a template's content and its latest git revision
type template struct {
	revision string
	content  []byte
}

// GenerateTiers processes the given metadata and files, generates TierTemplates and NSTemplateTiers, and ensures them via the provided EnsureObject function
func GenerateTiers(s *runtime.Scheme, ensureObject EnsureObject, namespace string, metadata map[string]string, files map[string][]byte) error {
	generator, err := newNSTemplateTierGenerator(s, ensureObject, namespace, metadata, files)
	if err != nil {
		return errors.Wrap(err, "unable to init NSTemplateTier generator")
	}

	// create the TierTemplate resources
	err = generator.createTierTemplates()
	if err != nil {
		return errors.Wrap(err, "unable to create TierTemplates")
	}

	// create the NSTemplateTier resources
	err = generator.createNSTemplateTiers()
	if err != nil {
		return errors.Wrap(err, "unable to create NSTemplateTiers")
	}
	return nil
}

// newNSTemplateTierGenerator loads templates from the provided assets and processes the tierTemplates and NSTemplateTiers
func newNSTemplateTierGenerator(s *runtime.Scheme, ensureObject EnsureObject, namespace string, metadata map[string]string, files map[string][]byte) (*TierGenerator, error) {
	templatesByTier, err := loadTemplatesByTiers(metadata, files)
	if err != nil {
		return nil, err
	}

	c := &TierGenerator{
		ensureObject:    ensureObject,
		namespace:       namespace,
		scheme:          s,
		templatesByTier: templatesByTier,
	}

	// process tierTemplates
	if err := c.initTierTemplates(); err != nil {
		return nil, err
	}

	// process NSTemplateTiers
	if err := c.initNSTemplateTiers(); err != nil {
		return nil, err
	}

	return c, nil
}

// BasedOnTier defines which tier is supposed to be reused and which parameters should be modified
// An example:
//
// from: base
// parameters:
//   - name: IDLER_TIMEOUT_SECONDS
//     value: 43200
//
// Which defines that for creating baseextendedidling tier the base tier should be used and
// the parameter IDLER_TIMEOUT_SECONDS should be set to 43200
type BasedOnTier struct {
	Revision   string
	From       string                 `json:"from"`
	Parameters []templatev1.Parameter `json:"parameters,omitempty" protobuf:"bytes,4,rep,name=parameters"`
}

// loadTemplatesByTiers loads the files and dispatches them by tiers, assuming the given files has the following structure:
//
// advanced/
//
//	based_on_tier.yaml
//
// base1ns/
//
//	cluster.yaml
//	ns_dev.yaml
//	ns_stage.yaml
//	spacerole_admin.yaml
//	tier.yaml
//
// team/
//
//	based_on_tier.yaml
//
// The output is a map of `tierData` indexed by tier.
// Each `tierData` object contains itself a map of `template` objects indexed by the namespace type (`namespaceTemplates`);
// an optional `template` for the cluster resources (`clusterTemplate`) and the NSTemplateTier resource object.
// Each `template` object contains a `revision` (`string`) and the `content` of the template to apply (`[]byte`)
func loadTemplatesByTiers(metadata map[string]string, files map[string][]byte) (map[string]*tierData, error) {
	results := make(map[string]*tierData)
	for name, content := range files {
		// split the name using the `/` separator
		parts := strings.Split(name, "/")
		// skip any name that does not have 2 parts
		if len(parts) != 2 {
			return nil, fmt.Errorf("unable to load templates: invalid name format for file '%s'", name)
		}
		tier := parts[0]
		filename := parts[1]
		if _, exists := results[tier]; !exists {
			results[tier] = &tierData{
				name: tier,
				rawTemplates: &templates{
					namespaceTemplates: map[string]template{},
					spaceroleTemplates: map[string]template{},
				},
			}
		}

		tmpl := template{
			revision: metadata[strings.TrimSuffix(name, ".yaml")],
			content:  content,
		}
		switch {
		case filename == "tier.yaml":
			results[tier].rawTemplates.nsTemplateTier = &tmpl
		case filename == "cluster.yaml":
			results[tier].rawTemplates.clusterTemplate = &tmpl
		case strings.HasPrefix(filename, "ns_"):
			kind := strings.TrimSuffix(strings.TrimPrefix(filename, "ns_"), ".yaml")
			results[tier].rawTemplates.namespaceTemplates[kind] = tmpl
		case strings.HasPrefix(filename, "spacerole_"):
			role := strings.TrimSuffix(strings.TrimPrefix(filename, "spacerole_"), ".yaml")
			results[tier].rawTemplates.spaceroleTemplates[role] = tmpl
		case filename == "based_on_tier.yaml":
			basedOnTier := &BasedOnTier{}
			if err := yaml.Unmarshal(content, basedOnTier); err != nil {
				return nil, errors.Wrapf(err, "unable to unmarshal '%s'", name)
			}
			results[tier].rawTemplates.basedOnTier = &tmpl
			results[tier].basedOnTier = basedOnTier
		default:
			return nil, errors.Errorf("unable to load templates: unknown scope for file '%s'", name)
		}
	}

	// check that none of the tiers uses combination of based_on_tier.yaml file together with any template file
	for tier, tierData := range results {
		if tierData.rawTemplates.basedOnTier != nil &&
			(tierData.rawTemplates.clusterTemplate != nil ||
				len(tierData.rawTemplates.namespaceTemplates) > 0 ||
				tierData.rawTemplates.nsTemplateTier != nil) {
			return nil, fmt.Errorf("the tier %s contains a mix of based_on_tier.yaml file together with a regular template file", tier)
		}
	}
	return results, nil
}

// initTierTemplates generates all TierTemplate resources, and adds them to the tier map indexed by tier name
func (t *TierGenerator) initTierTemplates() error {
	// process tiers in alphabetical order
	tiers := make([]string, 0, len(t.templatesByTier))
	for tier := range t.templatesByTier {
		tiers = append(tiers, tier)
	}
	sort.Strings(tiers)
	for _, tier := range tiers {
		tierData := t.templatesByTier[tier]
		basedOnTierFileRevision := ""
		var parameters []templatev1.Parameter
		if tierData.basedOnTier != nil {
			parameters = tierData.basedOnTier.Parameters
			basedOnTierFileRevision = tierData.rawTemplates.basedOnTier.revision
			tierData = t.templatesByTier[tierData.basedOnTier.From]
		}
		tierTemplates, err := t.newTierTemplates(basedOnTierFileRevision, tierData, tier, parameters)
		if err != nil {
			return err
		}
		t.templatesByTier[tier].tierTemplates = tierTemplates
	}

	return nil
}

func (t *TierGenerator) newTierTemplates(basedOnTierFileRevision string, tierData *tierData, tier string, parameters []templatev1.Parameter) ([]*toolchainv1alpha1.TierTemplate, error) {
	decoder := serializer.NewCodecFactory(t.scheme).UniversalDeserializer()

	// namespace templates
	kinds := make([]string, 0, len(tierData.rawTemplates.namespaceTemplates))
	for kind := range tierData.rawTemplates.namespaceTemplates {
		kinds = append(kinds, kind)
	}
	tierTmpls := []*toolchainv1alpha1.TierTemplate{}
	sort.Strings(kinds)
	for _, kind := range kinds {
		tmpl := tierData.rawTemplates.namespaceTemplates[kind]
		tierTmpl, err := t.newTierTemplate(decoder, basedOnTierFileRevision, tier, kind, tmpl, parameters)
		if err != nil {
			return nil, err
		}
		tierTmpls = append(tierTmpls, tierTmpl)
	}
	// space roles templates
	roles := make([]string, 0, len(tierData.rawTemplates.spaceroleTemplates))
	for role := range tierData.rawTemplates.spaceroleTemplates {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	for _, role := range roles {
		tmpl := tierData.rawTemplates.spaceroleTemplates[role]
		tierTmpl, err := t.newTierTemplate(decoder, basedOnTierFileRevision, tier, role, tmpl, parameters)
		if err != nil {
			return nil, err
		}
		tierTmpls = append(tierTmpls, tierTmpl)
	}
	// cluster resources templates
	if tierData.rawTemplates.clusterTemplate != nil {
		tierTmpl, err := t.newTierTemplate(decoder, basedOnTierFileRevision, tier, toolchainv1alpha1.ClusterResourcesTemplateType, *tierData.rawTemplates.clusterTemplate, parameters)
		if err != nil {
			return nil, err
		}
		tierTmpls = append(tierTmpls, tierTmpl)
	}
	return tierTmpls, nil
}

// createTierTemplates creates all TierTemplate resources from the tier map
func (t *TierGenerator) createTierTemplates() error {
	// create the templates
	for tierName, tierTmpls := range t.templatesByTier {
		for _, tierTmpl := range tierTmpls.tierTemplates {
			log.Info("creating TierTemplate", "namespace", tierTmpl.Namespace, "name", tierTmpl.Name)
			if err := t.ensureObject(tierTmpl, tierName); err != nil {
				return errors.Wrapf(err, "unable to create the '%s' TierTemplate in namespace '%s'", tierTmpl.Name, tierTmpl.Namespace)
			}
			log.Info("TierTemplate resource created", "namespace", tierTmpl.Namespace, "name", tierTmpl.Name)
		}
	}
	return nil
}

// newTierTemplate generates a TierTemplate resource for a given tier and kind
func (t *TierGenerator) newTierTemplate(decoder runtime.Decoder, basedOnTierFileRevision, tier, kind string, tmpl template, parameters []templatev1.Parameter) (*toolchainv1alpha1.TierTemplate, error) {
	if basedOnTierFileRevision == "" {
		basedOnTierFileRevision = tmpl.revision
	}
	revision := fmt.Sprintf("%s-%s", basedOnTierFileRevision, tmpl.revision)
	name := newTierTemplateName(tier, kind, revision)
	tmplObj := &templatev1.Template{}
	_, _, err := decoder.Decode(tmpl.content, nil, tmplObj)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to generate '%s' TierTemplate manifest", name)
	}
	setParams(parameters, tmplObj)

	return &toolchainv1alpha1.TierTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      name, // link to the TierTemplate resource, whose name is: `<tierName>-<nsType>-<revision>`,
		},
		Spec: toolchainv1alpha1.TierTemplateSpec{
			Revision: revision,
			TierName: tier,
			Type:     kind,
			Template: *tmplObj,
		},
	}, nil
}

// setParams sets the value for each of the keys in the given parameter set to the template, but only if the key exists there
func setParams(parametersToSet []templatev1.Parameter, tmpl *templatev1.Template) {
	for _, paramToSet := range parametersToSet {
		for i, param := range tmpl.Parameters {
			if param.Name == paramToSet.Name {
				tmpl.Parameters[i].Value = paramToSet.Value
				break
			}
		}
	}
}

// newTierTemplateName a utility func to generate a TierTemplate name, based on the given tier, kind and revision.
// note: the resource name must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character
func newTierTemplateName(tier, kind, revision string) string {
	return strings.ToLower(fmt.Sprintf("%s-%s-%s", tier, kind, revision))
}

// newNSTemplateTiers generates all NSTemplateTier resources and adds them to the tier map
func (t *TierGenerator) initNSTemplateTiers() error {
	for tierName, tierData := range t.templatesByTier {
		nsTemplateTier := tierData.rawTemplates.nsTemplateTier
		tierTemplates := tierData.tierTemplates
		sourceTierName := tierName
		var parameters []templatev1.Parameter
		if tierData.basedOnTier != nil {
			parameters = tierData.basedOnTier.Parameters
			fromData := t.templatesByTier[tierData.basedOnTier.From]
			nsTemplateTier = fromData.rawTemplates.nsTemplateTier
			sourceTierName = fromData.name
		}
		objs, err := t.newNSTemplateTier(sourceTierName, tierName, nsTemplateTier, tierTemplates, parameters)
		if err != nil {
			return err
		}
		t.templatesByTier[tierName].objects = objs
	}

	return nil
}

// createNSTemplateTiers creates the NSTemplateTier resources from the tier map
func (t *TierGenerator) createNSTemplateTiers() error {
	for tierName, tierData := range t.templatesByTier {
		if len(tierData.objects) != 1 {
			return fmt.Errorf("there is an unexpected number of NSTemplateTier object to be applied for tier name '%s'; expected: 1; actual: %d", tierName, len(tierData.objects))
		}

		unstructuredObj, ok := tierData.objects[0].(*unstructured.Unstructured)
		if !ok {
			return fmt.Errorf("unable to cast NSTemplateTier '%s' to Unstructured object '%+v'", tierName, tierData.objects[0])
		}
		tier := &toolchainv1alpha1.NSTemplateTier{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, tier); err != nil {
			return err
		}

		labels := tier.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[toolchainv1alpha1.ProviderLabelKey] = toolchainv1alpha1.ProviderLabelValue
		err := t.ensureObject(tier, tierName)
		if err != nil {
			return errors.Wrapf(err, "unable to create or update the '%s' NSTemplateTier", tierName)
		}
		tierLog := log.WithValues("name", tierName)
		if tier.Spec.ClusterResources != nil {
			tierLog = tierLog.WithValues("clusterResourcesTemplate", tier.Spec.ClusterResources.TemplateRef)
		}
		for i, nsTemplate := range tier.Spec.Namespaces {
			tierLog = tierLog.WithValues(fmt.Sprintf("namespaceTemplate-%d", i), nsTemplate.TemplateRef)
		}
		for role, nsTemplate := range tier.Spec.SpaceRoles {
			tierLog = tierLog.WithValues(fmt.Sprintf("spaceRoleTemplate-%s", role), nsTemplate.TemplateRef)
		}
		tierLog.Info("NSTemplateTier was patched")
	}
	return nil
}

// NewNSTemplateTier generates a complete NSTemplateTier object via Openshift Template based on the contents of tier.yaml and
// by embedding the `<tier>-code.yaml`, `<tier>-dev.yaml` and `<tier>-stage.yaml` and cluster.yaml references.
//
// After processing the Openshift Template the NSTemplateTier should look something like:
// ------
// kind: NSTemplateTier
//
//	metadata:
//	  name: appstudio
//	spec:
//	  deactivationTimeoutDays: 30
//	  clusterResources:
//	    templateRef: appstudio-clusterresources-07cac69-07cac69
//	  namespaces:
//	  - templateRef: appstudio-code-cb6fbd2-cb6fbd2
//	  - templateRef: appstudio-dev-4d49fe0-4d49fe0
//	  - templateRef: appstudio-stage-4d49fe0-4d49fe0
//	  spaceRoles:
//	    admin:
//	      templateRef: appstudio-admin-ab12cd34-ab12cd34
//	    viewer:
//	      templateRef: appstudio-admin-ab12cd34-ab12cd34
//
// ------
func (t *TierGenerator) newNSTemplateTier(sourceTierName, tierName string, nsTemplateTier *template, tierTemplates []*toolchainv1alpha1.TierTemplate, parameters []templatev1.Parameter) ([]runtimeclient.Object, error) {
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()
	if nsTemplateTier == nil {
		return nil, fmt.Errorf("tier %s is missing a tier.yaml file", tierName)
	}

	tmplObj := &templatev1.Template{}
	_, _, err := decoder.Decode(nsTemplateTier.content, nil, tmplObj)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to generate '%s' NSTemplateTier manifest", tierName)
	}

	tmplProcessor := commonTemplate.NewProcessor(t.scheme)
	params := map[string]string{"NAMESPACE": t.namespace}

	for _, tierTmpl := range tierTemplates {
		switch tierTmpl.Spec.Type {
		// ClusterResources
		case toolchainv1alpha1.ClusterResourcesTemplateType:
			params["CLUSTER_TEMPL_REF"] = tierTmpl.Name
		// Namespaces and Space Roles
		default:
			tmplType := strings.ToUpper(tierTmpl.Spec.Type) // code, dev, stage
			key := tmplType + "_TEMPL_REF"                  // eg. CODE_TEMPL_REF
			params[key] = tierTmpl.Name
		}
	}
	setParams(parameters, tmplObj)
	toolchainObjects, err := tmplProcessor.Process(tmplObj.DeepCopy(), params)
	if err != nil {
		return nil, err
	}
	for i := range toolchainObjects {
		toolchainObjects[i].SetName(strings.Replace(toolchainObjects[i].GetName(), sourceTierName, tierName, 1))
	}
	return toolchainObjects, nil
}
