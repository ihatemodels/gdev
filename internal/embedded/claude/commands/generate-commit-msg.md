---
description: Generate a commit message from current git changes
---

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

* <first change/feature>
* <second change/feature>
* <etc>

Body rules:
- Break down by functionality - one bullet per logical change
- Each line starts with * (bullet point)
- Keep each bullet under 72 chars
- Focus on WHY not WHAT

Types: feat, fix, refactor, docs, style, test, chore

Your response must start directly with the type (feat/fix/etc). Nothing else.
