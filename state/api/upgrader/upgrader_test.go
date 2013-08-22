// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package upgrader_test

import (
	stdtesting "testing"

	gc "launchpad.net/gocheck"

	"launchpad.net/juju-core/errors"
	"launchpad.net/juju-core/juju/testing"
	"launchpad.net/juju-core/state"
	"launchpad.net/juju-core/state/api"
	"launchpad.net/juju-core/state/api/params"
	"launchpad.net/juju-core/state/api/upgrader"
	statetesting "launchpad.net/juju-core/state/testing"
	coretesting "launchpad.net/juju-core/testing"
	"launchpad.net/juju-core/testing/checkers"
	"launchpad.net/juju-core/tools"
	"launchpad.net/juju-core/version"
)

func TestAll(t *stdtesting.T) {
	coretesting.MgoTestPackage(t)
}

type upgraderSuite struct {
	testing.JujuConnSuite

	stateAPI *api.State

	// These are raw State objects. Use them for setup and assertions, but
	// should never be touched by the API calls themselves
	rawMachine *state.Machine
	rawCharm   *state.Charm
	rawService *state.Service
	rawUnit    *state.Unit

	st *upgrader.State
}

var _ = gc.Suite(&upgraderSuite{})

func (s *upgraderSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)

	// Create a machine to work with
	var err error
	s.rawMachine, err = s.State.AddMachine("series", state.JobHostUnits)
	c.Assert(err, gc.IsNil)
	err = s.rawMachine.SetProvisioned("foo", "fake_nonce", nil)
	c.Assert(err, gc.IsNil)
	err = s.rawMachine.SetPassword("test-password")
	c.Assert(err, gc.IsNil)

	// Login as the machine agent of the created machine.
	s.stateAPI = s.OpenAPIAsMachine(c, s.rawMachine.Tag(), "test-password", "fake_nonce")
	c.Assert(s.stateAPI, gc.NotNil)

	// Create the upgrader facade.
	s.st = s.stateAPI.Upgrader()
	c.Assert(s.st, gc.NotNil)
}

func (s *upgraderSuite) TearDownTest(c *gc.C) {
	if s.stateAPI != nil {
		err := s.stateAPI.Close()
		c.Check(err, gc.IsNil)
	}
	s.JujuConnSuite.TearDownTest(c)
}

// Note: This is really meant as a unit-test, this isn't a test that should
//       need all of the setup we have for this test suite
func (s *upgraderSuite) TestNew(c *gc.C) {
	upgrader := upgrader.NewState(s.stateAPI)
	c.Assert(upgrader, gc.NotNil)
}

func (s *upgraderSuite) TestSetToolsWrongMachine(c *gc.C) {
	err := s.st.SetTools("42", &tools.Tools{
		Version: version.Current,
	})
	c.Assert(err, gc.ErrorMatches, "permission denied")
	c.Assert(params.ErrCode(err), gc.Equals, params.CodeUnauthorized)
}

func (s *upgraderSuite) TestSetTools(c *gc.C) {
	cur := version.Current
	agentTools, err := s.rawMachine.AgentTools()
	c.Assert(err, checkers.Satisfies, errors.IsNotFoundError)
	c.Assert(agentTools, gc.IsNil)
	err = s.st.SetTools(s.rawMachine.Tag(), &tools.Tools{Version: cur})
	c.Assert(err, gc.IsNil)
	s.rawMachine.Refresh()
	agentTools, err = s.rawMachine.AgentTools()
	c.Assert(err, gc.IsNil)
	c.Check(agentTools.Version, gc.Equals, cur)
}

func (s *upgraderSuite) TestToolsWrongMachine(c *gc.C) {
	tools, err := s.st.Tools("42")
	c.Assert(err, gc.ErrorMatches, "permission denied")
	c.Assert(params.ErrCode(err), gc.Equals, params.CodeUnauthorized)
	c.Assert(tools, gc.IsNil)
}

func (s *upgraderSuite) TestTools(c *gc.C) {
	cur := version.Current
	curTools := &tools.Tools{Version: cur, URL: ""}
	curTools.Version.Minor++
	s.rawMachine.SetAgentTools(curTools)
	// Upgrader.Tools returns the *desired* set of tools, not the currently
	// running set. We want to be upgraded to cur.Version
	stateTools, err := s.st.Tools(s.rawMachine.Tag())
	c.Assert(err, gc.IsNil)
	c.Assert(stateTools.Version, gc.Equals, cur)
	c.Assert(stateTools.URL, gc.Not(gc.Equals), "")
}

func (s *upgraderSuite) TestWatchAPIVersion(c *gc.C) {
	w, err := s.st.WatchAPIVersion(s.rawMachine.Tag())
	c.Assert(err, gc.IsNil)
	defer statetesting.AssertStop(c, w)
	wc := statetesting.NewNotifyWatcherC(c, s.BackingState, w)
	// Initial event
	wc.AssertOneChange()
	vers := version.MustParse("10.20.34")
	err = statetesting.SetAgentVersion(s.BackingState, vers)
	c.Assert(err, gc.IsNil)
	// One change noticing the new version
	wc.AssertOneChange()
	// Setting the version to the same value doesn't trigger a change
	err = statetesting.SetAgentVersion(s.BackingState, vers)
	c.Assert(err, gc.IsNil)
	wc.AssertNoChange()
	vers = version.MustParse("10.20.35")
	err = statetesting.SetAgentVersion(s.BackingState, vers)
	c.Assert(err, gc.IsNil)
	wc.AssertOneChange()
	statetesting.AssertStop(c, w)
	wc.AssertClosed()
}
