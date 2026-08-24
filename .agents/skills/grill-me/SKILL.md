---
name: grill-me
description: Interviews the user in rounds, until a plan, decision, or design is fully understood. Use when the user wants a stress test of their plan ("grill me"), or when another skill needs open decisions settled by interview.
argument-hint: "the topic to grill; no argument means the session's current topic"
disallowed-tools: AskUserQuestion
---

# Grill Me

Interview the user about each aspect of the topic, until you have a shared understanding. Find facts yourself. The user makes the decisions. Never ask the user a question that the environment can answer. Never answer a decision for the user. An interviewer who answers their own questions has failed.

The decisions make a tree. Each answer opens the decisions that hang below it. Work the tree in **rounds**.

## Rounds

In each round, ask each question whose prerequisites are settled — the questions that you can ask now, without a guess at answers that you did not hear yet. Number the questions. Attach your recommended answer to each question. If a question depends on another question that is open in this round, keep it for a later round. Then wait. The user's answers change the tree, unblock new questions, and the next round asks those. A round is often a single question. That is normal when everything hangs on one choice.

Ask the questions as plain text in your reply. Never use a structured question tool or a multiple-choice tool that the host offers (AskUserQuestion, request_user_input, or similar). Those UIs force one answer in one step. This is an interview. The user can answer some questions and not others, push back, answer with a question, or take a question in a direction that you did not offer. Engage with that in conversation. Carry the unanswered questions into the next round.

When a question waits on a fact from the environment (filesystem, repository, tools), send a sub-agent to find the fact. In a host without sub-agents, find it directly. Do not block the round on the fact. Only the questions below that fact wait. Ask the remainder now.

## During the interview

- **Make unclear language sharp.** When a term is vague or has many meanings, propose a precise term immediately. Example: "You say 'account' — do you mean the Customer or the User?"
- **Test against the glossary.** When the user's words conflict with `docs/GLOSSARY.md`, tell the user immediately. Settle which is correct.
- **Record terms when they settle.** A settled term goes into `docs/GLOSSARY.md` immediately (follow the format rules at the top of the file). Do not collect terms for later. The glossary stays a glossary: no implementation details.
- **Test with concrete scenarios.** Invent edge cases that force precision about the boundaries between concepts. Example: "A user cancels during payment — what occurs?"
- **Compare with the code.** When the user tells how something operates, examine whether the code agrees. This is a quick search. Show the contradictions. Example: "The code cancels full Orders, but you said that partial cancellation is possible — which is correct?"
- **Offer an ADR rarely.** Offer one only when a decision is hard to reverse, and surprising without context, _and_ a real trade-off. Path: `docs/adr/NNNN-{{slug}}.md`. A title plus 1–3 sentences (context, decision, reason) is a complete ADR.

In a repository without `docs/GLOSSARY.md` or `docs/adr/`, interview without the records. Suggest invoking the `set-up-for-agents` skill one time, at the end.

## Done

The interview is done when no askable question remains: you visited each branch of the tree, nothing stayed a silent assumption, and no decision remains that would make an implementer stop and ask. Do not act on the outcome until the user confirms that you have a shared understanding.
