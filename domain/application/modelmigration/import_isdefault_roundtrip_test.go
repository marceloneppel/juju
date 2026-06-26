// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelmigration

import (
	"strings"
	"testing"
	"time"

	"github.com/juju/description/v12"
	"github.com/juju/tc"
)

// descRelation is a minimal description.CharmMetadataRelation, used to build a
// model for serialization the same way a model export does.
type descRelation struct {
	name, role, iface, scope string
	optional, isDefault      bool
	limit                    int
}

func (r descRelation) Name() string      { return r.name }
func (r descRelation) Role() string      { return r.role }
func (r descRelation) Interface() string { return r.iface }
func (r descRelation) Optional() bool    { return r.optional }
func (r descRelation) IsDefault() bool   { return r.isDefault }
func (r descRelation) Limit() int        { return r.limit }
func (r descRelation) Scope() string     { return r.scope }

type isDefaultRoundTripSuite struct{}

func TestIsDefaultRoundTripSuite(t *testing.T) {
	tc.Run(t, &isDefaultRoundTripSuite{})
}

// TestIsDefaultSurvivesDescriptionRoundTrip drives the per-endpoint default flag
// through the charm-metadata model-migration serialization chain: it builds a
// description model carrying is-default, serializes and deserializes it with the
// real description package (the version Juju depends on via the go.mod replace),
// then runs Juju's importRelations over the result. This exercises the exact code
// the feature touches end to end; only the cross-controller transport is omitted,
// which is unrelated to the feature and currently unimplemented upstream.
func (s *isDefaultRoundTripSuite) TestIsDefaultSurvivesDescriptionRoundTrip(c *tc.C) {
	model := description.NewModel(description.ModelArgs{Type: "iaas"})
	app := model.AddApplication(description.ApplicationArgs{
		Name:     "db-app",
		CharmURL: "ch:db-app-1",
	})
	app.SetStatus(description.StatusArgs{
		Value:   "active",
		Updated: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	app.SetCharmMetadata(description.CharmMetadataArgs{
		Name: "relation-default-requirer",
		Requires: map[string]description.CharmMetadataRelation{
			"certs-main": descRelation{
				name: "certs-main", role: roleRequirer, iface: "shared",
				scope: scopeGlobal, isDefault: true,
			},
			"certs-extra": descRelation{
				name: "certs-extra", role: roleRequirer, iface: "shared",
				scope: scopeGlobal,
			},
		},
	})

	// Serialize with the real description package. is-default is a v2-only field,
	// so its presence in the wire format proves the charm metadata serialized at
	// schema version 2.
	data, err := description.Serialize(model)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(strings.Contains(string(data), "is-default: true"), tc.IsTrue,
		tc.Commentf("serialized model is missing the is-default flag:\n%s", data))

	// Deserialize with the real description package.
	imported, err := description.Deserialize(data)
	c.Assert(err, tc.ErrorIsNil)
	apps := imported.Applications()
	c.Assert(apps, tc.HasLen, 1)
	meta := apps[0].CharmMetadata()
	c.Assert(meta, tc.NotNil)

	// Run Juju's real import conversion over the round-tripped relations: the
	// default flag must survive on certs-main and stay false on certs-extra.
	relations, err := importRelations(meta.Requires())
	c.Assert(err, tc.ErrorIsNil)
	c.Check(relations["certs-main"].IsDefault, tc.IsTrue)
	c.Check(relations["certs-extra"].IsDefault, tc.IsFalse)
}
