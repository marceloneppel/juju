// Copyright 2011-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

// Export meaningful bits for tests only.

var (
	IfaceExpander = ifaceExpander

	ResourceSchema            = resourceSchema
	ExtraBindingsSchema       = extraBindingsSchema
	ValidateMetaExtraBindings = validateMetaExtraBindings
	ParseResourceMeta         = parseResourceMeta
)

// BundleEndpoint is an exported view of the internal endpoint struct, used
// only in tests that need to call InferEndpointsForTest directly.
type BundleEndpoint struct {
	Application string
	Relation    string
}

// InferEndpointsForTest exposes inferEndpoints for white-box testing.
func InferEndpointsForTest(ep0, ep1 BundleEndpoint, get func(svc string) (*Meta, error)) (BundleEndpoint, BundleEndpoint, error) {
	r0, r1, err := inferEndpoints(
		endpoint{application: ep0.Application, relation: ep0.Relation},
		endpoint{application: ep1.Application, relation: ep1.Relation},
		get,
	)
	return BundleEndpoint{Application: r0.application, Relation: r0.relation},
		BundleEndpoint{Application: r1.application, Relation: r1.relation},
		err
}
