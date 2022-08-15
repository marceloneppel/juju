// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package jujuc_test

import (
	"github.com/juju/cmd/v3"
	"github.com/juju/cmd/v3/cmdtesting"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/worker/uniter/runner/jujuc"
)

type SecretRevokeSuite struct {
	ContextSuite
}

var _ = gc.Suite(&SecretRevokeSuite{})

func (s *SecretRevokeSuite) TestRevokeSecretInvalidArgs(c *gc.C) {
	hctx, _ := s.ContextSuite.NewHookContext()

	for _, t := range []struct {
		args []string
		err  string
	}{
		{
			args: []string{},
			err:  "ERROR missing secret URI",
		}, {
			args: []string{"secret:9m4e2mr0ui3e8a215n4g"},
			err:  `ERROR missing relation or application or unit`,
		}, {
			args: []string{"secret:9m4e2mr0ui3e8a215n4g", "--app", "0/foo"},
			err:  `ERROR application "0/foo" not valid`,
		}, {
			args: []string{"secret:9m4e2mr0ui3e8a215n4g", "--unit", "foo"},
			err:  `ERROR unit "foo" not valid`,
		}, {
			args: []string{"secret:9m4e2mr0ui3e8a215n4g", "--relation", "-666"},
			err:  `ERROR invalid value "-666" for option --relation: relation not found`,
		},
	} {
		com, err := jujuc.NewCommand(hctx, "secret-revoke")
		c.Assert(err, jc.ErrorIsNil)
		ctx := cmdtesting.Context(c)
		code := cmd.Main(jujuc.NewJujucCommandWrappedForTest(com), ctx, t.args)

		c.Assert(code, gc.Equals, 2)
		c.Assert(bufferString(ctx.Stderr), gc.Equals, t.err+"\n")
	}
}

func (s *SecretRevokeSuite) TestRevokeSecretForApp(c *gc.C) {
	hctx, _ := s.ContextSuite.NewHookContext()

	com, err := jujuc.NewCommand(hctx, "secret-revoke")
	c.Assert(err, jc.ErrorIsNil)
	ctx := cmdtesting.Context(c)
	code := cmd.Main(jujuc.NewJujucCommandWrappedForTest(com), ctx, []string{
		"secret:9m4e2mr0ui3e8a215n4g", "--app", "foo",
	})

	c.Assert(code, gc.Equals, 0)
	args := &jujuc.SecretGrantRevokeArgs{
		ApplicationName: ptr("foo"),
	}
	s.Stub.CheckCallNames(c, "HookRelation", "RevokeSecret")
	s.Stub.CheckCall(c, 1, "RevokeSecret", "secret:9m4e2mr0ui3e8a215n4g", args)
}

func (s *SecretRevokeSuite) TestRevokeSecretForRelation(c *gc.C) {
	hctx, info := s.ContextSuite.NewHookContext()
	info.SetNewRelation(1, "db", s.Stub)
	info.SetAsRelationHook(1, "mediawiki/0")

	com, err := jujuc.NewCommand(hctx, "secret-revoke")
	c.Assert(err, jc.ErrorIsNil)
	ctx := cmdtesting.Context(c)
	code := cmd.Main(jujuc.NewJujucCommandWrappedForTest(com), ctx, []string{
		"secret:9m4e2mr0ui3e8a215n4g", "--relation", "1",
	})

	c.Assert(code, gc.Equals, 0)
	args := &jujuc.SecretGrantRevokeArgs{
		ApplicationName: ptr("mediawiki"),
		RelationKey:     ptr("wordpress:db mediawiki:db"),
	}
	s.Stub.CheckCallNames(c, "HookRelation", "Id", "FakeId", "Relation", "Relation", "RelationTag", "RemoteApplicationName", "RemoteUnitName", "RevokeSecret")
	s.Stub.CheckCall(c, 8, "RevokeSecret", "secret:9m4e2mr0ui3e8a215n4g", args)
}
