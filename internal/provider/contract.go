// Package provider is nerv-task's provider contract: the remote
// operations a task provider implements, the canonical stage keys, the
// pipeline events with the operations each needs, and the shape of a
// provider's provider.yaml manifest (MCP tools, config keys, stages,
// operations, events). It is a trimmed, standalone port of nerv's
// internal/taskprovider plus the provider-manifest slice of nerv's
// internal/extension: no exec runtime, no slots, no requires and no
// suggestion order, because nerv-task ships exactly one provider slot
// (task providers, all agent-runtime) and installs independently of nerv.
//
// The prose side of this contract lives in
// plugin/task/skills/task/SKILL.md.
package provider

// ContractVersion is the only contract_version this build understands.
const ContractVersion = 1

// NotImplemented declares an operation the provider does not support. An
// event that needs it makes the manifest invalid.
const NotImplemented = "not_implemented"

// ManifestFile is the manifest's name at the root of a provider's folder.
const ManifestFile = "provider.yaml"

// RuntimeAgent is the only runtime nerv-task's providers run: an LLM
// agent follows the provider's procedure files. There is no exec
// runtime in this trimmed contract.
const RuntimeAgent = "agent"

// Operations are the remote primitives a provider implements. Timers,
// timelog consolidation and the composite operations (block, cancel,
// close, done, stopAll) belong to the task plugin itself and are built
// from these by the task:dispatcher agent and the /task:* commands.
var Operations = []string{
	"getTask",
	"assign",
	"moveStage",
	"logTime",
	"updateTimelog",
	"deleteTimelog",
	"listTimelogs",
	"createTask",
	"createSubtask",
	"comment",
	"complete",
	"setPriority",
	"listTasks",
}

// StageKeys are the canonical stages events and commands speak; a
// provider maps each one to its own stage name through its stages block
// and config.
var StageKeys = []string{"pending", "analysis", "inDev", "testing", "implemented", "blocked", "canceled"}

// Statuses are the outcomes the task:dispatcher agent reports per
// provider for one event. needs_input carries a question the orchestrator
// relays verbatim before dispatching again.
const (
	StatusOK         = "ok"
	StatusSkipped    = "skipped"
	StatusBlocked    = "blocked"
	StatusFailed     = "failed"
	StatusNeedsInput = "needs_input"
)

// Results lists every status in Statuses, in the fixed order every
// dispatcher envelope and every piece of documentation must use.
var Results = []string{StatusOK, StatusSkipped, StatusBlocked, StatusFailed, StatusNeedsInput}

// Events maps each pipeline event nerv (or any other orchestrator
// following nerv-events.yaml) emits to the operations a provider must
// implement to handle it. run.closed never completes the task: that
// stays an explicit user action (/task:close).
var Events = map[string][]string{
	"run.started":          {"getTask", "assign", "moveStage"},
	"verification.started": {"moveStage"},
	"audit.ranked":         {"comment"},
	"issue.deferred":       {"createTask"},
	"wave.started":         {"createSubtask"},
	"run.closed":           {"moveStage", "logTime"},
	"run.halted":           {"comment", "moveStage", "logTime"},
}

// ConfigScopes are the two places a provider's config key can live:
// "repo" in the repository's task.yaml, "user" in the global task.yaml.
var ConfigScopes = []string{"user", "repo"}

// ConfigTypes are the value shapes a provider's config key can declare.
var ConfigTypes = []string{"string", "int", "bool", "map", "list"}
