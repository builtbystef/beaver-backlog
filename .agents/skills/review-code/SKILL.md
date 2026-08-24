---
name: review-code
description: "Reviews a set of changes on two independent axes: the spec or issue that motivated them, and the project's coding standards. Use when you review a diff, a branch, a PR, or uncommitted work."
argument-hint: "a ref, a range, or paths; no argument means uncommitted changes"
---

# Review Code

Review the changes on two independent axes. A change can obey each standard and build the wrong thing. A change can also do exactly what was asked and break each convention. Separate reports prevent one axis from a mask over the other. Each axis runs as its own sub-agent, so that their file reads do not pollute each other's context. This session pins the scope, writes the briefs, and collects the results.

The skill operates with or without a tracker. If `docs/TRACKER.md` exists, the spec search and the filed findings resolve through the tracker. If not, skip those parts — the report is then the deliverable — and suggest invoking the `set-up-for-agents` skill one time, at the end.

## Steps

1. **Pin the scope.** From the argument:
   - a ref (`git rev-parse` resolves it) → `git diff {{ref}}...HEAD` (three dots, so that the comparison is against the merge base), plus `git log {{ref}}..HEAD --oneline`
   - an explicit range, or a set of paths → as given
   - nothing → uncommitted work: `git diff HEAD`, plus untracked files

   Make sure that the diff is not empty. Tell the user what you review. This step is complete when the diff and the commit list are captured.

2. **Find the intent source.** Search in this order: issue references in the commit messages or the branch name (fetch the issue, and the spec issue that it belongs to — its parent); a ref or path that the user gave; a spec issue (label `spec`) in the tracker that matches the work. If you find nothing, ask the user. If the user says that no source exists, the Spec axis reports "no spec available". This step is complete when the source is pinned, or recorded as absent.

3. **Start both axes as parallel sub-agents.** One message, two sub-agents. Each brief contains the diff command and the commit list from step 1, plus its own instructions. A sub-agent can read the repository, but it cannot see this skill or this session's context. For this reason the brief must be complete.

   - **Spec axis** — include the spec or issue contents from step 2. The brief: examine the diff for (a) requirements that are missing or partial; (b) behavior that nobody asked for — scope creep; (c) requirements that look implemented, but are wrong. Quote the spec or issue line adjacent to each finding, with file:line. If no spec is available, do not start this sub-agent. Note the absence in the report.
   - **Standards axis** — include the paths of `docs/CODING_STANDARDS.md` and `docs/adr/` (one missing → say so; both missing → note the absence in the report). The brief: examine each hunk against the documented standards, and against each ADR that covers the diff's area — a change that contradicts an ADR is a hard violation. Skip what a linter or formatter already enforces. Cite the standard or the ADR for each violation, with file:line.

   If the host has no sub-agents, do the two briefs yourself, one after the other, independently. Complete one axis's findings, and write them down, before you read for the other axis. Then neither axis steers the other. This step is complete when both axes reported.

4. **Report.** Two sections: `## Spec` and `## Standards`. Present each sub-agent's findings as returned, with light cleanup at most, with file:line and severity (hard violation, or judgement call). Never merge the axes. Never rerank across them. That is the mask that the separation prevents. End with a one-line summary for each axis. Offer to file the findings that the user accepts: publish one issue for each accepted finding, with the label `review`, and the evidence with file:line as the body. This step is complete when the user has the report, and each accepted finding is filed.
