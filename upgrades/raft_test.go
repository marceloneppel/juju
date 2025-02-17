// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package upgrades_test

import (
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/juju/names/v4"
	"github.com/juju/replicaset/v2"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/controller"
	"github.com/juju/juju/core/lease"
	"github.com/juju/juju/core/raftlease"
	raftleasestore "github.com/juju/juju/state/raftlease"
	coretesting "github.com/juju/juju/testing"
	"github.com/juju/juju/upgrades"
	raftworker "github.com/juju/juju/worker/raft"
	"github.com/juju/juju/worker/raft/rafttest"
)

type raftSuite struct {
	coretesting.BaseSuite
}

var _ = gc.Suite(&raftSuite{})

func (s *raftSuite) TestBootstrapRaft(c *gc.C) {
	dataDir := c.MkDir()
	context := makeContext(dataDir)
	err := upgrades.BootstrapRaft(context)
	c.Assert(err, jc.ErrorIsNil)

	// Now make the raft node and check that the configuration is as
	// we expect.
	checkRaftConfiguration(c, dataDir)

	// Check the upgrade is idempotent.
	err = upgrades.BootstrapRaft(context)
	c.Assert(err, jc.ErrorIsNil)

	checkRaftConfiguration(c, dataDir)
}

func (s *raftSuite) TestBootstrapRaftWithEmptyDir(c *gc.C) {
	dataDir := c.MkDir()
	raftDir := filepath.Join(dataDir, "raft")
	c.Assert(os.Mkdir(raftDir, 0777), jc.ErrorIsNil)

	context := makeContext(dataDir)
	err := upgrades.BootstrapRaft(context)
	c.Assert(err, jc.ErrorIsNil)

	// Now make the raft node and check that the configuration is as
	// we expect.
	checkRaftConfiguration(c, dataDir)
}

func (s *raftSuite) TestBootStrapRaftWithEmptyLog(c *gc.C) {
	dataDir := c.MkDir()
	raftDir := filepath.Join(dataDir, "raft")
	c.Assert(os.Mkdir(raftDir, 0777), jc.ErrorIsNil)

	logStore, err := raftworker.NewLogStore(raftDir, raftworker.SyncAfterWrite)
	c.Assert(err, jc.ErrorIsNil)
	// Have to close it here or the open in the code hangs!
	logStore.Close()

	context := makeContext(dataDir)
	err = upgrades.BootstrapRaft(context)
	c.Assert(err, jc.ErrorIsNil)

	// Now make the raft node and check that the configuration is as
	// we expect.
	checkRaftConfiguration(c, dataDir)
}

func makeContext(dataDir string) *mockContext {
	votes := 1
	noVotes := 0
	return &mockContext{
		agentConfig: &mockAgentConfig{
			tag:     names.NewMachineTag("23"),
			dataDir: dataDir,
		},
		state: &mockState{
			members: []replicaset.Member{{
				Address: "somewhere.else:37012",
				Tags:    map[string]string{"juju-machine-id": "42"},
			}, {
				Address: "nowhere.else:37012",
				Tags:    map[string]string{"juju-machine-id": "23"},
				Votes:   &votes,
			}, {
				Address: "everywhere.else:37012",
				Tags:    map[string]string{"juju-machine-id": "7"},
				Votes:   &noVotes,
			}},
			info: controller.StateServingInfo{APIPort: 1234},
		},
	}
}

func withRaft(c *gc.C, dataDir string, fsm raft.FSM, checkFunc func(*raft.Raft)) {
	// Capture logging to include in test output.
	output := captureWriter{c}
	config := raft.DefaultConfig()
	config.LocalID = "23"
	config.Logger = hclog.New(&hclog.LoggerOptions{
		Output: output,
		Level:  hclog.DefaultLevel,
	})
	c.Assert(raft.ValidateConfig(config), jc.ErrorIsNil)

	raftDir := filepath.Join(dataDir, "raft")
	logStore, err := raftboltdb.New(raftboltdb.Options{
		Path: filepath.Join(raftDir, "logs"),
	})
	c.Assert(err, jc.ErrorIsNil)
	defer logStore.Close()

	snapshotStore, err := raft.NewFileSnapshotStore(raftDir, 1, output)
	c.Assert(err, jc.ErrorIsNil)
	_, transport := raft.NewInmemTransport(raft.ServerAddress("nowhere.else"))

	r, err := raft.NewRaft(
		config,
		fsm,
		logStore,
		logStore,
		snapshotStore,
		transport,
	)
	c.Assert(err, jc.ErrorIsNil)
	defer func() {
		c.Assert(r.Shutdown().Error(), jc.ErrorIsNil)
	}()
	checkFunc(r)
}

func checkRaftConfiguration(c *gc.C, dataDir string) {
	withRaft(c, dataDir, &raftworker.SimpleFSM{},
		func(r *raft.Raft) {
			rafttest.CheckConfiguration(c, r, []raft.Server{{
				ID:       "42",
				Address:  "somewhere.else:1234",
				Suffrage: raft.Voter,
			}, {
				ID:       "23",
				Address:  "nowhere.else:1234",
				Suffrage: raft.Voter,
			}, {
				ID:       "7",
				Address:  "everywhere.else:1234",
				Suffrage: raft.Nonvoter,
			}})
		},
	)
}

type mockState struct {
	upgrades.StateBackend
	stub    testing.Stub
	members []replicaset.Member
	info    controller.StateServingInfo
	config  controller.Config
	leases  map[lease.Key]lease.Info
	target  *mockTarget
}

func (s *mockState) ReplicaSetMembers() ([]replicaset.Member, error) {
	return s.members, s.stub.NextErr()
}

func (s *mockState) StateServingInfo() (controller.StateServingInfo, error) {
	return s.info, s.stub.NextErr()
}

func (s *mockState) ControllerConfig() (controller.Config, error) {
	return s.config, s.stub.NextErr()
}

func (s *mockState) LegacyLeases(localTime time.Time) (map[lease.Key]lease.Info, error) {
	s.stub.AddCall("LegacyLeases", localTime)
	return s.leases, s.stub.NextErr()
}

func (s *mockState) DropLeasesCollection() error {
	s.stub.AddCall("DropLeasesCollection")
	return s.stub.NextErr()
}

func (s *mockState) LeaseNotifyTarget(logger raftleasestore.Logger) (raftlease.NotifyTarget, error) {
	s.stub.AddCall("LeaseNotifyTarget", logger)
	return s.target, s.stub.NextErr()
}

type mockTarget struct {
	raftlease.NotifyTarget
	stub testing.Stub
}

func (t *mockTarget) Claimed(key lease.Key, holder string) error {
	t.stub.AddCall("Claimed", key, holder)
	return t.stub.NextErr()
}

func (t *mockTarget) assertClaimed(c *gc.C, claims map[lease.Key]string) {
	c.Assert(t.stub.Calls(), gc.HasLen, len(claims))
	for _, call := range t.stub.Calls() {
		c.Assert(call.FuncName, gc.Equals, "Claimed")
		c.Assert(call.Args, gc.HasLen, 2)
		key, ok := call.Args[0].(lease.Key)
		c.Assert(ok, gc.Equals, true)
		holder, ok := call.Args[1].(string)
		c.Assert(ok, gc.Equals, true)
		expectedHolder, found := claims[key]
		c.Assert(found, gc.Equals, true)
		c.Assert(holder, gc.Equals, expectedHolder)
		delete(claims, key)
	}
}

type captureWriter struct {
	c *gc.C
}

func (w captureWriter) Write(p []byte) (int, error) {
	w.c.Logf("%s", p[:len(p)-1]) // omit trailing newline
	return len(p), nil
}
