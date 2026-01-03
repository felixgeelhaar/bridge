package bridge.policy

# Default decisions
default allowed = true
default requires_approval = false

# Require approval for destructive operations
requires_approval {
    input.capabilities[_] == "file-write"
}

requires_approval {
    input.capabilities[_] == "shell-exec"
}

requires_approval {
    input.capabilities[_] == "git-push"
}

# Require approval for code generation affecting production paths
requires_approval {
    input.capabilities[_] == "code-generation"
    contains(input.context.path, "src/")
}

# Block access to sensitive files
allowed = false {
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
allowed = false {
    input.context.branch == "main"
    input.capabilities[_] == "git-push"
    not input.context.approved
}

allowed = false {
    input.context.branch == "master"
    input.capabilities[_] == "git-push"
    not input.context.approved
}

# Violations
violation[msg] {
    not allowed
    path := input.context.path
    sensitive_patterns[pattern]
    contains(path, pattern)
    msg := sprintf("Access to sensitive path blocked: %s (matches pattern: %s)", [path, pattern])
}

violation[msg] {
    input.context.branch == "main"
    input.capabilities[_] == "git-push"
    not input.context.approved
    msg := "Direct push to main branch requires approval"
}

violation[msg] {
    input.context.max_tokens > 100000
    msg := sprintf("Token limit exceeded: %d > 100000", [input.context.max_tokens])
}

# Warnings
warning[msg] {
    input.capabilities[_] == "file-write"
    msg := "File write operation detected - changes will be audited"
}

warning[msg] {
    input.context.tokens_used > 50000
    msg := sprintf("High token usage: %d tokens", [input.context.tokens_used])
}
