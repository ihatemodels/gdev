---
description: Generate a commit message from current git changes
allowed-tools: Bash(git diff:*), Bash(git log:*), Bash(git status:*)
---

## Context

- Current git diff (staged and unstaged changes): !`git diff HEAD`
- Current git status: !`git status --short`
- Recent commits for style reference: !`git log --oneline -5`

## Your task

Generate a commit message for the changes shown above.

CRITICAL RULES:
1. Output ONLY the raw commit message text
2. NO preamble like "Here is..." or "Based on..."
3. NO markdown formatting or code blocks
4. NO explanations before or after
5. First line is the subject, then blank line, then body

Format:
<type>: <subject max 50 chars>

<body explaining why, wrapped at 72 chars>

Types: feat, fix, refactor, docs, style, test, chore

Your response must start directly with the type (feat/fix/etc). Nothing else.
