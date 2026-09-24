---
name: jira-updates
description: Rules for ANY interaction with Jira (song88.atlassian.net) via the Atlassian MCP tools. MUST be followed whenever creating, editing, commenting on, linking, or transitioning a Jira issue, or when a task might touch Jira as a side effect.
---

# Jira Updates - Approval Required

Jira is a shared, team-visible system. Every write is immediately visible to the whole team under the user's own name. These rules are absolute.

## Rules

1. **Never write to Jira unless the user explicitly asks for that write in this conversation.**
   - "Explicitly asks" means the user requests the Jira change itself (e.g. "update the ticket", "add a comment to GM-123").
   - Providing a ticket number, connecting the MCP server, or discussing a ticket is NOT permission to write to it.
   - Read operations (get issue, search, list transitions) are always fine.

2. **Before any write: show the exact content first, then wait for approval.**
   - Present verbatim what will be saved (summary, description, comment body, field values) in the chat.
   - Ask for approval and stop the turn. Only perform the write after the user approves in a later message.
   - If the user edits the proposal, show the revised version again unless they say to just send it.

3. **NEVER update issue status.** Do not call transition tools, set resolutions, or otherwise move an issue through its workflow, even if asked implicitly ("mark it done"). If the user explicitly requests a transition, remind them of this rule and ask them to do it themselves in Jira (or to explicitly override this rule for that one action).

4. **Content style:** no em dash character ("—") in anything written to Jira or Confluence, use "-" or "," instead; strictly no emojis.

5. Scope: applies to all Atlassian write tools (`createJiraIssue`, `editJiraIssue`, `addOrEditJiraIssueComment`, `transitionJiraIssue`, `createConfluenceContent`, `updateConfluenceContent`, and any `executeWrite`/`executeDestructive` operation such as worklogs or issue links).

## Suggested flow

1. Read the issue (allowed without asking).
2. Draft the change; show it verbatim: "This is exactly what I would save to GM-XXX: ...".
3. Ask: "Shall I save this?" and end the turn.
4. On approval, perform ONLY the approved write, then report the result (issue key, comment id).
