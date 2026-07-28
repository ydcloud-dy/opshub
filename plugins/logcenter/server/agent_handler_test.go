package server

import "testing"

func TestBuildAgentConfigETagIsStableForEquivalentConfig(t *testing.T) {
	result := agentLogConfigResponse{
		ConfigVersion: 12, ReloadGeneration: 3, PollInterval: 30,
		Assignments: []agentDesiredAssignment{{PolicyID: 7, PolicyVersion: 2, DesiredState: "active"}},
	}
	result.LogCollection.Enabled = true
	result.LogCollection.GatewayURL = "https://logs.example.com"
	result.LogCollection.GatewayToken = "secret-token"
	first := buildAgentConfigETag(result)
	second := buildAgentConfigETag(result)
	if first != second {
		t.Fatalf("equivalent config produced different ETags: %s != %s", first, second)
	}
	result.Assignments[0].PolicyVersion++
	if changed := buildAgentConfigETag(result); changed == first {
		t.Fatal("policy version change must invalidate ETag")
	}
}

func TestBuildAgentConfigETagTracksGatewayRotation(t *testing.T) {
	result := agentLogConfigResponse{ConfigVersion: 1, PollInterval: 30}
	result.LogCollection.GatewayURL = "https://logs.example.com"
	result.LogCollection.GatewayToken = "old-token"
	first := buildAgentConfigETag(result)
	result.LogCollection.GatewayToken = "new-token"
	if changed := buildAgentConfigETag(result); changed == first {
		t.Fatal("gateway token rotation must invalidate ETag")
	}
}
