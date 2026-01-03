package bridge.policy

# Default decisions
default allowed = true
default requires_approval = false

# Require approval for destructive operations
requires_approval if {
	input.capabilities[_] == "file-write"
}

requires_approval if {
	input.capabilities[_] == "shell-exec"
}

requires_approval if {
	input.capabilities[_] == "git-push"
}

# Require approval for code generation affecting production paths
requires_approval if {
	input.capabilities[_] == "code-generation"
	contains(input.context.path, "src/")
}

# Block access to sensitive files
allowed = false if {
	path := input.context.path
	sensitive_patterns[pattern]
	contains(path, pattern)
}

sensitive_patterns := {
	".env",
	".env.local",
	".env.production",
	"secrets/",
	".ssh/",
	"credentials",
	"private_key",
	".npmrc",
	".pypirc",
}

# Block operations on protected branches
allowed = false if {
	input.context.branch == "main"
	input.capabilities[_] == "git-push"
	not input.context.approved
}

allowed = false if {
	input.context.branch == "master"
	input.capabilities[_] == "git-push"
	not input.context.approved
}

# Violations
violation contains msg if {
	not allowed
	path := input.context.path
	sensitive_patterns[pattern]
	contains(path, pattern)
	msg := sprintf("Access to sensitive path blocked: %s (matches pattern: %s)", [path, pattern])
}

violation contains msg if {
	input.context.branch == "main"
	input.capabilities[_] == "git-push"
	not input.context.approved
	msg := "Direct push to main branch requires approval"
}

violation contains msg if {
	input.context.max_tokens > 100000
	msg := sprintf("Token limit exceeded: %d > 100000", [input.context.max_tokens])
}

# Warnings
warning contains msg if {
	input.capabilities[_] == "file-write"
	msg := "File write operation detected - changes will be audited"
}

warning contains msg if {
	input.context.tokens_used > 50000
	msg := sprintf("High token usage: %d tokens", [input.context.tokens_used])
}
