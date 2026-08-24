---
name: research
description: "Answers a question from primary sources and cites each claim. Use when a decision waits on a question that only primary sources can answer: library or API behavior to verify, a technology choice to examine, or claims that need citations, not recall."
argument-hint: "the question, or an issue ref"
---

# Research

A research session answers a question that blocks a decision. The session operates with or without a tracker. If `docs/TRACKER.md` exists, intake and the report resolve through the tracker. If not, the findings go in `docs/research/`.

## Steps

1. **Pin the question.** If the argument is an issue reference, fetch the issue and claim it before you start work. The issue body is the brief. Divide the question into the specific sub-questions that block the decision — two to four. More than four means the question is too broad; narrow it to what the decision actually waits on. This step is complete when the sub-questions are listed.

2. **Investigate with parallel sub-agents.** If the question is a single verifiable fact, skip the sub-agents: answer it yourself from one or two primary sources and go to step 3. Otherwise, start one sub-agent for each sub-question. Start all of them in one message. Then the bulk of the source material stays out of this session's context. Each brief contains the sub-question and the full source rule, because the sub-agent cannot see this skill. The source rule:
   - Work from primary sources: official documentation, source code, specifications, first-party APIs, and the repository itself.
   - Never use a secondary description of a primary source.
   - Trace each claim back to the source that owns it.
   - Stop at the first authoritative source that answers the sub-question. Do not corroborate an already-cited claim with further sources, and do not survey alternatives the question did not ask about.
   - Return a compact list of claims, each with its source and the version or date for which the claim was true. Quote nothing beyond the sentence that supports the claim; never return the source material itself.

   Compare the answers when they come back. A contradiction between sources is itself a finding. Record both sides. If a sub-agent left a claim unverified, verify it yourself only when the verdict depends on that claim; otherwise it goes under Unresolved. If the host has no sub-agents, the rule does not change, only the mechanics: work the sub-questions yourself, one at a time. Carry forward only claims and citations, never the source material. This step is complete when each sub-question has an answer with a primary source, or a recorded reason why no answer exists.

3. **Write the report** with these sections:
   - **Question** — the brief.
   - **Answer** — the verdict, first, in a few sentences.
   - **Findings** — each claim, with its source adjacent to it.
   - **Unresolved** — the claims that you could not verify, and why.
   - **Sources** — each source that you examined.

   This step is complete when each sub-question from step 1 appears under Findings or Unresolved.

4. **Record the report.** Select the correct path:
   - The session started from an issue → write the full report in a note on the issue. Then close the issue. Exception: the body says that closure waits for user review. Then end the note with what the user must do: close to approve, or note the requested changes and remove the label. Apply `needs-review`, and release the claim. Do not close.
   - No issue, but the tracker exists → publish an issue. The title is the question. The label is `research`. The body is the report. Then close the issue. A closed `research` issue is the record that later sessions search for.
   - No tracker → write the report to `docs/research/{{slug}}.md`. If the slug is taken, select a fresh slug. Never overwrite a file.
